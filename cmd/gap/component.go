package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	gapLockPath   = "gap.lock"
	gapTOMLPath   = "gap.toml"
	vendorDirName = ".vendor"
)

// component is one pinned git checkout recorded in gap.lock.
type component struct {
	Name, Repo, Ref, Commit, Tree, Source string
}

type skillPaths struct {
	repoRoot string
	bySource map[string]component
}

type componentStatus struct {
	Dir   string
	Head  string
	Pin   string
	Dirty bool
	Off   bool
}

func loadSkillPaths(repoRoot string) (skillPaths, error) {
	p := skillPaths{repoRoot: repoRoot, bySource: map[string]component{}}
	lockBytes, lockErr := os.ReadFile(filepath.Join(repoRoot, gapLockPath))
	tomlBytes, tomlErr := os.ReadFile(filepath.Join(repoRoot, gapTOMLPath))
	if errors.Is(lockErr, os.ErrNotExist) && errors.Is(tomlErr, os.ErrNotExist) {
		return p, nil
	}
	if errors.Is(lockErr, os.ErrNotExist) {
		if tomlErr == nil {
			return p, fmt.Errorf("%s: missing %s", gapTOMLPath, gapLockPath)
		}
		return p, tomlErr
	}
	if lockErr != nil {
		return p, lockErr
	}
	lock, err := parseTOML(lockBytes)
	if err != nil {
		return p, fmt.Errorf("%s: %w", gapLockPath, err)
	}
	if tomlString(lock, "schema") != "2" {
		return p, fmt.Errorf("%s: schema must be \"2\"", gapLockPath)
	}
	comps := tomlChild(lock, "components")
	if comps == nil {
		return p, fmt.Errorf("%s: missing [components] tables", gapLockPath)
	}
	byName := map[string]component{}
	for name, raw := range comps {
		spec, ok := raw.(tomlTable)
		if !ok {
			return p, fmt.Errorf("%s: component %s is not a table", gapLockPath, name)
		}
		c, err := componentFromTable(name, spec)
		if err != nil {
			return p, fmt.Errorf("%s: %w", gapLockPath, err)
		}
		p.bySource[c.Source] = c
		byName[name] = c
	}
	if errors.Is(tomlErr, os.ErrNotExist) {
		return p, nil
	}
	if tomlErr != nil {
		return p, tomlErr
	}
	req, err := parseTOML(tomlBytes)
	if err != nil {
		return p, fmt.Errorf("%s: %w", gapTOMLPath, err)
	}
	reqComps := tomlChild(req, "components")
	if reqComps == nil {
		return p, fmt.Errorf("%s: missing [components] tables", gapTOMLPath)
	}
	for name, raw := range reqComps {
		spec, ok := raw.(tomlTable)
		if !ok {
			return p, fmt.Errorf("%s: component %s is not a table", gapTOMLPath, name)
		}
		got, ok := byName[name]
		if !ok {
			return p, fmt.Errorf("%s: component %s is not in %s", gapTOMLPath, name, gapLockPath)
		}
		repo := tomlString(spec, "repo")
		ref := tomlString(spec, "ref")
		if repo == "" || ref == "" {
			return p, fmt.Errorf("%s: component %s needs repo and ref", gapTOMLPath, name)
		}
		if repo != got.Repo || ref != got.Ref {
			return p, fmt.Errorf("%s: component %s repo/ref does not match %s", gapTOMLPath, name, gapLockPath)
		}
	}
	return p, nil
}

func componentFromTable(name string, spec tomlTable) (component, error) {
	c := component{
		Name:   name,
		Repo:   tomlString(spec, "repo"),
		Ref:    tomlString(spec, "ref"),
		Commit: tomlString(spec, "commit"),
		Tree:   tomlString(spec, "tree"),
		Source: tomlString(spec, "source"),
	}
	if c.Repo == "" || c.Ref == "" || c.Commit == "" {
		return component{}, fmt.Errorf("component %s needs repo, ref, and commit", name)
	}
	if err := requirePinCommit(c); err != nil {
		return component{}, err
	}
	if c.Source == "" {
		c.Source = githubSourceFromURL(c.Repo)
	}
	if c.Source == "" {
		return component{}, fmt.Errorf("component %s needs source", name)
	}
	return c, nil
}

func githubSourceFromURL(url string) string {
	repo, _, _ := githubIdentity(url)
	return repo
}

func (p skillPaths) sourceRoot(source string) (string, bool) {
	if source == "." {
		return filepath.Clean(p.repoRoot), true
	}
	c, ok := p.bySource[source]
	if !ok {
		return "", false
	}
	return filepath.Clean(componentDir(p.repoRoot, c.Name)), true
}

func (p skillPaths) needed(entries []manifestEntry) []component {
	var out []component
	seen := map[string]bool{}
	for _, e := range entries {
		c, ok := p.bySource[e.Source]
		if !ok || seen[c.Name] {
			continue
		}
		seen[c.Name] = true
		out = append(out, c)
	}
	return out
}

func componentDir(repoRoot, name string) string {
	return filepath.Join(repoRoot, vendorDirName, name)
}

func localSourceRoot(repoRoot, source string) (string, bool) {
	paths, err := loadSkillPaths(repoRoot)
	if err != nil {
		if source == "." {
			return filepath.Clean(repoRoot), true
		}
		return "", false
	}
	return paths.sourceRoot(source)
}

func skillDir(repoRoot, source, name string) string {
	root, ok := localSourceRoot(repoRoot, source)
	if !ok {
		return ""
	}
	return localSkillDir(root, name)
}

