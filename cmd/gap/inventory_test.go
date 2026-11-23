package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func inventoryRow(t *testing.T, rows []itemRow, id string) itemRow {
	t.Helper()
	for _, r := range rows {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no row %s in %+v", id, rows)
	return itemRow{}
}

func TestInventoryMatchesStatus(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	captureStd(t, func() {
		if err := runAdd(addGlobalArgs(e.remote, "--id", "util")); err != nil {
			t.Fatal(err)
		}
	})
	compare := func(wantDrift int) {
		t.Helper()
		rows, err := inventory(e.home, e.repo, e.project)
		if err != nil {
			t.Fatal(err)
		}
		r := inventoryRow(t, rows, "util")
		if r.Source != "user" || r.Scope != "global" || r.Store != storeOK || r.Drift != wantDrift || !r.Pinned {
			t.Fatalf("row = %+v", r)
		}
		status, _ := captureStd(t, func() { _ = statusAll() })
		want := fmt.Sprintf("util ref=%s commit=%s store=%s", r.Ref, r.Commit, r.Store)
		if !strings.Contains(status, want) || !strings.Contains(status, fmt.Sprintf("drift=%d", wantDrift)) {
			t.Fatalf("status does not match inventory %q:\n%s", want, status)
		}
		list, _ := captureStd(t, func() { _ = listUtilities() })
		if !strings.Contains(list, r.Commit) || !strings.Contains(list, "trusted") {
			t.Fatalf("list does not match inventory:\n%s", list)
		}
	}
	compare(0)
	write(t, filepath.Join(e.home, ".agents", "skills", "util-skill", "SKILL.md"), "edited\n")
	compare(1)
}

func TestInventoryDisabledDistro(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	write(t, filepath.Join(e.repo, "gap.toml"), "schema = \"2\"\n\n[components.dist]\nrepo = \""+e.remote+"\"\nref = \"master\"\n")
	write(t, userConfigFile(e.home), "[components.dist]\nenabled = false\n")
	rows, err := inventory(e.home, e.repo, e.project)
	if err != nil {
		t.Fatal(err)
	}
	r := inventoryRow(t, rows, "dist")
	if r.Source != "distro" || !r.Disabled || r.Pinned || r.Store != storeUnpinned {
		t.Fatalf("row = %+v", r)
	}
}

func TestProjectInventoryMissingRoot(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	v := projectInventory(e.home, e.repo, projectRecord{Root: filepath.Join(e.project, "gone")})
	if !v.Missing || v.Err != nil {
		t.Fatalf("view = %+v", v)
	}
}

func TestProjectInventoryProjectScope(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	captureStd(t, func() {
		if err := runAdd(addProjectArgs(e.remote, "--id", "util")); err != nil {
			t.Fatal(err)
		}
	})
	list, err := readProjects(e.home)
	if err != nil || len(list) != 1 {
		t.Fatalf("registry = %v %v", list, err)
	}
	v := projectInventory(e.home, e.repo, list[0])
	if v.Missing || v.Err != nil || len(v.Items) != 1 || v.Items[0].ID != "util" || v.Items[0].Drift != 0 {
		t.Fatalf("view = %+v", v)
	}
}
