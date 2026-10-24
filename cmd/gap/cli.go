package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

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

gap publishes the checkout it was built from. Reinstall with
curl -fsSL https://raw.githubusercontent.com/edheltzel/GAP/master/scripts/install.sh | sh
after cmd/gap changes.
Component skills install from the pinned store with no Node. Skills in
global/skills.json that are not pinned still install through skills.sh, which needs Node (npx).
sync never moves pins. pin is the only command that writes gap.lock.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			exitCode = 2
			return cmd.Help()
		},
	}
	root.AddCommand(syncCmd(), statusCmd(), restoreCmd(), pinCmd(), doctorCmd(), initCommand())
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
	return err
}

func syncCmd() *cobra.Command {
	var removeLegacy bool
	var profile string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Publish the policy and commands, install missing skills, check skills for drift",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return finish(syncAll(removeLegacy, profile))
		},
	}
	cmd.Flags().BoolVar(&removeLegacy, "remove-codex-legacy", false, "strip the old GAP text from Codex AGENTS.md once both hooks are trusted")
	cmd.Flags().StringVar(&profile, "profile", "", "apply profiles/<name>.toml over the distro for this run")
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
			if code := initCmd(append(adoptFlag(adopt), dir)); code != 0 {
				exitCode = code
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&adopt, "adopt", false, "preserve existing project files while adding missing GAP configuration")
	return cmd
}

func adoptFlag(adopt bool) []string {
	if adopt {
		return []string{"--adopt"}
	}
	return nil
}
