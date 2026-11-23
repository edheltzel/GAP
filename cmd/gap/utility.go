package main

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
)

// defaultBranch is the branch the remote's HEAD points at.
func defaultBranch(url string) (string, error) {
	out, err := spin("Reading the default branch of "+url, func() (string, error) { return git(".", "ls-remote", "--symref", url, "HEAD") })
	if err != nil {
		return "", fmt.Errorf("reading the default branch of %s: %w", url, err)
	}
	for _, line := range strings.Split(out, "\n") {
		if rest, ok := strings.CutPrefix(line, "ref: refs/heads/"); ok {
			if branch, _, ok := strings.Cut(rest, "\t"); ok {
				return branch, nil
			}
		}
	}
	return "", fmt.Errorf("%s has no default branch; pass owner/repo#ref", url)
}

func validID(id string) bool {
	if id == "" || id == "policy" || id == "manifest" {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return id != "." && id != ".."
}

func userConfigFile(home string) string {
	return filepath.Join(userConfigDir(home), "gap.toml")
}

func projectConfigFile(root string) string {
	return filepath.Join(root, ".gap", "gap.toml")
}

type componentConfig struct {
	Repo, Ref, Path, Load    string
	Enabled                  *bool
	Pin                      bool
	Names, Agents, Subagents []string
	Copy                     bool
	From                     map[string]string
	Skills                   map[string]map[string]any
}

func readComponentConfig(text, id string) (componentConfig, bool, error) {
	doc, err := parseTOML(text)
	if err != nil {
		return componentConfig{}, false, err
	}
	components, _ := doc["components"].(map[string]any)
	table, ok := components[id].(map[string]any)
	if !ok {
		return componentConfig{}, false, nil
	}
	cc := componentConfig{From: map[string]string{}, Skills: map[string]map[string]any{}}
	cc.Repo, _ = table["repo"].(string)
	cc.Ref, _ = table["ref"].(string)
	cc.Path, _ = table["path"].(string)
	cc.Load, _ = table["load"].(string)
	if enabled, ok := table["enabled"].(bool); ok {
		cc.Enabled = &enabled
	}
	cc.Pin, _ = table["pin"].(bool)
	cc.Names, _ = table["names"].([]string)
	cc.Agents, _ = table["agents"].([]string)
	cc.Copy, _ = table["copy"].(bool)
	cc.Subagents, _ = table["subagents"].([]string)
	skills, _ := table["skills"].(map[string]any)
	for name, raw := range skills {
		spec, ok := raw.(map[string]any)
		if strings.HasPrefix(name, "\x00") || !ok {
			continue
		}
		rest := map[string]any{}
		for key, value := range spec {
			switch {
			case strings.HasPrefix(key, "\x00"):
			case key == "from":
				if from, ok := value.(string); ok {
					cc.From[name] = from
				}
			default:
				rest[key] = value
			}
		}
		cc.Skills[name] = rest
	}
	return cc, true, nil
}

func tomlString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(s) + `"`
}

