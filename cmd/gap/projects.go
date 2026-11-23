package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type projectRecord struct {
	Root  string `json:"root"`
	Added string `json:"added,omitempty"`
}

type projectsFile struct {
	Projects []projectRecord `json:"projects"`
}

func projectsPath(home string) string {
	return filepath.Join(home, ".gap", "projects.json")
}

func readProjects(home string) ([]projectRecord, error) {
	path := projectsPath(home)
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return legacyProjects(home)
	}
	if err != nil {
		return nil, err
	}
	var f projectsFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return sortProjects(f.Projects), nil
}

func legacyProjects(home string) ([]projectRecord, error) {
	state, err := readState(home)
	if err != nil {
		return nil, err
	}
	var out []projectRecord
	for _, entry := range state.Components {
		for root := range entry.ProjectSkills {
			if !slices.ContainsFunc(out, func(p projectRecord) bool { return p.Root == root }) {
				out = append(out, projectRecord{Root: filepath.Clean(root)})
			}
		}
	}
	return sortProjects(out), nil
}

func sortProjects(list []projectRecord) []projectRecord {
	slices.SortFunc(list, func(a, b projectRecord) int { return strings.Compare(a.Root, b.Root) })
	return slices.CompactFunc(list, func(a, b projectRecord) bool { return a.Root == b.Root })
}

func writeProjects(home string, list []projectRecord) error {
	b, err := json.MarshalIndent(projectsFile{Projects: sortProjects(list)}, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(projectsPath(home), string(b)+"\n")
}

func canonicalRoot(path string) string {
	path = filepath.Clean(path)
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real
	}
	return path
}

func recordProject(home, root string) error {
	if !filepath.IsAbs(root) {
		return fmt.Errorf("project root %s is not absolute", root)
	}
	root = canonicalRoot(root)
	if root == canonicalRoot(home) {
		return nil
	}
	unlock, err := lockProjects(home)
	if err != nil {
		return err
	}
	defer unlock()
	list, err := readProjects(home)
	if err != nil {
		return err
	}
	merged := coalesceProjects(list)
	known := slices.ContainsFunc(merged, func(p projectRecord) bool { return p.Root == root })
	if _, statErr := os.Stat(projectsPath(home)); statErr == nil && known && slices.Equal(merged, list) {
		return nil
	}
	if !known {
		merged = append(merged, projectRecord{Root: root, Added: time.Now().Format(time.DateOnly)})
	}
	return writeProjects(home, merged)
}

func coalesceProjects(list []projectRecord) []projectRecord {
	byRoot := map[string]projectRecord{}
	for _, p := range list {
		root := canonicalRoot(p.Root)
		prev, seen := byRoot[root]
		if !seen || (p.Added != "" && (prev.Added == "" || p.Added < prev.Added)) {
			byRoot[root] = projectRecord{Root: root, Added: p.Added}
		}
	}
	out := make([]projectRecord, 0, len(byRoot))
	for _, p := range byRoot {
		out = append(out, p)
	}
	return sortProjects(out)
}

func forgetProject(home, root string) error {
	unlock, err := lockProjects(home)
	if err != nil {
		return err
	}
	defer unlock()
	list, err := readProjects(home)
	if err != nil {
		return err
	}
	stored, real := filepath.Clean(root), canonicalRoot(root)
	return writeProjects(home, slices.DeleteFunc(list, func(p projectRecord) bool { return p.Root == stored || canonicalRoot(p.Root) == real }))
}

func noteProject(root string) {
	home, err := os.UserHomeDir()
	if err == nil {
		err = recordProject(home, root)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not record %s in %s: %v\n", root, projectsPath(home), err)
	}
}
