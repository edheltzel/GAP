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

func main() {
	if err := fang.Execute(context.Background(), rootCmd(),
		fang.WithoutManpage(),
		fang.WithoutCompletions(),
		fang.WithErrorHandler(func(w io.Writer, _ fang.Styles, err error) {
			fmt.Fprintln(w, err)
		}),
	); err != nil {
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
those harnesses already read, installs the skills listed in global/skills.json,
and installs each component pinned in gap.toml from gap.lock.

Installation, source checkout requirements, and updates:
https://gap.edheltzel.codes/docs/#install-gap
Component skills install from the pinned store with no Node. Skills in
global/skills.json that are not pinned still install through skills.sh, which needs Node (npx).
sync never moves pins. pin is the only command that writes the distro's gap.lock.
add, update, and remove manage utilities you add yourself: a GitHub repo with a
gap-utility.toml, recorded in ~/.config/gap/gap.toml and pinned in ~/.config/gap/gap.lock.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			exitCode = 2
			return cmd.Help()
		},
	}
	root.AddCommand(syncCmd(), statusCmd(), restoreCmd(), pinCmd(), doctorCmd(), initCommand(), addCmd(), updateCmd(), removeCmd(), listCmd(), uninstallCmd(), searchCmd())
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
		Short: "Preview, then publish the policy and commands, install component and manifest skills, check skills for drift",
		Long: `sync prints what it will touch, grouped as policy, each component in gap.toml,
and manifest (global/skills.json). On a terminal it opens a picker with every group
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
	cmd.Flags().StringSliceVar(&opts.Only, "only", nil, "sync only these groups (policy, manifest, or a component id)")
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
	cmd := &cobra.Command{
		Use:   "add <owner/repo[@ref]>",
		Short: "Add a utility: pin a repo with a gap-utility.toml and install it",
		Long: `add reads the repo's gap-utility.toml, records the repo in ~/.config/gap/gap.toml,
pins it in ~/.config/gap/gap.lock, and installs its skills, commands, and install hook.
The ref defaults to the repo's default branch. A hook asks before it runs unless the
repo is listed under [trust] repos in ~/.config/gap/gap.toml.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { return finish(addUtility(args[0], id)) },
	}
	cmd.Flags().StringVar(&id, "id", "", "id to add the utility under (default: the repo name, lowercased)")
	return cmd
}

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update [id]",
		Short: "Move a utility's pin to the newest commit on its ref, then install it",
		Args:  cobra.MaximumNArgs(1),
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
		Args:  cobra.MaximumNArgs(1),
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
