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
	query := `"` + strings.ReplaceAll(note, `"`, `""`) + `"`
	search := exec.Command(recall, "search", query, "-t", "breadcrumbs", "-p", slug, "--include-duplicates", "-l", "1")
	out, err := search.Output()
	if err != nil {
		return fmt.Errorf("recall search breadcrumbs: %w", err)
	}
	if strings.Contains(string(out), "[breadcrumbs#") {
		return nil
	}
	add := exec.Command(recall, "add", "breadcrumb", note, "-p", slug, "-c", "setup")
	add.Stdout, add.Stderr = os.Stdout, os.Stderr
	if err := add.Run(); err != nil {
		return fmt.Errorf("recall add breadcrumb: %w", err)
	}
	return nil
}
