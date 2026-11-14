package main

import (
	"cmp"
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

type manifestEntry struct {
	Source    string
	Skills    []string
	Path      string
	SkillPath string
	Project   string
	Owner     string
}

// installSkills copies each missing skill out of its pinned store and hands unedited
// skills.sh installs of the same name to their component.
func installSkills(home string, entries []manifestEntry) bool {
	ok := true
	for _, e := range entries {
		target := entryHome(home, e)
		for _, name := range e.Skills {
			exists := false
			for _, dir := range skillDirs(target, name) {
				if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
					exists = true
				}
			}
			if !exists {
				replaced, err := replaceSkill(target, e, name)
				if err != nil || !replaced {
					fmt.Fprintf(os.Stderr, "Installing %s: %v\n", name, err)
					ok = false
				}
				continue
			}
			if e.Project == "" {
				if err := adoptSkill(home, e, name); err != nil {
					fmt.Fprintf(os.Stderr, "Migrating %s from skills.sh: %v\n", name, err)
					ok = false
				}
			}
		}
	}
	return ok
}

// adoptSkill takes over a skill skills.sh installed when both copies match the pinned store.
// It records the component as owner and drops the skills.sh lock entry so skills.sh stops updating it.
// An edited copy is left for drift review, and a missing lock is not a migration.
// A lock that exists but cannot be read or parsed is an error, and nothing is changed.
func adoptSkill(home string, e manifestEntry, name string) error {
	skills, err := readSkillLock(home)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("skills.sh lock %s: %w", skillLockPath(home), err)
	}
	locked, found := skills[name]
	if !found || entryDriftCount(e, home, name) != 0 {
		return nil
	}
	lockPath := skillLockPath(home)
	before, err := os.ReadFile(lockPath)
	if err != nil {
		return fmt.Errorf("skills.sh lock %s: %w", lockPath, err)
	}
	if err := dropSkillLockEntry(home, name); err != nil {
		return err
	}
	if err := recordOwnership(home, e.Owner, []string{name}, nil, e.Path); err != nil {
		if restoreErr := writeAtomic(lockPath, string(before)); restoreErr != nil {
			return fmt.Errorf("%w; restoring skills.sh lock %s also failed: %v", err, lockPath, restoreErr)
		}
		return err
	}
	fmt.Printf("Migrated %s from skills.sh (%s) to %s\n", name, lockLabel(locked), cmp.Or(e.Owner, e.Source))
	return nil
}

// entryDriftCount is the number of installed copies of name that differ from the entry's pinned store.
func entryDriftCount(e manifestEntry, home, name string) int {
	src := entrySkillDir(e, name)
	if src == "" {
		return 1
	}
	want, err := treeHash(src)
	if err != nil {
		return 1
	}
	n := 0
	for _, dir := range skillDirs(home, name) {
		got, err := treeHash(dir)
		if err != nil || got != want {
			n++
		}
	}
	return n
}

// lockEntry is the part of a skills.sh ~/.agents/.skill-lock.json entry that migration reports.
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

