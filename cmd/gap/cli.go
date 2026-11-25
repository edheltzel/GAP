package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"
)

// version is set by the release build (-X main.version). Empty means a source build, where fang reads the module version.
var version string

func main() {
	opts := []fang.Option{
		fang.WithoutManpage(),
		fang.WithoutCompletions(),
		fang.WithColorSchemeFunc(gapColorScheme),
		fang.WithErrorHandler(func(w io.Writer, _ fang.Styles, err error) {
			fmt.Fprintln(w, err)
		}),
	}
	if version != "" {
		opts = append(opts, fang.WithVersion(version))
	}
	if err := fang.Execute(context.Background(), rootCmd(), opts...); err != nil {
		os.Exit(1)
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "gap",
		Short: "Publish one set of agent rules to every harness",
		Long: `gap publishes global/AGENTS.md and each global/commands/*.md into the files
those harnesses already read and installs each component pinned in gap.toml from gap.lock.

Installation, source checkout requirements, and updates:
https://gap.edheltzel.codes/docs/#install-gap
Selected skills install directly from the pinned store without Node.
sync never moves pins. pin is the only command that writes the distro's gap.lock.
add, update, and remove manage utilities you add yourself: any git repo or local folder
of skills, recorded in ~/.config/gap/gap.toml and pinned in ~/.config/gap/gap.lock.

Global vs project:
  global   ~/.config/gap/gap.toml. Distro skills go to ~/.agents/skills and ~/.claude/skills.
           gap add -g owner/repo adds a utility here; its skills go to ~/.agents/skills,
           linked into each agent's own folder.
  project  <repo>/.gap/gap.toml. gap add owner/repo (the default, or -p) inside the repo
           adds it here; its skills go to <repo>/.agents/skills, linked into each agent's
           project folder. A distro skill with load = "project" installs into
           <repo>/.agents/skills and <repo>/.claude/skills when the repo matches its
           when.files. enabled = false leaves a component out.
Commands, hooks, and MCP servers are always machine-wide.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if tuiReady() {
				return runTUI()
			}
			exitCode = 2
			return cmd.Help()
		},
	}
	root.AddCommand(tuiCmd(), syncCmd(), statusCmd(), sourceCmd(), restoreCmd(), pinCmd(), doctorCmd(), initCommand(), addCmd(), checkCmd(), updateCmd(), removeCmd(), listCmd(), uninstallCmd(), searchCmd())
	return root
}

var exitCode int

func finish(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errSkipped) {
		exitCode = 1
		return nil
	}
	if errors.Is(err, errSyncCancelled) {
		fmt.Println("Nothing synced.")
		return nil
	}
	if errors.Is(err, errAddCancelled) {
		fmt.Println("Installation cancelled")
		return nil
	}
	return err
}

func tuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the terminal UI: update, check, remove, and disable items; see status and projects",
		Long: `tui lists every component and utility with its pin, store, drift, and new commits.
u updates the selected item, c checks it, C checks all. space marks utilities and d removes
the marked ones (or the selected one) after one confirmation; files move to ~/.gap/backups.
Distro components are locked: x disables or enables one in ~/.config/gap/gap.toml.
The Status view shows gap status. The Projects view lists each repo recorded in
~/.gap/projects.json by gap init, gap add --project, and gap sync; f forgets one.
Plain gap opens it too when stdin and stdout are a terminal and NO_COLOR and TERM=dumb are unset.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !tuiReady() {
				exitCode = 2
				fmt.Fprintln(os.Stderr, "gap tui needs a terminal on stdin and stdout, with NO_COLOR and TERM=dumb unset; use gap list and gap status instead.")
				return nil
			}
			return runTUI()
		},
	}
}

