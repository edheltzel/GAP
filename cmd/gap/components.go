package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

type distro struct {
	Schema     string
	Name       string
	Profile    string
	Components map[string]component
}

type component struct {
	ID          string
	Repo        string
	Ref         string
	Skills      []string
	SkillSet    map[string]skillSpec
	Doctor      string
	Install     string
	Disabled    bool
	RefOverride bool
}

// skillSpec is one skill's load level. WhenFiles names the repo files that make a
// project skill install. Empty Load means listed, which is what M1 installed.
type skillSpec struct {
	Load      string
	WhenFiles []string
}

type lockFile struct {
	Components map[string]lockEntryM1
}

type lockEntryM1 struct {
	Ref, Commit, Tree string
}

func readDistro(repoRoot string) (distro, error) {
	path := filepath.Join(repoRoot, "gap.toml")
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return distro{Components: map[string]component{}}, nil
	}
	if err != nil {
		return distro{}, err
	}
	doc, err := parseTOML(string(b))
	if err != nil {
		return distro{}, fmt.Errorf("gap.toml: %w", err)
	}
	d := distro{Components: map[string]component{}}
	d.Schema, _ = doc["schema"].(string)
	if d.Schema != "2" {
		return distro{}, fmt.Errorf("gap.toml: schema must be \"2\"")
	}
	if raw, exists := doc["name"]; exists {
		var ok bool
		d.Name, ok = raw.(string)
		if !ok {
			return distro{}, fmt.Errorf("gap.toml: name must be a string")
		}
	}
	tables, ok := doc["components"].(map[string]any)
	if _, exists := doc["components"]; exists && !ok {
		return distro{}, fmt.Errorf("gap.toml: components must be a table")
	}
	for id, raw := range tables {
		if strings.HasPrefix(id, "\x00") {
			continue
		}
		table, ok := raw.(map[string]any)
		if !ok {
			return distro{}, fmt.Errorf("gap.toml: component %s must be a table", id)
		}
		c := component{ID: id}
		c.Repo, _ = table["repo"].(string)
		c.Ref, _ = table["ref"].(string)
		c.Doctor, _ = table["doctor"].(string)
		c.Install, _ = table["install"].(string)
		if c.Repo == "" || c.Ref == "" {
			return distro{}, fmt.Errorf("gap.toml: component %s needs nonempty string repo and ref", id)
		}
		c.Skills, c.SkillSet, err = componentSkills(table)
		if err != nil {
			return distro{}, fmt.Errorf("gap.toml: component %s: %w", id, err)
		}
		sort.Strings(c.Skills)
		d.Components[id] = c
	}
	return d, nil
}

// componentSkills reads names plus an optional [components.<id>.skills] table.
// names is the install list. The skills table only sets a load level for a name already in it.
func componentSkills(table map[string]any) ([]string, map[string]skillSpec, error) {
	raw, ok := table["names"]
	if !ok {
		return nil, nil, fmt.Errorf("names must be a string array")
	}
	list, ok := raw.([]string)
	if !ok {
		return nil, nil, fmt.Errorf("names must be a string array")
	}
	seen := map[string]bool{}
	var names []string
	for _, name := range list {
		if name == "" || seen[name] {
			return nil, nil, fmt.Errorf("names has an empty or repeated entry")
		}
		seen[name] = true
		names = append(names, name)
	}
	set := map[string]skillSpec{}
	raw, exists := table["skills"]
	if !exists {
		return names, set, nil
	}
	skills, ok := raw.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("skills must be a table")
	}
	for name, value := range skills {
		if strings.HasPrefix(name, "\x00") {
			continue
		}
		spec, ok := value.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("skill %s must be a table", name)
		}
		s := skillSpec{}
		if load, ok := spec["load"].(string); ok {
			switch load {
			case "always", "listed", "project", "command", "off":
				s.Load = load
			default:
				return nil, nil, fmt.Errorf("skill %s load must be always, listed, project, command, or off", name)
			}
		}
		if when, ok := spec["when"].(map[string]any); ok {
			if files, ok := when["files"].([]string); ok {
				s.WhenFiles = files
			} else if _, present := when["files"]; present {
				return nil, nil, fmt.Errorf("skill %s when.files must be a string array", name)
			}
		}
		if !seen[name] {
			return nil, nil, fmt.Errorf("skill %s is not in names", name)
		}
		set[name] = s
	}
	return names, set, nil
}

