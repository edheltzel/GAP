package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Path, when set, is the directory skills are read from. Source stays the identity used for lock matching.
type manifestEntry struct {
	Source string   `json:"source"`
	Skills []string `json:"skills"`
	Path   string   `json:"-"`
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
			// A pinned copy that already matches its store needs nothing, whatever its lock says.
			// A checkout source still migrates, so its lock moves off the old source.
			if e.Path != "" && skillUsable(repoRoot, home, e, name) {
				// The bytes are right but the lock may still name the old source. Record the store.
				if e.Path != "" {
					if _, foreign := foreignLockEntry(entryRoot(repoRoot, e), home, e, name); foreign {
						src := localSkillSource(repoRoot, e, name)
						hash, herr := contentHash(src)
						if herr == nil {
							_ = writeSkillLockEntry(home, name, lockEntry{Source: e.Path, SourceType: "local", SourceURL: e.Path, SkillFolderHash: hash})
						}
					}
				}
				continue
			}
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
			// Only an unedited copy is replaced here. An edit, or a hash we cannot check, waits for drift review.
			if locked, foreign := foreignLockEntry(entryRoot(repoRoot, e), home, e, name); foreign && copiesMatchLock(home, name, locked) {
				migrate = append(migrate, name)
				migrateFrom = append(migrateFrom, locked)
			}
		}
		if len(missing) == 0 && len(migrate) == 0 {
			continue
		}
		// A pinned store is copied directly. A checkout source and anything gap has to fetch still go through skills.sh.
		if e.Path != "" && localSkillSource(repoRoot, e, firstName(missing, migrate)) != "" {
			for _, name := range append(append([]string{}, missing...), migrate...) {
				src := localSkillSource(repoRoot, e, name)
				if src == "" {
					fmt.Fprintf(os.Stderr, "Skill %s has no source in %s\n", name, addSource(repoRoot, e))
					ok = false
					continue
				}
				if err := installLocal(home, e, name, src); err != nil {
					fmt.Fprintln(os.Stderr, err)
					ok = false
					continue
				}
				fmt.Println(filepath.Join(home, ".agents", "skills", name))
			}
			continue
		}
		if npx == "" {
			if npx, err = exec.LookPath("npx"); err != nil {
				fmt.Fprintln(os.Stderr, "Skipping skills: npx not found; install Node to let gap install skills")
				return false
			}
		}
		if len(missing) > 0 && !skillsAdd(npx, entryRoot(repoRoot, e), home, e, missing) {
			ok = false
		} else {
			for _, name := range missing {
				fmt.Println(filepath.Join(home, ".agents", "skills", name))
			}
		}
		for i, name := range migrate {
			replaced, err := replaceSkill(npx, entryRoot(repoRoot, e), home, e, name)
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

func firstName(groups ...[]string) string {
	for _, g := range groups {
		if len(g) > 0 {
			return g[0]
		}
	}
	return ""
}

func skillsAdd(npx, repoRoot, home string, e manifestEntry, names []string) bool {
	// Path is the pinned store (or a local checkout). Source is only the lock identity.
	src := addSource(repoRoot, e)
	// skills.sh installs into ~/.agents/skills (read by Codex, Cursor, Pi, OMP) and links Claude Code to it.
	args := []string{"-y", "skills", "add", src, "-g", "-y", "-a", "claude-code", "-a", "codex"}
	for _, name := range names {
		args = append(args, "-s", name)
	}
	cmd := exec.Command(npx, args...)
	cmd.Dir = home
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Skills install failed for %s (%s): %v\n%s", e.Source, strings.Join(names, ", "), err, out)
		return false
	}
	ok := true
	for _, name := range names {
		if !installed(home, name) {
			fmt.Fprintf(os.Stderr, "Skills install incomplete for %s: canonical skill and Claude mirror are required\n", name)
			ok = false
			continue
		}
		want, _, _, err := skillBaseline(repoRoot, home, e, name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			ok = false
			continue
		}
		for _, dir := range skillDirs(home, name) {
			got, err := hashLike(dir, want)
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
	return foreignLockEntry(repoRoot, home, manifestEntry{Source: source}, name)
}

func foreignLockEntry(repoRoot, home string, e manifestEntry, name string) (lockEntry, bool) {
	skills, err := readSkillLock(home)
	if err != nil {
		return lockEntry{}, false
	}
	locked, found := skills[name]
	if !found || sameSource(repoRoot, e, locked) {
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
func sameSource(repoRoot string, e manifestEntry, l lockEntry) bool {
	manifest := e.Source
	if manifest == "." {
		p, ok := localLockPath(l)
		if !ok {
			return false
		}
		root := filepath.Clean(addSource(repoRoot, e))
		return p == root || storeSibling(root, p) || checkoutLock(root, p)
	}
	// Path is the pinned store. skills.sh records that directory, including an older commit of the same component.
	// A lock that still names the GAP checkout is not this store, so an unedited copy stays foreign and migrates.
	if e.Path != "" {
		if p, ok := localLockPath(l); ok && storeSibling(e.Path, p) {
			return true
		}
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
	case strings.HasPrefix(s, "github.com/"):
		s = strings.TrimPrefix(s, "github.com/")
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

func skillBaseline(repoRoot, home string, e manifestEntry, name string) (string, string, lockEntry, error) {
	if locked, foreign := foreignLockEntry(repoRoot, home, e, name); foreign {
		return "", "", lockEntry{}, fmt.Errorf("Skill %s is locked to %s, not %s", name, lockLabel(locked), e.Source)
	}
	if local := e.Path != "" || e.Source == "."; local {
		dir := localSkillDir(addSource(repoRoot, e), name)
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
	// Recorded as-is: 40-hex Git tree, or 64-hex content hash when the tree API was unavailable.
	return l.SkillFolderHash, "", l, nil
}

func checkDrift(repoRoot, home string, entries []manifestEntry) (bool, error) {
	defer func() {
		for _, e := range entries {
			for _, name := range e.Skills {
				if warning := empryoShadow(home, name); warning != "" {
					fmt.Fprintln(os.Stderr, warning)
				}
			}
		}
	}()
	tty := termReady()
	in := bufio.NewReader(os.Stdin)
	ok := true
	for _, e := range entries {
		for _, name := range e.Skills {
			var sourceDir string
			var external lockEntry
			drifted := false
			foreignReview := false
			root := entryRoot(repoRoot, e)
			if locked, foreign := foreignLockEntry(root, home, e, name); foreign {
				foreignReview = true
				fmt.Fprintf(os.Stderr, "Skill %s is locked to %s, not %s\n", name, lockLabel(locked), e.Source)
				drifted = true
				if e.Path != "" || e.Source == "." {
					sourceDir = localSkillDir(addSource(repoRoot, e), name)
				}
			} else {
				want, dir, ext, err := skillBaseline(root, home, e, name)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					ok = false
					continue
				}
				sourceDir, external = dir, ext
				dirs := skillDirs(home, name)
				drifted = !installed(home, name)
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
					got, err := hashLike(dir, want)
					if err != nil {
						return false, err
					}
					drifted = drifted || got != want
				}
			}
			if !drifted {
				continue
			}
			fmt.Printf("Skill drifted from its source or is incomplete: %s\n", name)
			if !tty {
				ok = false
				continue
			}
			if foreignReview && sourceDir == "" {
				var cleanup func()
				sourceDir, cleanup = openSourceSkill(e, name)
				defer cleanup()
			}
			for _, dir := range skillDirs(home, name) {
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
				replaced, err := replaceSkill(npx, entryRoot(repoRoot, e), home, e, name)
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

func replaceSkill(npx, repoRoot, home string, e manifestEntry, name string) (bool, error) {
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
	snap := snapshotSkillLock(home, name)
	if skillsAdd(npx, repoRoot, home, e, []string{name}) {
		return true, nil
	}
	var err error
	for _, dir := range dirs {
		err = errors.Join(err, os.RemoveAll(dir))
	}
	return false, errors.Join(err, restoreSkillLock(snap), restore())
}

type skillLockSnap struct {
	path, name string
	hadFile    bool
	hadSkill   bool
	entry      json.RawMessage
}

func snapshotSkillLock(home, name string) skillLockSnap {
	path := skillLockPath(home)
	snap := skillLockSnap{path: path, name: name}
	b, err := os.ReadFile(path)
	if err != nil {
		return snap
	}
	snap.hadFile = true
	entry, ok := rawSkillEntry(b, name)
	snap.hadSkill, snap.entry = ok, entry
	return snap
}

func rawSkillEntry(doc []byte, name string) (json.RawMessage, bool) {
	var root map[string]json.RawMessage
	if json.Unmarshal(doc, &root) != nil {
		return nil, false
	}
	var skills map[string]json.RawMessage
	if json.Unmarshal(root["skills"], &skills) != nil {
		return nil, false
	}
	entry, ok := skills[name]
	return entry, ok
}

func restoreSkillLock(snap skillLockSnap) error {
	if !snap.hadFile {
		if _, err := os.Lstat(snap.path); err == nil {
			return os.Remove(snap.path)
		}
		return nil
	}
	b, err := os.ReadFile(snap.path)
	if err != nil {
		if !snap.hadSkill {
			return nil
		}
		b = []byte(`{"skills":{}}`)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(b, &root); err != nil {
		return err
	}
	skills := map[string]json.RawMessage{}
	if raw := root["skills"]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &skills); err != nil {
			return err
		}
	}
	if snap.hadSkill {
		skills[snap.name] = snap.entry
	} else {
		delete(skills, snap.name)
	}
	encoded, err := json.Marshal(skills)
	if err != nil {
		return err
	}
	root["skills"] = encoded
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(snap.path, append(out, '\n'), 0o644)
}

// openSourceSkill returns the skill directory for a diff.
// Path selects the pinned checkout to read from. Otherwise the manifest source is cloned.
// The cleanup removes a clone. A failed fetch returns an empty dir after reporting what take would replace.
func openSourceSkill(e manifestEntry, name string) (string, func()) {
	noop := func() {}
	if e.Path != "" {
		dir := localSkillDir(e.Path, name)
		if dir == "" {
			fmt.Fprintf(os.Stderr, "No diff for %s: %s has no %s skill. Take would reinstall from %s.\n", name, e.Path, name, e.Path)
		}
		return dir, noop
	}
	source := e.Source
	repo, sub, ok := githubIdentity(source)
	if !ok {
		fmt.Fprintf(os.Stderr, "No diff for %s: cannot fetch %s. Take would reinstall from %s.\n", name, source, source)
		return "", noop
	}
	tmp, err := os.MkdirTemp("", "gap-src-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "No diff for %s: %v. Take would reinstall from %s.\n", name, err, source)
		return "", noop
	}
	cleanup := func() { os.RemoveAll(tmp) }
	url := "https://github.com/" + repo + ".git"
	if exec.Command("git", "clone", "-q", "--depth", "1", url, tmp).Run() != nil {
		cleanup()
		fmt.Fprintf(os.Stderr, "No diff for %s: could not fetch %s. Take would reinstall from %s.\n", name, url, source)
		return "", noop
	}
	dir := findSkillDir(tmp, sub, name)
	if dir == "" {
		cleanup()
		fmt.Fprintf(os.Stderr, "No diff for %s: %s has no %s skill. Take would reinstall from %s.\n", name, url, name, source)
		return "", noop
	}
	return dir, cleanup
}

func findSkillDir(root, sub, name string) string {
	bases := []string{root}
	if sub != "" {
		bases = []string{filepath.Join(root, filepath.FromSlash(sub)), root}
	}
	for _, base := range bases {
		var found string
		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil || found != "" {
				return err
			}
			if d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git") {
				return filepath.SkipDir
			}
			if d.Name() == "SKILL.md" && filepath.Base(filepath.Dir(path)) == name {
				found = filepath.Dir(path)
				return filepath.SkipAll
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}

// localSkillDir finds skills/<scope>/<group>/<name>, or a component checkout that is the skill itself.
func localSkillDir(repoRoot, name string) string {
	if _, err := os.Stat(filepath.Join(repoRoot, "SKILL.md")); err == nil && filepath.Base(repoRoot) == name {
		return repoRoot
	}
	matches, _ := filepath.Glob(filepath.Join(repoRoot, "skills", "*", "*", name, "SKILL.md"))
	if len(matches) != 1 {
		return findSkillDir(repoRoot, "", name)
	}
	return filepath.Dir(matches[0])
}

// entryRoot is the component store when Path is set, otherwise the GAP checkout.
func entryRoot(repoRoot string, e manifestEntry) string {
	return addSource(repoRoot, e)
}

// addSource prefers the pinned Path over Source, which remains the lock identity.
func addSource(repoRoot string, e manifestEntry) string {
	if e.Path != "" {
		return e.Path
	}
	if e.Source == "." {
		return repoRoot
	}
	return e.Source
}

// storeSibling reports that locked is another commit of the same component store as root
// (home/.gap/store/<id>/<commit>). A checkout-path lock is the same identity only when root is that checkout.
func storeSibling(root, locked string) bool {
	root, locked = cleanPath(root), cleanPath(locked)
	if root == locked {
		return true
	}
	id := filepath.Base(filepath.Dir(root))
	store := cleanPath(filepath.Dir(filepath.Dir(root)))
	if filepath.Base(store) != "store" || id == "" || id == "." {
		return false
	}
	rel, err := filepath.Rel(filepath.Join(store, id), locked)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return !strings.Contains(rel, string(filepath.Separator))
}

// cleanPath resolves symlinks. macOS reports /var as /private/var.
func cleanPath(path string) string {
	path = filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved)
	}
	// A removed commit dir still has to compare. Resolve the parent that exists.
	parent := filepath.Dir(path)
	if resolved, err := filepath.EvalSymlinks(parent); err == nil {
		return filepath.Clean(filepath.Join(resolved, filepath.Base(path)))
	}
	return path
}

// checkoutLock reports a skills.sh lock whose source is the GAP checkout while the
// manifest now points at that component's store. Hashes still have to match before migration.
func checkoutLock(store, locked string) bool {
	if _, err := os.Stat(filepath.Join(locked, "global", "AGENTS.md")); err != nil {
		return false
	}
	return filepath.Base(filepath.Dir(filepath.Dir(store))) == "store"
}

// treeHash is the Git tree id of dir. skills.sh records that when the GitHub tree API works.
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
	if err := dropInstallExcludes(env, tmp); err != nil {
		return "", err
	}
	return run(env, "--git-dir="+tmp, "--work-tree="+dir, "write-tree")
}

// indexPathExcluded is a path skills.sh copyDirectory does not install, plus node_modules.
// installer.ts excludes metadata.json, .git, __pycache__, and __pypackages__. A skill's
// node_modules is not in that set, but skills.sh's install of do-art omits it and a Git
// tree of it never matches the copy (symlinks are dereferenced). Hash both sides without it.
func indexPathExcluded(rel string) bool {
	rel = filepath.ToSlash(rel)
	if rel == "metadata.json" || strings.HasSuffix(rel, "/metadata.json") {
		return true
	}
	for _, part := range strings.Split(rel, "/") {
		switch part {
		case "node_modules", ".git", "__pycache__", "__pypackages__":
			return true
		}
	}
	return false
}

func dropInstallExcludes(env []string, gitDir string) error {
	cmd := exec.Command("git", "--git-dir="+gitDir, "ls-files", "-z")
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("listing skill index: %w", err)
	}
	var drop []string
	for _, rel := range strings.Split(string(out), "\x00") {
		if rel != "" && indexPathExcluded(rel) {
			drop = append(drop, rel)
		}
	}
	if len(drop) == 0 {
		return nil
	}
	args := append([]string{"--git-dir=" + gitDir, "rm", "--cached", "-q", "--ignore-unmatch", "--"}, drop...)
	rm := exec.Command("git", args...)
	rm.Env = env
	if msg, err := rm.CombinedOutput(); err != nil {
		return fmt.Errorf("dropping excluded skill files: %w\n%s", err, msg)
	}
	return nil
}

func copiesMatchLock(home, name string, locked lockEntry) bool {
	if !isHexLen(locked.SkillFolderHash, 40) && !isHexLen(locked.SkillFolderHash, 64) {
		return false
	}
	for _, dir := range skillDirs(home, name) {
		if _, err := os.Lstat(dir); err != nil {
			return false
		}
		got, err := hashLike(dir, locked.SkillFolderHash)
		if err != nil || got != locked.SkillFolderHash {
			return false
		}
	}
	return true
}

func hashLike(dir, recorded string) (string, error) {
	if isHexLen(recorded, 64) {
		return contentHash(dir)
	}
	return treeHash(dir)
}

func isHexLen(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if c < '0' || (c > '9' && c < 'a') || c > 'f' {
			return false
		}
	}
	return true
}

// contentHash is skills.sh computeSkillFolderHash: SHA-256 of each relative path plus its bytes, sorted.
// .git, node_modules, and non-regular files are skipped.
// ponytail: ASCII order matches Node en-US localeCompare; non-ASCII code points sort after ASCII. Revisit if a non-en-US lock hash does not verify.
func contentHash(dir string) (string, error) {
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	var rels []string
	err = filepath.WalkDir(resolved, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != resolved && (d.Name() == ".git" || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(resolved, path)
		if err != nil {
			return err
		}
		rels = append(rels, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(rels, func(i, j int) bool { return skillPathLess(rels[i], rels[j]) })
	sum := sha256.New()
	for _, rel := range rels {
		b, err := os.ReadFile(filepath.Join(resolved, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		sum.Write([]byte(rel))
		sum.Write(b)
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// skillPrimary is the en-US localeCompare order of ASCII 1..127. Letters share one weight; case breaks ties.
var skillPrimary [128]int

func init() {
	order := []byte{
		1, 2, 3, 4, 5, 6, 7, 8, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 127,
		9, 10, 11, 12, 13, 32, 95, 45, 44, 59, 58, 33, 63, 46, 39, 34, 40, 41, 91, 93, 123, 125, 64, 42, 47, 92, 38, 35, 37, 96, 94, 43, 60, 61, 62, 124, 126, 36,
		48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
		97, 65, 98, 66, 99, 67, 100, 68, 101, 69, 102, 70, 103, 71, 104, 72, 105, 73, 106, 74, 107, 75, 108, 76, 109, 77, 110, 78, 111, 79, 112, 80, 113, 81, 114, 82, 115, 83, 116, 84, 117, 85, 118, 86, 119, 87, 120, 88, 121, 89, 122, 90,
	}
	for i, c := range order {
		if c >= 'A' && c <= 'Z' {
			continue
		}
		skillPrimary[c] = i + 1
	}
	for c := byte('A'); c <= 'Z'; c++ {
		skillPrimary[c] = skillPrimary[c+'a'-'A']
	}
}

func skillWeight(r rune) int {
	if r < 128 && skillPrimary[r] != 0 {
		return skillPrimary[r]
	}
	return 256 + int(r)
}

func skillPathLess(a, b string) bool {
	ar, br := []rune(a), []rune(b)
	n := min(len(ar), len(br))
	for i := range n {
		if wa, wb := skillWeight(ar[i]), skillWeight(br[i]); wa != wb {
			return wa < wb
		}
	}
	if len(ar) != len(br) {
		return len(ar) < len(br)
	}
	for i := range ar {
		if ar[i] == br[i] {
			continue
		}
		au, bu := ar[i] >= 'A' && ar[i] <= 'Z', br[i] >= 'A' && br[i] <= 'Z'
		if au != bu {
			return !au
		}
		return ar[i] < br[i]
	}
	return false
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

// empryoShadow warns when ~/.empryo/skills/<name> differs from the managed copy. Empryo looks there first, so that copy wins. gap never writes it.
func empryoShadow(home, name string) string {
	if name == "empryo" {
		return ""
	}
	shadow := filepath.Join(home, ".empryo", "skills", name)
	if info, err := os.Stat(shadow); err != nil || !info.IsDir() {
		return ""
	}
	managed := skillDirs(home, name)[0]
	got, err := contentHash(shadow)
	want, werr := contentHash(managed)
	if err == nil && werr == nil && got == want {
		return ""
	}
	return fmt.Sprintf("Empryo loads %s before the managed %s, and the copies differ", shadow, managed)
}

func installed(home, name string) bool {
	for _, dir := range skillDirs(home, name) {
		if info, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return false
		}
	}
	return true
}