func syncCmd() *cobra.Command {
	var opts syncOptions
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Preview, then publish the policy and commands, install component skills, check skills for drift",
		Long: `sync prints what it will touch, grouped as policy and each component in gap.toml.
On a terminal it opens a picker with every group
checked: space toggles a group, enter syncs, ctrl+c quits before anything is written.
--only, --skip, or --yes skip the picker. Without a terminal every group syncs unless
a flag leaves it out.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return finish(syncAll(opts))
		},
	}
	cmd.Flags().BoolVar(&opts.RemoveLegacy, "remove-codex-legacy", false, "strip the old GAP text from Codex AGENTS.md once both hooks are trusted")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "apply profiles/<name>.toml over the distro for this run")
	cmd.Flags().StringSliceVar(&opts.Only, "only", nil, "sync only these groups (policy or a component id)")
	cmd.Flags().StringSliceVar(&opts.Skip, "skip", nil, "leave these groups out of this run")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "sync every selected group without asking")
	cmd.MarkFlagsMutuallyExclusive("only", "skip")
	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show each component pin, store, and skill drift. Writes nothing",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			err := statusAll()
			if errors.Is(err, errCrew) {
				return nil
			}
			if err == nil || errors.Is(err, errSkipped) {
				printRecallCandidates()
			}
			return finish(err)
		},
	}
}

var recallTimeout = 5 * time.Second

func printRecallCandidates() {
	recall, err := exec.LookPath("recall")
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), recallTimeout*4/5)
	defer cancel()
	cmd := exec.CommandContext(ctx, recall, "promote", "--json")
	cmd.WaitDelay = recallTimeout / 5
	out, err := cmd.Output()
	if err != nil {
		return
	}
	var report struct {
		Candidates []json.RawMessage `json:"candidates"`
	}
	if json.Unmarshal(out, &report) != nil || report.Candidates == nil {
		return
	}
	fmt.Printf("recall: %d promotion candidates\n", len(report.Candidates))
}

func sourceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "source",
		Short: "Print the GAP checkout that sync publishes from. Writes nothing",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := sourceRoot()
			if err != nil {
				return err
			}
			fmt.Println(root)
			return nil
		},
	}
}

func restoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore [id]",
		Short: "Reinstall drifted component skills from the lock. No network when the store exists",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) == 1 {
				id = args[0]
			}
			return finish(restoreAll(id))
		},
	}
}

func pinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pin <id>@<ref>",
		Short: "Record id's commit and tree in gap.lock",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return pinComponent(args[0]) },
	}
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run each component's doctor and report which harnesses are present",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return finish(doctorAll()) },
	}
}

func addCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add <source>",
		Aliases: []string{"install", "i", "a"},
		Short:   "Add skills from any git repo or local folder and install them for your agents",
		Long: `add installs skills from any git repository or local folder, with the same flags as
npx skills add. It records the repo in ~/.config/gap/gap.toml, pins it in
~/.config/gap/gap.lock, and installs the skills you pick. A repo with a gap-utility.toml
also installs its commands, MCP servers, and hooks; a hook asks before it runs unless the
repo is listed under [trust] repos in ~/.config/gap/gap.toml.

Sources:
  owner/repo, owner/repo/path, owner/repo@skill, owner/repo@v1.2.3, owner/repo#ref[@skill],
  GitHub, GitLab, and Azure Repos URLs (a tree URL picks the ref and path), git@host:path,
  ssh:// and https:// git URLs, and a local folder inside a git repository.
  Download, well-known, and Notion sources are refused.

Scope:
  project (default, -p, --local)  skills go to <repo>/.agents/skills of the repo you run it
                                  in, recorded with load = "project" in ~/.config/gap/gap.toml
                                  and with the selection in that repo's .gap/gap.toml.
  global (-g)                     skills go to ~/.agents/skills.
  Without -g, -p, or -y, gap asks. Each agent outside .agents/skills gets a symlink to that
  copy, or a copy of its own with --copy. gap sync keeps every copy current.

Commands, hooks, and MCP servers are machine-wide in both scopes. gap remove <id>
removes the utility everywhere, including every project copy.`,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if slices.Contains(args, "-h") || slices.Contains(args, "--help") {
				return cmd.Help()
			}
			return finish(runAdd(args))
		},
	}
	flags := cmd.Flags()
	flags.BoolP("global", "g", false, "install for every project, under home")
	flags.BoolP("project", "p", false, "install into the current repo (the default)")
	flags.Bool("local", false, "same as --project")
	flags.BoolP("yes", "y", false, "install without prompts")
	flags.BoolP("list", "l", false, "list the source's skills and exit")
	flags.Bool("all", false, "every skill to every agent, without prompts")
	flags.StringSliceP("agent", "a", nil, "agents to install to; '*' for all")
	flags.StringSliceP("skill", "s", nil, "skills to install; '*' for all")
	flags.String("metadata", "", "accepted for compatibility; unused")
	flags.Bool("full-depth", false, "search the whole repo for skills")
	flags.Bool("json", false, "print the results as JSON (needs -y or --all)")
	flags.Bool("copy", false, "copy into each agent folder instead of linking")
	flags.StringSlice("subagent", nil, "Eve subagents to install to")
	flags.String("rename", "", "install the one selected skill under a new name")
	flags.String("id", "", "utility id (default: the repo name, lowercased)")
	return cmd
}

func checkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check [id]",
		Short: "List the commits waiting on each component's ref. Changes no pin",
		Long: `check fetches each component's ref into ~/.gap/cache and lists the commits after its pin.
It changes no pin and installs nothing; gap status shows the counts from the last check.
A component with pin = true is skipped unless it is named.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) == 1 {
				id = args[0]
			}
			return finish(checkComponents(id))
		},
	}
}

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [id]",
		Short: "Move pins to the newest commit on each ref; install utilities, lock distro components for a PR",
		Long: `update resolves the newest commit on each component's ref, like lazy.nvim's :Lazy update.
A distro component's new pin is written to gap.lock in the GAP checkout with the commits it
brings in; commit and merge gap.lock, then gap sync installs it. A utility added with gap add
moves its pin in ~/.config/gap/gap.lock and installs right away. A component with pin = true
is skipped unless it is named.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) == 1 {
				id = args[0]
			}
			return finish(updateUtilities(id))
		},
	}
}

func removeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Run a utility's uninstall hook, move its files to ~/.gap/backups, and drop it",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return finish(removeUtility(args[0])) },
	}
}

func uninstallCmd() *cobra.Command {
	var opts uninstallOptions
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove everything gap sync and gap add put on this machine, from GAP's records",
		Long: `uninstall removes every utility the way gap remove does, then the MCP servers and
