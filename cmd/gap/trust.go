package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// trustList reads repos under [trust] in ~/.config/gap/gap.toml: owner/repo, or owner/* for every repo of an owner.
func trustList(home string) ([]string, error) {
	path := filepath.Join(userConfigDir(home), "gap.toml")
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	doc, err := parseTOML(string(b))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	raw, exists := doc["trust"]
	if !exists {
		return nil, nil
	}
	table, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: trust must be a table", path)
	}
	repos, ok := table["repos"].([]string)
	if _, present := table["repos"]; present && !ok {
		return nil, fmt.Errorf("%s: trust.repos must be a string array", path)
	}
	return repos, nil
}

// repoIdentity is owner/repo for a GitHub repo, otherwise the repo string as written.
func repoIdentity(repo string) string {
	if id, ok := strings.CutPrefix(repo, "github.com/"); ok {
		return strings.TrimSuffix(strings.ToLower(id), ".git")
	}
	return repo
}

func isTrusted(repo string, trust []string) bool {
	id := repoIdentity(repo)
	for _, t := range trust {
		t = strings.ToLower(strings.TrimPrefix(t, "github.com/"))
		if t == id || t == strings.ToLower(id) {
			return true
		}
		if owner, ok := strings.CutSuffix(t, "/*"); ok && strings.HasPrefix(strings.ToLower(id), owner+"/") {
			return true
		}
	}
	return false
}

// hookGate decides whether a component's hook may run. Distro components and trusted repos run.
type hookGate struct {
	trust []string
	state stateFile
}

func newHookGate(home string) (*hookGate, error) {
	trust, err := trustList(home)
	if err != nil {
		return nil, err
	}
	state, err := readState(home)
	if err != nil {
		return nil, err
	}
	return &hookGate{trust: trust, state: state}, nil
}

func (g *hookGate) trusted(c component) bool {
	return !c.User || isTrusted(c.Repo, g.trust)
}

func (g *hookGate) approve(c component, hook, command string) bool {
	if g.trusted(c) {
		return true
	}
	if !termReady() {
		fmt.Fprintf(os.Stderr, "%s: skipped its %s hook; it is not approved. Run gap from a terminal to approve it, or add %q to [trust] repos in ~/.config/gap/gap.toml.\n", c.ID, hook, repoIdentity(c.Repo))
		return false
	}
	fmt.Printf("%s (%s) wants to run its %s hook:\n  %s\n", c.ID, repoIdentity(c.Repo), hook, command)
	fmt.Print("Run it? [y/N] ")
	if strings.ToLower(strings.TrimSpace(confirmLine())) != "y" {
		fmt.Fprintf(os.Stderr, "%s: %s hook not approved; skipped\n", c.ID, hook)
		return false
	}
	return true
}

func (g *hookGate) recordInstalled(home, id, commit string) error {
	fresh, err := readState(home)
	if err != nil {
		return err
	}
	entry := fresh.Components[id]
	entry.Installed = commit
	fresh.Components[id] = entry
	if err := writeState(home, fresh); err != nil {
		return err
	}
	g.state = fresh
	return nil
}

// installHook is the hook a pin needs: install the first time, then update when the utility has one.
func installHook(c component, entry stateEntry) (string, string) {
	if entry.Installed != "" && c.Update != "" {
		return "update", c.Update
	}
	return "install", c.Install
}