func gitRun(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd = exec.Command("git", append([]string{"-C", dir}, args...)...)
	}
	cmd.Env = gitEnv()
	out, err := cmd.CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		if s != "" {
			return s, fmt.Errorf("%s", s)
		}
		return s, err
	}
	return s, nil
}

func isGitRepo(dir string) bool {
	_, err := gitRun(dir, "rev-parse", "--git-dir")
	return err == nil
}

func inspectComponent(dir, pin string) (componentStatus, error) {
	st := componentStatus{Dir: dir, Pin: pin}
	if !isGitRepo(dir) {
		return st, fmt.Errorf("pinned component at %s is not a git checkout", dir)
	}
	head, err := gitRun(dir, "rev-parse", "HEAD")
	if err != nil {
		return st, fmt.Errorf("reading HEAD in %s: %w", dir, err)
	}
	st.Head = head
	st.Off = head != pin
	porcelain, err := gitRun(dir, "status", "--porcelain")
	if err != nil {
		return st, fmt.Errorf("reading status in %s: %w", dir, err)
	}
	st.Dirty = porcelain != ""
	return st, nil
}

func reportComponentStatus(st componentStatus, name string) {
	if st.Dirty {
		fmt.Fprintf(os.Stderr, "Pinned component %s is dirty at %s\n", name, st.Dir)
	}
	if st.Off {
		fmt.Fprintf(os.Stderr, "Pinned component %s is at %s, not pinned %s\n", name, st.Head, st.Pin)
	}
}

func ensureNeeded(repoRoot string, paths skillPaths, entries []manifestEntry) bool {
	ok := true
	for _, c := range paths.needed(entries) {
		if err := ensureComponent(repoRoot, c); err != nil {
			fmt.Fprintln(os.Stderr, err)
			ok = false
		}
	}
	return ok
}

func requirePinCommit(c component) error {
	if len(c.Commit) != 40 {
		return fmt.Errorf("component %s commit must be a 40-character lowercase hex SHA", c.Name)
	}
	for i := 0; i < 40; i++ {
		ch := c.Commit[i]
		if ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' {
			continue
		}
		return fmt.Errorf("component %s commit must be a 40-character lowercase hex SHA", c.Name)
	}
	return nil
}

func ensureComponent(repoRoot string, c component) error {
	if err := requirePinCommit(c); err != nil {
		return err
	}
	dir := componentDir(repoRoot, c.Name)
	if !isGitRepo(dir) {
		if err := initComponent(dir, c); err != nil {
			return err
		}
	}
	origin, err := gitRun(dir, "remote", "get-url", "origin")
	if err != nil {
		return fmt.Errorf("pinned component %s: %w", c.Name, err)
	}
	if !sameGitURL(origin, c.Repo) {
		return fmt.Errorf("pinned component %s origin is %s, not %s", c.Name, origin, c.Repo)
	}
	st, err := inspectComponent(dir, c.Commit)
	if err == nil && !st.Dirty && !st.Off {
		return verifyComponentTree(dir, c)
	}
	if err == nil && st.Dirty {
		// Never reset a dirty checkout onto the pin.
		return nil
	}
	return checkoutPin(dir, c)
}

func initComponent(dir string, c component) error {
	if info, err := os.Stat(dir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("pinned component %s: %s exists and is not a directory", c.Name, dir)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf("pinned component %s: %s exists and is not a git checkout", c.Name, dir)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if _, err := gitRun(dir, "init", "-q"); err != nil {
		return fmt.Errorf("pinned component %s: init: %w", c.Name, err)
	}
	if _, err := gitRun(dir, "remote", "add", "origin", c.Repo); err != nil {
		return fmt.Errorf("pinned component %s: remote: %w", c.Name, err)
	}
	if err := checkoutPin(dir, c); err != nil {
		return err
	}
	fmt.Printf("using %s at %s\n", c.Source, c.Commit)
	return nil
}

func checkoutPin(dir string, c component) error {
	if err := requirePinCommit(c); err != nil {
		return err
	}
	if _, err := gitRun(dir, "fetch", "-q", "origin", c.Commit); err != nil {
		return fmt.Errorf("pinned component %s: fetch %s: %w", c.Name, c.Commit, err)
	}
	if _, err := gitRun(dir, "checkout", "-q", "--detach", c.Commit); err != nil {
		return fmt.Errorf("pinned component %s: checkout %s: %w", c.Name, c.Commit, err)
	}
	return verifyComponentTree(dir, c)
}

func verifyComponentTree(dir string, c component) error {
	if c.Tree == "" {
		return nil
	}
	tree, err := gitRun(dir, "rev-parse", "HEAD^{tree}")
	if err != nil {
		return fmt.Errorf("pinned component %s: tree: %w", c.Name, err)
	}
	if tree != c.Tree {
		return fmt.Errorf("pinned component %s tree is %s, not %s", c.Name, tree, c.Tree)
	}
	return nil
}

func sameGitURL(a, b string) bool {
	return normalizeGitURL(a) == normalizeGitURL(b)
}

func normalizeGitURL(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")
	if filepath.IsAbs(s) {
		return filepath.Clean(s)
	}
	return s
}

func sourceInstallable(repoRoot, source string, paths skillPaths) bool {
	c, ok := paths.bySource[source]
	if !ok {
		return true
	}
	st, err := inspectComponent(componentDir(repoRoot, c.Name), c.Commit)
	return err == nil && !st.Dirty && !st.Off
}
