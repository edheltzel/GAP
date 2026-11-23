package main

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

type discoveredSkill struct {
	Name, Description, Dir, Plugin string
}

var discoverSkipDirs = []string{"node_modules", ".git", "dist", "build", "__pycache__"}

func agentProjectSkillDirs() []string {
	set := map[string]bool{}
	for _, a := range agentTable {
		set[a.Dir] = true
	}
	for _, dir := range []string{".cline/skills", ".codex/skills", ".factory/skills", ".github/skills", ".kilo/skills", ".kilocode/skills", ".opencode/skills", ".pi/skills"} {
		set[dir] = true
	}
	delete(set, "skills")
	return slices.Sorted(maps.Keys(set))
}

func unsafeSubpath(sub string) error {
	for _, segment := range strings.Split(strings.ReplaceAll(sub, "\\", "/"), "/") {
		if segment == ".." {
			return fmt.Errorf("Unsafe subpath: \"%s\" contains path traversal segments. Subpaths must not contain \"..\" components.", sub)
		}
	}
	return nil
}

func hasSkillMD(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && info.Mode().IsRegular()
}

func sourceLockedSkillNames(base string) map[string]bool {
	b, err := readUtilityFile(base, "skills-lock.json")
	if err != nil {
		return nil
	}
	var lock struct {
		Version any                        `json:"version"`
		Skills  map[string]json.RawMessage `json:"skills"`
	}
	if json.Unmarshal(b, &lock) != nil || lock.Skills == nil || lock.Version != float64(1) && lock.Version != float64(2) {
		return nil
	}
	names := map[string]bool{}
	for name := range lock.Skills {
		names[normalizeSkillName(name)] = true
	}
	return names
}

func relativeInside(base, target string) (string, bool) {
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func readInCheckout(checkout, file string) ([]byte, error) {
	rel, ok := relativeInside(checkout, file)
	if !ok {
		return nil, fmt.Errorf("%s is outside %s", file, checkout)
	}
	return readUtilityFile(checkout, rel)
}

func pluginManifest(checkout, base string) (pluginRoot string, plugins []map[string]any, ok bool) {
	b, err := readInCheckout(checkout, filepath.Join(base, ".claude-plugin", "marketplace.json"))
	if err != nil {
		return "", nil, false
	}
	var manifest struct {
		Metadata struct {
			PluginRoot *string `json:"pluginRoot"`
		} `json:"metadata"`
		Plugins []map[string]any `json:"plugins"`
	}
	if json.Unmarshal(b, &manifest) != nil {
		return "", nil, false
	}
	if manifest.Metadata.PluginRoot != nil {
		if !strings.HasPrefix(*manifest.Metadata.PluginRoot, "./") {
			return "", nil, false
		}
		pluginRoot = *manifest.Metadata.PluginRoot
	}
	return pluginRoot, manifest.Plugins, true
}

func pluginBase(base, pluginRoot string, plugin map[string]any) (string, bool) {
	source, present := plugin["source"]
	rel, isString := source.(string)
	if present && (!isString || !strings.HasPrefix(rel, "./")) {
		return "", false
	}
	return filepath.Join(base, pluginRoot, rel), true
}

func pluginSkills(plugin map[string]any) []string {
	list, _ := plugin["skills"].([]any)
	var out []string
	for _, item := range list {
		if s, ok := item.(string); ok && strings.HasPrefix(s, "./") {
			out = append(out, s)
		}
	}
	return out
}

func pluginSkillPaths(checkout, base string) []string {
	var dirs []string
	add := func(root string, skills []string) {
		if _, inside := relativeInside(base, root); !inside {
			return
		}
		for _, skill := range skills {
			dir := filepath.Dir(filepath.Join(root, skill))
			if _, inside := relativeInside(base, dir); inside {
				dirs = append(dirs, dir)
			}
		}
		dirs = append(dirs, filepath.Join(root, "skills"))
	}
	if root, plugins, ok := pluginManifest(checkout, base); ok {
		for _, plugin := range plugins {
			if dir, ok := pluginBase(base, root, plugin); ok {
				add(dir, pluginSkills(plugin))
			}
		}
	}
	if b, err := readInCheckout(checkout, filepath.Join(base, ".claude-plugin", "plugin.json")); err == nil {
		var plugin map[string]any
		if json.Unmarshal(b, &plugin) == nil {
			add(base, pluginSkills(plugin))
		}
	}
	return dirs
}

func pluginGroupings(checkout, base string) map[string]string {
	groups := map[string]string{}
	if root, plugins, ok := pluginManifest(checkout, base); ok {
		for _, plugin := range plugins {
			name, _ := plugin["name"].(string)
			dir, ok := pluginBase(base, root, plugin)
			if name == "" || !ok {
				continue
			}
			if _, inside := relativeInside(base, dir); !inside {
				continue
			}
			for _, skill := range pluginSkills(plugin) {
				skillDir := filepath.Join(dir, skill)
				if _, inside := relativeInside(base, skillDir); inside {
					groups[skillDir] = name
				}
			}
		}
	}
	if b, err := readInCheckout(checkout, filepath.Join(base, ".claude-plugin", "plugin.json")); err == nil {
		var plugin map[string]any
		if json.Unmarshal(b, &plugin) == nil {
			if name, _ := plugin["name"].(string); name != "" {
				for _, skill := range pluginSkills(plugin) {
					skillDir := filepath.Join(base, skill)
					if _, inside := relativeInside(base, skillDir); inside {
						groups[skillDir] = name
					}
				}
			}
		}
	}
	return groups
}

func skillsShSearchDirs(checkout, base string) []string {
	b, err := readInCheckout(checkout, filepath.Join(base, "skills.sh.json"))
	if err != nil {
		return nil
	}
	var config map[string]any
	if json.Unmarshal(b, &config) != nil {
		return nil
	}
	list, _ := config["paths"].([]any)
	var dirs, seen []string
	for _, item := range list {
		entry, ok := item.(string)
		if !ok || !strings.HasPrefix(entry, "./") || strings.Contains(entry, "\\") {
			continue
		}
		clean := strings.TrimRight(path.Clean(entry), "/")
		if clean == "." || clean == "" || clean == ".." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) || slices.Contains(seen, clean) {
			continue
		}
		seen = append(seen, clean)
		dirs = append(dirs, filepath.Join(base, filepath.FromSlash(clean)))
	}
	return dirs
}

