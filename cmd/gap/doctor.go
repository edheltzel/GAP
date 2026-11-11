package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

type stateFile struct {
	Components map[string]stateEntry `json:"components"`
}

type stateEntry struct {
	Commit    string            `json:"commit"`
	Adapters  []string          `json:"adapters"`
	Installed string            `json:"installed,omitempty"`
	Approved  map[string]string `json:"approved,omitempty"`
	Skills    []string          `json:"skills,omitempty"`
	Commands  []string          `json:"commands,omitempty"`
}

func statePath(home string) string {
	return filepath.Join(home, ".gap", "state.json")
}

func readState(home string) (stateFile, error) {
	path := statePath(home)
	state := stateFile{Components: map[string]stateEntry{}}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return stateFile{}, err
	}
	if err := json.Unmarshal(b, &state); err != nil {
		return stateFile{}, fmt.Errorf("%s: %w", path, err)
	}
	if state.Components == nil {
		state.Components = map[string]stateEntry{}
	}
	return state, nil
}

// recordOwnership writes, per synced component, the home skills and command files gap placed for it.
// gap remove deletes only these paths.
func recordOwnership(home, projectRoot string, d distro, commands map[string][]string) error {
	if !slices.ContainsFunc(sortedIDs(d.Components), func(id string) bool { return d.Components[id].Store != "" }) {
		return nil
	}
	state, err := readState(home)
	if err != nil {
		return err
	}
	for _, id := range sortedIDs(d.Components) {
		c := d.Components[id]
		if c.Store == "" {
			continue
		}
		var skills []string
		for _, entry := range selectedEntries(c, c.Store, projectRoot) {
			if entry.Project == "" {
				skills = append(skills, entry.Skills...)
			}
		}
		entry := state.Components[id]
		entry.Skills, entry.Commands = skills, commands[id]
		state.Components[id] = entry
	}
	return writeState(home, state)
}

func writeState(home string, state stateFile) error {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	text := string(b) + "\n"
	if old, err := os.ReadFile(statePath(home)); err == nil && string(old) == text {
		return nil
	}
	return writeAtomic(statePath(home), text)
}

func recordEchoAdapters(home, dir string, lk lockFile) error {
	if dir == "" {
		fmt.Println("echo: no pinned store; no adapters recorded")
		return nil
	}
	script := filepath.Join(dir, "scripts", "install.sh")
	cmd := exec.Command(script, "--list-installed")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("echo reported nothing; no adapters recorded")
		return nil
	}
	var adapters []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			adapters = append(adapters, line)
		}
	}
	sort.Strings(adapters)
	return writeEchoState(home, lk.Components["echo"].Commit, adapters)
}

func writeEchoState(home, commit string, adapters []string) error {
	state, err := readState(home)
	if err != nil {
		return err
	}
	entry := state.Components["echo"]
	entry.Commit, entry.Adapters = commit, adapters
	state.Components["echo"] = entry
	return writeState(home, state)
}
