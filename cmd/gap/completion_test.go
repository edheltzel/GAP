package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runGap(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	cmd := rootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestCompletionInstallPerShell(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	want := map[string][2]string{
		"bash": {".local/share/bash-completion/completions/gap", "__start_gap"},
		"zsh":  {".local/share/zsh/site-functions/_gap", "#compdef gap"},
		"fish": {".config/fish/completions/gap.fish", "complete -c gap"},
	}
	for shell, w := range want {
		t.Setenv("SHELL", "/usr/bin/"+shell)
		if _, err := runGap(t, "completion", "--install"); err != nil {
			t.Fatalf("%s: %v", shell, err)
		}
		got, err := os.ReadFile(filepath.Join(home, w[0]))
		if err != nil || !strings.Contains(string(got), w[1]) {
			t.Fatalf("%s completions at %s: %v\n%.200s", shell, w[0], err, got)
		}
	}
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	if _, err := runGap(t, "completion", "--install", "zsh"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(data, "zsh", "site-functions", "_gap")); err != nil {
		t.Fatalf("XDG_DATA_HOME ignored: %v", err)
	}
}

func TestCompletionRefusesUnknownShell(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/sh")
	if _, err := runGap(t, "completion", "--install"); err == nil || !strings.Contains(err.Error(), "bash, zsh, and fish") {
		t.Fatalf("unknown $SHELL: %v", err)
	}
	if _, err := runGap(t, "completion"); err == nil || !strings.Contains(err.Error(), "name a shell") {
		t.Fatalf("bare completion: %v", err)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Fatalf("a refused install wrote files: %v", entries)
	}
}

func TestCompletionListsCommandsAndIDs(t *testing.T) {
	out, err := runGap(t, "__complete", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"completion\t", "sync\t", "update\t"} {
		if !strings.Contains(out, want) {
			t.Fatalf("subcommand completions lack %q:\n%s", want, out)
		}
	}
	out, err = runGap(t, "__complete", "completion", "")
	if err != nil || !strings.Contains(out, "bash\nzsh\nfish\n") {
		t.Fatalf("shell completions: %v\n%s", err, out)
	}
}

func TestCompletionOffersItemIDs(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	quietStdout(t)
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	if err := runAdd(addGlobalArgs(e.remote, "--id", "util")); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"__complete", "update", ""}, "util\t"},
		{[]string{"__complete", "remove", ""}, "util\t"},
		{[]string{"__complete", "pin", ""}, "util@\t"},
	} {
		out, err := runGap(t, c.args...)
		if err != nil || !strings.Contains(out, c.want) {
			t.Fatalf("%v: %v\n%s", c.args, err, out)
		}
	}
	out, _ := runGap(t, "__complete", "update", "util", "")
	if strings.Contains(out, "util\t") {
		t.Fatalf("update takes one id, but a second was offered:\n%s", out)
	}
	out, err := runGap(t, "__complete", "sync", "--only", "")
	if err != nil || !strings.Contains(out, "policy\t") || !strings.Contains(out, "util\t") {
		t.Fatalf("sync --only must offer policy and every enabled group, utilities included: %v\n%s", err, out)
	}
	out, _ = runGap(t, "__complete", "sync", "--skip", "policy,")
	if strings.Contains(out, "policy,policy") {
		t.Fatalf("sync --skip offered a group already chosen:\n%s", out)
	}
}
