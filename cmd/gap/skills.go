package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// manifestEntry is one source in global/skills.json. Source "." is the checkout gap was built from.
type manifestEntry struct {
	Source string   `json:"source"`
	Skills []string `json:"skills"`
}

func readManifest(repoRoot string) ([]manifestEntry, error) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "global", "skills.json"))
	if err != nil {
		return nil, err
	}
	var entries []manifestEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, fmt.Errorf("global/skills.json: %w", err)
	}
	return entries, nil
}

func installSkills(repoRoot, home string, entries []manifestEntry) bool {
	var npx string
	var err error
	ok := true
	for _, e := range entries {
		var missing, migrate []string
		var migrateFrom []lockEntry
		for _, name := range e.Skills {
			exists := false
			for _, dir := range skillDirs(home, name) {
				if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
					exists = true
				}
			}
			if !exists {
				missing = append(missing, name)
				continue
			}
			if locked, foreign := foreignLock(repoRoot, home, e.Source, name); foreign {
				migrate = append(migrate, name)
				migrateFrom = append(migrateFrom, locked)
			}
		}
		if len(missing) == 0 && len(migrate) == 0 {
			continue
		}
		if npx == "" {
			if npx, err = exec.LookPath("npx"); err != nil {
				fmt.Fprintln(os.Stderr, "Skipping skills: npx not found; install Node to let gap install skills")
				return false
			}
		}
		if len(missing) > 0 && !skillsAdd(npx, repoRoot, home, e.Source, missing) {
			ok = false
		} else {
			for _, name := range missing {
				fmt.Println(filepath.Join(home, ".agents", "skills", name))
			}
		}
		for i, name := range migrate {
			replaced, err := replaceSkill(npx, repoRoot, home, e.Source, name)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return false
			}
			if !replaced {
				ok = false
				continue
			}
			fmt.Printf("Migrated %s from %s to %s\n", name, lockLabel(migrateFrom[i]), e.Source)
		}
	}
	return ok
}

func skillsAdd(npx, repoRoot, home, source string, names []string) bool {
	src := source
	if src == "." {
		src = repoRoot
	}
	// skills.sh installs into ~/.agents/skills (read by Codex, Cursor, Pi, OMP) and links Claude Code to it.
	args := []string{"-y", "skills", "add", src, "-g", "-y", "-a", "claude-code", "-a", "codex"}
	for _, name := range names {
		args = append(args, "-s", name)
	}
	cmd := exec.Command(npx, args...)
	cmd.Dir = home
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Skills install failed for %s (%s): %v\n%s", source, strings.Join(names, ", "), err, out)
		return false
	}
	ok := true
	for _, name := range names {
		if !installed(home, name) {
			fmt.Fprintf(os.Stderr, "Skills install incomplete for %s: canonical skill and Claude mirror are required\n", name)
			ok = false
			continue
		}
		want, _, _, err := skillBaseline(repoRoot, home, source, name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			ok = false
			continue
		}
		for _, dir := range skillDirs(home, name) {
			got, err := treeHash(dir)
			if err != nil || got != want {
				fmt.Fprintf(os.Stderr, "Skills install does not match its source: %s\n", dir)
				ok = false
			}
		}
	}
	return ok
}

// lockEntry is the part of skills.sh's ~/.agents/.skill-lock.json drift needs.
// Source is owner/repo, a GitHub URL, or the absolute path skills.sh records for a local add.
type lockEntry struct {
	Source          string `json:"source"`
	SourceType      string `json:"sourceType"`
	SourceURL       string `json:"sourceUrl"`
	SkillPath       string `json:"skillPath"`
	SkillFolderHash string `json:"skillFolderHash"`
}

func skillLockPath(home string) string {
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		return filepath.Join(stateHome, "skills", ".skill-lock.json")
	}
	return filepath.Join(home, ".agents", ".skill-lock.json")
}

func readSkillLock(home string) (map[string]lockEntry, error) {
	b, err := os.ReadFile(skillLockPath(home))
	if err != nil {
		return nil, err
	}
	var lock struct {
		Skills map[string]lockEntry `json:"skills"`
	}
	if err := json.Unmarshal(b, &lock); err != nil {
		return nil, err
	}
	return lock.Skills, nil
}

// foreignLock reports a parsed lock entry whose source is not the manifest source.
// A missing or unreadable lock is not a mismatch; external baselines still require a hash.
func foreignLock(repoRoot, home, source, name string) (lockEntry, bool) {
	skills, err := readSkillLock(home)
	if err != nil {
		return lockEntry{}, false
	}
	locked, found := skills[name]
	if !found || sameSource(repoRoot, source, locked) {
		return lockEntry{}, false
	}
	return locked, true
}

