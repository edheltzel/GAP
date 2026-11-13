package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// catalogEntry is one [utilities.<id>] table in catalog.toml.
type catalogEntry struct {
	ID          string
	Repo        string
	Description string
	Tags        []string
}

// parseCatalog reads catalog.toml: [utilities.<id>] tables with repo, description, and tags.
// Ids must pass validID and repos must be the github.com/owner/repo form gap add resolves to.
func parseCatalog(text string) ([]catalogEntry, error) {
	doc, err := parseTOML(text)
	if err != nil {
		return nil, err
	}
	for key := range doc {
		if key != "utilities" {
			return nil, fmt.Errorf("catalog: unknown key %q", key)
		}
	}
	raw := doc["utilities"]
	tables, ok := raw.(map[string]any)
	if raw != nil && !ok {
		return nil, fmt.Errorf("catalog: utilities must be a table")
	}
	entries := make([]catalogEntry, 0, len(tables))
	for id, v := range tables {
		if id == "\x00header" {
			continue
		}
		if !validID(id) {
			return nil, fmt.Errorf("catalog: invalid id %q", id)
		}
		t, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("catalog: utilities.%s must be a table", id)
		}
		e := catalogEntry{ID: id, Tags: []string{}}
		for key, val := range t {
			switch key {
			case "repo":
				e.Repo, ok = val.(string)
			case "description":
				e.Description, ok = val.(string)
			case "tags":
				e.Tags, ok = val.([]string)
			case "\x00header":
			default:
				return nil, fmt.Errorf("catalog: utilities.%s: unknown key %q", id, key)
			}
			if !ok {
				return nil, fmt.Errorf("catalog: utilities.%s.%s has the wrong type", id, key)
			}
		}
		if err := validCatalogRepo(e.Repo); err != nil {
			return nil, fmt.Errorf("catalog: utilities.%s: %w", id, err)
		}
		if strings.TrimSpace(e.Description) == "" || strings.Contains(e.Description, "\n") {
			return nil, fmt.Errorf("catalog: utilities.%s needs a one-line description", id)
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return entries, nil
}

// validCatalogRepo accepts only the canonical github.com/owner/repo that parseUtilitySpec resolves a spec to.
func validCatalogRepo(repo string) error {
	if repo == "" {
		return fmt.Errorf("repo is required")
	}
	if strings.HasPrefix(repo, "file://") || filepath.IsAbs(repo) {
		return fmt.Errorf("repo must be github.com/owner/repo: %s", repo)
	}
	got, ref, _, err := parseUtilitySpec(repo)
	if err != nil {
		return err
	}
	if _, sub, _ := githubIdentity(got); ref != "" || sub != "" || got != repo {
		return fmt.Errorf("repo must be github.com/owner/repo: %s", repo)
	}
	return nil
}

// searchCatalog returns the entries matching query case-insensitively: id matches first, then tag, then description.
// Ties keep id order. An empty query returns every entry.
func searchCatalog(entries []catalogEntry, query string) []catalogEntry {
	q := strings.ToLower(strings.TrimSpace(query))
	type hit struct {
		rank  int
		entry catalogEntry
	}
	var hits []hit
	for _, e := range entries {
		rank := -1
		switch {
		case q == "" || strings.Contains(strings.ToLower(e.ID), q):
			rank = 0
		case tagMatches(e.Tags, q):
			rank = 1
		case strings.Contains(strings.ToLower(e.Description), q):
			rank = 2
		}
		if rank >= 0 {
			hits = append(hits, hit{rank, e})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].rank != hits[j].rank {
			return hits[i].rank < hits[j].rank
		}
		return hits[i].entry.ID < hits[j].entry.ID
	})
	out := make([]catalogEntry, len(hits))
	for i, h := range hits {
		out[i] = h.entry
	}
	return out
}

func tagMatches(tags []string, q string) bool {
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}

// addCommand is the gap add line for an entry, with --id when gap add would pick a different id.
func addCommand(e catalogEntry) string {
	if _, _, id, err := parseUtilitySpec(e.Repo); err == nil && id != e.ID {
		return "gap add " + e.Repo + " --id " + e.ID
	}
	return "gap add " + e.Repo
}

// installedSet holds the ids in ~/.gap/state.json and the ids and repos of components in ~/.config/gap/gap.toml.
func installedSet(home string) (ids, repos map[string]bool, err error) {
	ids, repos = map[string]bool{}, map[string]bool{}
	state, err := readState(home)
	if err != nil {
		return nil, nil, err
	}
	for id := range state.Components {
		ids[id] = true
	}
	text, err := readOptional(userConfigFile(home))
	if err != nil {
		return nil, nil, err
	}
	doc, err := parseTOML(text)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", userConfigFile(home), err)
	}
	comps, _ := doc["components"].(map[string]any)
	for id, v := range comps {
		t, _ := v.(map[string]any)
		if repo, ok := t["repo"].(string); ok {
			ids[id] = true
			repos[repoIdentity(repo)] = true
		}
	}
	return ids, repos, nil
}

// runSearch prints the catalog entries matching args. It exits 0 with matches and 1 with none or on error.
func runSearch(args []string) int {
	return searchTo(os.Stdout, os.Stderr, strings.Join(args, " "))
}

func searchTo(stdout, stderr io.Writer, query string) int {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(stderr, "gap search:", err)
		return 1
	}
	root, err := sourceRoot()
	if err != nil {
		fmt.Fprintln(stderr, "gap search:", err)
		return 1
	}
	b, err := os.ReadFile(filepath.Join(root, "catalog.toml"))
	if err != nil {
		fmt.Fprintln(stderr, "gap search:", err)
		return 1
	}
	entries, err := parseCatalog(string(b))
	if err != nil {
		fmt.Fprintln(stderr, "gap search:", err)
		return 1
	}
	ids, repos, err := installedSet(home)
	if err != nil {
		fmt.Fprintln(stderr, "gap search:", err)
		return 1
	}
	matches := searchCatalog(entries, query)
	if len(matches) == 0 {
		fmt.Fprintf(stderr, "No utility matches %q.\n", query)
		return 1
	}
	out := newReport(stdout)
	rows := make([][]string, 0, len(matches))
	for _, e := range matches {
		status := "-"
		if ids[e.ID] || repos[repoIdentity(e.Repo)] {
			status = "installed"
		}
		rows = append(rows, []string{e.ID, status, e.Description, strings.Join(e.Tags, ", "), addCommand(e)})
	}
	if !out.tty {
		for _, row := range rows {
			out.printf("%s %s - %s\n  %s\n", row[0], row[1], row[2], row[4])
		}
		return 0
	}
	for _, row := range rows {
		hex := colorDim
		if row[1] == "installed" {
			hex = colorOK
		}
		row[1] = out.tone(row[1], hex)
	}
	out.table([]string{"id", "status", "description", "tags", "add"}, rows)
	return 0
}
