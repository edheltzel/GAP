package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type uninstallOptions struct {
	Yes    bool
	DryRun bool
}

type policyAction int

const (
	policyAbsent policyAction = iota
	policyLeave
	policyStrip
	policyDelete
)

type policyTarget struct {
	path   string
	cursor bool
}

// uninstallPlan is what gap uninstall found in GAP's records before it writes anything.
type uninstallPlan struct {
	home, codexHome, codexPolicy string
	utilities                    []string
	userIDs                      map[string]bool
	state                        stateFile
	ledger                       map[string]string
	policies                     []policyTarget
	commands                     []string
	skills                       []string
}

// classifyPolicy decides what uninstall does with one policy file and returns the text that remains after a strip.
func classifyPolicy(path string, cursor bool, ledger map[string]string) (policyAction, string, string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return policyAbsent, "", "", nil
	}
	if err != nil {
		return policyLeave, "", "", err
	}
	if isSymlink(info) {
		return policyLeave, "", "a symlink", nil
	}
	if !info.Mode().IsRegular() {
		return policyLeave, "", "not a regular file", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return policyLeave, "", "", err
	}
	text := string(b)
	if _, _, _, _, ok := gapSpan(text); ok {
		next := text
		for {
			start, _, _, end, ok := gapSpan(next)
			if !ok {
				break
			}
			next = next[:start] + next[end:]
		}
		rest := next
		if cursor {
			rest = strings.TrimPrefix(rest, cursorFront)
		}
		if strings.TrimSpace(rest) == "" {
			return policyDelete, "", "only the GAP block", nil
		}
		return policyStrip, next, "remove the GAP block", nil
	}
	if last := ledger[path]; last != "" && last != cleaned {
		if digest(text) == last {
			return policyDelete, "", "whole-file GAP copy", nil
		}
		if cut, ok := ledgerPrefixEnd(text, last); ok {
			if strings.TrimSpace(text[cut:]) == "" {
				return policyDelete, "", "only the recorded GAP prefix", nil
			}
			return policyStrip, text[cut:], "remove the recorded GAP prefix", nil
		}
	}
	return policyLeave, "", "no GAP markers and not a recorded GAP copy", nil
}

// backupTarget is where path goes under this run's backup folder.
func backupTarget(home, backup, path string) string {
	rel, err := filepath.Rel(home, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		rel = strings.TrimPrefix(path, string(filepath.Separator))
	}
	return filepath.Join(backup, rel)
}

var renameBackup = os.Rename

// moveToBackup moves path into the backup folder. A missing path is not an error.
func moveToBackup(home, backup, path string) (bool, error) {
	if _, err := os.Lstat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	dest := backupTarget(home, backup, path)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return false, err
	}
	if err := renameBackup(path, dest); err != nil {
		if !isCrossDevice(err) {
			return false, err
		}
		if err := copyTree(path, dest); err != nil {
			os.RemoveAll(dest)
			return false, err
		}
		if err := os.RemoveAll(path); err != nil {
			return false, err
		}
	}
	return true, nil
}

// dropSkillLockEntry removes name from the skills.sh lock when the lock records it.
func dropSkillLockEntry(home, name string) error {
	path := skillLockPath(home)
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(b, &root); err != nil {
		return err
	}
	var skills map[string]json.RawMessage
	if raw, ok := root["skills"]; ok {
		if err := json.Unmarshal(raw, &skills); err != nil {
			return err
		}
	}
	if _, ok := skills[name]; !ok {
		return nil
	}
	delete(skills, name)
	root["skills"], err = json.Marshal(skills)
	if err != nil {
		return err
	}
	b, err = json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, string(b)+"\n")
}

