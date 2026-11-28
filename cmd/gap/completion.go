package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var completionShells = []string{"bash", "zsh", "fish"}

func completionCmd() *cobra.Command {
	var install bool
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Print or install shell completions for bash, zsh, or fish",
		Long: `completion prints the completion script for a shell to stdout. With --install it
writes the script where that shell loads completions for your user, and the shell
defaults to the one in $SHELL:

  bash  $XDG_DATA_HOME/bash-completion/completions/gap (loaded by bash-completion 2)
  zsh   $XDG_DATA_HOME/zsh/site-functions/_gap (add that folder to fpath)
  fish  $XDG_CONFIG_HOME/fish/completions/gap.fish

$XDG_DATA_HOME defaults to ~/.local/share and $XDG_CONFIG_HOME to ~/.config. The
install script runs gap completion --install for you.`,
		Example:   "  gap completion zsh > _gap\n  gap completion --install\n  gap completion --install fish",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: completionShells,
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := ""
			if len(args) == 1 {
				shell = args[0]
			}
			if shell == "" && install {
				shell = filepath.Base(os.Getenv("SHELL"))
			}
			if !slices.Contains(completionShells, shell) {
				if shell == "" {
					return fmt.Errorf("name a shell: gap completion bash, zsh, or fish")
				}
				return fmt.Errorf("no completions for %q; gap supports bash, zsh, and fish", shell)
			}
			if !install {
				return writeCompletion(cmd.Root(), shell, os.Stdout)
			}
			return installCompletion(cmd.Root(), shell, newReport(os.Stdout))
		},
	}
	cmd.Flags().BoolVar(&install, "install", false, "write the script where the shell loads it (shell defaults to $SHELL)")
	return cmd
}

func writeCompletion(root *cobra.Command, shell string, w io.Writer) error {
	switch shell {
	case "bash":
		return root.GenBashCompletionV2(w, true)
	case "zsh":
		return root.GenZshCompletion(w)
	}
	return root.GenFishCompletion(w, true)
}

func completionPath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	xdg := func(env, fallback string) string {
		if v := os.Getenv(env); filepath.IsAbs(v) {
			return v
		}
		return filepath.Join(home, fallback)
	}
	switch shell {
	case "bash":
		return filepath.Join(xdg("XDG_DATA_HOME", ".local/share"), "bash-completion", "completions", "gap"), nil
	case "zsh":
		return filepath.Join(xdg("XDG_DATA_HOME", ".local/share"), "zsh", "site-functions", "_gap"), nil
	}
	return filepath.Join(xdg("XDG_CONFIG_HOME", ".config"), "fish", "completions", "gap.fish"), nil
}

func installCompletion(root *cobra.Command, shell string, out *report) error {
	path, err := completionPath(shell)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := writeCompletion(root, shell, &buf); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, buf.Bytes()) {
		out.done(true, fmt.Sprintf("%s completions already current at %s", shell, tildePath(path)))
		return nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	out.done(true, fmt.Sprintf("Installed %s completions at %s", shell, tildePath(path)))
	switch shell {
	case "bash":
		out.println("  Bash loads them through bash-completion 2, which most Linux distros ship.")
		out.println("  On macOS: brew install bash-completion@2, then follow its caveats.")
	case "zsh":
		out.println("  Unless that folder is already in fpath, add this to ~/.zshrc before compinit:")
		out.println("    fpath=(" + tildePath(filepath.Dir(path)) + " $fpath)")
	}
	out.println("  Open a new terminal to use them.")
	return nil
}

func completeSyncGroups(cmd *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	prefix := ""
	if i := strings.LastIndex(toComplete, ","); i >= 0 {
		prefix = toComplete[:i+1]
	}
	chosen := strings.Split(prefix, ",")
	groups := []cobra.Completion{cobra.CompletionWithDesc(prefix+"policy", "global/AGENTS.md and global/commands")}
	if d, err := completionDistro(cmd); err == nil {
		for _, id := range sortedIDs(d.Components) {
			if !slices.Contains(chosen, id) {
				groups = append(groups, cobra.CompletionWithDesc(prefix+id, d.Components[id].Repo))
			}
		}
	}
	if slices.Contains(chosen, "policy") {
		groups = groups[1:]
	}
	return groups, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

func completeProfiles(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	_, repoRoot, err := gapHomes()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	entries, _ := os.ReadDir(filepath.Join(repoRoot, "profiles"))
	var names []cobra.Completion
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".toml"); ok && !e.IsDir() {
			names = append(names, name)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func completionDistro(cmd *cobra.Command) (distro, error) {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return distro{}, err
	}
	projectRoot, err := invokingProjectRoot()
	if err != nil {
		return distro{}, err
	}
	profile, _ := cmd.Flags().GetString("profile")
	d, _, err := readPins(repoRoot)
	if err != nil {
		return distro{}, err
	}
	return applyOverrides(d, home, repoRoot, projectRoot, profile)
}

func completeIDs(userOnly bool, suffix string) cobra.CompletionFunc {
	return func(_ *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		home, repoRoot, err := gapHomes()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		items, err := registeredRows(home, repoRoot)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var ids []cobra.Completion
		for _, r := range items {
			if userOnly && !r.user() {
				continue
			}
			ids = append(ids, cobra.CompletionWithDesc(r.ID+suffix, r.Source+" "+r.Repo))
		}
		directive := cobra.ShellCompDirectiveNoFileComp
		if suffix != "" {
			directive |= cobra.ShellCompDirectiveNoSpace
		}
		return ids, directive
	}
}
