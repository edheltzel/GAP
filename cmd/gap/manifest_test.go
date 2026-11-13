package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeUtilityRepo(t *testing.T, dir, manifest string) {
	t.Helper()
	write(t, filepath.Join(dir, "skills", "util-skill", "SKILL.md"), "---\nname: util-skill\ndescription: test\n---\n")
	write(t, filepath.Join(dir, "commands", "util-cmd.md"), "Run the utility.\n")
	if manifest != "" {
		write(t, filepath.Join(dir, utilityManifestName), manifest)
	}
}

func commitAll(t *testing.T, dir string) (commit, tree string) {
	t.Helper()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = gitEnv()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		run("init", "-q", "-b", "master")
	}
	run("-c", "user.name=gap", "-c", "user.email=gap@example.com", "add", "-A")
	run("-c", "user.name=gap", "-c", "user.email=gap@example.com", "commit", "-q", "--allow-empty", "-m", "utility")
	return run("rev-parse", "HEAD"), run("rev-parse", "HEAD^{tree}")
}

func TestReadmeManifestExample(t *testing.T) {
	readme := readString(t, "../../README.md")
	_, section, ok := strings.Cut(readme, "## Publish a utility")
	if !ok {
		t.Fatal("README has no Publish a utility section")
	}
	_, block, ok := strings.Cut(section, "```toml\n")
	if !ok {
		t.Fatal("Publish a utility has no toml example")
	}
	example, _, _ := strings.Cut(block, "```")
	dir := t.TempDir()
	write(t, filepath.Join(dir, utilityManifestName), example)
	write(t, filepath.Join(dir, "skills", "my-skill", "SKILL.md"), "---\nname: my-skill\ndescription: x\n---\n")
	write(t, filepath.Join(dir, "commands", "my-command.md"), "Do it.\n")
	m, err := readUtilityManifest(dir)
	if err != nil {
		t.Fatalf("README example manifest: %v", err)
	}
	if m.Hooks.Install == "" || m.Hooks.Uninstall == "" || len(m.MCP) != 1 {
		t.Fatalf("README example lost its hooks or mcp server: %+v", m)
	}
}

func TestUtilityManifest(t *testing.T) {
	full := `schema = "1"
name = "util"
description = "A test utility"
skills = ["util-skill"]
commands = ["commands/util-cmd.md"]

[hooks]
install = "./install.sh"
update = "./update.sh"
uninstall = "./uninstall.sh"
doctor = "./doctor.sh"
`
	dir := t.TempDir()
	writeUtilityRepo(t, dir, full)
	m, err := readUtilityManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "util" || m.Description != "A test utility" || len(m.Skills) != 1 || m.Skills[0] != "util-skill" || len(m.Commands) != 1 {
		t.Fatalf("manifest = %+v", m)
	}
	if m.Hooks != (utilityHooks{Install: "./install.sh", Update: "./update.sh", Uninstall: "./uninstall.sh", Doctor: "./doctor.sh"}) {
		t.Fatalf("hooks = %+v", m.Hooks)
	}

	minimal := t.TempDir()
	writeUtilityRepo(t, minimal, "schema = \"1\"\nname = \"mini\"\nskills = [\"util-skill\"]\n")
	if m, err := readUtilityManifest(minimal); err != nil || m.Hooks != (utilityHooks{}) || len(m.Commands) != 0 {
		t.Fatalf("minimal = %+v, %v", m, err)
	}

	missing := t.TempDir()
	writeUtilityRepo(t, missing, "")
	if _, err := readUtilityManifest(missing); !errors.Is(err, errNotUtility) {
		t.Fatalf("missing manifest error = %v", err)
	}

	bad := map[string]string{
		"schema":         "schema = \"2\"\nname = \"x\"\nskills = [\"util-skill\"]\n",
		"name":           "schema = \"1\"\nskills = [\"util-skill\"]\n",
		"unknown key":    "schema = \"1\"\nname = \"x\"\nskills = [\"util-skill\"]\nextra = \"y\"\n",
		"no skill dir":   "schema = \"1\"\nname = \"x\"\nskills = [\"absent\"]\n",
		"repeated skill": "schema = \"1\"\nname = \"x\"\nskills = [\"util-skill\", \"util-skill\"]\n",
		"command escape": "schema = \"1\"\nname = \"x\"\ncommands = [\"../x.md\"]\n",
		"command ext":    "schema = \"1\"\nname = \"x\"\ncommands = [\"commands/util-cmd.txt\"]\n",
		"missing cmd":    "schema = \"1\"\nname = \"x\"\ncommands = [\"commands/nope.md\"]\n",
		"unknown hook":   "schema = \"1\"\nname = \"x\"\n[hooks]\nbuild = \"make\"\n",
		"empty hook":     "schema = \"1\"\nname = \"x\"\n[hooks]\ninstall = \" \"\n",
		"empty":          "schema = \"1\"\nname = \"x\"\n",
	}
	for label, text := range bad {
		dir := t.TempDir()
		writeUtilityRepo(t, dir, text)
		if _, err := readUtilityManifest(dir); err == nil || !strings.Contains(err.Error(), utilityManifestName) {
			t.Errorf("%s: error = %v", label, err)
		}
	}
}