// planUninstall reads GAP's records and lists what uninstall would remove. It writes nothing.
func planUninstall() (*uninstallPlan, error) {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return nil, err
	}
	codexHome, err := resolveCodexHome(os.Getenv("CODEX_HOME"), home)
	if err != nil {
		return nil, err
	}
	p := &uninstallPlan{home: home, codexHome: codexHome, userIDs: map[string]bool{}}
	p.codexPolicy = filepath.Join(codexHome, "gap", "global-agent-policy.md")
	d, _, err := registeredPins(repoRoot, home)
	if err != nil {
		return nil, err
	}
	for _, id := range sortedIDs(d.Components) {
		if d.Components[id].User {
			p.utilities = append(p.utilities, id)
			p.userIDs[id] = true
		}
	}
	if p.state, err = readState(home); err != nil {
		return nil, err
	}
	if p.ledger, err = readLedger(filepath.Join(home, ".gap", "published.json")); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, dest := range append(policyDests(home, p.codexPolicy), policyDest{dest: filepath.Join(codexHome, "AGENTS.md")}) {
		if !seen[dest.dest] {
			seen[dest.dest] = true
			p.policies = append(p.policies, policyTarget{dest.dest, dest.cursor})
		}
	}
	commandDirs := map[string]bool{}
	for _, dest := range commandDests(home, "") {
		commandDirs[filepath.Clean(dest)] = true
	}
	commands := map[string]bool{}
	names, err := globalCommandNames(repoRoot)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		for _, dest := range commandDests(home, name) {
			commands[dest] = true
		}
	}
	for path := range p.ledger {
		if commandDirs[filepath.Dir(path)] {
			commands[path] = true
		}
	}
	for _, entry := range p.state.Components {
		for _, path := range entry.Commands {
			delete(commands, path)
		}
	}
	p.commands = slices.Sorted(maps.Keys(commands))
	p.skills, err = gapSkills(home, repoRoot, p.state)
	return p, err
}

