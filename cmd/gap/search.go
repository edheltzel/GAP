package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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

func searchCatalog(entries []catalogEntry, query string) []catalogEntry {
	q := strings.ToLower(strings.TrimSpace(query))
	var matches []catalogEntry
	for _, e := range entries {
		if q == "" || strings.Contains(strings.ToLower(e.ID), q) || tagMatches(e.Tags, q) || strings.Contains(strings.ToLower(e.Description), q) {
			matches = append(matches, e)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].ID < matches[j].ID })
	return matches
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
// An entry the distro already ships gets no add line, because gap add refuses a distro id.
func addCommand(e catalogEntry, distroIDs map[string]bool) string {
	if distroIDs[e.ID] {
		return "included with GAP; gap sync installs it"
	}
	if _, _, id, err := parseUtilitySpec(e.Repo); err == nil && id != e.ID {
		return "gap add " + e.Repo + " --id " + e.ID
	}
	return "gap add " + e.Repo
}

// runSearch prints the catalog entries matching args. It exits 0 with matches and 1 with none or on error.
func runSearch(args []string) int {
	return searchTo(os.Stdout, os.Stderr, strings.Join(args, " "))
}

func searchTo(stdout, stderr io.Writer, query string) int {
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
	d, err := readDistro(root)
	if err != nil {
		fmt.Fprintln(stderr, "gap search:", err)
		return 1
	}
	distroIDs := map[string]bool{}
	for id := range d.Components {
		distroIDs[id] = true
	}
	matches := searchCatalog(entries, query)
	if len(matches) == 0 {
		fmt.Fprintf(stderr, "No utility matches %q.\n", query)
		return 1
	}
	rows := make([][]string, 0, len(matches))
	for _, e := range matches {
		rows = append(rows, []string{e.ID, e.Description, strings.Join(e.Tags, ", "), addCommand(e, distroIDs)})
	}
	printSearch(newReport(stdout), query, rows)
	return 0
}

// printSearch writes search matches: two plain lines per match when piped, a framed table with a match count on a terminal.
func printSearch(out *report, query string, rows [][]string) {
	if !out.tty {
		for _, row := range rows {
			out.printf("%s - %s\n  %s\n", row[0], row[1], row[3])
		}
		return
	}
	detail := "utility catalog"
	if q := strings.TrimSpace(query); q != "" {
		detail = strconv.Quote(q)
	}
	out.title("gap search", detail)
	out.framed([]string{"id", "description", "tags", "add"}, rows)
	noun := "matches"
	if len(rows) == 1 {
		noun = "match"
	}
	out.footer(fmt.Sprintf("%d %s", len(rows), noun))
}