func readLock(repoRoot string) (lockFile, error) {
	path := filepath.Join(repoRoot, "gap.lock")
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return lockFile{Components: map[string]lockEntryM1{}}, nil
	}
	if err != nil {
		return lockFile{}, err
	}
	doc, err := parseTOML(string(b))
	if err != nil {
		return lockFile{}, fmt.Errorf("gap.lock: %w", err)
	}
	lk := lockFile{Components: map[string]lockEntryM1{}}
	if doc["schema"] != "2" {
		return lockFile{}, fmt.Errorf("gap.lock: schema must be \"2\"")
	}
	tables, ok := doc["components"].(map[string]any)
	if _, exists := doc["components"]; exists && !ok {
		return lockFile{}, fmt.Errorf("gap.lock: components must be a table")
	}
	for id, raw := range tables {
		if strings.HasPrefix(id, "\x00") {
			continue
		}
		table, ok := raw.(map[string]any)
		if !ok {
			return lockFile{}, fmt.Errorf("gap.lock: component %s must be a table", id)
		}
		e := lockEntryM1{}
		e.Ref, _ = table["ref"].(string)
		if e.Ref == "" {
			return lockFile{}, fmt.Errorf("gap.lock: %s ref must be a nonempty string", id)
		}
		e.Commit, _ = table["commit"].(string)
		e.Tree, _ = table["tree"].(string)
		if !isHexLen(e.Commit, 40) {
			return lockFile{}, fmt.Errorf("gap.lock: %s commit must be 40 hex digits", id)
		}
		if !isHexLen(e.Tree, 40) {
			return lockFile{}, fmt.Errorf("gap.lock: %s tree must be 40 hex digits", id)
		}
		lk.Components[id] = e
	}
	return lk, nil
}

func writeLock(repoRoot string, lk lockFile) error {
	if lk.Components == nil {
		lk.Components = map[string]lockEntryM1{}
	}
	return writeAtomic(filepath.Join(repoRoot, "gap.lock"), writeLockTOML(lk.Components))
}

// lockFor reports the pin for every distro component. A missing pin names `gap pin`.
func lockFor(d distro, lk lockFile) (map[string]lockEntryM1, error) {
	out := map[string]lockEntryM1{}
	for id := range d.Components {
		e, ok := lk.Components[id]
		if !ok {
			c := d.Components[id]
			return nil, fmt.Errorf("component %s is not pinned; run `gap pin %s@%s`", id, id, c.Ref)
		}
		out[id] = e
	}
	return out, nil
}

// cloneURL turns a component repo into something git can fetch.
// github.com/owner/repo becomes https://github.com/owner/repo.git.
// file:// and an absolute path are kept for tests.
func cloneURL(repo string) (string, error) {
	if strings.HasPrefix(repo, "file://") || filepath.IsAbs(repo) {
		return repo, nil
	}
	owner, _, ok := githubIdentity(repo)
	if !ok {
		return "", fmt.Errorf("component repo must be github.com/owner/repo, a file:// URL, or an absolute path: %s", repo)
	}
	// githubIdentity returns owner/repo with no host. Require the host form the distro writes.
	if !strings.HasPrefix(repo, "github.com/") {
		return "", fmt.Errorf("component repo must be github.com/owner/repo, a file:// URL, or an absolute path: %s", repo)
	}
	return "https://github.com/" + owner + ".git", nil
}

func storePath(home, id, commit string) string {
	return filepath.Join(home, ".gap", "store", id, commit)
}

// componentSource is owner/repo for a GitHub component and "." for a local fixture.
// "." installs through the store path the way a checkout skill does.
func componentSource(repo string) string {
	owner, _, ok := githubIdentity(repo)
	if ok && strings.HasPrefix(repo, "github.com/") {
		return owner
	}
	return "."
}

// ensureStore returns a verified checkout of l.Commit. A dir whose tree matches is reused with no fetch.
func ensureStore(home string, c component, l lockEntryM1) (string, error) {
	if !isHexLen(l.Commit, 40) || !isHexLen(l.Tree, 40) {
		return "", fmt.Errorf("component %s: commit and tree must be 40 hex digits", c.ID)
	}
	dest := storePath(home, c.ID, l.Commit)
	if tree, err := worktreeID(dest); err == nil && tree == l.Tree {
		return dest, nil
	}
	tmp, err := fetchCommit(home, c, l)
	if err != nil {
		return "", err
	}
	keep := false
	defer func() {
		if !keep {
			os.RemoveAll(tmp)
		}
	}()
	tree, err := git(tmp, "rev-parse", "HEAD^{tree}")
	if err != nil || tree != l.Tree {
		return "", fmt.Errorf("component %s: tree %s does not match lock %s", c.ID, tree, l.Tree)
	}
	if err := placeStore(tmp, dest); err != nil {
		return "", err
	}
	keep = true
	got, err := git(dest, "rev-parse", "HEAD^{tree}")
	if err != nil || got != l.Tree {
		return "", fmt.Errorf("component %s: store tree does not match lock", c.ID)
	}
	return dest, nil
}

