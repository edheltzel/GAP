package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchCatalogParsesRealCatalog(t *testing.T) {
	entries, err := parseCatalog(readString(t, filepath.Join("..", "..", "catalog.toml")))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"do-skills": "github.com/edheltzel/Do-Skills",
		"echo":      "github.com/edheltzel/Echo",
		"forge":     "github.com/edheltzel/Forge",
		"recall":    "github.com/edheltzel/Recall",
	}
	if len(entries) != len(want) {
		t.Fatalf("entries = %+v", entries)
	}
	for _, e := range entries {
		if want[e.ID] != e.Repo || e.Description == "" || len(e.Tags) == 0 {
			t.Fatalf("entry %+v", e)
		}
	}
}

func TestSearchCatalogRejectsBadEntries(t *testing.T) {
	for name, text := range map[string]string{
		"uppercase id":  "[utilities.Forge]\nrepo = \"github.com/edheltzel/Forge\"\ndescription = \"x\"\n",
		"reserved id":   "[utilities.policy]\nrepo = \"github.com/edheltzel/Forge\"\ndescription = \"x\"\n",
		"bare repo":     "[utilities.forge]\nrepo = \"edheltzel/Forge\"\ndescription = \"x\"\n",
		"ref":           "[utilities.forge]\nrepo = \"github.com/edheltzel/Forge@v1\"\ndescription = \"x\"\n",
		"git suffix":    "[utilities.forge]\nrepo = \"github.com/edheltzel/Forge.git\"\ndescription = \"x\"\n",
		"subpath":       "[utilities.forge]\nrepo = \"github.com/edheltzel/Forge/skills\"\ndescription = \"x\"\n",
		"no owner":      "[utilities.forge]\nrepo = \"github.com/Forge\"\ndescription = \"x\"\n",
		"file url":      "[utilities.forge]\nrepo = \"file:///tmp/forge\"\ndescription = \"x\"\n",
		"absolute path": "[utilities.forge]\nrepo = \"/tmp/forge\"\ndescription = \"x\"\n",
		"missing repo":  "[utilities.forge]\ndescription = \"x\"\n",
		"missing desc":  "[utilities.forge]\nrepo = \"github.com/edheltzel/Forge\"\n",
		"string tags":   "[utilities.forge]\nrepo = \"github.com/edheltzel/Forge\"\ndescription = \"x\"\ntags = \"a\"\n",
		"unknown key":   "[utilities.forge]\nrepo = \"github.com/edheltzel/Forge\"\ndescription = \"x\"\nref = \"master\"\n",
		"unknown table": "[other.forge]\nrepo = \"github.com/edheltzel/Forge\"\ndescription = \"x\"\n",
	} {
		if _, err := parseCatalog(text); err == nil {
			t.Errorf("%s: parsed", name)
		}
	}
}

func TestSearchCatalogRanksIDThenTagThenDescription(t *testing.T) {
	entries := []catalogEntry{
		{ID: "zeta", Description: "Stores MEMORY on disk"},
		{ID: "alpha", Description: "unrelated", Tags: []string{"Memory"}},
		{ID: "other", Description: "nothing here"},
		{ID: "memory-x", Description: "unrelated"},
		{ID: "beta", Description: "unrelated", Tags: []string{"memory"}},
	}
	var got []string
	for _, e := range searchCatalog(entries, "MEMORY") {
		got = append(got, e.ID)
	}
	if strings.Join(got, ",") != "memory-x,alpha,beta,zeta" {
		t.Fatalf("order = %v", got)
	}
	if all := searchCatalog(entries, "  "); len(all) != len(entries) || all[0].ID != "alpha" {
		t.Fatalf("empty query = %+v", all)
	}
	if none := searchCatalog(entries, "absent"); len(none) != 0 {
		t.Fatalf("absent = %+v", none)
	}
}

func TestSearchMarksInstalledAndStaysPlainWhenPiped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	write(t, filepath.Join(home, ".gap", "state.json"), `{"components":{"echo":{"commit":"x","adapters":[]}}}`)
	write(t, filepath.Join(home, ".config", "gap", "gap.toml"), "[components.mem]\nrepo = \"github.com/EdHeltzel/Recall\"\nref = \"main\"\n")
	var stdout, stderr bytes.Buffer
	if code := searchTo(&stdout, &stderr, ""); code != 0 {
		t.Fatalf("code = %d stderr = %s", code, stderr.String())
	}
	out := stdout.String()
	if strings.Contains(out, "\x1b") || strings.ContainsAny(out, "│─") {
		t.Fatalf("piped output is styled:\n%s", out)
	}
	for _, want := range []string{
		"echo installed - ",
		"recall installed - ",
		"forge - - ",
		"do-skills - - ",
		"  gap add github.com/edheltzel/Forge\n",
		"  gap add github.com/edheltzel/Do-Skills\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	stdout.Reset()
	stderr.Reset()
	if code := searchTo(&stdout, &stderr, "no-such-utility"); code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "No utility matches") {
		t.Fatalf("no match: code = %d stdout = %q stderr = %q", code, stdout.String(), stderr.String())
	}
}
