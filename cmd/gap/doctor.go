package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type stateFile struct {
	Components map[string]stateEntry `json:"components"`
}

type stateEntry struct {
	Commit   string   `json:"commit"`
	Adapters []string `json:"adapters"`
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
	path := filepath.Join(home, ".gap", "state.json")
	state := stateFile{Components: map[string]stateEntry{}}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &state); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if state.Components == nil {
			state.Components = map[string]stateEntry{}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	state.Components["echo"] = stateEntry{Commit: commit, Adapters: adapters}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, string(b)+"\n")
}
