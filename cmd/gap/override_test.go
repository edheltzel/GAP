package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOverrideDisablesComponent(t *testing.T) {
	home, repo := t.TempDir(), t.TempDir()
	d := distro{Components: map[string]component{
		"echo":      {ID: "echo", Ref: "v0.13.0"},
		"do-skills": {ID: "do-skills", Ref: "master", Skills: []string{"do-astro"}, SkillSet: map[string]skillSpec{"do-astro": {Load: "project"}}},
	}}
	write(t, filepath.Join(home, ".config", "gap", "gap.toml"), "[components.echo]\nenabled = false\n")
	got, err := applyOverrides(d, home, repo, repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, kept := got.Components["echo"]; kept {
		t.Fatal("a user override did not disable echo")
	}
	if got.Components["do-skills"].Ref != "master" {
		t.Fatal("an override changed a component it does not name")
	}
}

func TestProjectOverrideWinsOverUser(t *testing.T) {
	home, repo := t.TempDir(), t.TempDir()
	d := distro{Components: map[string]component{
		"do-skills": {ID: "do-skills", Skills: []string{"do-astro"}, SkillSet: map[string]skillSpec{"do-astro": {Load: "project"}}},
	}}
	write(t, filepath.Join(home, ".config", "gap", "gap.toml"), "[components.do-skills.skills.do-astro]\nload = \"off\"\n")
	write(t, filepath.Join(repo, ".gap", "gap.toml"), "[components.do-skills.skills.do-astro]\nload = \"always\"\n")
	got, err := applyOverrides(d, home, repo, repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Components["do-skills"].SkillSet["do-astro"].Load != "always" {
		t.Fatalf("project override lost: %q", got.Components["do-skills"].SkillSet["do-astro"].Load)
	}
}

func TestCrewProfileDropsMachineComponents(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, "profiles", "crew.toml"), "[components.recall]\nenabled = false\n[components.echo]\nenabled = false\n")
	d := distro{Components: map[string]component{
		"recall":    {ID: "recall"},
		"echo":      {ID: "echo"},
		"do-skills": {ID: "do-skills"},
	}}
	got, err := applyOverrides(d, t.TempDir(), repo, repo, "crew")
	if err != nil {
		t.Fatal(err)
	}
	if _, kept := got.Components["recall"]; kept {
		t.Fatal("the crew profile kept recall")
	}
	if _, kept := got.Components["echo"]; kept {
		t.Fatal("the crew profile kept echo")
	}
	if _, kept := got.Components["do-skills"]; !kept {
		t.Fatal("the crew profile dropped do-skills")
	}
}

func TestUserComponent(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	repo := t.TempDir()
	project := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote")
	writeUtilityRepo(t, remote, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	commit, tree := commitAll(t, remote)
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n")
	userToml := filepath.Join(home, ".config", "gap", "gap.toml")
	write(t, userToml, "[components.util]\nrepo = \""+remote+"\"\nref = \"master\"\n")
	if _, err := loadTestComponents(repo, home, project); err == nil || !strings.Contains(err.Error(), "gap update util") {
		t.Fatalf("missing user pin: %v", err)
	}
	if err := writeLock(userConfigDir(home), lockFile{Components: map[string]lockEntryM1{"util": {Ref: "master", Commit: commit, Tree: tree}}}); err != nil {
		t.Fatal(err)
	}
	entries, err := loadTestComponents(repo, home, project)
	if err != nil || len(entries) != 1 || entries[0].Skills[0] != "util-skill" {
		t.Fatalf("user component: %+v %v", entries, err)
	}
	if _, err := os.Stat(filepath.Join(repo, "gap.lock")); !os.IsNotExist(err) {
		t.Fatalf("distro lock written: %v", err)
	}

	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n\n[components.util]\nrepo = \""+remote+"\"\nref = \"master\"\nnames = [\"util-skill\"]\n")
	if _, err := loadTestComponents(repo, home, project); err == nil || !strings.Contains(err.Error(), "already a distro component") {
		t.Fatalf("collision: %v", err)
	}
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n")
	write(t, filepath.Join(project, ".gap", "gap.toml"), "[components.other]\nrepo = \""+remote+"\"\nref = \"master\"\n")
	if _, err := loadTestComponents(repo, home, project); err == nil || !strings.Contains(err.Error(), "only ~/.config/gap/gap.toml") {
		t.Fatalf("project add: %v", err)
	}
}

func TestNoOverrideChangesNothing(t *testing.T) {
	d := distro{Components: map[string]component{"echo": {ID: "echo", Ref: "v0.13.0"}}}
	got, err := applyOverrides(d, t.TempDir(), t.TempDir(), t.TempDir(), "")
	if err != nil || len(got.Components) != 1 {
		t.Fatalf("missing overrides must leave the distro: %+v %v", got, err)
	}
}
