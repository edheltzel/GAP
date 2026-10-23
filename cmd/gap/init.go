package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func provisionBreadcrumb(slug string) error {
	recall, err := exec.LookPath("recall")
	if err != nil {
		fmt.Println("recall is not installed; no breadcrumb written")
		return nil
	}
	note := "gap provisioned " + slug
	recent := exec.Command(recall, "recent", "breadcrumbs", "-p", slug, "-l", "1000")
	out, err := recent.Output()
	if err != nil {
		return fmt.Errorf("recall recent breadcrumbs: %w", err)
	}
	if strings.Contains(string(out), note) {
		return nil
	}
	add := exec.Command(recall, "add", "breadcrumb", note, "-p", slug, "-c", "setup")
	add.Stdout, add.Stderr = os.Stdout, os.Stderr
	if err := add.Run(); err != nil {
		return fmt.Errorf("recall add breadcrumb: %w", err)
	}
	return nil
}