func lockLabel(l lockEntry) string {
	if l.Source != "" {
		return l.Source
	}
	return l.SourceURL
}

// sameSource matches a manifest source to a skills.sh lock entry.
// "." matches the absolute path passed to skills add. owner/repo, a github.com URL, and a subpath are one identity.
func sameSource(repoRoot, manifest string, l lockEntry) bool {
	if manifest == "." {
		p, ok := localLockPath(l)
		return ok && p == filepath.Clean(repoRoot)
	}
	wantRepo, wantSub, ok := githubIdentity(manifest)
	if !ok {
		return false
	}
	gotRepo, gotSub, ok := lockGitHub(l)
	if !ok || !strings.EqualFold(gotRepo, wantRepo) {
		return false
	}
	if wantSub == "" {
		return true
	}
	sub := skillSubpath(l.SkillPath)
	if sub == "" {
		sub = gotSub
	}
	return sub == wantSub || strings.HasPrefix(sub, wantSub+"/")
}

func localLockPath(l lockEntry) (string, bool) {
	if l.SourceType != "" && l.SourceType != "local" {
		return "", false
	}
	p := l.Source
	if !filepath.IsAbs(p) {
		p = l.SourceURL
	}
	if !filepath.IsAbs(p) {
		return "", false
	}
	return filepath.Clean(p), true
}

func lockGitHub(l lockEntry) (string, string, bool) {
	repo, sub, ok := githubIdentity(l.Source)
	uRepo, uSub, uOK := githubIdentity(l.SourceURL)
	if ok && uOK && !strings.EqualFold(repo, uRepo) {
		return "", "", false
	}
	if !ok && uOK {
		return uRepo, uSub, true
	}
	return repo, sub, ok
}

func githubIdentity(s string) (repo, sub string, ok bool) {
	s = strings.TrimSuffix(strings.TrimSpace(s), "/")
	if s == "" || s == "." || filepath.IsAbs(s) {
		return "", "", false
	}
	switch {
	case strings.HasPrefix(s, "https://github.com/"):
		s = strings.TrimPrefix(s, "https://github.com/")
	case strings.HasPrefix(s, "http://github.com/"):
		s = strings.TrimPrefix(s, "http://github.com/")
	case strings.HasPrefix(s, "git@github.com:"):
		s = strings.TrimPrefix(s, "git@github.com:")
	case strings.HasPrefix(s, "ssh://git@github.com/"):
		s = strings.TrimPrefix(s, "ssh://git@github.com/")
	case strings.Contains(s, "://") || strings.Contains(s, ":"):
		return "", "", false
	}
	s = strings.TrimSuffix(s, ".git")
	parts := strings.Split(s, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	rest := parts[2:]
	if len(rest) >= 2 && rest[0] == "tree" {
		rest = rest[2:]
	}
	return parts[0] + "/" + parts[1], strings.Join(rest, "/"), true
}

func skillSubpath(skillPath string) string {
	skillPath = filepath.ToSlash(skillPath)
	return strings.Trim(strings.TrimSuffix(skillPath, "/SKILL.md"), "/")
}

func skillBaseline(repoRoot, home, source, name string) (string, string, lockEntry, error) {
	if locked, foreign := foreignLock(repoRoot, home, source, name); foreign {
		return "", "", lockEntry{}, fmt.Errorf("Skill %s is locked to %s, not %s", name, lockLabel(locked), source)
	}
	if source == "." {
		dir := localSkillDir(repoRoot, name)
		if dir == "" {
			return "", "", lockEntry{}, fmt.Errorf("Skill baseline unavailable for %s: expected one local source", name)
		}
		hash, err := treeHash(dir)
		return hash, dir, lockEntry{}, err
	}
	skills, lockErr := readSkillLock(home)
	l, found := skills[name]
	if lockErr != nil || !found || l.SkillFolderHash == "" {
		return "", "", lockEntry{}, fmt.Errorf("Skill baseline unavailable for %s in %s", name, skillLockPath(home))
	}
	return l.SkillFolderHash, "", l, nil
}

func checkDrift(repoRoot, home string, entries []manifestEntry) (bool, error) {
	tty := stdinIsTerminal()
	in := bufio.NewReader(os.Stdin)
	ok := true
	for _, e := range entries {
		for _, name := range e.Skills {
			want, sourceDir, external, err := skillBaseline(repoRoot, home, e.Source, name)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				ok = false
				continue
			}
			dirs := skillDirs(home, name)
			drifted := !installed(home, name)
			for _, dir := range dirs {
				info, err := os.Stat(dir)
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				if err != nil {
					return false, err
				}
				if !info.IsDir() {
					drifted = true
					continue
				}
				got, err := treeHash(dir)
				if err != nil {
					return false, err
				}
				drifted = drifted || got != want
			}
			if !drifted {
				continue
			}
			fmt.Printf("Skill drifted from its source or is incomplete: %s\n", name)
			if !tty {
				ok = false
				continue
			}
			for _, dir := range dirs {
				if _, err := os.Stat(dir); err == nil {
					showDiff(sourceDir, external, dir)
				} else {
					fmt.Println("Missing skill destination: " + dir)
				}
			}
			fmt.Printf("%s: [k]eep installed, [t]ake source, [s]kip? ", name)
			line, _ := in.ReadString('\n')
			switch strings.ToLower(strings.TrimSpace(line)) {
			case "k":
				ok = installed(home, name) && ok
			case "t":
				npx, err := exec.LookPath("npx")
				if err != nil {
					fmt.Fprintln(os.Stderr, "Cannot take source: npx not found")
					ok = false
					continue
				}
				replaced, err := replaceSkill(npx, repoRoot, home, e.Source, name)
				if err != nil {
					return false, err
				}
				ok = replaced && ok
			default:
				ok = false
			}
		}
	}
	return ok, nil
}

