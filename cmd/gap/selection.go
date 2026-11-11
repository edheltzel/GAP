package main

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"charm.land/huh/v2"
)

// errSyncCancelled means the user quit the sync picker before anything was written.
var errSyncCancelled = errors.New("sync cancelled")

type syncOptions struct {
	RemoveLegacy bool
	Profile      string
	Only, Skip   []string
	Yes          bool
}

// syncGroup is one unit the user can include in or leave out of a sync run.
type syncGroup struct {
	ID, Summary string
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func pinLabel(pin lockEntryM1) string {
	if pin.Commit == "" {
		return ""
	}
	short := pin.Commit[:min(7, len(pin.Commit))]
	return fmt.Sprintf(" @ %s (%s)", pin.Ref, short)
}

func syncGroups(d distro, pins map[string]lockEntryM1, manifest []manifestEntry, projectRoot string, harnesses, commands int) []syncGroup {
	policy := "publish AGENTS.md to " + plural(harnesses, "harness file")
	if commands > 0 {
		policy += " and " + plural(commands, "command")
	}
	groups := []syncGroup{{"policy", policy}}
	for _, id := range sortedIDs(d.Components) {
		c := d.Components[id]
		var parts []string
		skills := 0
		for _, name := range c.Skills {
			if renderLoad(c, name, projectRoot) {
				skills++
			}
		}
		if skills > 0 {
			parts = append(parts, plural(skills, "skill"))
		}
		if c.Install != "" {
			parts = append(parts, "run its installer")
		}
		if len(parts) == 0 {
			parts = append(parts, "nothing to install")
		}
		summary := strings.Join(parts, ", ") + pinLabel(pins[id])
		groups = append(groups, syncGroup{id, summary})
	}
	if len(manifest) > 0 {
		skills := 0
		for _, e := range manifest {
			skills += len(e.Skills)
		}
		groups = append(groups, syncGroup{"manifest", fmt.Sprintf("%s from %s in global/skills.json (skills.sh, needs npx)", plural(skills, "skill"), plural(len(manifest), "source"))})
	}
	return groups
}

func printPreview(groups []syncGroup, sel map[string]bool) {
	r := newReport(os.Stdout)
	if r.tty {
		r.preview(groups, sel)
		return
	}
	r.heading("gap sync will:")
	width := 0
	for _, g := range groups {
		width = max(width, len(g.ID))
	}
	for _, g := range groups {
		line := fmt.Sprintf("  %-*s  %s", width, g.ID, g.Summary)
		if !sel[g.ID] {
			line = r.tone(line+"  (skipped)", "#767676")
		}
		r.println(line)
	}
}

// chooseGroups resolves --only and --skip, then asks on a terminal unless a flag already decided. End of input cancels.
func chooseGroups(groups []syncGroup, only, skip []string, yes bool) (map[string]bool, error) {
	ids := make([]string, len(groups))
	for i, g := range groups {
		ids[i] = g.ID
	}
	for _, id := range append(slices.Clone(only), skip...) {
		if !slices.Contains(ids, id) {
			return nil, fmt.Errorf("unknown sync group %q; choose from: %s", id, strings.Join(ids, ", "))
		}
	}
	sel := map[string]bool{}
	for _, id := range ids {
		sel[id] = (len(only) == 0 || slices.Contains(only, id)) && !slices.Contains(skip, id)
	}
	if yes || len(only) > 0 || len(skip) > 0 || !termReady() {
		printPreview(groups, sel)
		return sel, nil
	}
	if pickReady() {
		sel, err := pickGroups(groups, sel)
		if err != nil {
			return nil, err
		}
		printPreview(groups, sel)
		return sel, nil
	}
	printPreview(groups, sel)
	fmt.Print("Sync all of it? [a]ll, [c]hoose, [q]uit [a] ")
	line := confirmLine()
	if line == "" {
		fmt.Println()
		return nil, errSyncCancelled
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "", "a", "y", "yes", "all":
		return sel, nil
	case "c":
		for _, g := range groups {
			for asked := false; ; asked = true {
				if asked {
					fmt.Println("Answer y, n, or q.")
				}
				fmt.Printf("Sync %s (%s)? [Y/n/q] ", g.ID, g.Summary)
				line := confirmLine()
				if line == "" {
					fmt.Println()
					return nil, errSyncCancelled
				}
				answer := strings.ToLower(strings.TrimSpace(line))
				if answer == "q" || answer == "quit" {
					return nil, errSyncCancelled
				}
				if answer == "" || answer == "y" || answer == "yes" || answer == "n" || answer == "no" {
					sel[g.ID] = answer != "n" && answer != "no"
					break
				}
			}
		}
		if !slices.Contains(slices.Collect(maps.Values(sel)), true) {
			return nil, errSyncCancelled
		}
		return sel, nil
	default:
		return nil, errSyncCancelled
	}
}

func skippedIDs(groups []syncGroup, sel map[string]bool) []string {
	var out []string
	for _, g := range groups {
		if !sel[g.ID] {
			out = append(out, g.ID)
		}
	}
	return out
}

// pickReady and pickGroups are the full-screen picker boundary. Tests replace them.
var (
	pickReady  = func() bool { return usesPicker(os.Getenv("TERM"), isTerminal(os.Stdin), isTerminal(os.Stdout)) }
	pickGroups = huhPick
)

// usesPicker keeps dumb terminals on the line prompt, where end of input cancels. huh's accessible mode reads it as confirm.
func usesPicker(term string, stdin, stdout bool) bool {
	return term != "dumb" && stdin && stdout
}

// huhPick shows every group as a checked option. Only enter syncs the checked groups; ctrl+c, a signal, or an empty pick cancels.
func huhPick(groups []syncGroup, sel map[string]bool) (map[string]bool, error) {
	return runPicker(groups, sel, os.Stdin, os.Stdout)
}

func runPicker(groups []syncGroup, sel map[string]bool, in io.Reader, out io.Writer) (map[string]bool, error) {
	width := 0
	for _, g := range groups {
		width = max(width, len(g.ID))
	}
	opts := make([]huh.Option[string], len(groups))
	for i, g := range groups {
		opts[i] = huh.NewOption(fmt.Sprintf("%-*s  %s", width, g.ID, g.Summary), g.ID).Selected(sel[g.ID])
	}
	var chosen []string
	field := huh.NewMultiSelect[string]().
		Title("gap sync").
		Description("Pick what to sync. space toggles, enter syncs, ctrl+c quits.").
		Options(opts...).
		Height(len(groups) + 2).
		Value(&chosen)
	form := huh.NewForm(huh.NewGroup(field)).
		WithShowHelp(false).
		WithOutput(out)
	if in != os.Stdin {
		form = form.WithInput(in)
	}
	err := form.Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return nil, errSyncCancelled
	}
	if err != nil {
		return nil, err
	}
	if form.State != huh.StateCompleted || len(chosen) == 0 {
		return nil, errSyncCancelled
	}
	picked := map[string]bool{}
	for _, g := range groups {
		picked[g.ID] = slices.Contains(chosen, g.ID)
	}
	return picked, nil
}
