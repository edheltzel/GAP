package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// applyOverrides layers the user's and the repo's overrides onto the distro.
// Later wins: ~/.config/gap/gap.toml, then <repo>/.gap/gap.toml. A missing file changes nothing.
// Scalars replace. A skill's load replaces. enabled = false drops the component.
func applyOverrides(d distro, home, distroRoot, projectRoot, profile string) (distro, error) {
	paths := []string{
		filepath.Join(home, ".config", "gap", "gap.toml"),
		filepath.Join(projectRoot, ".gap", "gap.toml"),
	}
	if profile != "" {
		if filepath.Base(profile) != profile || profile == "." || profile == ".." {
			return distro{}, fmt.Errorf("invalid profile %q", profile)
		}
		path := filepath.Join(distroRoot, "profiles", profile+".toml")
		if _, err := os.Stat(path); err != nil {
			return distro{}, err
		}
		paths = append(paths, path)
	}
	for _, path := range paths {
		next, err := mergeOverride(d, path)
		if err != nil {
			return distro{}, err
		}
		d = next
	}
	if profile == "" && d.Profile == "crew" {
		var err error
		d, err = mergeOverride(d, filepath.Join(distroRoot, "profiles", "crew.toml"))
		if err != nil {
			return distro{}, err
		}
	}
	for id, c := range d.Components {
		if c.Disabled {
			delete(d.Components, id)
		}
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
	if profile, ok := doc["profile"].(string); ok {
		d.Profile = profile
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
		if enabled, ok := table["enabled"].(string); ok {
			c.Disabled = enabled == "false"
		}
		if ref, ok := table["ref"].(string); ok && ref != "" {
			c.Ref = ref
			c.RefOverride = true
		}
		if skills, ok := table["skills"].(map[string]any); ok {
			for name, value := range skills {
				if len(name) > 0 && name[0] == 0 {
					continue
				}
				if !slices.Contains(c.Skills, name) {
					return distro{}, fmt.Errorf("%s: skill %s is not in component %s", path, name, id)
				}
				spec, ok := value.(map[string]any)
				if !ok {
					continue
				}
				s := c.SkillSet[name]
				if load, ok := spec["load"].(string); ok {
					switch load {
					case "always", "listed", "project", "command", "off":
					default:
						return distro{}, fmt.Errorf("%s: invalid load %q for %s", path, load, name)
					}
					s.Load = load
				}
				if when, ok := spec["when"].(map[string]any); ok {
					if files, exists := when["files"]; exists {
						var ok bool
						s.WhenFiles, ok = files.([]string)
						if !ok {
							return distro{}, fmt.Errorf("%s: %s when.files must be a string array", path, name)
						}
					}
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
