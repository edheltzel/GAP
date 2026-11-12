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
	Commit      string            `json:"commit"`
	Adapters    []string          `json:"adapters"`
	Installed   string            `json:"installed,omitempty"`
	Skills      []string          `json:"skills,omitempty"`
	Commands    []string          `json:"commands,omitempty"`
	SkillCopies map[string]string `json:"skillCopies,omitempty"`
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

func recordDeclarativeInstalls(home, projectRoot string, d distro, commands map[string][]string) error {
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
		entry := state.Components[id]
		complete := true
		for _, selected := range selectedEntries(c, c.Store, projectRoot) {
			for _, name := range selected.Skills {
				target := entryHome(home, selected)
				matches := installed(target, name) && skillDriftCount(c.Store, target, name) == 0
				complete = matches && complete
				if matches && selected.Project == "" {
					if entry.SkillCopies == nil {
						entry.SkillCopies = map[string]string{}
					}
					entry.SkillCopies[name] = filepath.Base(c.Store)
				}
			}
		}
		_, command := installHook(c, entry)
		if command == "" && len(commands[id]) == len(commandDests(home, ""))*len(c.Commands) && complete {
			entry.Installed = filepath.Base(c.Store)
		}
		state.Components[id] = entry
	}
	return writeState(home, state)
}

func recordOwnership(home, id string, skills, commands []string, store string) error {
	if id == "" {
		return nil
	}
	state, err := readState(home)
	if err != nil {
		return err
	}
	entry := state.Components[id]
	entry.Skills = append(entry.Skills, skills...)
	slices.Sort(entry.Skills)
	entry.Skills = slices.Compact(entry.Skills)
	if store != "" {
		if entry.SkillCopies == nil {
			entry.SkillCopies = map[string]string{}
		}
		for _, name := range skills {
			entry.SkillCopies[name] = filepath.Base(store)
		}
	}
	entry.Commands = append(entry.Commands, commands...)
	slices.Sort(entry.Commands)
	entry.Commands = slices.Compact(entry.Commands)
	state.Components[id] = entry
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
