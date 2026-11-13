package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const installHint = "curl -fsSL https://raw.githubusercontent.com/edheltzel/GAP/master/scripts/install.sh | sh"

// checkoutRoot is the GAP checkout a release install keeps at $GAP_HOME/checkout, or ~/.gap/checkout when GAP_HOME is unset.
// A prebuilt binary has no source checkout of its own, so sourceRoot falls back to this one.
func checkoutRoot() (string, error) {
	base := os.Getenv("GAP_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".gap")
	} else if !filepath.IsAbs(base) {
		return "", fmt.Errorf("GAP_HOME must be an absolute path, got %q", base)
	}
	root := filepath.Join(base, "checkout")
	for _, rel := range []string{".git", filepath.Join("global", "AGENTS.md"), filepath.Join("cmd", "gap", "main.go")} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			return "", fmt.Errorf("no GAP checkout at %s (missing %s); run the installer: %s", root, rel, installHint)
		}
	}
	return root, nil
}
