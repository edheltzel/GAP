package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
add, update, and remove manage utilities you add yourself: a GitHub repo with a
gap-utility.toml, recorded in ~/.config/gap/gap.toml and pinned in ~/.config/gap/gap.lock.

Global vs project:
  global   ~/.config/gap/gap.toml. Skills go to ~/.agents/skills and ~/.claude/skills.
           gap add owner/repo (or --global) adds a utility here.
  project  <repo>/.gap/gap.toml. Skills go to <repo>/.agents/skills and
           <repo>/.claude/skills. gap add --project (or --local) inside the repo adds
           a utility here. A distro skill with load = "project" installs here when
           the repo matches its when.files. enabled = false leaves a component out.
Commands, hooks, and MCP servers are always machine-wide.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			exitCode = 2
			return cmd.Help()
		},
	}
	root.AddCommand(syncCmd(), statusCmd(), restoreCmd(), pinCmd(), doctorCmd(), initCommand(), addCmd(), checkCmd(), updateCmd(), removeCmd(), listCmd(), uninstallCmd(), searchCmd())
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
	return err
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
		RunE:  func(cmd *cobra.Command, _ []string) error { return finish(statusAll()) },
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
	var id string
	var global, project, local bool
	cmd := &cobra.Command{
		Use:   "add <owner/repo[@ref]>",
		Short: "Add a utility: pin a repo with a gap-utility.toml and install it",
		Long: `add reads the repo's gap-utility.toml, records the repo in ~/.config/gap/gap.toml,
pins it in ~/.config/gap/gap.lock, and installs its skills, commands, and install hook.
The ref defaults to the repo's default branch. A hook asks before it runs unless the
repo is listed under [trust] repos in ~/.config/gap/gap.toml.

Scope:
  --global (default)    skills go to ~/.agents/skills and ~/.claude/skills.
  --project, --local    skills go to <repo>/.agents/skills and <repo>/.claude/skills
                        of the repo you run it in, recorded as load = "project" in
                        ~/.config/gap/gap.toml and in that repo's .gap/gap.toml.
                        Run it again in another repo to add the same utility there;
                        gap sync inside each repo keeps its copy current.
Commands, hooks, and MCP servers are machine-wide in both scopes. gap remove <id>
removes the utility everywhere, including every project copy.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return finish(addUtility(args[0], id, project || local))
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "id to add the utility under (default: the repo name, lowercased)")
	cmd.Flags().BoolVarP(&global, "global", "g", false, "install its skills under home for every project (default)")
	cmd.Flags().BoolVarP(&project, "project", "p", false, "install its skills into the current repo only")
	cmd.Flags().BoolVarP(&local, "local", "l", false, "same as --project")
	cmd.MarkFlagsMutuallyExclusive("global", "project", "local")
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
gap add --project writes this file too.`,
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