commands recorded for distro components, the GAP block from each policy file (a file
with no markers is deleted only when it matches ~/.gap/published.json), the GAP Codex
hooks, published global commands that match the ledger, and the skills GAP recorded
installing. Anything GAP did not write is left and listed. It previews first and asks
on a terminal. ~/.gap/store, state.json, and published.json are removed only after
every step succeeds; a failure exits 1 and keeps them so the command can run again.
~/.gap/backups, ~/.gap/recovery, your ~/.config/gap settings, and the gap binary stay.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return finish(uninstallAll(opts)) },
	}
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "remove without asking")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "print what would be removed and change nothing")
	return cmd
}

func searchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Search the utility catalog",
		Args:  cobra.MaximumNArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { exitCode = runSearch(args); return nil },
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List every component and utility with its source, pin, and hook trust",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return finish(listUtilities()) },
	}
}

func initCommand() *cobra.Command {
	var adopt bool
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Write .gap/gap.toml and a policy pointer",
		Long: `init creates <dir>/.gap/gap.toml, the project layer read after ~/.config/gap/gap.toml.
It can set a component's enabled and ref, and a distro skill's load and when.files.
A skill with load = "project" installs into <repo>/.agents/skills and
<repo>/.claude/skills when gap sync runs inside a repo that matches its when.files.
gap add writes this file too when it adds a utility to this repo.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			root, err := filepath.Abs(dir)
			if err != nil {
				return err
			}
			return initProject(root, "project", adopt)
		},
	}
	cmd.Flags().BoolVar(&adopt, "adopt", false, "preserve existing project files while adding missing GAP configuration")
	return cmd
}
