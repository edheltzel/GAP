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

func recordProject(home, root string) error {
	if !filepath.IsAbs(root) {
		return fmt.Errorf("project root %s is not absolute", root)
	}
	root = filepath.Clean(root)
	if root == filepath.Clean(home) {
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
	if _, statErr := os.Stat(projectsPath(home)); statErr == nil && slices.ContainsFunc(list, func(p projectRecord) bool { return p.Root == root }) {
		return nil
	}
	if !slices.ContainsFunc(list, func(p projectRecord) bool { return p.Root == root }) {
		list = append(list, projectRecord{Root: root, Added: time.Now().Format(time.DateOnly)})
	}
	return writeProjects(home, list)
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
	root = filepath.Clean(root)
	return writeProjects(home, slices.DeleteFunc(list, func(p projectRecord) bool { return p.Root == root }))
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