func replaceSkill(npx, repoRoot, home, source, name string) (bool, error) {
	dirs := skillDirs(home, name)
	backups := map[string]string{}
	restore := func() error {
		var result error
		for _, dir := range dirs {
			if backup, found := backups[dir]; found {
				result = errors.Join(result, os.RemoveAll(dir))
				result = errors.Join(result, os.Rename(backup, dir))
			}
		}
		return result
	}
	for _, dir := range dirs {
		if _, err := os.Lstat(dir); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return false, errors.Join(err, restore())
		}
		backup := backupPath(dir, home)
		if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
			return false, errors.Join(err, restore())
		}
		if err := os.Rename(dir, backup); err != nil {
			return false, errors.Join(err, restore())
		}
		backups[dir] = backup
		fmt.Println("Backed up to " + backup)
	}
	if skillsAdd(npx, repoRoot, home, source, []string{name}) {
		return true, nil
	}
	var err error
	for _, dir := range dirs {
		err = errors.Join(err, os.RemoveAll(dir))
	}
	return false, errors.Join(err, restore())
}

// localSkillDir finds skills/<scope>/<group>/<name> in this repo.
func localSkillDir(repoRoot, name string) string {
	matches, _ := filepath.Glob(filepath.Join(repoRoot, "skills", "*", "*", name, "SKILL.md"))
	if len(matches) != 1 {
		return ""
	}
	return filepath.Dir(matches[0])
}

// treeHash is the git tree id of dir's files, the same value skills.sh records as skillFolderHash.
func treeHash(dir string) (string, error) {
	tmp, err := os.MkdirTemp("", "gap-tree-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	base := gitEnv()
	run := func(env []string, args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir, cmd.Env = dir, env
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("hashing %s: git %s: %w", dir, args[len(args)-1], err)
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err := run(base, "init", "-q", "--bare", tmp); err != nil {
		return "", err
	}
	env := append(base, "GIT_INDEX_FILE="+filepath.Join(tmp, "index"))
	if _, err := run(env, "--git-dir="+tmp, "--work-tree="+dir, "add", "--force", "-A", "."); err != nil {
		return "", err
	}
	return run(env, "--git-dir="+tmp, "--work-tree="+dir, "write-tree")
}

// showDiff prints source vs installed. An external source is cloned shallowly from its lock entry.
func showDiff(sourceDir string, external lockEntry, installedDir string) {
	if sourceDir == "" {
		if external.SourceURL == "" {
			return
		}
		tmp, err := os.MkdirTemp("", "gap-src-")
		if err != nil {
			return
		}
		defer os.RemoveAll(tmp)
		if exec.Command("git", "clone", "-q", "--depth", "1", external.SourceURL, tmp).Run() != nil {
			fmt.Fprintln(os.Stderr, "Could not fetch "+external.SourceURL+" to show the diff")
			return
		}
		sourceDir = filepath.Join(tmp, filepath.Dir(external.SkillPath))
	}
	cmd := exec.Command("git", "--no-pager", "diff", "--no-index", "--", sourceDir, installedDir)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	_ = cmd.Run()
}

func skillDirs(home, name string) []string {
	return []string{filepath.Join(home, ".agents", "skills", name), filepath.Join(home, ".claude", "skills", name)}
}

func installed(home, name string) bool {
	for _, dir := range skillDirs(home, name) {
		if info, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return false
		}
	}
	return true
}
