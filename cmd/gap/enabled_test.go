package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSetComponentEnabled(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	write(t, filepath.Join(e.repo, "gap.toml"), "schema = \"2\"\n\n[components.echo]\nrepo = \""+e.remote+"\"\nref = \"master\"\n")
	config := userConfigFile(e.home)
	write(t, config, "# mine\n[trust]\nrepos = []\n\n[components.echo]\nref = \"master\" # keep\n\n[components.echo.skills.x]\nload = \"off\"\n")
	if err := setComponentEnabled(e.home, e.repo, "echo", false); err != nil {
		t.Fatal(err)
	}
	want := "# mine\n[trust]\nrepos = []\n\n[components.echo]\nenabled = false\nref = \"master\" # keep\n\n[components.echo.skills.x]\nload = \"off\"\n"
	if got := readString(t, config); got != want {
		t.Fatalf("config:\n%s", got)
	}
	if err := setComponentEnabled(e.home, e.repo, "echo", false); err != nil || readString(t, config) != want {
		t.Fatalf("repeat changed the file: %v\n%s", err, readString(t, config))
	}
	rows, err := inventory(e.home, e.repo, e.project)
	if err != nil || !inventoryRow(t, rows, "echo").Disabled {
		t.Fatalf("inventory after disable: %+v %v", rows, err)
	}
	if err := setComponentEnabled(e.home, e.repo, "echo", true); err != nil || !strings.Contains(readString(t, config), "[components.echo]\nenabled = true\n") {
		t.Fatalf("enable: %v\n%s", err, readString(t, config))
	}
	if err := setComponentEnabled(e.home, e.repo, "nope", false); err == nil {
		t.Fatal("unknown id accepted")
	}
}

func TestSetComponentEnabledNewTableAndUtility(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	write(t, filepath.Join(e.repo, "gap.toml"), "schema = \"2\"\n\n[components.echo]\nrepo = \""+e.remote+"\"\nref = \"master\"\n")
	if err := setComponentEnabled(e.home, e.repo, "echo", false); err != nil {
		t.Fatal(err)
	}
	if got := readString(t, userConfigFile(e.home)); got != "[components.echo]\nenabled = false\n" {
		t.Fatalf("config: %q", got)
	}
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	captureStd(t, func() {
		if err := runAdd(addGlobalArgs(e.remote, "--id", "util")); err != nil {
			t.Fatal(err)
		}
	})
	if err := setComponentEnabled(e.home, e.repo, "util", false); err == nil || !strings.Contains(err.Error(), "gap remove util") {
		t.Fatalf("utility disable: %v", err)
	}
}
