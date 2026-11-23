package main

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

type renderOpts struct {
	Eve    bool
	Rename string
	Quiet  bool
}

func renderSkill(src, dst string, opt renderOpts) error {
	root, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	if err := renderDir(src, dst, root, map[string]bool{}, opt); err != nil {
		return err
	}
	if opt.Rename == "" {
		return nil
	}
	file := filepath.Join(dst, "SKILL.md")
	info, err := os.Lstat(file)
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	if renamed := setFrontmatterName(string(b), opt.Rename); renamed != string(b) {
		return os.WriteFile(file, []byte(renamed), info.Mode().Perm())
	}
	return nil
}

func renderDir(src, dst, root string, ancestors map[string]bool, opt renderOpts) error {
	warn := func(format string, args ...any) {
		if !opt.Quiet {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}
	real, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	if ancestors[real] {
		warn("Skipping symlink that creates a directory cycle: %s", src)
		return nil
	}
	branch := maps.Clone(ancestors)
	branch[real] = true
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		isDir := entry.IsDir()
		if skillCopyExcluded(name, isDir) {
			continue
		}
		from, to := filepath.Join(src, name), filepath.Join(dst, name)
		if entry.Type()&fs.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(from)
			if err != nil {
				warn("Skipping broken symlink: %s", from)
				continue
			}
			if !withinRoot(root, target) {
				warn("Skipping symlink that points outside the skill directory: %s", from)
				continue
			}
			info, err := os.Stat(target)
			if err != nil {
				return err
			}
			if isDir = info.IsDir(); isDir && skillCopyExcluded(name, true) {
				continue
			}
		} else if !isDir && !entry.Type().IsRegular() {
			continue
		}
		if isDir {
			if err := renderDir(from, to, root, branch, opt); err != nil {
				return err
			}
			continue
		}
		b, err := os.ReadFile(from)
		if err != nil {
			return err
		}
		if opt.Eve && strings.ToLower(name) == "skill.md" {
			text, err := eveFrontmatter(string(b))
			if err != nil {
				return fmt.Errorf("%s: %w", from, err)
			}
			if err := os.WriteFile(to, []byte(text), 0o666); err != nil {
				return err
			}
			continue
		}
		info, err := os.Stat(from)
		if err != nil {
			return err
		}
		if err := os.WriteFile(to, b, info.Mode().Perm()); err != nil {
			return err
		}
		if err := os.Chmod(to, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

func realOr(path string) string {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real
	}
	return filepath.Clean(path)
}

func linkSkill(target, link string) error {
	realTarget, realLink := realOr(target), realOr(link)
	if realTarget == realLink {
		return nil
	}
	if filepath.Join(realOr(filepath.Dir(target)), filepath.Base(target)) == filepath.Join(realOr(filepath.Dir(link)), filepath.Base(link)) {
		return nil
	}
	if info, err := os.Lstat(link); err == nil {
		if info.Mode()&fs.ModeSymlink != 0 {
			err = os.Remove(link)
		} else {
			err = os.RemoveAll(link)
		}
		if err != nil {
			return err
		}
	}
	dir := filepath.Dir(link)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	rel, err := filepath.Rel(realOr(dir), realTarget)
	if err != nil {
		return err
	}
	return os.Symlink(rel, link)
}

func categoryConflict(dir string) []string {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || fileExists(filepath.Join(dir, "SKILL.md")) {
		return nil
	}
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, e := range entries {
		if fileExists(filepath.Join(dir, e.Name(), "SKILL.md")) {
			names = append(names, e.Name())
		}
	}
	return names
}

// renderLoad reports whether a skill should be installed for this run.
// listed and always install. project installs only inside a repo that matches when.
// command and off never install here. An entry with no load installs, as M1 did.
func renderLoad(c component, name, repoRoot string) bool {
	s, ok := c.SkillSet[name]
	if !ok || s.Load == "" || s.Load == "listed" || s.Load == "always" {
		return true
	}
	if s.Load == "project" {
		return projectMatches(repoRoot, s.WhenFiles)
	}
	return false
}

// projectMatches reports that repoRoot holds a file matching one of the globs.
func projectMatches(repoRoot string, globs []string) bool {
	if repoRoot == "" {
		return false
	}
	for _, g := range globs {
		matches, _ := filepath.Glob(filepath.Join(repoRoot, g))
		if len(matches) > 0 {
			return true
		}
	}
	return false
}
