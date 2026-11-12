package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const utilityManifestName = "gap-utility.toml"

type utilityHooks struct {
	Install, Update, Uninstall, Doctor string
}

type utilityManifest struct {
	Name        string
	Description string
	Skills      []string
	Commands    []string
	Hooks       utilityHooks
	MCP         []mcpServer
}

var errNotUtility = errors.New("not a GAP utility")

func hasUtilityManifest(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, utilityManifestName))
	return err == nil
}

func readUtilityManifest(dir string) (utilityManifest, error) {
	b, err := readUtilityFile(dir, utilityManifestName)
	if errors.Is(err, os.ErrNotExist) {
		return utilityManifest{}, fmt.Errorf("%w: no %s at the repo root", errNotUtility, utilityManifestName)
	}
	if err != nil {
		return utilityManifest{}, err
	}
	doc, err := parseTOML(string(b))
	if err != nil {
		return utilityManifest{}, fmt.Errorf("%s: %w", utilityManifestName, err)
	}
	fail := func(format string, args ...any) (utilityManifest, error) {
		return utilityManifest{}, fmt.Errorf(utilityManifestName+": "+format, args...)
	}
	for key := range doc {
		switch key {
		case "schema", "name", "description", "skills", "commands", "hooks", "mcp":
		default:
			if !strings.HasPrefix(key, "\x00") {
				return fail("unknown key %s", key)
			}
		}
	}
	if s, _ := doc["schema"].(string); s != "1" {
		return fail("schema must be \"1\"")
	}
	var m utilityManifest
	var ok bool
	if m.Name, ok = doc["name"].(string); !ok || m.Name == "" {
		return fail("name must be a nonempty string")
	}
	if raw, exists := doc["description"]; exists {
		if m.Description, ok = raw.(string); !ok {
			return fail("description must be a string")
		}
	}
	if m.Skills, err = manifestList(doc, "skills"); err != nil {
		return fail("%v", err)
	}
	for _, name := range m.Skills {
		if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
			return fail("skill %s is not a plain name", name)
		}
		if localSkillDir(dir, name) == "" {
			return fail("skill %s has no %s/SKILL.md in the repo", name, name)
		}
	}
	if m.Commands, err = manifestList(doc, "commands"); err != nil {
		return fail("%v", err)
	}
	for _, rel := range m.Commands {
		clean := filepath.Clean(filepath.FromSlash(rel))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fail("command %s is outside the repo", rel)
		}
		if filepath.Ext(clean) != ".md" {
			return fail("command %s must be a .md file", rel)
		}
		if _, err := readUtilityFile(dir, clean); err != nil {
			return fail("command %s is not a file in the repo: %v", rel, err)
		}
	}
	if raw, exists := doc["hooks"]; exists {
		hooks, ok := raw.(map[string]any)
		if !ok {
			return fail("hooks must be a table")
		}
		for key, value := range hooks {
			if strings.HasPrefix(key, "\x00") {
				continue
			}
			cmd, ok := value.(string)
			if !ok || strings.TrimSpace(cmd) == "" {
				return fail("hooks.%s must be a nonempty string", key)
			}
			switch key {
			case "install":
				m.Hooks.Install = cmd
			case "update":
				m.Hooks.Update = cmd
			case "uninstall":
				m.Hooks.Uninstall = cmd
			case "doctor":
				m.Hooks.Doctor = cmd
			default:
				return fail("unknown hook %s", key)
			}
		}
	}
	if m.MCP, err = manifestMCP(doc); err != nil {
		return fail("%v", err)
	}
	if len(m.Skills) == 0 && len(m.Commands) == 0 && m.Hooks == (utilityHooks{}) && len(m.MCP) == 0 {
		return fail("declares no skills, commands, hooks, or mcp servers")
	}
	return m, nil
}

func readUtilityFile(dir, rel string) ([]byte, error) {
	f, err := os.OpenInRoot(dir, filepath.FromSlash(rel))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", rel)
	}
	return io.ReadAll(f)
}

func manifestList(doc map[string]any, key string) ([]string, error) {
	raw, exists := doc[key]
	if !exists {
		return nil, nil
	}
	list, ok := raw.([]string)
	if !ok {
		return nil, fmt.Errorf("%s must be a string array", key)
	}
	seen := map[string]bool{}
	for _, v := range list {
		if v == "" || seen[v] {
			return nil, fmt.Errorf("%s has an empty or repeated entry", key)
		}
		seen[v] = true
	}
	out := append([]string(nil), list...)
	sort.Strings(out)
	return out, nil
}
