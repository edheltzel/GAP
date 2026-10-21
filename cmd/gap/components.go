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
	Components map[string]component
}

type component struct {
	ID     string
	Repo   string
	Ref    string
	Skills []string
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
		if c.Repo == "" || c.Ref == "" {
			return distro{}, fmt.Errorf("gap.toml: component %s needs nonempty string repo and ref", id)
		}
		c.Skills, ok = table["names"].([]string)
		if !ok {
			return distro{}, fmt.Errorf("gap.toml: component %s names must be a string array", id)
		}
		sort.Strings(c.Skills)
		d.Components[id] = c
	}
	return d, nil
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

// worktreeID is the tree of the files on disk, not the commit. A tampered store must not match.
func worktreeID(dir string) (string, error) {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return "", err
	}
	return treeHash(dir)
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

func loadComponents(repoRoot, home string) ([]manifestEntry, error) {
	d, err := readDistro(repoRoot)
	if err != nil {
		return nil, err
	}
	if len(d.Components) == 0 {
		return nil, nil
	}
	lk, err := readLock(repoRoot)
	if err != nil {
		return nil, err
	}
	pins, err := lockFor(d, lk)
	if err != nil {
		return nil, err
	}
	return componentEntries(home, d, pins)
}

func statusAll() error {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return err
	}
	d, lk, err := readPins(repoRoot)
	if err != nil {
		return err
	}
	bad := false
	ids := sortedIDs(d.Components)
	for _, id := range ids {
		c := d.Components[id]
		e, pinned := lk.Components[id]
		ref := c.Ref
		short := "unpinned"
		store := "missing"
		if pinned {
			ref = e.Ref
			short = e.Commit
			if len(short) > 7 {
				short = short[:7]
			}
			if tree, err := worktreeID(storePath(home, id, e.Commit)); err == nil && tree == e.Tree {
				store = "ok"
			} else {
				bad = true
			}
		} else {
			bad = true
		}
		fmt.Printf("%s ref=%s commit=%s store=%s\n", id, ref, short, store)
		if !pinned {
			continue
		}
		dir := storePath(home, id, e.Commit)
		for _, name := range c.Skills {
			n := skillDriftCount(dir, home, name)
			fmt.Printf("  %s drift=%d\n", name, n)
			if n > 0 {
				bad = true
			}
		}
	}
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
	d, lk, err := readPins(repoRoot)
	if err != nil {
		return err
	}
	if id != "" {
		if _, ok := d.Components[id]; !ok {
			return fmt.Errorf("component %s is not in gap.toml", id)
		}
	}
	npx, err := exec.LookPath("npx")
	if err != nil {
		return errors.New("npx not found; install Node to let gap restore skills")
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
		entry := manifestEntry{Source: componentSource(c.Repo), Skills: c.Skills, Path: dir}
		for _, name := range c.Skills {
			if skillDriftCount(dir, home, name) == 0 {
				continue
			}
			replaced, err := replaceSkill(npx, repoRoot, home, entry, name)
			if err != nil {
				return err
			}
			if !replaced {
				ok = false
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
func componentEntries(home string, d distro, pins map[string]lockEntryM1) ([]manifestEntry, error) {
	ids := make([]string, 0, len(d.Components))
	for id := range d.Components {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var entries []manifestEntry
	for _, id := range ids {
		c := d.Components[id]
		dir, err := ensureStore(home, c, pins[id])
		if err != nil {
			return nil, err
		}
		source := componentSource(c.Repo)
		entries = append(entries, manifestEntry{Source: source, Skills: c.Skills, Path: dir})
	}
	return entries, nil
}