func tomlStrings(list []string) string {
	quoted := make([]string, len(list))
	for i, s := range list {
		quoted[i] = tomlString(s)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func componentTOML(id string, cc componentConfig) string {
	var b strings.Builder
	key := "components." + tomlKey(id)
	b.WriteString("[" + key + "]\n")
	for _, kv := range [][2]string{{"repo", cc.Repo}, {"ref", cc.Ref}, {"path", cc.Path}, {"load", cc.Load}} {
		if kv[1] != "" {
			b.WriteString(kv[0] + " = " + tomlString(kv[1]) + "\n")
		}
	}
	if cc.Enabled != nil {
		fmt.Fprintf(&b, "enabled = %t\n", *cc.Enabled)
	}
	if cc.Pin {
		b.WriteString("pin = true\n")
	}
	for _, kv := range []struct {
		key  string
		list []string
	}{{"names", cc.Names}, {"agents", cc.Agents}} {
		if kv.list != nil {
			b.WriteString(kv.key + " = " + tomlStrings(kv.list) + "\n")
		}
	}
	if cc.Copy {
		b.WriteString("copy = true\n")
	}
	if cc.Subagents != nil {
		b.WriteString("subagents = " + tomlStrings(cc.Subagents) + "\n")
	}
	names := appendMissing(slices.Collect(maps.Keys(cc.Skills)), slices.Collect(maps.Keys(cc.From))...)
	slices.Sort(names)
	for _, name := range names {
		b.WriteString("\n[" + key + ".skills." + tomlKey(name) + "]\n")
		if from := cc.From[name]; from != "" {
			b.WriteString("from = " + tomlString(from) + "\n")
		}
		spec := cc.Skills[name]
		if load, ok := spec["load"].(string); ok {
			b.WriteString("load = " + tomlString(load) + "\n")
		}
		if when, ok := spec["when"].(map[string]any); ok {
			if files, ok := when["files"].([]string); ok {
				b.WriteString("when.files = " + tomlStrings(files) + "\n")
			}
		}
	}
	return b.String()
}

func upsertComponent(text, id string, cc componentConfig) (string, error) {
	text, _ = removeTOMLComponent(text, id)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if text != "" {
		text += "\n"
	}
	text += componentTOML(id, cc)
	if _, err := parseTOML(text); err != nil {
		return "", err
	}
	return text, nil
}

type addSelection struct {
	names, agents, subagents, manifest []string
	from                               map[string]string
	copy                               bool
}

func mergeSelection(cc componentConfig, declared bool, sel addSelection) componentConfig {
	names := slices.Clone(cc.Names)
	if declared && cc.Names == nil && sel.manifest != nil {
		names = slices.Clone(sel.manifest)
	}
	names = appendMissing(names, sel.names...)
	slices.Sort(names)
	if cc.From == nil {
		cc.From = map[string]string{}
	}
	for _, name := range sel.names {
		delete(cc.From, name)
	}
	maps.Copy(cc.From, sel.from)
	covers := sel.manifest != nil && len(cc.From) == 0 && !slices.ContainsFunc(sel.manifest, func(m string) bool { return !slices.Contains(names, m) })
	cc.Names = names
	if covers {
		cc.Names = nil
	}
	cc.Agents = appendMissing(cc.Agents, sel.agents...)
	cc.Copy = sel.copy
	cc.Subagents = appendMissing(cc.Subagents, sel.subagents...)
	if len(cc.Subagents) == 0 || len(cc.Subagents) == 1 && cc.Subagents[0] == "" {
		cc.Subagents = nil
	}
	return cc
}

func appendMissing(list []string, items ...string) []string {
	for _, item := range items {
		if !slices.Contains(list, item) {
			list = append(list, item)
		}
	}
	return list
}

func updatedConfig(path, initial, id string, update func(componentConfig, bool) componentConfig) (string, error) {
	text, err := readOptional(path)
	if err != nil {
		return "", err
	}
	if text == "" {
		text = initial
	}
	cc, declared, err := readComponentConfig(text, id)
	if err != nil {
		return "", fmt.Errorf("fix %s before gap add: %w", path, err)
	}
	next, err := upsertComponent(text, id, update(cc, declared))
	if err != nil {
		return "", fmt.Errorf("adding %s to %s: %w", id, path, err)
	}
	return next, nil
}

func writeAddConfig(home, projectRoot, id string, global bool, base componentConfig, sel addSelection) error {
	user, err := updatedConfig(userConfigFile(home), "", id, func(cc componentConfig, declared bool) componentConfig {
		cc.Repo, cc.Ref, cc.Path = base.Repo, base.Ref, base.Path
		if global {
			cc.Load = ""
			return mergeSelection(cc, declared, sel)
		}
		cc.Load = "project"
		cc.Names, cc.Agents, cc.Copy, cc.Subagents, cc.From = nil, nil, false, nil, nil
		return cc
	})
	if err != nil {
		return err
	}
	if global {
		return writeAtomic(userConfigFile(home), user)
	}
	initial := "profile = \"project\"\nslug = " + strconv.Quote(filepath.Base(projectRoot)) + "\n"
	project, err := updatedConfig(projectConfigFile(projectRoot), initial, id, func(cc componentConfig, declared bool) componentConfig {
		cc.Load = "project"
		return mergeSelection(cc, declared, sel)
	})
	if err != nil {
		return err
	}
	if err := writeAtomic(userConfigFile(home), user); err != nil {
		return err
	}
	if err := writeAtomic(projectConfigFile(projectRoot), project); err != nil {
		return err
	}
	noteProject(projectRoot)
	return nil
}

func describeUtility(m utilityManifest) string {
	var parts []string
	if len(m.Skills) > 0 {
		parts = append(parts, plural(len(m.Skills), "skill"))
	}
	if len(m.Commands) > 0 {
		parts = append(parts, plural(len(m.Commands), "command"))
	}
	var hooks []string
	for _, h := range [][2]string{{"install", m.Hooks.Install}, {"update", m.Hooks.Update}, {"uninstall", m.Hooks.Uninstall}, {"doctor", m.Hooks.Doctor}} {
		if h[1] != "" {
			hooks = append(hooks, h[0])
		}
	}
	if len(hooks) > 0 {
		parts = append(parts, "hooks: "+strings.Join(hooks, ", "))
	}
	if len(m.MCP) > 0 {
		names := make([]string, len(m.MCP))
		for i, s := range m.MCP {
			names[i] = s.Name
		}
		parts = append(parts, "mcp: "+strings.Join(names, ", "))
	}
	return strings.Join(parts, ", ")
}

func readOptional(path string) (string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return string(b), err
}

func setUserPin(home, id string, pin lockEntryM1) error {
	lk, err := readLock(userConfigDir(home))
	if err != nil {
		return err
	}
	lk.Components[id] = pin
	return writeLock(userConfigDir(home), lk)
}

// installUtility syncs one component: its hook, its skills, its commands, and the ownership record.
func installUtility(home, repoRoot, projectRoot, id, prevStore string, inv mcpInventory) error {
	d, lk, err := effectivePins(repoRoot, home, projectRoot, "")
	if err != nil {
		return err
	}
	c, ok := d.Components[id]
	if !ok {
		return nil
	}
	d.Components = map[string]component{id: c}
	pins, err := lockFor(d, lk)
	if err != nil {
		return err
	}
	if err := prepareComponentSources(home, d, pins); err != nil {
		return err
	}
	if err := checkCommandOwners(repoRoot, home, d); err != nil {
		return err
	}
	entries, hooksOK, err := componentEntries(home, projectRoot, d, pins)
	if err != nil {
		return err
	}
	ledgerPath := filepath.Join(home, ".gap", "published.json")
	ledger, err := readLedger(ledgerPath)
	if err != nil {
		return err
	}
	owned, commandsOK, err := publishComponentCommands(repoRoot, home, d, ledger, ledgerPath)
	if err != nil {
		return err
	}
	skillsOK := true
	state, err := readState(home)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		target := entryHome(home, entry)
		for _, name := range entry.Skills {
			baseline := prevStore
			if commit := state.Components[id].SkillCopies[name]; commit != "" {
				baseline = storePath(home, id, commit)
			}
			if baseline == "" || baseline == entry.Path || !slices.Contains(state.Components[id].Skills, name) {
				continue
			}
			if !installed(target, entry, name) || entryDriftCount(entry, target, name) == 0 {
				continue
			}
			previous := entry
			previous.Path = baseline
			if entryDriftCount(previous, target, name) > 0 {
				fmt.Fprintf(os.Stderr, "%s: your edited copy of %s was kept; gap restore %s takes the new version\n", id, name, id)
				skillsOK = false
				continue
			}
			if _, err := replaceSkill(target, entry, name); err != nil {
				fmt.Fprintf(os.Stderr, "Updating %s: %v\n", name, err)
				skillsOK = false
			}
		}
	}
	mcpFailures, err := publishComponentMCP(home, d, inv)
	if err != nil {
		return err
	}
	skillsOK = installSkills(home, entries) && skillsOK
	driftOK, err := checkDrift(home, entries)
	if err != nil {
		return err
	}
	if skillsOK {
		if err := recordDeclarativeInstalls(home, projectRoot, d, owned, mcpFailures); err != nil {
			return err
		}
	}
	if !hooksOK || !commandsOK || len(mcpFailures) > 0 || !skillsOK || !driftOK {
		return errSkipped
	}
	return nil
}

func updateUtilities(id string) (resultErr error) {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return err
	}
	projectRoot, err := invokingProjectRoot()
	if err != nil {
		return err
	}
	d, err := readDistro(repoRoot)
	if err != nil {
		return err
	}
	d, err = mergeOverrides(d, home, repoRoot, projectRoot, "")
	if err != nil {
		return err
	}
	var ids []string
	for _, cid := range sortedIDs(d.Components) {
		if c := d.Components[cid]; c.User && ((id == "" && !c.Pin) || cid == id) {
			ids = append(ids, cid)
		}
	}
	if id != "" && len(ids) == 0 {
		if _, distroID := d.Components[id]; distroID {
			_, err := updateDistro(id)
			return err
		}
		return fmt.Errorf("utility %s is not added; run gap add owner/repo", id)
	}
	if id == "" {
		if _, err := updateDistro(""); err != nil {
			return err
		}
	}
	out := newReport(os.Stdout)
	if len(ids) == 0 {
		out.println("No utilities added; gap add owner/repo adds one.")
		return nil
	}
	checked, updated := 0, 0
	defer func() {
		for _, cid := range ids[checked:] {
			out.done(false, cid+" was not checked; update stopped")
		}
		noun := "utilities"
		if checked == 1 {
			noun = "utility"
		}
		summary := fmt.Sprintf("%d %s checked: %d updated, %d up to date", checked, noun, updated, checked-updated)
		if checked < len(ids) {
			summary += fmt.Sprintf("; %d not checked", len(ids)-checked)
		}
		switch {
		case errors.Is(resultErr, errSkipped):
			summary += "; some install steps were left over, see above"
		case resultErr != nil:
			summary += "; update failed"
		}
		out.done(resultErr == nil, summary)
	}()
	lk, err := readLock(userConfigDir(home))
	if err != nil {
		return err
	}
	state, err := readState(home)
	if err != nil {
		return err
	}
	baselines := map[string]string{}
	for _, cid := range ids {
		c := d.Components[cid]
		url, err := cloneURL(c.Repo)
		if err != nil {
			return err
		}
		commit, err := resolveRef(url, c.Ref)
		if err != nil {
			return err
		}
		prev, pinned := lk.Components[cid]
		if installedCommit := state.Components[cid].Installed; installedCommit != "" {
			baselines[cid] = storePath(home, cid, installedCommit)
		} else if pinned {
			baselines[cid] = storePath(home, cid, prev.Commit)
		}
		if pinned && prev.Commit == commit && prev.Ref == c.Ref {
			out.done(true, cid+" is up to date"+pinLabel(prev))
		} else {
			pin, err := pinCommit(home, c, c.Ref, commit, prev)
			if err != nil {
				return err
			}
			if _, err := storeHooks(storePath(home, cid, commit)); err != nil {
				return fmt.Errorf("%s at %s: %w", cid, commit[:7], err)
			}
			lk.Components[cid] = pin
			if err := writeLock(userConfigDir(home), lk); err != nil {
				return err
			}
			updated++
			if pinned {
				out.done(true, fmt.Sprintf("%s moved from %s to %s @ %s", cid, prev.Commit[:min(7, len(prev.Commit))], commit[:7], c.Ref))
			} else {
				out.done(true, cid+" pinned"+pinLabel(pin))
			}
		}
		checked++
	}
	requested := distro{Components: map[string]component{}}
	for _, cid := range ids {
		requested.Components[cid] = d.Components[cid]
	}
	requestedPins, err := lockFor(requested, lk)
	if err != nil {
		return err
	}
	if err := prepareComponentSources(home, requested, requestedPins); err != nil {
		return err
	}
	failed := false
	inv := mcpInventory{}
	for _, cid := range ids {
		if err := installUtility(home, repoRoot, projectRoot, cid, baselines[cid], inv); err != nil {
			if !errors.Is(err, errSkipped) {
				return err
			}
			failed = true
		}
	}
	if failed {
		return errSkipped
	}
	return nil
}