func lockLabel(l lockEntry) string {
	if l.Source != "" {
		return l.Source
	}
	return l.SourceURL
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

func checkDrift(home string, entries []manifestEntry) (bool, error) {
	defer func() {
		for _, e := range entries {
			if e.Project != "" {
				continue
			}
			for _, name := range e.Skills {
				for _, warning := range shadowWarnings(home, name) {
					fmt.Fprintln(os.Stderr, warning)
				}
			}
		}
	}()
	tty := termReady()
	ok := true
	for _, e := range entries {
		target := entryHome(home, e)
		for _, name := range e.Skills {
			sourceDir := entrySkillDir(e, name)
			if sourceDir == "" {
				fmt.Fprintf(os.Stderr, "Skill baseline unavailable for %s: %s has no %s skill\n", name, e.Path, name)
				ok = false
				continue
			}
			want, err := treeHash(sourceDir)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				ok = false
				continue
			}
			drifted := !installed(target, name)
			for _, dir := range skillDirs(target, name) {
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
			newReport(os.Stdout).warn("Skill drifted from its source or is incomplete: "+name, name, "drifted from its source or incomplete")
			if e.Project == "" {
				if skills, err := readSkillLock(home); err == nil {
					if locked, found := skills[name]; found {
						fmt.Fprintf(os.Stderr, "Skill %s was installed by skills.sh from %s; take source moves it to %s\n", name, lockLabel(locked), cmp.Or(e.Owner, e.Source))
					}
				}
			}
			if !tty {
				ok = false
				continue
			}
			for _, dir := range skillDirs(target, name) {
				if _, err := os.Stat(dir); err == nil {
					showDiff(sourceDir, dir)
				} else {
					fmt.Println("Missing skill destination: " + dir)
				}
			}
			fmt.Printf("%s: [k]eep installed, [t]ake source, [s]kip? ", name)
			switch strings.ToLower(strings.TrimSpace(stdinLine())) {
			case "k":
				ok = installed(target, name) && ok
			case "t":
				replaced, err := replaceSkill(target, e, name)
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

// replaceSkill backs up both destinations, then copies name out of the entry's pinned store.
// A global install also drops any skills.sh lock entry for name. A failed copy restores the backups.
func replaceSkill(home string, e manifestEntry, name string) (bool, error) {
	src := entrySkillDir(e, name)
	if src == "" {
		return false, fmt.Errorf("skill %s has no source in pinned store %s", name, e.Path)
	}
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
	var installErr error
	for _, dir := range dirs {
		if installErr = copySkill(src, dir); installErr != nil {
			break
		}
	}
	if installErr == nil && e.Project == "" {
		installErr = dropSkillLockEntry(home, name)
	}
	if installErr == nil {
		return true, recordOwnership(home, e.Owner, []string{name}, nil, e.Path)
	}
	for _, dir := range dirs {
		installErr = errors.Join(installErr, os.RemoveAll(dir))
	}
	return false, errors.Join(installErr, restore())
}

type skillLockSnap struct {
	path, name string
	hadFile    bool
	hadSkill   bool
	entry      json.RawMessage
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

// findSkillDir returns the folder named name that holds a regular SKILL.md, searching root/sub before root.
// Within one base the shallowest match wins, and a tie goes to the first path in lexical order,
// so a repo that ships several copies of a skill always resolves to the same one.
func findSkillDir(root, sub, name string) string {
	bases := []string{root}
	if sub != "" {
		bases = []string{filepath.Join(root, filepath.FromSlash(sub)), root}
	}
	for _, base := range bases {
		found, best := "", -1
		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			depth := strings.Count(filepath.ToSlash(strings.TrimPrefix(path, base)), "/")
			if d.IsDir() {
				if d.Name() == "node_modules" || d.Name() == ".git" || (best >= 0 && depth >= best) {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Name() == "SKILL.md" && d.Type().IsRegular() && filepath.Base(filepath.Dir(path)) == name && (best < 0 || depth-1 < best) {
				found, best = filepath.Dir(path), depth-1
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}

func localSkillDir(repoRoot, name string) string {
	return findSkillDir(repoRoot, "skills", name)
}

func entrySkillDir(e manifestEntry, name string) string {
	if e.SkillPath != "" {
		return findSkillDir(filepath.Join(e.Path, filepath.FromSlash(e.SkillPath)), "", name)
	}
	return localSkillDir(e.Path, name)
}

// treeHash is the Git tree id of dir, without the paths indexPathExcluded names.
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

// indexPathExcluded is a path an installed skill copy leaves out: metadata.json, .git,
// __pycache__, __pypackages__, and node_modules (a Git tree of node_modules never matches
// a copy, because symlinks are dereferenced). Both sides are hashed without them.
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

// contentHash is SHA-256 of each relative path plus its bytes, sorted the way skills.sh sorts them.
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

// showDiff prints the pinned source against an installed copy.
func showDiff(sourceDir, installedDir string) {
	cmd := exec.Command("git", "--no-pager", "diff", "--no-index", "--", sourceDir, installedDir)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	_ = cmd.Run()
}

func skillDirs(home, name string) []string {
	return []string{filepath.Join(home, ".agents", "skills", name), filepath.Join(home, ".claude", "skills", name)}
}

// shadowWarnings reports harness-only skill copies that differ from the managed copy. gap never writes them.
// Empryo's harness copy wins; jcode's copy precedence is unverified.
func shadowWarnings(home, name string) []string {
	managed := skillDirs(home, name)[0]
	var out []string
	if name != "empryo" {
		if shadow := filepath.Join(home, ".empryo", "skills", name); differs(shadow, managed) {
			out = append(out, fmt.Sprintf("Empryo loads %s before the managed %s, and the copies differ", shadow, managed))
		}
	}
	if base, err := harnessHome(harnessNamed("jcode"), home); err == nil {
		if shadow := filepath.Join(base, "skills", name); differs(shadow, managed) {
			out = append(out, fmt.Sprintf("jcode skill copy %s differs from the managed %s", shadow, managed))
		}
	}
	return out
}

func differs(shadow, managed string) bool {
	if info, err := os.Stat(shadow); err != nil || !info.IsDir() {
		return false
	}
	got, err := contentHash(shadow)
	want, werr := contentHash(managed)
	return err != nil || werr != nil || got != want
}

func installed(home, name string) bool {
	for _, dir := range skillDirs(home, name) {
		if info, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return false
		}
	}
	return true
}

func entryHome(home string, e manifestEntry) string {
	if e.Project != "" {
		return e.Project
	}
	return home
}
