package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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
		rows, err := inventory(e.home, e.repo, e.project, nil)
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
	rows, err := inventory(e.home, e.repo, e.project, nil)
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
	v := projectInventory(e.home, e.repo, projectRecord{Root: filepath.Join(e.project, "gone")}, nil)
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
	v := projectInventory(e.home, e.repo, list[0], nil)
	if v.Missing || v.Err != nil || len(v.Items) != 1 || v.Items[0].ID != "util" || v.Items[0].Drift != 0 {
		t.Fatalf("view = %+v", v)
	}
}

func addedUtility(t *testing.T, e *utilityEnv, args []string) lockEntryM1 {
	t.Helper()
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	captureStd(t, func() {
		if err := runAdd(args); err != nil {
			t.Fatal(err)
		}
	})
	lk, err := readLock(userConfigDir(e.home))
	if err != nil {
		t.Fatal(err)
	}
	return lk.Components["util"]
}

func TestProjectInventoryKeepsItemWithoutStore(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	pin := addedUtility(t, e, addProjectArgs(e.remote, "--id", "util"))
	list, err := readProjects(e.home)
	if err != nil || len(list) != 1 {
		t.Fatalf("registry = %v %v", list, err)
	}
	if err := os.RemoveAll(storePath(e.home, "util", pin.Commit)); err != nil {
		t.Fatal(err)
	}
	v := projectInventory(e.home, e.repo, list[0], nil)
	if v.Err != nil || len(v.Items) != 1 || v.Items[0].ID != "util" || v.Items[0].Store != storeMissing || !v.unchecked() {
		t.Fatalf("view = %+v", v)
	}
	m := newTUIModel(e.home, e.repo, e.project, darkPalette)
	m.Update(tea.WindowSizeMsg{Width: 240, Height: 30})
	m.Update(loadedMsg{projects: []projectView{v}})
	press(m, "3")
	s := screen(m)
	if !strings.Contains(s, "unchecked") || strings.Contains(s, " ok") || !strings.Contains(s, "store missing: drift not checked") || strings.Contains(s, "No project-scope skills") {
		t.Fatalf("projects view:\n%s", s)
	}
}

func TestInventoryUsesEffectiveRef(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	pin := addedUtility(t, e, addGlobalArgs(e.remote, "--id", "util"))
	machine := inventoryRow(t, mustInventory(t, e), "util").Ref
	pin.Ref = "beta"
	if machine == pin.Ref {
		t.Fatalf("machine ref already %s", machine)
	}
	if err := writeLock(userConfigDir(e.home), lockFile{Components: map[string]lockEntryM1{"util": pin}}); err != nil {
		t.Fatal(err)
	}
	write(t, projectConfigFile(e.project), "[components.util]\nref = \"beta\"\n")
	r := inventoryRow(t, mustInventory(t, e), "util")
	if r.Ref != "beta" || r.Commit != pin.Commit[:7] || r.Store != storeOK || r.Inactive {
		t.Fatalf("row = %+v", r)
	}
}

func TestInventoryProjectDisabledKeepsPinAndStore(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	pin := addedUtility(t, e, addGlobalArgs(e.remote, "--id", "util"))
	write(t, projectConfigFile(e.project), "[components.util]\nenabled = false\n")
	r := inventoryRow(t, mustInventory(t, e), "util")
	if !r.Inactive || r.Disabled || r.Store != storeOK || r.Commit != pin.Commit[:7] || !r.Pinned {
		t.Fatalf("row = %+v", r)
	}
	m := newTUIModel(e.home, e.repo, e.project, darkPalette)
	m.Update(loadedMsg{items: []itemRow{r}})
	if s := screen(m); strings.Contains(s, "unpinned") || !strings.Contains(s, "off here") {
		t.Fatalf("items view:\n%s", s)
	}
}

func TestInventoryDamagedInactiveStoreKeepsOtherRows(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	pin := addedUtility(t, e, addGlobalArgs(e.remote, "--id", "util"))
	write(t, filepath.Join(e.repo, "gap.toml"), "schema = \"2\"\n\n[components.dist]\nrepo = \""+e.remote+"\"\nref = \"master\"\n")
	write(t, projectConfigFile(e.project), "[components.util]\nenabled = false\n")
	if err := os.RemoveAll(filepath.Join(storePath(e.home, "util", pin.Commit), ".git")); err != nil {
		t.Fatal(err)
	}
	rows := mustInventory(t, e)
	if r := inventoryRow(t, rows, "util"); !r.Inactive || r.Store != storeUnreadable {
		t.Fatalf("util row = %+v", r)
	}
	if r := inventoryRow(t, rows, "dist"); r.Store != storeUnpinned || r.Inactive {
		t.Fatalf("dist row = %+v", r)
	}
}

func mustInventory(t *testing.T, e *utilityEnv) []itemRow {
	t.Helper()
	rows, err := inventory(e.home, e.repo, e.project, nil)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestInventoryUnpinnedCheckCount(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	write(t, filepath.Join(e.repo, "gap.toml"), "schema = \"2\"\n\n[components.dist]\nrepo = \""+e.remote+"\"\nref = \"master\"\n")
	if err := writeChecks(e.home, map[string]checkRecord{"dist": {Pin: "", New: 1}}); err != nil {
		t.Fatal(err)
	}
	if r := inventoryRow(t, mustInventory(t, e), "dist"); r.Pinned || r.New != 1 {
		t.Fatalf("row = %+v", r)
	}
	if err := writeChecks(e.home, map[string]checkRecord{"dist": {Pin: strings.Repeat("a", 40), New: 4}}); err != nil {
		t.Fatal(err)
	}
	if r := inventoryRow(t, mustInventory(t, e), "dist"); r.New != 0 {
		t.Fatalf("stale record shown: %+v", r)
	}
}

func TestTUILoadVerifiesEachStoreOnce(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	addedUtility(t, e, addProjectArgs(e.remote, "--id", "util"))
	for range 3 {
		if err := recordProject(e.home, t.TempDir()); err != nil {
			t.Fatal(err)
		}
	}
	stores := newStoreCache()
	items := loadTUIItems(e.home, e.repo, e.project, stores)
	projects := loadTUIProjects(e.home, e.repo, stores)
	if items.err != nil || projects.err != nil || len(projects.projects) != 4 {
		t.Fatalf("items %v projects %+v", items.err, projects)
	}
	if stores.misses != 1 {
		t.Fatalf("verified %d stores for one pin, want 1", stores.misses)
	}
}

func TestLoadTUIItemsKeepsInventoryError(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	write(t, checkPath(e.home), "not json\n")
	msg := loadTUIItems(e.home, e.repo, e.project, nil)
	if msg.err == nil || strings.Contains(msg.err.Error(), "%!w") || !strings.Contains(msg.err.Error(), "check.json") {
		t.Fatalf("err = %v", msg.err)
	}
}
