package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// installLocal copies a skill out of a pinned store (or this checkout) into both
// harness dirs and records the lock entry skills.sh would have written.
// It is the Go path. skills.sh stays only for sources gap does not have on disk.
func installLocal(home string, e manifestEntry, name, src string) error {
	hash, err := contentHash(src)
	if err != nil {
		return fmt.Errorf("hashing %s: %w", name, err)
	}
	for _, dir := range skillDirs(home, name) {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		if err := copySkill(src, dir); err != nil {
			return fmt.Errorf("installing %s: %w", dir, err)
		}
	}
	// The lock names the store, which is the path skills.sh recorded. A checkout source has no store path.
	recorded := e.Path
	if recorded == "" {
		recorded = src
	}
	return writeSkillLockEntry(home, name, lockEntry{
		Source:          recorded,
		SourceType:      "local",
		SourceURL:       recorded,
		SkillFolderHash: hash,
	})
}

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

// writeSkillLockEntry sets one skill in the skills.sh lock, leaving every other entry as it was.
func writeSkillLockEntry(home, name string, entry lockEntry) error {
	path := skillLockPath(home)
	root := map[string]json.RawMessage{}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &root); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	skills := map[string]json.RawMessage{}
	if raw := root["skills"]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &skills); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	skills[name] = encoded
	raw, err := json.Marshal(skills)
	if err != nil {
		return err
	}
	root["skills"] = raw
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

// localSkillSource is the on-disk skill dir when the entry has one. External manifest
// sources have none, and those still go through skills.sh.
func localSkillSource(repoRoot string, e manifestEntry, name string) string {
	if e.Path == "" && e.Source != "." {
		return ""
	}
	return localSkillDir(addSource(repoRoot, e), name)
}

// skillUsable reports that both harness copies exist and match the pinned source.
func skillUsable(repoRoot, home string, e manifestEntry, name string) bool {
	src := localSkillSource(repoRoot, e, name)
	if src == "" || !installed(home, name) {
		return false
	}
	want, err := treeHash(src)
	if err != nil {
		return false
	}
	for _, dir := range skillDirs(home, name) {
		got, err := treeHash(dir)
		if err != nil || got != want {
			return false
		}
	}
	return true
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
		// A glob with no slash still has to see the repo root, not only its top directory.
		if !strings.Contains(g, "/") {
			found := false
			_ = filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
				if err != nil || found {
					return err
				}
				if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
					return filepath.SkipDir
				}
				ok, _ := filepath.Match(g, d.Name())
				if ok && !d.IsDir() {
					found = true
					return filepath.SkipAll
				}
				return nil
			})
			if found {
				return true
			}
		}
	}
	return false
}