func findSkillDirs(dir string, depth int) []string {
	if depth > 5 {
		return nil
	}
	var out []string
	if hasSkillMD(dir) {
		out = append(out, dir)
	}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if entry.IsDir() && !slices.Contains(discoverSkipDirs, entry.Name()) {
			out = append(out, findSkillDirs(filepath.Join(dir, entry.Name()), depth+1)...)
		}
	}
	return out
}

func discoverSkills(base, subpath string, fullDepth, includeInternal bool, warn func(string)) ([]discoveredSkill, error) {
	if err := unsafeSubpath(subpath); err != nil {
		return nil, err
	}
	base, err := filepath.Abs(base)
	if err != nil {
		return nil, err
	}
	search := base
	if subpath != "" {
		search = filepath.Join(base, filepath.FromSlash(subpath))
	}
	escapes := func(path string) bool {
		if _, err := os.Stat(path); err != nil || resolvesInside(base, path) {
			return false
		}
		if warn != nil {
			warn(fmt.Sprintf("Skipping %s: it resolves outside the repository", path))
		}
		return true
	}
	if escapes(search) {
		return nil, nil
	}
	locked := sourceLockedSkillNames(base)
	groups := pluginGroupings(base, search)
	projectDirs := agentProjectSkillDirs()
	var skills []discoveredSkill
	seen := map[string]bool{}
	parsed := map[string]bool{}
	installedProjectSkill := func(m skillMeta) bool {
		if len(locked) == 0 {
			return false
		}
		rel, _ := filepath.Rel(base, m.Dir)
		rel = filepath.ToSlash(rel)
		if !slices.ContainsFunc(projectDirs, func(dir string) bool { return rel == dir || strings.HasPrefix(rel, dir+"/") }) {
			return false
		}
		return locked[normalizeSkillName(m.Name)] || locked[normalizeSkillName(filepath.Base(m.Dir))]
	}
	parseAt := func(dir string) (skillMeta, bool) {
		file := filepath.Join(dir, "SKILL.md")
		if parsed[file] {
			return skillMeta{}, false
		}
		parsed[file] = true
		if escapes(dir) || escapes(file) {
			return skillMeta{}, false
		}
		return parseSkillMD(file, includeInternal, warn)
	}
	add := func(m skillMeta) {
		skills = append(skills, discoveredSkill{Name: m.Name, Description: m.Description, Dir: m.Dir, Plugin: groups[m.Dir]})
		seen[m.Name] = true
	}
	if hasSkillMD(search) {
		if m, ok := parseAt(search); ok && !installedProjectSkill(m) {
			add(m)
			if !fullDepth {
				return skills, nil
			}
		}
	}
	priority := []string{search}
	for _, rel := range []string{"skills", "skills/.curated", "skills/.experimental", "skills/.system"} {
		priority = append(priority, filepath.Join(search, rel))
	}
	for _, rel := range projectDirs {
		priority = append(priority, filepath.Join(search, filepath.FromSlash(rel)))
	}
	priority = append(priority, skillsShSearchDirs(base, search)...)
	deep := map[string]bool{}
	for _, dir := range priority[1:] {
		deep[dir] = true
	}
	priority = append(priority, pluginSkillPaths(base, search)...)
	tryAdd := func(dir string) bool {
		if !hasSkillMD(dir) {
			return false
		}
		if m, ok := parseAt(dir); ok && !seen[m.Name] && !installedProjectSkill(m) {
			add(m)
		}
		return true
	}
	var walk func(dir string, maxDepth, depth int)
	walk = func(dir string, maxDepth, depth int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			child := filepath.Join(dir, entry.Name())
			if tryAdd(child) || depth >= maxDepth || slices.Contains(discoverSkipDirs, entry.Name()) {
				continue
			}
			walk(child, maxDepth, depth+1)
		}
	}
	for _, dir := range priority {
		if escapes(dir) {
			continue
		}
		maxDepth := 1
		if deep[dir] {
			maxDepth = 3
		}
		walk(dir, maxDepth, 1)
	}
	if len(skills) == 0 || fullDepth {
		for _, dir := range findSkillDirs(search, 0) {
			if m, ok := parseAt(dir); ok && !seen[m.Name] && !installedProjectSkill(m) {
				add(m)
			}
		}
	}
	return skills, nil
}

