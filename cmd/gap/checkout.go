package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const installHint = "curl -fsSL https://raw.githubusercontent.com/edheltzel/GAP/master/scripts/install.sh | sh"

func checkoutRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(home, ".gap", "checkout")
	for _, rel := range []string{".git", filepath.Join("global", "AGENTS.md"), filepath.Join("cmd", "gap", "main.go")} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			return "", fmt.Errorf("no GAP checkout at %s (missing %s); run the installer: %s", root, rel, installHint)
		}
	}
	return root, nil
}
