package main

import (
	"os"
	"path/filepath"
)

// copySkill copies the files skills.sh installs: regular files, skipping the
// dirs and metadata.json that indexPathExcluded names.
func copySkill(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if indexPathExcluded(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		return copyFile(path, target)
	})
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
