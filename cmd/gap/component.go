package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var componentNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

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
	repoRoot  string
	bySource  map[string]component
	requested map[string]bool
}

type componentStatus struct {
	Dir   string
	Head  string
	Pin   string
	Dirty bool
	Off   bool
}

func loadSkillPaths(repoRoot string) (skillPaths, error) {
	p := skillPaths{repoRoot: repoRoot, bySource: map[string]component{}, requested: map[string]bool{}}
	lockBytes, lockErr := os.ReadFile(filepath.Join(repoRoot, gapLockPath))
	tomlBytes, tomlErr := os.ReadFile(filepath.Join(repoRoot, gapTOMLPath))
	if errors.Is(lockErr, os.ErrNotExist) && errors.Is(tomlErr, os.ErrNotExist) {
		return p, fmt.Errorf("%s and %s are missing", gapLockPath, gapTOMLPath)
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
	if err := checkTOMLKeys(lock, map[string]bool{"schema": true, "components": true}, gapLockPath); err != nil {
		return p, err
	}
	comps := tomlChild(lock, "components")
	if comps == nil {
		return p, fmt.Errorf("%s: missing [components] tables", gapLockPath)
	}
	byName := map[string]component{}
	for name, raw := range comps {
		if err := validComponentName(name); err != nil {
			return p, fmt.Errorf("%s: %w", gapLockPath, err)
		}
		spec, ok := raw.(tomlTable)
		if !ok {
			return p, fmt.Errorf("%s: component %s is not a table", gapLockPath, name)
		}
		if err := checkTOMLKeys(spec, map[string]bool{"repo": true, "ref": true, "commit": true, "tree": true, "source": true}, gapLockPath+": component "+name); err != nil {
			return p, err
		}
		c, err := componentFromTable(name, spec)
		if err != nil {
			return p, fmt.Errorf("%s: %w", gapLockPath, err)
		}
		if err := p.indexComponent(c); err != nil {
			return p, fmt.Errorf("%s: %w", gapLockPath, err)
		}
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
	if err := checkTOMLKeys(req, map[string]bool{"components": true}, gapTOMLPath); err != nil {
		return p, err
	}
	reqComps := tomlChild(req, "components")
	if reqComps == nil {
		return p, fmt.Errorf("%s: missing [components] tables", gapTOMLPath)
	}
	for name, raw := range reqComps {
		if err := validComponentName(name); err != nil {
			return p, fmt.Errorf("%s: %w", gapTOMLPath, err)
		}
		spec, ok := raw.(tomlTable)
		if !ok {
			return p, fmt.Errorf("%s: component %s is not a table", gapTOMLPath, name)
		}
		if err := checkTOMLKeys(spec, map[string]bool{"repo": true, "ref": true}, gapTOMLPath+": component "+name); err != nil {
			return p, err
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
		if !sameGitURL(repo, got.Repo) || ref != got.Ref {
			return p, fmt.Errorf("%s: component %s repo/ref does not match %s", gapTOMLPath, name, gapLockPath)
		}
		if key := normalizeSource(repo); key != "" {
			p.requested[key] = true
		}
		if key := normalizeSource(got.Source); key != "" {
			p.requested[key] = true
		}
	}
	return p, nil
}

func (p *skillPaths) indexComponent(c component) error {
	if p.bySource == nil {
		p.bySource = map[string]component{}
	}
	for _, key := range componentSourceKeys(c) {
		if existing, ok := p.bySource[key]; ok && existing.Name != c.Name {
			return fmt.Errorf("component %s duplicates source %s of %s", c.Name, key, existing.Name)
		}
		p.bySource[key] = c
	}
	return nil
}

func validComponentName(name string) error {
	if !componentNameRe.MatchString(name) {
		return fmt.Errorf("component name %s must match %s", name, componentNameRe.String())
	}
	return nil
}

func componentSourceKeys(c component) []string {
	seen := map[string]bool{}
	var keys []string
	for _, raw := range []string{c.Source, c.Repo} {
		key := normalizeSource(raw)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	return keys
}

func (p skillPaths) lookup(source string) (component, bool) {
	c, ok := p.bySource[normalizeSource(source)]
	return c, ok
}

func (p skillPaths) requestedSource(source string) bool {
	return p.requested[normalizeSource(source)]
}

func normalizeSource(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "." || filepath.IsAbs(s) {
		return ""
	}
	repo, sub, ok := githubIdentity(s)
	if !ok {
		s = strings.TrimSuffix(strings.TrimSuffix(s, "/"), ".git")
		return strings.ToLower(s)
	}
	repo = strings.ToLower(repo)
	if sub == "" {
		return repo
	}
	return repo + "/" + sub
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
	if _, present := spec["tree"]; present {
		if err := lowercaseHexSHA(c.Tree, "component "+name+" tree"); err != nil {
			return component{}, err
		}
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
	c, ok := p.lookup(source)
	if !ok {
		return "", false
	}
	return filepath.Clean(componentDir(p.repoRoot, c.Name)), true
}

func (p skillPaths) needed(entries []manifestEntry) []component {
	var out []component
	seen := map[string]bool{}
	for _, e := range entries {
		c, ok := p.lookup(e.Source)
		if !ok || seen[c.Name] {
			continue
		}
		seen[c.Name] = true
		out = append(out, c)
	}
	return out
}

func resolveInstallSource(repoRoot, source string) (string, error) {
	if source == "." {
		return filepath.Clean(repoRoot), nil
	}
	paths, err := loadSkillPaths(repoRoot)
	if err != nil {
		return "", err
	}
	if c, ok := paths.lookup(source); ok {
		return filepath.Clean(componentDir(repoRoot, c.Name)), nil
	}
	if paths.requestedSource(source) {
		return "", fmt.Errorf("component source %s has no lock entry", source)
	}
	if err := paths.componentSubfolder(source); err != nil {
		return "", err
	}
	return source, nil
}

func (p skillPaths) componentSubfolder(source string) error {
	repo, sub, ok := githubIdentity(source)
	if !ok || sub == "" {
		return nil
	}
	c, hit := p.lookup(repo)
	if !hit && !p.requested[strings.ToLower(repo)] {
		return nil
	}
	name := c.Name
	if name == "" {
		name = repo
	}
	return fmt.Errorf("component source %s names a subfolder of %s", source, name)
}

func componentDir(repoRoot, name string) string {
	return filepath.Join(repoRoot, vendorDirName, name)
}

func (p skillPaths) component(name string) (component, bool) {
	for _, c := range p.bySource {
		if c.Name == name {
			return c, true
		}
	}
	return component{}, false
}

func lockField(repoRoot, name, field string) (string, error) {
	if err := validComponentName(name); err != nil {
		return "", err
	}
	paths, err := loadSkillPaths(repoRoot)
	if err != nil {
		return "", err
	}
	c, ok := paths.component(name)
	if !ok {
		return "", fmt.Errorf("component %s is not in %s", name, gapLockPath)
	}
	switch field {
	case "repo":
		return c.Repo, nil
	case "ref":
		return c.Ref, nil
	case "commit":
		return c.Commit, nil
	case "tree":
		return c.Tree, nil
	case "source":
		return c.Source, nil
	default:
		return "", fmt.Errorf("unknown lock field %s", field)
	}
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

func checkTOMLKeys(t tomlTable, allowed map[string]bool, where string) error {
	for k := range t {
		if !allowed[k] {
			return fmt.Errorf("%s: unknown key %s", where, k)
		}
	}
	return nil
}

func lowercaseHexSHA(s, what string) error {
	if len(s) != 40 {
		return fmt.Errorf("%s must be a 40-character lowercase hex SHA", what)
	}
	for i := 0; i < 40; i++ {
		ch := s[i]
		if ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' {
			continue
		}
		return fmt.Errorf("%s must be a 40-character lowercase hex SHA", what)
	}
	return nil
}

func requirePinCommit(c component) error {
	return lowercaseHexSHA(c.Commit, "component "+c.Name+" commit")
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
		// Never reset a dirty checkout onto the pin, and never install from it.
		return fmt.Errorf("pinned component %s is dirty at %s", c.Name, dir)
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
	return strings.EqualFold(normalizeGitURL(a), normalizeGitURL(b))
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
	c, ok := paths.lookup(source)
	if !ok {
		return true
	}
	st, err := inspectComponent(componentDir(repoRoot, c.Name), c.Commit)
	return err == nil && !st.Dirty && !st.Off
}

func reportSkippedComponent(repoRoot, source string, paths skillPaths) {
	c, ok := paths.lookup(source)
	if !ok {
		return
	}
	st, err := inspectComponent(componentDir(repoRoot, c.Name), c.Commit)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	reportComponentStatus(st, c.Name)
}