var discoveredBySource = map[string][]discoveredSkill{}

func insideCheckout(root, dir string) bool {
	return resolvesInside(root, dir) && resolvesInside(root, filepath.Join(dir, "SKILL.md"))
}

func resolvesInside(root, path string) bool {
	resolve := func(path string) (string, error) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		return filepath.EvalSymlinks(abs)
	}
	realRoot, err := resolve(root)
	if err != nil {
		return false
	}
	real, err := resolve(path)
	return err == nil && withinRoot(realRoot, real)
}

func discoverByName(base, subpath, want string) string {
	fallback := ""
	for _, fullDepth := range []bool{false, true} {
		key := fmt.Sprintf("%s\x00%s\x00%t", base, subpath, fullDepth)
		skills, cached := discoveredBySource[key]
		if !cached {
			skills, _ = discoverSkills(base, subpath, fullDepth, true, nil)
			discoveredBySource[key] = skills
		}
		for _, s := range skills {
			if s.Name == want {
				return s.Dir
			}
			if fallback == "" && sanitizeName(s.Name) == sanitizeName(want) {
				fallback = s.Dir
			}
		}
	}
	return fallback
}

func filterSkills(all []discoveredSkill, requested []string) []discoveredSkill {
	selected := map[int]bool{}
	for _, input := range requested {
		var matches []int
		for i, s := range all {
			if strings.ToLower(s.Name) == strings.ToLower(input) {
				matches = append(matches, i)
			}
		}
		if len(matches) == 0 {
			for i, s := range all {
				if normalizeSkillName(s.Name) == normalizeSkillName(input) {
					matches = append(matches, i)
				}
			}
		}
		for _, i := range matches {
			selected[i] = true
		}
	}
	var out []discoveredSkill
	for i, s := range all {
		if selected[i] {
			out = append(out, s)
		}
	}
	return out
}