func gapSkills(home, repoRoot string, state stateFile) ([]string, error) {
	skills := map[string]bool{}
	for _, entry := range state.Components {
		for _, name := range entry.Skills {
			skills[name] = true
		}
		for name := range entry.SkillCopies {
			skills[name] = true
		}
	}
	manifest, err := readManifest(repoRoot)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	lock, err := readSkillLock(home)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s: %w", skillLockPath(home), err)
	}
	for _, e := range manifest {
		for _, name := range e.Skills {
			if l, ok := lock[name]; ok && sameSource(entryRoot(repoRoot, e), e, l) {
				skills[name] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(skills)), nil
}

// preview prints what uninstall will remove and what it leaves, in the report style.
func (p *uninstallPlan) preview(out *report) {
	item := func(text string) { out.printf("  %s %s\n", out.tone("-", colorAccent), text) }
	leave := func(text string) { out.printf("  %s %s\n", out.tone("○", colorDim), text) }
	out.heading("gap uninstall will remove")
	if len(p.utilities) > 0 {
		out.println(out.bold("Utilities", colorAccent))
		for _, id := range p.utilities {
			e := p.state.Components[id]
			item(fmt.Sprintf("%s: uninstall hook, %d skills, %d commands, %d mcp servers", id, len(e.Skills), len(e.Commands), len(e.MCP)))
		}
	}
	var records []string
	for _, id := range slices.Sorted(maps.Keys(p.state.Components)) {
		if p.userIDs[id] {
			continue
		}
		e := p.state.Components[id]
		for _, key := range slices.Sorted(maps.Keys(e.MCP)) {
			records = append(records, id+": mcp server "+key)
		}
		for _, path := range e.Commands {
			records = append(records, id+": "+tildePath(path))
		}
	}
	if len(records) > 0 {
		out.println(out.bold("Component records", colorAccent))
		for _, r := range records {
			item(r)
		}
	}
	var left []string
	out.println(out.bold("Policy", colorAccent))
	for _, t := range p.policies {
		action, _, reason, err := classifyPolicy(t.path, t.cursor, p.ledger)
		switch {
		case err != nil:
			left = append(left, fmt.Sprintf("%s: %v", tildePath(t.path), err))
		case action == policyStrip:
			item(tildePath(t.path) + ": " + reason)
		case action == policyDelete:
			item(tildePath(t.path) + ": delete, " + reason)
		case action == policyLeave:
			left = append(left, tildePath(t.path)+": "+reason)
		}
	}
	hooksPath := filepath.Join(p.codexHome, "hooks.json")
	if b, err := os.ReadFile(hooksPath); err == nil {
		if _, _, res := removeGapHook(string(b), p.codexPolicy); res == hookChanged {
			item(tildePath(hooksPath) + ": GAP SessionStart and SubagentStart hooks")
		}
	}
	out.println(out.bold("Commands", colorAccent))
	for _, path := range p.commands {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if digest(string(b)) == p.ledger[path] {
			item(tildePath(path))
		} else {
			left = append(left, tildePath(path)+": not the recorded GAP publish")
		}
	}
	if len(p.skills) > 0 {
		out.println(out.bold("Skills", colorAccent))
		for _, name := range p.skills {
			item(name + " in ~/.agents/skills and ~/.claude/skills")
		}
	}
	out.println(out.bold("Records", colorAccent))
	for _, rel := range []string{"store", "state.json", "published.json"} {
		item(tildePath(filepath.Join(p.home, ".gap", rel)))
	}
	out.heading("Left in place")
	for _, l := range left {
		leave(l)
	}
	for _, rel := range []string{"backups", "recovery", "checkouts"} {
		leave(tildePath(filepath.Join(p.home, ".gap", rel)))
	}
	leave(tildePath(userConfigDir(p.home)) + " (your settings and trust list)")
	leave("the gap binary")
}

// uninstallAll removes what gap sync and gap add put on this machine, using only GAP's records.
// The ~/.gap records are removed last, and only when every earlier step succeeded, so a failed run can be repeated.
// pendingUninstall is a state entry no utility id can match. It holds every utility's skills and commands before
// the utilities are removed, so an interrupted run still knows to clean them up when it is repeated.
const pendingUninstall = ":uninstall"

// holdUtilityCleanup records the utilities' skills and commands under pendingUninstall before any of them is removed.
func holdUtilityCleanup(home string, ids []string) error {
	state, err := readState(home)
	if err != nil {
		return err
	}
	held := state.Components[pendingUninstall]
	skills, commands := map[string]bool{}, map[string]bool{}
	for _, name := range held.Skills {
		skills[name] = true
	}
	for _, path := range held.Commands {
		commands[path] = true
	}
	for _, id := range ids {
		entry := state.Components[id]
		for _, name := range entry.Skills {
			skills[name] = true
		}
		for name := range entry.SkillCopies {
			skills[name] = true
		}
		for _, path := range entry.Commands {
			commands[path] = true
		}
	}
	if len(skills) == 0 && len(commands) == 0 {
		return nil
	}
	held.Skills = slices.Sorted(maps.Keys(skills))
	held.Commands = slices.Sorted(maps.Keys(commands))
	state.Components[pendingUninstall] = held
	return writeState(home, state)
}

func uninstallAll(opts uninstallOptions) error {
	p, err := planUninstall()
	if err != nil {
		return err
	}
	out := newReport(os.Stdout)
	p.preview(out)
	if opts.DryRun {
		out.println()
		out.println("Dry run: nothing removed.")
		return nil
	}
	if !opts.Yes {
		if !termReady() {
			return errors.New("gap uninstall needs a terminal to confirm; rerun with --yes")
		}
		fmt.Print("\nRemove these? [y/N] ")
		if strings.ToLower(strings.TrimSpace(confirmLine())) != "y" {
			fmt.Println("Nothing removed.")
			return nil
		}
	}
	home := p.home
	backup := backupPath(home, home)
	failed := false
	fail := func(format string, args ...any) {
		failed = true
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
	var kept []string

	if err := holdUtilityCleanup(home, p.utilities); err != nil {
		return err
	}
	for _, id := range p.utilities {
		if err := removeUtility(id); err != nil {
			fail("%s: %v", id, err)
		}
	}

	state, err := readState(home)
	if err != nil {
		return err
	}
	if failed {
		return errors.New("gap uninstall did not finish removing utilities; records were kept for a retry")
	}
	for _, id := range slices.Sorted(maps.Keys(state.Components)) {
		entry := state.Components[id]
		if len(entry.MCP) > 0 {
			if err := removeComponentMCP(home, id); err != nil {
				fail("%v", err)
				if fresh, err := readState(home); err == nil {
					entry.MCP = fresh.Components[id].MCP
				}
			} else {
				entry.MCP = nil
			}
		}
		var remaining []string
		for _, path := range entry.Commands {
			moved, err := moveToBackup(home, backup, path)
			if err != nil {
				fail("%s: %v", path, err)
				remaining = append(remaining, path)
				continue
			}
			if moved {
				out.wrote(path)
			}
		}
		entry.Commands = remaining
		state.Components[id] = entry
	}
	if err := writeState(home, state); err != nil {
		fail("%v", err)
	}

	for _, t := range p.policies {
		action, next, reason, err := classifyPolicy(t.path, t.cursor, p.ledger)
		if err != nil {
			fail("%s: %v", t.path, err)
			continue
		}
		switch action {
		case policyStrip:
			if err := copyFile(t.path, backupTarget(home, backup, t.path)); err != nil {
				fail("%s: %v", t.path, err)
			} else if err := writeAtomic(t.path, next); err != nil {
				fail("%s: %v", t.path, err)
			} else {
				out.wrote(t.path)
			}
		case policyDelete:
			if _, err := moveToBackup(home, backup, t.path); err != nil {
				fail("%s: %v", t.path, err)
			} else {
				out.wrote(t.path)
			}
		case policyLeave:
			kept = append(kept, fmt.Sprintf("Left %s: %s", t.path, reason))
		}
	}
	os.Remove(filepath.Dir(p.codexPolicy))

	hooksPath := filepath.Join(p.codexHome, "hooks.json")
	if info, err := os.Lstat(hooksPath); err == nil {
		if isSymlink(info) {
			kept = append(kept, "Left "+hooksPath+": a symlink")
		} else if b, err := os.ReadFile(hooksPath); err != nil {
			fail("%s: %v", hooksPath, err)
		} else {
			next, empty, res := removeGapHook(string(b), p.codexPolicy)
			switch res {
			case hookMalformed:
				kept = append(kept, "Left "+hooksPath+": not JSON")
			case hookChanged:
				if err := copyFile(hooksPath, backupTarget(home, backup, hooksPath)); err != nil {
					fail("%s: %v", hooksPath, err)
				} else if empty {
					if err := os.Remove(hooksPath); err != nil {
						fail("%s: %v", hooksPath, err)
					}
				} else if err := writeAtomic(hooksPath, next); err != nil {
					fail("%s: %v", hooksPath, err)
				}
				out.wrote(hooksPath)
			}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		fail("%s: %v", hooksPath, err)
	}

	for _, path := range p.commands {
		b, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			fail("%s: %v", path, err)
			continue
		}
		if digest(string(b)) != p.ledger[path] {
			kept = append(kept, "Left "+path+": not the recorded GAP publish")
			continue
		}
		if _, err := moveToBackup(home, backup, path); err != nil {
			fail("%s: %v", path, err)
		} else {
			out.wrote(path)
		}
	}

	for _, name := range p.skills {
		ok := true
		for _, dir := range skillDirs(home, name) {
			if moved, err := moveToBackup(home, backup, dir); err != nil {
				fail("%s: %v", dir, err)
				ok = false
			} else if moved {
				out.wrote(dir)
			}
		}
		if ok {
			if err := dropSkillLockEntry(home, name); err != nil {
				fail("%s: %v", skillLockPath(home), err)
			}
		}
	}

	for _, line := range kept {
		fmt.Println(line)
	}
	if failed {
		return fmt.Errorf("gap uninstall did not finish; %s and %s were kept so you can fix the errors above and run gap uninstall again", statePath(home), filepath.Join(home, ".gap", "published.json"))
	}
	for _, path := range []string{filepath.Join(home, ".gap", "store"), statePath(home), filepath.Join(home, ".gap", "published.json")} {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	fmt.Println("Kept " + filepath.Join(home, ".gap", "backups"))
	fmt.Println("Kept " + filepath.Join(home, ".gap", "recovery"))
	fmt.Println("Kept " + filepath.Join(home, ".gap", "checkouts") + " and " + userConfigDir(home))
	if bin, err := os.Executable(); err == nil {
		fmt.Println("To remove the gap binary, run: trash " + shellQuote(bin))
	}
	out.done(true, "Uninstall complete.")
	return nil
}
