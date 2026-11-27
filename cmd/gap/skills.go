package main

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
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
	ID        string
	Agents    []string
	Copy      bool
	Subagents []string
	From      map[string]string
	Discover  bool
}

// installSkills copies each missing skill out of its pinned store and hands unedited
// skills.sh installs of the same name to their component.
func skillCount(entries []manifestEntry) int {
	n := 0
	for _, e := range entries {
		n += len(e.Skills)
	}
	return n
}

func installSkills(home string, entries []manifestEntry) bool {
	ok := true
	n, total := 0, skillCount(entries)
	for _, e := range entries {
		target := entryHome(home, e)
		for _, name := range e.Skills {
			n++
			activeProgress.item(n, total, name)
			exists := false
			for _, d := range entryDests(target, e, name) {
				if _, err := os.Lstat(d.Dir); !errors.Is(err, os.ErrNotExist) {
					exists = true
				}
			}
			if !exists {
				if _, err := replaceSkill(target, e, name); err != nil {
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
	err = recordDests(e, home, name, destDirs(entryDests(home, e, name)))
	if err == nil {
		err = recordOwnership(home, e.Owner, []string{name}, nil, e.Path)
	}
	if err != nil {
		if restoreErr := writeAtomic(lockPath, string(before)); restoreErr != nil {
			return fmt.Errorf("%w; restoring skills.sh lock %s also failed: %v", err, lockPath, restoreErr)
		}
		return err
	}
	fmt.Printf("Migrated %s from skills.sh (%s) to %s\n", name, lockLabel(locked), cmp.Or(e.Owner, e.Source))
	return nil
}

// upgradeUntouched replaces installed copies that still match an earlier store of their component
// with the pinned version, so a moved pin upgrades them without asking. A copy that matches no
// earlier store was edited, and drift review handles it. A skills.sh install is adopted on the way.
func upgradeUntouched(home string, entries []manifestEntry) bool {
	ok := true
	n, total := 0, skillCount(entries)
	for _, e := range entries {
		target := entryHome(home, e)
		for _, name := range e.Skills {
			n++
			activeProgress.item(n, total, name)
			if !installed(target, e, name) || entryDriftCount(e, target, name) == 0 || !matchesEarlierStore(e, target, name) {
				continue
			}
			if _, err := replaceSkill(target, e, name); err != nil {
				fmt.Fprintf(os.Stderr, "Updating %s: %v\n", name, err)
				ok = false
				continue
			}
			fmt.Printf("Updated %s to %s; it matched an earlier version with no local edits\n", name, short(filepath.Base(e.Path)))
		}
	}
	return ok
}

// matchesEarlierStore reports whether every installed copy of name equals the skill in one
// other verified commit of the same component under ~/.gap/store/<id>/.
func matchesEarlierStore(e manifestEntry, home, name string) bool {
	root := filepath.Dir(e.Path)
	stores, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, s := range stores {
		dir := filepath.Join(root, s.Name())
		if !s.IsDir() || !isHexLen(s.Name(), 40) || dir == e.Path || !verifiedStore(dir, s.Name()) {
			continue
		}
		earlier := e
		earlier.Path = dir
		if entrySkillDir(earlier, name) != "" && entryDriftCount(earlier, home, name) == 0 {
			return true
		}
	}
	return false
}

// verifiedStore reports whether the checkout at dir still holds exactly the tree of commit,
// so its files can stand in for an unedited earlier version.
func verifiedStore(dir, commit string) bool {
	tree, err := git(dir, "rev-parse", commit+"^{tree}")
	if err != nil {
		return false
	}
	state, err := storeState(dir, tree)
	return err == nil && state == storeOK
}

// entryDriftCount is the number of installed copies of name that differ from the entry's pinned store.
func entryDriftCount(e manifestEntry, home, name string) int {
	if entrySkillDir(e, name) == "" {
		return 1
	}
	wants := map[bool]string{}
	n := 0
	for _, d := range entryDests(home, e, name) {
		want, cached := wants[d.Eve]
		if !cached {
			var err error
			if want, err = expectedTree(e, name, d.Eve); err != nil {
				return 1
			}
			wants[d.Eve] = want
		}
		if got, err := treeHash(d.Dir); err != nil || got != want {
			n++
		}
	}
	return n
}

func renamedTo(e manifestEntry, name string) string {
	if from := e.From[name]; from != "" && (!e.Discover || sanitizeName(from) != name) {
		return name
	}
	return ""
}

func renderExpected(e manifestEntry, name string, eve bool) (string, func(), error) {
	src := entrySkillDir(e, name)
	if src == "" {
		return "", nil, fmt.Errorf("skill %s has no source in pinned store %s", name, e.Path)
	}
	tmp, err := os.MkdirTemp("", "gap-render-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(tmp) }
	dir := filepath.Join(tmp, name)
	if err := renderSkill(src, dir, renderOpts{Eve: eve, Rename: renamedTo(e, name), Quiet: true}); err != nil {
		cleanup()
		return "", nil, err
	}
	return dir, cleanup, nil
}

func expectedTree(e manifestEntry, name string, eve bool) (string, error) {
	dir, cleanup, err := renderExpected(e, name, eve)
	if err != nil {
		return "", err
	}
	defer cleanup()
	return treeHash(dir)
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
				if err := refreshJcodeShadows(home, e, name); err != nil {
					fmt.Fprintln(os.Stderr, err)
				}
				for _, warning := range shadowWarnings(home, e, name) {
					fmt.Fprintln(os.Stderr, warning)
				}
			}
		}
	}()
	tty := termReady()
	ok := true
	n, total := 0, skillCount(entries)
	for _, e := range entries {
		target := entryHome(home, e)
		for _, name := range e.Skills {
			n++
			activeProgress.item(n, total, name)
			if entrySkillDir(e, name) == "" {
				fmt.Fprintf(os.Stderr, "Skill baseline unavailable for %s: %s has no %s skill\n", name, e.Path, name)
				ok = false
				continue
			}
			dests := entryDests(target, e, name)
			wants := map[bool]string{}
			var wantErr error
			for _, d := range dests {
				if _, done := wants[d.Eve]; !done {
					want, err := expectedTree(e, name, d.Eve)
					wants[d.Eve], wantErr = want, errors.Join(wantErr, err)
				}
			}
			if wantErr != nil {
				fmt.Fprintln(os.Stderr, wantErr)
				ok = false
				continue
			}
			drifted := !installed(target, e, name)
			for _, d := range dests {
				info, err := os.Stat(d.Dir)
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
				got, err := treeHash(d.Dir)
				if err != nil {
					return false, err
				}
				drifted = drifted || got != wants[d.Eve]
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
			var choice string
			activeProgress.suspended(func() {
				for _, d := range dests {
					if _, err := os.Stat(d.Dir); err == nil {
						showExpectedDiff(e, name, d)
					} else {
						fmt.Println("Missing skill destination: " + d.Dir)
					}
				}
				fmt.Printf("%s: [k]eep installed, [t]ake source, [s]kip? ", name)
				choice = strings.ToLower(strings.TrimSpace(stdinLine()))
			})
			switch choice {
			case "k":
				ok = installed(target, e, name) && ok
			case "t":
				if _, err := replaceSkill(target, e, name); err != nil {
					return false, err
				}
			default:
				ok = false
			}
		}
	}
	return ok, nil
}

func showExpectedDiff(e manifestEntry, name string, d skillDest) {
	if renamedTo(e, name) == "" && !d.Eve {
		showDiff(entrySkillDir(e, name), d.Dir)
		return
	}
	expected, cleanup, err := renderExpected(e, name, d.Eve)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer cleanup()
	showDiff(expected, d.Dir)
}

type destResult struct {
	Dir                string
	Agents             []string
	Linked, LinkFailed bool
}

func categoryConflictError(path string, skills []string) error {
	shown, more := skills, ""
	if len(shown) > 8 {
		shown, more = shown[:8], ", ..."
	}
	return fmt.Errorf("Refusing to replace %s: it is a category directory with nested skills (%s%s). Installing would delete them.", path, strings.Join(shown, ", "), more)
}

// replaceSkill backs up every destination, then renders name out of the entry's pinned store and links the agent folders.
// A global install also drops any skills.sh lock entry for name. A failed install restores the backups.
func replaceSkill(home string, e manifestEntry, name string) ([]destResult, error) {
	src := entrySkillDir(e, name)
	if src == "" {
		return nil, fmt.Errorf("skill %s has no source in pinned store %s", name, e.Path)
	}
	dests := entryDests(home, e, name)
	if len(dests) == 0 {
		return nil, fmt.Errorf("skill %s has nowhere to install: none of its agents (%s) supports this scope", name, strings.Join(e.Agents, ", "))
	}
	for _, d := range dests {
		if skills := categoryConflict(d.Dir); len(skills) > 0 {
			return nil, categoryConflictError(d.Dir, skills)
		}
	}
	backups := map[string]string{}
	restore := func() error {
		var result error
		for _, d := range dests {
			if backup, found := backups[d.Dir]; found {
				result = errors.Join(result, os.RemoveAll(d.Dir))
				result = errors.Join(result, os.Rename(backup, d.Dir))
			}
		}
		return result
	}
	for _, d := range dests {
		if _, err := os.Lstat(d.Dir); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, errors.Join(err, restore())
		}
		backup := backupPath(d.Dir, home)
		if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
			return nil, errors.Join(err, restore())
		}
		if err := os.Rename(d.Dir, backup); err != nil {
			return nil, errors.Join(err, restore())
		}
		backups[d.Dir] = backup
		fmt.Println("Backed up to " + backup)
	}
	opt := renderOpts{Rename: renamedTo(e, name)}
	var results []destResult
	var installErr error
	for _, d := range dests {
		if d.LinkTo != "" {
			continue
		}
		copied := opt
		copied.Eve = d.Eve
		if installErr = renderSkill(src, d.Dir, copied); installErr != nil {
			break
		}
		results = append(results, destResult{Dir: d.Dir, Agents: d.Agents})
	}
	for _, d := range dests {
		if installErr != nil {
			break
		}
		if d.LinkTo == "" {
			continue
		}
		result := destResult{Dir: d.Dir, Agents: d.Agents, Linked: true}
		if linkSkill(d.LinkTo, d.Dir) != nil {
			result.Linked, result.LinkFailed = false, true
			if installErr = errors.Join(os.RemoveAll(d.Dir), renderSkill(src, d.Dir, opt)); installErr != nil {
				break
			}
		}
		results = append(results, result)
	}
	if installErr == nil && e.Project == "" {
		installErr = dropSkillLockEntry(home, name)
	}
	if installErr == nil {
		if err := recordDests(e, entryHome(home, e), name, destDirs(dests)); err != nil {
			return results, err
		}
		return results, recordOwnership(home, e.Owner, []string{name}, nil, e.Path)
	}
	for _, d := range dests {
		installErr = errors.Join(installErr, os.RemoveAll(d.Dir))
	}
	return nil, errors.Join(installErr, restore())
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

// findSkillDir returns the folder named name that holds a regular SKILL.md, searching each base under checkout in order.
// Within one base the shallowest match wins, and a tie goes to the first path in lexical order,
// so a repo that ships several copies of a skill always resolves to the same one.
func findSkillDir(checkout, name string, bases ...string) string {
	root, err := os.OpenRoot(checkout)
	if err != nil {
		return ""
	}
	defer root.Close()
	for _, base := range bases {
		found, best := "", -1
		var walk func(dir string, depth int)
		walk = func(dir string, depth int) {
			if best >= 0 && depth >= best {
				return
			}
			entries, err := readDirOnly(root, dir)
			if err != nil {
				return
			}
			slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
			if best < 0 || depth < best {
				for _, entry := range entries {
					if entry.Name() == "SKILL.md" && entry.Type().IsRegular() && path.Base(dir) == name {
						found, best = dir, depth
						break
					}
				}
			}
			for _, entry := range entries {
				if entry.IsDir() && entry.Name() != "node_modules" && entry.Name() != ".git" {
					walk(path.Join(dir, entry.Name()), depth+1)
				}
			}
		}
		walk(base, 0)
		if found != "" {
			return filepath.Join(checkout, filepath.FromSlash(found))
		}
	}
	return ""
}

func localSkillDir(repoRoot, name string) string {
	return findSkillDir(repoRoot, name, "skills", ".")
}

func entrySkillDir(e manifestEntry, name string) string {
	want := cmp.Or(e.From[name], name)
	var dir string
	switch {
	case e.Discover:
		dir = discoverByName(e.Path, e.SkillPath, want)
	case e.SkillPath != "":
		dir = findSkillDir(e.Path, want, e.SkillPath)
	default:
		dir = localSkillDir(e.Path, want)
	}
	if dir == "" || !insideCheckout(e.Path, dir) {
		return ""
	}
	return dir
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

type skillDest struct {
	Dir, LinkTo string
	Eve         bool
	Agents      []string
}

func skillDests(root string, global bool, name string, e manifestEntry, env agentEnv) []skillDest {
	var out []skillDest
	if len(e.Agents) == 0 {
		for _, dir := range skillDirs(root, name) {
			out = append(out, skillDest{Dir: dir})
		}
		return out
	}
	add := func(d skillDest) {
		for i := range out {
			if out[i].Dir == d.Dir {
				out[i].Agents = appendMissing(out[i].Agents, d.Agents...)
				return
			}
		}
		out = append(out, d)
	}
	canonical := filepath.Join(root, ".agents", "skills", name)
	for _, id := range e.Agents {
		subagents := []string{""}
		if id == "eve" {
			subagents = eveSubagents(e.Subagents)
		}
		for _, sub := range subagents {
			base := agentBaseDir(id, global, root, sub, env)
			if base == "" {
				continue
			}
			agentDir := filepath.Join(base, name)
			eve := id == "eve"
			if e.Copy {
				add(skillDest{Dir: agentDir, Eve: eve, Agents: []string{id}})
				continue
			}
			target := canonical
			if eve {
				target = agentDir
			}
			add(skillDest{Dir: target, Eve: eve})
			if global && readsCanonicalGlobally(id) || agentDir == target {
				add(skillDest{Dir: target, Agents: []string{id}})
			} else {
				add(skillDest{Dir: agentDir, LinkTo: target, Agents: []string{id}})
			}
		}
	}
	return out
}

func entryDests(home string, e manifestEntry, name string) []skillDest {
	return skillDests(entryHome(home, e), e.Project == "", name, e, currentAgentEnv())
}

func destDirs(dests []skillDest) []string {
	dirs := make([]string, len(dests))
	for i, d := range dests {
		dirs[i] = d.Dir
	}
	return dirs
}

func addDests(entry *stateEntry, root, name string, dirs []string) {
	if entry.Dests == nil {
		entry.Dests = map[string]map[string][]string{}
	}
	if entry.Dests[root] == nil {
		entry.Dests[root] = map[string][]string{}
	}
	merged := append(slices.Clone(entry.Dests[root][name]), dirs...)
	slices.Sort(merged)
	entry.Dests[root][name] = slices.Compact(merged)
}

func recordDests(e manifestEntry, root, name string, dirs []string) error {
	if e.ID == "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	state, err := readState(home)
	if err != nil {
		return err
	}
	entry := state.Components[e.ID]
	if len(e.Agents) == 0 && len(entry.Dests[root][name]) == 0 {
		return nil
	}
	backfillLegacyTrees(&entry, e, root, name)
	addInstalledDests(&entry, root, name, e.Project != "", dirs)
	noteTrees(&entry, dirs)
	state.Components[e.ID] = entry
	return writeState(home, state)
}

func addInstalledDests(entry *stateEntry, root, name string, project bool, dirs []string) {
	owned := project && slices.Contains(entry.ProjectSkills[root], name) || !project && slices.Contains(entry.Skills, name)
	if owned && len(entry.Dests[root][name]) == 0 {
		dirs = append(skillDirs(root, name), dirs...)
	}
	addDests(entry, root, name, dirs)
	if project {
		addProjectSkill(entry, root, name)
	}
}

func backfillLegacyTrees(entry *stateEntry, e manifestEntry, root, name string) {
	commit := entry.SkillCopies[name]
	if e.Project != "" || commit == "" || e.From[name] != "" || len(entry.Dests[root][name]) > 0 || !slices.Contains(entry.Skills, name) {
		return
	}
	earlier := e
	earlier.Path = filepath.Join(filepath.Dir(e.Path), commit)
	earlier.Discover = !hasUtilityManifest(earlier.Path)
	if !verifiedStore(earlier.Path, commit) || entrySkillDir(earlier, name) == "" {
		return
	}
	tree, err := expectedTree(earlier, name, false)
	if err != nil {
		return
	}
	for _, dir := range skillDirs(root, name) {
		if info, err := os.Lstat(dir); err == nil && info.IsDir() {
			if entry.DestTrees == nil {
				entry.DestTrees = map[string]string{}
			}
			entry.DestTrees[realOr(dir)] = tree
		}
	}
}

func addProjectSkill(entry *stateEntry, root, name string) {
	if slices.Contains(entry.ProjectSkills[root], name) {
		return
	}
	if entry.ProjectSkills == nil {
		entry.ProjectSkills = map[string][]string{}
	}
	entry.ProjectSkills[root] = append(entry.ProjectSkills[root], name)
	slices.Sort(entry.ProjectSkills[root])
}

func shareDests(home, from, to, root, name string, project bool) error {
	state, err := readState(home)
	if err != nil {
		return err
	}
	entry := state.Components[to]
	dirs := ownedDests(state, from, root, name)
	addInstalledDests(&entry, root, name, project, dirs)
	for _, dir := range dirs {
		key := realOr(dir)
		if tree := state.Components[from].DestTrees[key]; tree != "" && entry.DestTrees[key] == "" {
			if entry.DestTrees == nil {
				entry.DestTrees = map[string]string{}
			}
			entry.DestTrees[key] = tree
		}
	}
	state.Components[to] = entry
	return writeState(home, state)
}

func ownedDests(state stateFile, id, root, name string) []string {
	if dirs := state.Components[id].Dests[root][name]; len(dirs) > 0 {
		return dirs
	}
	return skillDirs(root, name)
}

func retireStaleDests(root string, e manifestEntry, name string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	state, err := readState(home)
	if err != nil {
		return err
	}
	entry := state.Components[e.ID]
	recorded := entry.Dests[root][name]
	project := e.Project != ""
	needed := map[string]bool{}
	for _, dir := range destDirs(entryDests(root, e, name)) {
		needed[realOr(dir)] = true
	}
	for oid, other := range state.Components {
		owns := project && slices.Contains(other.ProjectSkills[root], name) || !project && slices.Contains(other.Skills, name)
		if oid != e.ID && owns {
			for _, dir := range ownedDests(state, oid, root, name) {
				needed[realOr(dir)] = true
			}
		}
	}
	if !slices.ContainsFunc(recorded, func(dir string) bool { return !needed[realOr(dir)] }) || !layoutReady(e, root, name) || !installed(root, e, name) || entryDriftCount(e, root, name) != 0 {
		return nil
	}
	var keep []string
	for _, dir := range recorded {
		if needed[realOr(dir)] {
			keep = append(keep, dir)
			continue
		}
		info, err := os.Lstat(dir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		key := realOr(dir)
		if info.Mode()&os.ModeSymlink == 0 {
			if tree, err := treeHash(dir); err != nil || entry.DestTrees[key] == "" || tree != entry.DestTrees[key] {
				keep = append(keep, dir)
				continue
			}
		}
		backup := backupPath(dir, root)
		if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
			return err
		}
		if err := os.Rename(dir, backup); err != nil {
			return err
		}
		fmt.Println("Backed up to " + backup)
		if info.Mode()&os.ModeSymlink == 0 {
			delete(entry.DestTrees, key)
		}
	}
	if len(keep) == len(recorded) {
		return nil
	}
	entry.Dests[root][name] = keep
	state.Components[e.ID] = entry
	return writeState(home, state)
}

func noteTrees(entry *stateEntry, dirs []string) {
	for _, dir := range dirs {
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		tree, err := treeHash(dir)
		if err != nil {
			continue
		}
		if entry.DestTrees == nil {
			entry.DestTrees = map[string]string{}
		}
		entry.DestTrees[realOr(dir)] = tree
	}
}

func layoutReady(e manifestEntry, root, name string) bool {
	dests := entryDests(root, e, name)
	if len(dests) == 0 {
		return false
	}
	for _, d := range dests {
		info, err := os.Lstat(d.Dir)
		if err != nil || wrongLayout(d, info) {
			return false
		}
	}
	return true
}

func wrongLayout(d skillDest, info fs.FileInfo) bool {
	linked := info.Mode()&os.ModeSymlink != 0
	return linked != (d.LinkTo != "") || linked && realOr(d.Dir) != realOr(d.LinkTo)
}

func ownedDrift(e manifestEntry, root, name string, dirs []string) bool {
	eve := map[string]bool{}
	for _, d := range entryDests(root, e, name) {
		eve[d.Dir] = d.Eve
	}
	wants := map[bool]string{}
	for _, dir := range dirs {
		want, cached := wants[eve[dir]]
		if !cached {
			var err error
			if want, err = expectedTree(e, name, eve[dir]); err != nil {
				return true
			}
			wants[eve[dir]] = want
		}
		if got, err := treeHash(dir); err != nil || got != want {
			return true
		}
	}
	return false
}

func incompleteUnedited(e manifestEntry, home, name string) bool {
	incomplete := false
	wants := map[bool]string{}
	for _, d := range entryDests(home, e, name) {
		info, err := os.Lstat(d.Dir)
		if errors.Is(err, os.ErrNotExist) {
			incomplete = true
			continue
		}
		if err != nil {
			return false
		}
		if wrongLayout(d, info) {
			incomplete = true
		}
		want, cached := wants[d.Eve]
		if !cached {
			if want, err = expectedTree(e, name, d.Eve); err != nil {
				return false
			}
			wants[d.Eve] = want
		}
		if got, err := treeHash(d.Dir); err != nil || got != want {
			return false
		}
	}
	return incomplete
}

// shadowWarnings reports harness-only skill copies that differ from the managed copy.
// Empryo's copy wins, so it is only reported. jcode's copy is refreshed from the store first.
func shadowWarnings(home string, e manifestEntry, name string) []string {
	dests := entryDests(home, e, name)
	if len(dests) == 0 {
		return nil
	}
	managed := dests[0].Dir
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

var renameJcodeShadow = os.Rename

func refreshJcodeShadows(home string, e manifestEntry, name string) error {
	if len(entryDests(home, e, name)) == 0 {
		return nil
	}
	base, err := harnessHome(harnessNamed("jcode"), home)
	if err != nil {
		return nil
	}
	shadow := filepath.Join(base, "skills", name)
	if info, err := os.Stat(shadow); err != nil || !info.IsDir() {
		return nil
	}
	if info, err := os.Stat(filepath.Join(shadow, "SKILL.md")); err != nil || !info.Mode().IsRegular() {
		return nil
	}
	if sameFile(shadow, filepath.Join(home, ".empryo", "skills", name)) {
		return nil
	}
	for _, d := range entryDests(home, e, name) {
		if sameFile(shadow, d.Dir) {
			return nil
		}
	}
	rendered, cleanup, err := renderExpected(e, name, false)
	if err != nil {
		return err
	}
	defer cleanup()
	if !differs(shadow, rendered) {
		return nil
	}
	stage, err := os.MkdirTemp(filepath.Dir(shadow), ".gap-refresh-")
	if err != nil {
		return err
	}
	staged := filepath.Join(stage, "new")
	if err := renderSkill(rendered, staged, renderOpts{Quiet: true}); err != nil {
		os.RemoveAll(stage)
		return err
	}
	saved := filepath.Join(stage, "saved")
	if err := renameJcodeShadow(shadow, saved); err != nil {
		os.RemoveAll(stage)
		return err
	}
	if err := renameJcodeShadow(staged, shadow); err != nil {
		if back := renameJcodeShadow(saved, shadow); back != nil {
			return errors.Join(err, fmt.Errorf("the old copy is still at %s: %w", saved, back))
		}
		os.RemoveAll(stage)
		return err
	}
	if err := os.RemoveAll(stage); err != nil {
		return fmt.Errorf("refreshed %s, but could not remove staging directory %s: %w", shadow, stage, err)
	}
	return nil
}

func sameFile(a, b string) bool {
	left, err := os.Stat(a)
	if err != nil {
		return false
	}
	right, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(left, right)
}

func differs(shadow, managed string) bool {
	if info, err := os.Stat(shadow); err != nil || !info.IsDir() {
		return false
	}
	got, err := contentHash(shadow)
	want, werr := contentHash(managed)
	return err != nil || werr != nil || got != want
}

func installed(home string, e manifestEntry, name string) bool {
	dests := entryDests(home, e, name)
	if len(dests) == 0 {
		return false
	}
	for _, d := range dests {
		if info, err := os.Stat(filepath.Join(d.Dir, "SKILL.md")); err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
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
