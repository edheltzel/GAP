package main

import (
	"path/filepath"
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

func TestNoOverrideChangesNothing(t *testing.T) {
	d := distro{Components: map[string]component{"echo": {ID: "echo", Ref: "v0.13.0"}}}
	got, err := applyOverrides(d, t.TempDir(), t.TempDir(), t.TempDir(), "")
	if err != nil || len(got.Components) != 1 {
		t.Fatalf("missing overrides must leave the distro: %+v %v", got, err)
	}
}