var storeRename = os.Rename

// placeStore moves a verified fetch into dest. A previous dest is set aside first and put back if the move fails.
func placeStore(tmp, dest string) error {
	aside := ""
	if _, err := os.Lstat(dest); err == nil {
		aside = fmt.Sprintf("%s.old-%d", dest, time.Now().UnixNano())
		if err := os.Rename(dest, aside); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := storeRename(tmp, dest); err != nil {
		if aside != "" {
			if rerr := os.Rename(aside, dest); rerr != nil {
				return fmt.Errorf("%w; restore %s: %v", err, aside, rerr)
			}
		}
		return err
	}
	if aside != "" {
		os.RemoveAll(aside)
	}
	return nil
}

// worktreeID hashes tracked files and non-ignored additions, not just HEAD.
func worktreeID(dir string) (string, error) {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp("", "gap-store-index-")
	if err != nil {
		return "", fmt.Errorf("creating store index: %w", err)
	}
	defer os.RemoveAll(tmp)
	objectDir := filepath.Join(tmp, "objects")
	if err := os.Mkdir(objectDir, 0o700); err != nil {
		return "", fmt.Errorf("creating store objects: %w", err)
	}
	env := append(gitEnv(), "GIT_INDEX_FILE="+filepath.Join(tmp, "index"))
	var tree string
	for _, args := range [][]string{{"read-tree", "HEAD"}, {"add", "-A", "--", "."}, {"write-tree"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir, cmd.Env = dir, env
		if args[0] != "read-tree" {
			cmd.Env = append(cmd.Env, "GIT_OBJECT_DIRECTORY="+objectDir)
		}
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("hashing store %s: git %s: %w", dir, args[0], err)
		}
		tree = strings.TrimSpace(string(out))
	}
	return tree, nil
}

const (
	storeOK       = "ok"
	storeMissing  = "missing"
	storeChanged  = "changed"
	storeUnpinned = "unpinned"
)

// storeState names why a pinned checkout does not match its lock. A directory
// that is not there is missing; one whose files differ from the locked tree is
// changed. ok means the checkout matches.
func storeState(dir, tree string) (string, error) {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return storeMissing, nil
	} else if err != nil {
		return "", err
	}
	got, err := worktreeID(dir)
	if err != nil {
		return "", fmt.Errorf("reading store %s: %w", dir, err)
	}
	if got != tree {
		return storeChanged, nil
	}
	return storeOK, nil
}

// resolveRef returns the commit a ref names. A 40-hex ref is that commit.
func resolveRef(url, ref string) (string, error) {
	if isHexLen(ref, 40) {
		return ref, nil
	}
	refs := []string{ref}
	if ref != "HEAD" && !strings.HasPrefix(ref, "refs/") {
		refs = []string{"refs/heads/" + ref, "refs/tags/" + ref}
	}
	args := []string{"ls-remote", "--", url}
	for _, name := range refs {
		args = append(args, name, name+"^{}")
	}
	out, err := git(".", args...)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", ref, err)
	}
	commits := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || !isHexLen(fields[0], 40) {
			continue
		}
		name := strings.TrimSuffix(fields[1], "^{}")
		if !slices.Contains(refs, name) {
			continue
		}
		if _, found := commits[name]; !found || fields[1] != name {
			commits[name] = fields[0]
		}
	}
	if len(commits) > 1 {
		return "", fmt.Errorf("resolve %s: ambiguous ref; use refs/heads/ or refs/tags/", ref)
	}
	for _, commit := range commits {
		return commit, nil
	}
	return "", fmt.Errorf("resolve %s: no matching ref", ref)
}

func effectivePins(repoRoot, home, projectRoot, profile string) (distro, lockFile, error) {
	d, lk, err := readPins(repoRoot)
	if err != nil {
		return distro{}, lockFile{}, err
	}
	d, err = applyOverrides(d, home, repoRoot, projectRoot, profile)
	if err != nil {
		return distro{}, lockFile{}, err
	}
	for id, c := range d.Components {
		if pin, exists := lk.Components[id]; exists && c.RefOverride && c.Ref != pin.Ref {
			return distro{}, lockFile{}, fmt.Errorf("component %s requests ref %s but gap.lock pins %s; run `gap pin %s@%s`", id, c.Ref, pin.Ref, id, c.Ref)
		}
	}
	return d, lk, nil
}

func invokingProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if root, err := git(cwd, "rev-parse", "--show-toplevel"); err == nil {
		return root, nil
	}
	return cwd, nil
}

func statusAll() error {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return err
	}
	projectRoot, err := invokingProjectRoot()
	if err != nil {
		return err
	}
	d, lk, err := effectivePins(repoRoot, home, projectRoot, "")
	if err != nil {
		return err
	}
	out := newReport(os.Stdout)
	if d.Profile == "crew" {
		out.heading("crew: project files only")
		return nil
	}
	bad := false
	missing := false
	changed := false
	var pinSteps []string
	ids := sortedIDs(d.Components)
	for _, id := range ids {
		c := d.Components[id]
		e, pinned := lk.Components[id]
		ref := c.Ref
		short := "unpinned"
		store := storeUnpinned
		if pinned {
			ref = e.Ref
			short = e.Commit
			if len(short) > 7 {
				short = short[:7]
			}
			var err error
			store, err = storeState(storePath(home, id, e.Commit), e.Tree)
			if err != nil {
				return err
			}
			bad = bad || store != storeOK
		} else {
			bad = true
			pinSteps = append(pinSteps, "gap pin "+id+"@"+c.Ref+"    create the missing lock entry")
		}
		if store == storeChanged {
			changed = true
		}
		if store == storeMissing {
			missing = true
		}
		out.statusLine(id, ref, short, store)
		if store != storeOK {
			continue
		}
		dir := storePath(home, id, e.Commit)
		var rows [][]string
		for _, entry := range selectedEntries(c, dir, projectRoot) {
			for _, name := range entry.Skills {
				n := skillDriftCount(dir, entryHome(home, entry), name)
				rows = append(rows, []string{name, fmt.Sprintf("drift=%d", n)})
				if n > 0 {
					bad = true
				}
			}
		}
		if len(rows) > 0 {
			out.driftTable(rows)
		}
	}
	var steps []string
	steps = append(steps, pinSteps...)
	if missing {
		steps = append(steps, "gap sync    fetch each missing source checkout and install it")
	}
	if changed {
		steps = append(steps, "gap sync    replace changed source checkouts from the lock; inspect local changes first")
	}
	out.next(steps...)
	if bad {
		return errSkipped
	}
	return nil
}

func skillDriftCount(store, home, name string) int {
	src := localSkillDir(store, name)
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

func restoreAll(id string) error {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return err
	}
	projectRoot, err := invokingProjectRoot()
	if err != nil {
		return err
	}
	d, lk, err := effectivePins(repoRoot, home, projectRoot, "")
	if err != nil {
		return err
	}
	if d.Profile == "crew" {
		return nil
	}
	if id != "" {
		if _, ok := d.Components[id]; !ok {
			return fmt.Errorf("component %s is not in gap.toml", id)
		}
	}
	ok := true
	for _, cid := range sortedIDs(d.Components) {
		if id != "" && cid != id {
			continue
		}
		c := d.Components[cid]
		e, pinned := lk.Components[cid]
		if !pinned {
			return fmt.Errorf("component %s is not pinned; run `gap pin %s@%s`", cid, cid, c.Ref)
		}
		dir, err := ensureStore(home, c, e)
		if err != nil {
			return err
		}
		for _, entry := range selectedEntries(c, dir, projectRoot) {
			target := entryHome(home, entry)
			for _, name := range entry.Skills {
				if skillDriftCount(dir, target, name) == 0 {
					continue
				}
				replaced, err := replaceSkill("", repoRoot, target, entry, name)
				if err != nil || !replaced {
					fmt.Fprintf(os.Stderr, "Restoring %s: %v\n", name, err)
					ok = false
				}
			}
		}
	}
	if !ok {
		return errSkipped
	}
	return nil
}

