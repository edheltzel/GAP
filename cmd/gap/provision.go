package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func initCmd(args []string) int {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	adopt := flags.Bool("adopt", false, "report existing project files before adding GAP configuration")
	if err := flags.Parse(args); err != nil || flags.NArg() > 1 {
		return 2
	}
	dir := "."
	if flags.NArg() == 1 {
		dir = flags.Arg(0)
	}
	root, err := filepath.Abs(dir)
	if err == nil {
		err = initProject(root, "project", *adopt)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func initProject(root, profile string, adopt bool) error {
	if adopt {
		for _, name := range []string{"AGENTS.md", "CLAUDE.md", "todo.txt", "forge", ".codegraph", ".empryo", "package.json", "go.mod", "Cargo.toml", "Package.swift"} {
			if _, err := os.Lstat(filepath.Join(root, name)); err == nil {
				fmt.Println("Found " + name)
			}
		}
	}
	config := filepath.Join(root, ".gap", "gap.toml")
	if _, err := os.Lstat(config); errors.Is(err, os.ErrNotExist) {
		text := "profile = " + strconv.Quote(profile) + "\nslug = " + strconv.Quote(filepath.Base(root)) + "\n"
		if err := writeAtomic(config, text); err != nil {
			return err
		}
		fmt.Println(config)
	} else if err != nil {
		return err
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
	if strings.Contains(text, gapBegin) || strings.Contains(text, gapEnd) {
		fmt.Println("Keeping marked file " + path)
		return nil
	}
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
	bad := false
	for _, id := range sortedIDs(d.Components) {
		c := d.Components[id]
		if c.Doctor == "" {
			fmt.Printf("%s: no doctor configured\n", id)
			continue
		}
		pin, pinned := lk.Components[id]
		dir := storePath(home, id, pin.Commit)
		if tree, err := worktreeID(dir); !pinned || err != nil || tree != pin.Tree {
			fmt.Printf("%s: pinned store missing or changed; run gap sync\n", id)
			bad = true
			continue
		}
		fmt.Printf("%s: %s\n", id, c.Doctor)
		if err := runComponentCommand(dir, c.Doctor); err != nil {
			fmt.Fprintf(os.Stderr, "%s doctor: %v\n", id, err)
			bad = true
		}
	}
	for _, harness := range []string{"claude", "omp", "pi", "codex", "cursor", "empryo", "grok"} {
		if path, err := exec.LookPath(harness); err == nil {
			fmt.Printf("%s: present (%s)\n", harness, path)
		} else {
			fmt.Printf("%s: absent; install this harness separately if needed\n", harness)
		}
	}
	if bad {
		return errSkipped
	}
	return nil
}
