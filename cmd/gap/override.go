package main

import (
	"os"
	"path/filepath"
)

// applyOverrides layers the user's and the repo's overrides onto the distro.
// Later wins: ~/.config/gap/gap.toml, then <repo>/.gap/gap.toml. A missing file changes nothing.
// Scalars replace. A skill's load replaces. enabled = false drops the component.
func applyOverrides(d distro, home, repoRoot, profile string) (distro, error) {
	paths := []string{
		filepath.Join(home, ".config", "gap", "gap.toml"),
		filepath.Join(repoRoot, ".gap", "gap.toml"),
	}
	if profile != "" {
		// The profile is the run's choice, so it wins over the standing overrides.
		paths = append(paths, filepath.Join(repoRoot, "profiles", profile+".toml"))
	}
	for _, path := range paths {
		next, err := mergeOverride(d, path)
		if err != nil {
			return distro{}, err
		}
		d = next
	}
	return d, nil
}

// mergeOverride reads one override file. A missing file returns the distro unchanged.
func mergeOverride(d distro, path string) (distro, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return d, nil
	}
	if err != nil {
		return distro{}, err
	}
	doc, err := parseTOML(string(b))
	if err != nil {
		return distro{}, err
	}
	tables, ok := doc["components"].(map[string]any)
	if !ok {
		return d, nil
	}
	for id, raw := range tables {
		if len(id) > 0 && id[0] == 0 {
			continue
		}
		table, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		c, found := d.Components[id]
		if !found {
			continue
		}
		if table["enabled"] == "false" {
			delete(d.Components, id)
			continue
		}
		if ref, ok := table["ref"].(string); ok && ref != "" {
			c.Ref = ref
		}
		if skills, ok := table["skills"].(map[string]any); ok {
			for name, value := range skills {
				spec, ok := value.(map[string]any)
				if !ok {
					continue
				}
				s := c.SkillSet[name]
				if load, ok := spec["load"].(string); ok {
					s.Load = load
				}
				if c.SkillSet == nil {
					c.SkillSet = map[string]skillSpec{}
				}
				c.SkillSet[name] = s
			}
		}
		d.Components[id] = c
	}
	return d, nil
}