func pinComponent(spec string) error {
	id, ref, ok := strings.Cut(spec, "@")
	if !ok || id == "" || ref == "" || strings.Contains(ref, "@") {
		return fmt.Errorf("usage: gap pin <id>@<ref>")
	}
	home, repoRoot, err := gapHomes()
	if err != nil {
		return err
	}
	d, err := readDistro(repoRoot)
	if err != nil {
		return err
	}
	c, found := d.Components[id]
	if !found {
		return fmt.Errorf("component %s is not in gap.toml", id)
	}
	url, err := cloneURL(c.Repo)
	if err != nil {
		return err
	}
	commit, err := resolveRef(url, ref)
	if err != nil {
		return err
	}
	lk, err := readLock(repoRoot)
	if err != nil {
		return err
	}
	tree := ""
	if prev, ok := lk.Components[id]; ok && prev.Commit == commit && isHexLen(prev.Tree, 40) {
		tree = prev.Tree
	}
	if tree == "" {
		probe := lockEntryM1{Ref: ref, Commit: commit, Tree: strings.Repeat("0", 40)}
		dir, err := fetchCommit(home, c, probe)
		if err != nil {
			return err
		}
		tree, err = git(dir, "rev-parse", "HEAD^{tree}")
		if err != nil || !isHexLen(tree, 40) {
			os.RemoveAll(dir)
			return fmt.Errorf("component %s: no tree for %s", id, commit)
		}
		if err := placeStore(dir, storePath(home, id, commit)); err != nil {
			os.RemoveAll(dir)
			return err
		}
	}
	if lk.Components == nil {
		lk.Components = map[string]lockEntryM1{}
	}
	lk.Components[id] = lockEntryM1{Ref: ref, Commit: commit, Tree: tree}
	return writeLock(repoRoot, lk)
}

func gapHomes() (home, repoRoot string, err error) {
	home, err = os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	repoRoot, err = sourceRoot()
	return home, repoRoot, err
}

func readPins(repoRoot string) (distro, lockFile, error) {
	d, err := readDistro(repoRoot)
	if err != nil {
		return distro{}, lockFile{}, err
	}
	lk, err := readLock(repoRoot)
	return d, lk, err
}

func sortedIDs(components map[string]component) []string {
	ids := make([]string, 0, len(components))
	for id := range components {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// fetchCommit checks out commit without knowing its tree. pin uses it, then records the tree.
func fetchCommit(home string, c component, l lockEntryM1) (string, error) {
	url, err := cloneURL(c.Repo)
	if err != nil {
		return "", err
	}
	parent := filepath.Join(home, ".gap", "store", c.ID)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(parent, "fetch-")
	if err != nil {
		return "", err
	}
	if _, err := git(tmp, "init", "-q"); err != nil {
		os.RemoveAll(tmp)
		return "", err
	}
	if _, err := git(tmp, "fetch", "--depth", "1", url, l.Commit); err != nil {
		if _, err2 := git(tmp, "fetch", url, l.Commit); err2 != nil {
			os.RemoveAll(tmp)
			return "", fmt.Errorf("component %s: fetch %s: %w", c.ID, l.Commit, err2)
		}
	}
	if _, err := git(tmp, "checkout", "--detach", "FETCH_HEAD"); err != nil {
		os.RemoveAll(tmp)
		return "", err
	}
	head, err := git(tmp, "rev-parse", "HEAD")
	if err != nil || head != l.Commit {
		os.RemoveAll(tmp)
		return "", fmt.Errorf("component %s: fetched %s, want %s", c.ID, head, l.Commit)
	}
	return tmp, nil
}

// componentEntries turns pinned components into manifest entries installed from the store.
func componentEntries(home, projectRoot string, d distro, pins map[string]lockEntryM1) ([]manifestEntry, error) {
	var entries []manifestEntry
	for _, id := range sortedIDs(d.Components) {
		c := d.Components[id]
		dir, err := ensureStore(home, c, pins[id])
		if err != nil {
			return nil, err
		}
		if c.Install != "" {
			if err := runComponentCommand(dir, c.Install); err != nil {
				return nil, fmt.Errorf("component %s install: %w", id, err)
			}
		}
		entries = append(entries, selectedEntries(c, dir, projectRoot)...)
	}
	return entries, nil
}

func selectedEntries(c component, store, projectRoot string) []manifestEntry {
	global := manifestEntry{Source: componentSource(c.Repo), Path: store}
	project := manifestEntry{Source: componentSource(c.Repo), Path: store, Project: projectRoot}
	for _, name := range c.Skills {
		if !renderLoad(c, name, projectRoot) {
			continue
		}
		if c.SkillSet[name].Load == "project" {
			project.Skills = append(project.Skills, name)
		} else {
			global.Skills = append(global.Skills, name)
		}
	}
	var entries []manifestEntry
	for _, entry := range []manifestEntry{global, project} {
		if len(entry.Skills) > 0 {
			entries = append(entries, entry)
		}
	}
	return entries
}