func listUtilities() error {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return err
	}
	items, err := registeredRows(home, repoRoot)
	if err != nil {
		return err
	}
	var rows [][]string
	for _, r := range items {
		rows = append(rows, []string{r.ID, r.Source, r.Repo, r.Ref, r.Commit, r.Hooks})
	}
	return printList(newReport(os.Stdout), rows)
}

// printList writes gap list rows: tab-aligned columns when piped, a framed table with a total on a terminal.
func printList(out *report, rows [][]string) error {
	if out.tty {
		out.title("gap list", "components and utilities")
		out.framed([]string{"id", "source", "repo", "ref", "commit", "hooks"}, rows)
		out.footer(fmt.Sprintf("%d total", len(rows)))
		return nil
	}
	w := tabwriter.NewWriter(out.out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSOURCE\tREPO\tREF\tCOMMIT\tHOOKS")
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	return w.Flush()
}

func removeTOMLComponent(text, id string) (string, bool) {
	var kept []string
	var section []string
	found := false
	lines := strings.SplitAfter(text, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trim := strings.TrimSpace(stripComment(line))
		keys := section
		if strings.HasPrefix(trim, "[") {
			var err error
			section, err = splitKey(strings.TrimSuffix(strings.TrimPrefix(trim, "["), "]"), i+1)
			if err != nil {
				return text, false
			}
			keys = section
		} else if key, value, assignment := strings.Cut(trim, "="); assignment {
			parts, err := splitKey(strings.TrimSpace(key), i+1)
			if err != nil {
				return text, false
			}
			keys = append(slices.Clone(section), parts...)
			for arrayContinues(value) && i+1 < len(lines) {
				i++
				line += lines[i]
				value += "\n" + stripComment(lines[i])
			}
		}
		dropping := len(keys) >= 2 && keys[0] == "components" && keys[1] == id
		found = found || dropping
		if !dropping {
			kept = append(kept, line)
		}
	}
	out := strings.Join(kept, "")
	for strings.HasSuffix(out, "\n\n") {
		out = strings.TrimSuffix(out, "\n")
	}
	return out, found
}

func setComponentEnabled(home, repoRoot, id string, enabled bool) error {
	d, _, err := registeredPins(repoRoot, home)
	if err != nil {
		return err
	}
	c, ok := d.Components[id]
	switch {
	case !ok:
		return fmt.Errorf("component %s is not in gap.toml", id)
	case c.User:
		return fmt.Errorf("%s is a utility; run gap remove %s instead", id, id)
	}
	config := userConfigFile(home)
	text, err := readOptional(config)
	if err != nil {
		return err
	}
	next, err := withEnabled(text, id, enabled)
	if err != nil {
		return fmt.Errorf("%s: %w", config, err)
	}
	doc, err := parseTOML(next)
	if err != nil {
		return fmt.Errorf("setting enabled for %s in %s: %w", id, config, err)
	}
	components, _ := doc["components"].(map[string]any)
	table, _ := components[id].(map[string]any)
	if got, _ := table["enabled"].(bool); got != enabled || table == nil {
		return fmt.Errorf("%s: could not set enabled = %t for %s", config, enabled, id)
	}
	if next == text {
		return nil
	}
	return writeAtomic(config, next)
}

func withEnabled(text, id string, enabled bool) (string, error) {
	line := "enabled = " + strconv.FormatBool(enabled) + "\n"
	lines := strings.SplitAfter(text, "\n")
	header := -1
	var section []string
	for i, raw := range lines {
		trim := strings.TrimSpace(stripComment(raw))
		if strings.HasPrefix(trim, "[") {
			var err error
			if section, err = splitKey(strings.TrimSuffix(strings.TrimPrefix(trim, "["), "]"), i+1); err != nil {
				return "", err
			}
			if len(section) == 2 && section[0] == "components" && section[1] == id {
				header = i
			}
			continue
		}
		if header < 0 || len(section) != 2 || section[0] != "components" || section[1] != id {
			continue
		}
		if key, _, ok := strings.Cut(trim, "="); ok && strings.TrimSpace(key) == "enabled" {
			if !strings.HasSuffix(raw, "\n") {
				line = strings.TrimSuffix(line, "\n")
			}
			lines[i] = line
			return strings.Join(lines, ""), nil
		}
	}
	if header >= 0 {
		if !strings.HasSuffix(lines[header], "\n") {
			lines[header] += "\n"
		}
		lines = slices.Insert(lines, header+1, line)
		return strings.Join(lines, ""), nil
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if text != "" {
		text += "\n"
	}
	return text + "[components." + tomlKey(id) + "]\n" + line, nil
}

func removeUtility(id string) error {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return err
	}
	d, lk, err := registeredPins(repoRoot, home)
	if err != nil {
		return err
	}
	c, ok := d.Components[id]
	if !ok || !c.User {
		if ok {
			return fmt.Errorf("%s is a distro component; set enabled = false under [components.%s] in %s to leave it out", id, id, userConfigFile(home))
		}
		return fmt.Errorf("utility %s is not added", id)
	}
	config := userConfigFile(home)
	text, err := readOptional(config)
	if err != nil {
		return err
	}
	next, found := removeTOMLComponent(text, id)
	if !found {
		return fmt.Errorf("utility %s: cannot remove its configuration; nothing removed", id)
	}
	doc, err := parseTOML(next)
	if err != nil {
		return err
	}
	if components, ok := doc["components"].(map[string]any); ok {
		if _, remains := components[id]; remains {
			return fmt.Errorf("utility %s: configuration still declares it; nothing removed", id)
		}
	}
	if err := loadLocalManifests(home, d, lk); err != nil {
		return err
	}
	for _, oid := range sortedIDs(d.Components) {
		other := d.Components[oid]
		if oid != id && other.Store == "" && (other.User || !other.Named) {
			return fmt.Errorf("cannot determine %s selections; run gap sync before removing %s; nothing removed", oid, id)
		}
	}
	gate, err := newHookGate(home)
	if err != nil {
		return err
	}
	out := newReport(os.Stdout)
	if pin, pinned := lk.Components[id]; pinned {
		dir, err := ensureStore(home, c, pin)
		if err != nil {
			return err
		}
		if c, err = applyManifest(c, dir); err != nil {
			return err
		}
		c.Store = dir
	}
	entry := gate.state.Components[id]
	if entry.Installed != "" {
		pin, err := pinCommit(home, c, entry.Installed, entry.Installed, lk.Components[id])
		if err != nil {
			return err
		}
		dir, err := ensureStore(home, c, pin)
		if err != nil {
			return err
		}
		hooks, err := storeHooks(dir)
		if err != nil {
			return err
		}
		if hooks.Uninstall != "" {
			if !gate.approve(c, "uninstall", hooks.Uninstall) {
				return fmt.Errorf("%s: uninstall hook not approved; nothing removed", id)
			}
			if err := runComponentCommand(dir, hooks.Uninstall); err != nil {
				return fmt.Errorf("%s uninstall hook failed; nothing removed: %w", id, err)
			}
			out.result(true, id+" uninstall hook finished")
			if err := markUninstalled(home, id); err != nil {
				return err
			}
		} else if hooks.Install != "" && !gate.trusted(c) {
			fmt.Fprintf(os.Stderr, "%s has an install hook but no uninstall hook; gap cannot undo what its installer did.\n", id)
		}
	}
	if err := removeComponentMCP(home, id); err != nil {
		return err
	}
	shared := map[string]bool{}
	sharedSkillOwners := map[string][]string{}
	sharedCommands := map[string]bool{}
	sharedCommandOwners := map[string][]string{}
	for oid, other := range gate.state.Components {
		if oid != id {
			for _, name := range other.Skills {
				shared[name] = true
				sharedSkillOwners[name] = append(sharedSkillOwners[name], oid)
			}
			for _, path := range other.Commands {
				sharedCommands[path] = true
				sharedCommandOwners[path] = append(sharedCommandOwners[path], oid)
			}
		}
	}
	globals, err := globalCommandNames(repoRoot)
	if err != nil {
		return err
	}
	for _, name := range globals {
		for _, dest := range commandDests(home, name) {
			sharedCommands[dest] = true
		}
	}
	for oid, other := range d.Components {
		if oid != id {
			for _, name := range other.Skills {
				shared[name] = true
				sharedSkillOwners[name] = append(sharedSkillOwners[name], oid)
			}
			for _, rel := range other.Commands {
				for _, path := range commandDests(home, filepath.Base(rel)) {
					sharedCommands[path] = true
					sharedCommandOwners[path] = append(sharedCommandOwners[path], oid)
				}
			}
		}
	}
	backup := backupPath(home, home)
	var edited []string
	moved := 0
	move := func(path string, changed bool, project bool) error {
		if _, err := os.Lstat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		dest := backupTarget(home, backup, path)
		if project {
			dest = filepath.Join(backup, "projects", strings.TrimPrefix(path, string(filepath.Separator)))
		}
		dest = unusedPath(dest)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.Rename(path, dest); err != nil {
			return err
		}
		moved++
		if changed {
			edited = append(edited, dest)
		}
		return nil
	}
	moveOut := func(path string, changed bool) error { return move(path, changed, false) }
	for _, name := range entry.Skills {
		if shared[name] {
			for _, owner := range sharedSkillOwners[name] {
				if err := shareDests(home, id, owner, home, name, false); err != nil {
					return err
				}
				if err := recordOwnership(home, owner, []string{name}, nil, ""); err != nil {
					return err
				}
			}
			fmt.Printf("Kept %s: another component also selects it\n", name)
			continue
		}
		dirs := ownedDests(gate.state, id, home, name)
		changed := c.Store == "" || ownedDrift(componentEntry(c, c.Store), home, name, dirs)
		for _, dir := range dirs {
			if err := moveOut(dir, changed); err != nil {
				return err
			}
		}
		if err := restoreSkillLock(skillLockSnap{path: skillLockPath(home), name: name, hadFile: fileExists(skillLockPath(home))}); err != nil {
			return err
		}
	}
	for _, root := range slices.Sorted(maps.Keys(entry.ProjectSkills)) {
		for _, name := range entry.ProjectSkills[root] {
			others := slices.DeleteFunc(slices.Sorted(maps.Keys(gate.state.Components)), func(oid string) bool {
				return oid == id || !slices.Contains(gate.state.Components[oid].ProjectSkills[root], name)
			})
			if len(others) > 0 {
				for _, owner := range others {
					if err := shareDests(home, id, owner, root, name, true); err != nil {
						return err
					}
				}
				fmt.Printf("Kept %s in %s: another utility also installs it there\n", name, root)
				continue
			}
			dirs := ownedDests(gate.state, id, root, name)
			project := componentEntry(projectComponent(c, root), c.Store)
			project.Project = root
			changed := c.Store == "" || ownedDrift(project, root, name, dirs)
			for _, dir := range dirs {
				if err := move(dir, changed, true); err != nil {
					return err
				}
			}
		}
		if err := dropProjectComponent(root, id); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v; remove [components.%s] there by hand\n", root, err, tomlKey(id))
		}
	}
	ledgerPath := filepath.Join(home, ".gap", "published.json")
	ledger, err := readLedger(ledgerPath)
	if err != nil {
		return err
	}
	for _, path := range entry.Commands {
		if sharedCommands[path] {
			for _, owner := range sharedCommandOwners[path] {
				if err := recordOwnership(home, owner, nil, []string{path}, ""); err != nil {
					return err
				}
			}
			fmt.Printf("Kept %s: another source also owns it\n", path)
			continue
		}
		changed := true
		if c.Store != "" {
			for _, rel := range c.Commands {
				if filepath.Base(rel) == filepath.Base(path) {
					want, err := readUtilityFile(c.Store, rel)
					if err != nil {
						return err
					}
					got, _ := os.ReadFile(path)
					changed = string(want) != string(got)
				}
			}
		}
		if err := moveOut(path, changed); err != nil {
			return err
		}
		delete(ledger, path)
	}
	if err := writeLedger(ledgerPath, ledger); err != nil {
		return err
	}
	if err := writeAtomic(config, next); err != nil {
		return err
	}
	userLk, err := readLock(userConfigDir(home))
	if err != nil {
		return err
	}
	if _, pinned := userLk.Components[id]; pinned {
		delete(userLk.Components, id)
		if err := writeLock(userConfigDir(home), userLk); err != nil {
			return err
		}
	}
	state, err := readState(home)
	if err != nil {
		return err
	}
	delete(state.Components, id)
	if err := writeState(home, state); err != nil {
		return err
	}
	slices.Sort(edited)
	for _, path := range edited {
		fmt.Printf("Kept your edited copy at %s\n", path)
	}
	if moved > 0 {
		out.done(true, fmt.Sprintf("%s removed; its files moved to %s", id, backup))
	} else {
		out.done(true, id+" removed")
	}
	return nil
}

// markUninstalled records that a utility's uninstall hook has run, so a retry after a later failure does not run it again.
func markUninstalled(home, id string) error {
	state, err := readState(home)
	if err != nil {
		return err
	}
	entry := state.Components[id]
	entry.Installed = ""
	state.Components[id] = entry
	return writeState(home, state)
}

// dropProjectComponent removes id's table from a project's .gap/gap.toml. A missing file or table is fine.
func dropProjectComponent(root, id string) error {
	config := projectConfigFile(root)
	text, err := readOptional(config)
	if err != nil || text == "" {
		return err
	}
	next, found := removeTOMLComponent(text, id)
	if !found {
		return nil
	}
	if _, err := parseTOML(next); err != nil {
		return err
	}
	return writeAtomic(config, next)
}

func projectComponent(c component, root string) component {
	d, err := mergeOverride(distro{Components: map[string]component{c.ID: c}}, projectConfigFile(root), false)
	if err != nil {
		return c
	}
	return d.Components[c.ID]
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
