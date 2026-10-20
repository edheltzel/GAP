package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	componentsLockPath = "global/components.lock"
	vendorDirName      = ".vendor"
)

// component is one pinned git checkout recorded in global/components.lock.
type component struct {
	Name, URL, Ref, Source string
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
	b, err := os.ReadFile(filepath.Join(repoRoot, componentsLockPath))
	if errors.Is(err, os.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	var doc struct {
		Comment    string `json:"comment"`
		Components map[string]struct {
			URL    string `json:"url"`
			Ref    string `json:"ref"`
			Source string `json:"source"`
		} `json:"components"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return p, fmt.Errorf("%s: %w", componentsLockPath, err)
	}
	for name, spec := range doc.Components {
		if spec.URL == "" || spec.Ref == "" {
			return p, fmt.Errorf("%s: component %s needs url and ref", componentsLockPath, name)
		}
		src := spec.Source
		if src == "" {
			src = githubSourceFromURL(spec.URL)
		}
		if src == "" {
			return p, fmt.Errorf("%s: component %s needs source", componentsLockPath, name)
		}
		p.bySource[src] = component{Name: name, URL: spec.URL, Ref: spec.Ref, Source: src}
	}
	return p, nil
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

func inspectComponent(dir, ref string) (componentStatus, error) {
	st := componentStatus{Dir: dir}
	if !isGitRepo(dir) {
		return st, fmt.Errorf("pinned component at %s is not a git checkout", dir)
	}
	head, err := gitRun(dir, "rev-parse", "HEAD")
	if err != nil {
		return st, fmt.Errorf("reading HEAD in %s: %w", dir, err)
	}
	pin, err := gitRun(dir, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		pin = ref
		st.Off = true
	} else {
		st.Off = head != pin
	}
	st.Head, st.Pin = head, pin
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

func ensureComponent(repoRoot string, c component) error {
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
	if !sameGitURL(origin, c.URL) {
		return fmt.Errorf("pinned component %s origin is %s, not %s", c.Name, origin, c.URL)
	}
	st, err := inspectComponent(dir, c.Ref)
	if err == nil && !st.Dirty && !st.Off {
		return nil
	}
	if err == nil && st.Dirty {
		// Never reset a dirty checkout onto the pin.
		return nil
	}
	if _, err := gitRun(dir, "fetch", "-q", "origin", c.Ref); err != nil {
		return fmt.Errorf("pinned component %s: fetch %s: %w", c.Name, c.Ref, err)
	}
	pin, err := gitRun(dir, "rev-parse", "--verify", c.Ref+"^{commit}")
	if err != nil {
		return fmt.Errorf("pinned component %s: resolve %s: %w", c.Name, c.Ref, err)
	}
	if _, err := gitRun(dir, "checkout", "-q", "--detach", pin); err != nil {
		return fmt.Errorf("pinned component %s: checkout %s: %w", c.Name, pin, err)
	}
	return nil
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
	if _, err := gitRun(dir, "remote", "add", "origin", c.URL); err != nil {
		return fmt.Errorf("pinned component %s: remote: %w", c.Name, err)
	}
	if _, err := gitRun(dir, "fetch", "-q", "origin", c.Ref); err != nil {
		return fmt.Errorf("pinned component %s: fetch %s: %w", c.Name, c.Ref, err)
	}
	pin, err := gitRun(dir, "rev-parse", "--verify", c.Ref+"^{commit}")
	if err != nil {
		return fmt.Errorf("pinned component %s: resolve %s: %w", c.Name, c.Ref, err)
	}
	if _, err := gitRun(dir, "checkout", "-q", "--detach", pin); err != nil {
		return fmt.Errorf("pinned component %s: checkout %s: %w", c.Name, pin, err)
	}
	fmt.Printf("using %s at %s\n", c.Source, pin)
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
	st, err := inspectComponent(componentDir(repoRoot, c.Name), c.Ref)
	return err == nil && !st.Dirty && !st.Off
}
