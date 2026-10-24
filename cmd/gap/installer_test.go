package main

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallerRetainsSourcesAndHonorsLocalInvocation(t *testing.T) {
	cacheOutput, err := exec.Command("go", "env", "-json", "GOCACHE", "GOMODCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	var caches map[string]string
	if err := json.Unmarshal(cacheOutput, &caches); err != nil {
		t.Fatal(err)
	}
	repo, home, elsewhere := t.TempDir(), t.TempDir(), t.TempDir()
	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, src := range sources {
		if !strings.HasSuffix(src, "_test.go") {
			write(t, filepath.Join(repo, "cmd", "gap", src), readString(t, src))
		}
	}
	for _, name := range []string{"go.mod", "go.sum", "scripts/install.sh"} {
		write(t, filepath.Join(repo, name), readString(t, filepath.Join("../..", name)))
	}
	write(t, filepath.Join(repo, "global", "AGENTS.md"), "policy\n")
	const config = "schema = \"2\"\n[components.%s]\nrepo = \"github.com/example/component\"\nref = \"master\"\nnames = []\n"
	write(t, filepath.Join(repo, "gap.toml"), strings.Replace(config, "%s", "remote", 1))
	gitFixture(t, repo, "skill\n")
	write(t, filepath.Join(repo, "gap.toml"), strings.Replace(config, "%s", "local", 1))
	remoteURL := (&url.URL{Scheme: "file", Path: repo}).String()
	binDir := filepath.Join(home, "bin")
	env := append(os.Environ(),
		"HOME="+home, "GOBIN="+binDir, "GOENV=off", "GOFLAGS=", "GOWORK=off",
		"GOCACHE="+caches["GOCACHE"], "GOMODCACHE="+caches["GOMODCACHE"],
		"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=url."+remoteURL+".insteadOf",
		"GIT_CONFIG_VALUE_0=https://github.com/edheltzel/GAP.git",
	)
	installer := filepath.Join(repo, "scripts", "install.sh")
	var retained []string
	for _, mode := range []string{"local", "piped", "piped-again"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command("sh", installer)
			want := "local"
			if mode != "local" {
				cmd = exec.Command("/bin/sh")
				cmd.Stdin = strings.NewReader(readString(t, installer))
				want = "remote"
			}
			cmd.Dir, cmd.Env = elsewhere, env
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("installer: %v\n%s", err, out)
			}
			status := exec.Command(filepath.Join(binDir, "gap"), "status")
			status.Dir, status.Env = elsewhere, env
			out, err := status.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 || !strings.HasPrefix(string(out), want+" ref=master commit=unpinned store=unpinned\n") {
				t.Fatalf("installed binary must read the %s checkout after installer exits: %v\n%s", want, err, out)
			}
			checkouts, err := filepath.Glob(filepath.Join(home, ".gap", "checkouts", "*"))
			if err != nil {
				t.Fatal(err)
			}
			if mode == "local" {
				if len(checkouts) != 0 {
					t.Fatal("local invocation cloned instead of using its checkout")
				}
				return
			}
			if len(checkouts) != len(retained)+1 {
				t.Fatalf("curl install must retain a new checkout: %v", checkouts)
			}
			for _, dir := range retained {
				if _, err := os.Stat(filepath.Join(dir, "global", "AGENTS.md")); err != nil {
					t.Fatalf("reinstall removed an earlier source checkout: %v", err)
				}
			}
			retained = checkouts
		})
	}
}
