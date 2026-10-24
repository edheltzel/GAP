package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func initProject(root, profile string, adopt bool) error {
	if adopt {
		for _, name := range []string{"AGENTS.md", "CLAUDE.md", "todo.txt", "forge", ".codegraph", ".empryo", "package.json", "go.mod", "Cargo.toml", "Package.swift"} {
			if _, err := os.Lstat(filepath.Join(root, name)); err == nil {
				fmt.Println("Found " + name)
			}
		}
	}
	config := filepath.Join(root, ".gap", "gap.toml")
	_, err := os.Lstat(config)
	newConfig := errors.Is(err, os.ErrNotExist)
	if err != nil && !newConfig {
		return fmt.Errorf("read project configuration %s: %w", config, err)
	}
	configText := "profile = " + strconv.Quote(profile) + "\nslug = " + strconv.Quote(filepath.Base(root)) + "\n"
	if !newConfig {
		b, err := os.ReadFile(config)
		if err != nil {
			return fmt.Errorf("read project configuration %s: %w", config, err)
		}
		configText = string(b)
	}
	doc, err := parseTOML(configText)
	if err != nil {
		return fmt.Errorf("fix project configuration %s before running gap init: %w", config, err)
	}
	slug, ok := doc["slug"].(string)
	if !ok || strings.TrimSpace(slug) == "" {
		return fmt.Errorf("%s: set slug to a non-empty string before running gap init", config)
	}
	if newConfig {
		if err := writeAtomic(config, configText); err != nil {
			return err
		}
		fmt.Println(config)
	}
	path := filepath.Join(root, "AGENTS.md")
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to append policy pointer to non-regular file %s", path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	text := string(data)
	switch {
	case adopt && err == nil:
		fmt.Println("Keeping adopted file " + path)
	case strings.Contains(text, gapBegin) || strings.Contains(text, gapEnd):
		fmt.Println("Keeping marked file " + path)
	default:
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		if text != "" {
			text += "\n"
		}
		text += gapBegin + "\nFollow the [GAP global agent policy](https://github.com/edheltzel/GAP/blob/master/global/AGENTS.md).\n" + gapEnd + "\n"
		if err := writeAtomic(path, text); err != nil {
			return err
		}
		fmt.Println(path)
	}
	if profile != "crew" && doc["profile"] != "crew" {
		return provisionBreadcrumb(slug)
	}
	return nil
}

func runComponentCommand(dir, command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func doctorAll() error {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return err
	}
	projectRoot, err := invokingProjectRoot()
	if err != nil {
		return err
	}
	d, lk, err := effectivePins(repoRoot, home, projectRoot, "")
	if err != nil {
		return err
	}
	out := newReport(os.Stdout)
	bad := false
	missing := false
	var rows [][]string
	for _, id := range sortedIDs(d.Components) {
		c := d.Components[id]
		if c.Doctor == "" {
			rows = append(rows, []string{id, "no doctor configured"})
			continue
		}
		pin, pinned := lk.Components[id]
		if !pinned {
			rows = append(rows, []string{id, "not pinned; run gap pin " + id + "@" + c.Ref})
			bad = true
			continue
		}
		dir := storePath(home, id, pin.Commit)
		state, err := storeState(dir, pin.Tree)
		if err != nil {
			return err
		}
		if state != storeOK {
			detail := "pinned source checkout " + state + "; run gap sync"
			if state == storeChanged {
				detail += " after inspecting local changes"
			}
			rows = append(rows, []string{id, detail})
			bad = true
			if state == storeMissing {
				missing = true
			}
			continue
		}
		rows = append(rows, []string{id, c.Doctor})
		if err := runComponentCommand(dir, c.Doctor); err != nil {
			fmt.Fprintf(os.Stderr, "%s doctor: %v\n", id, err)
			bad = true
		}
		if id == "echo" {
			if err := recordEchoAdapters(home, dir, lk); err != nil {
				return err
			}
		}
	}
	if len(rows) > 0 {
		out.doctorTable(rows)
	}
	var harnesses [][]string
	for _, harness := range []string{"claude", "omp", "pi", "codex", "cursor", "empryo", "grok"} {
		if path, err := exec.LookPath(harness); err == nil {
			harnesses = append(harnesses, []string{harness, "present (" + path + ")"})
		} else {
			harnesses = append(harnesses, []string{harness, "absent; install this harness separately if needed"})
		}
	}
	out.doctorTable(harnesses)
	if missing {
		out.next("gap sync    fetch each missing source checkout, then rerun gap doctor")
	}
	if bad {
		return errSkipped
	}
	return nil
}
