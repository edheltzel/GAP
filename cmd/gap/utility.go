package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"
)

// parseUtilitySpec turns owner/repo[@ref], github.com/owner/repo[@ref], or a file:// URL or absolute path into a repo, ref, and default id.
func parseUtilitySpec(spec string) (repo, ref, id string, err error) {
	repo = spec
	if i := strings.LastIndex(spec, "@"); i > 0 {
		repo, ref = spec[:i], spec[i+1:]
		if ref == "" {
			return "", "", "", fmt.Errorf("usage: gap add owner/repo[@ref]")
		}
	}
	switch {
	case strings.HasPrefix(repo, "file://") || filepath.IsAbs(repo):
	case strings.HasPrefix(repo, "github.com/"):
	default:
		repo = "github.com/" + repo
	}
	repo = strings.TrimSuffix(strings.TrimSuffix(repo, "/"), ".git")
	if _, err := cloneURL(repo); err != nil {
		return "", "", "", err
	}
	id = strings.ToLower(filepath.Base(strings.TrimPrefix(repo, "file://")))
	return repo, ref, id, nil
}

// defaultBranch is the branch the remote's HEAD points at.
func defaultBranch(url string) (string, error) {
	out, err := git(".", "ls-remote", "--symref", url, "HEAD")
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
	return "", fmt.Errorf("%s has no default branch; pass owner/repo@ref", url)
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

func addUtility(spec, id string) error {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return err
	}
	projectRoot, err := invokingProjectRoot()
	if err != nil {
		return err
	}
	repo, ref, defaultID, err := parseUtilitySpec(spec)
	if err != nil {
		return err
	}
	if id == "" {
		id = defaultID
	}
	if !validID(id) {
		return fmt.Errorf("id %q must use lowercase letters, digits, dot, dash, or underscore; policy and manifest are reserved; pass --id", id)
	}
	d, _, err := registeredPins(repoRoot, home)
	if err != nil {
		return err
	}
	if c, taken := d.Components[id]; taken {
		if c.User {
			return fmt.Errorf("utility %s is already added; run gap update %s", id, id)
		}
		return fmt.Errorf("%s is already a distro component; pass --id to add this repo under another id", id)
	}
	config := userConfigFile(home)
	text, err := readOptional(config)
	if err != nil {
		return err
	}
	doc, err := parseTOML(text)
	if err != nil {
		return err
	}
	if components, ok := doc["components"].(map[string]any); ok {
		if _, exists := components[id]; exists {
			return fmt.Errorf("%s already declares components.%s; pass --id to use another id", config, tomlKey(id))
		}
	}
	url, err := cloneURL(repo)
	if err != nil {
		return err
	}
	if ref == "" {
		if ref, err = defaultBranch(url); err != nil {
			return err
		}
	}
	commit, err := resolveRef(url, ref)
	if err != nil {
		return err
	}
	c := component{ID: id, Repo: repo, Ref: ref, User: true}
	pin, err := pinCommit(home, c, ref, commit, lockEntryM1{})
	if err != nil {
		return err
	}
	m, err := readUtilityManifest(storePath(home, id, commit))
	if err != nil {
		return fmt.Errorf("%s: %w", repo, err)
	}
	c, err = applyManifest(c, storePath(home, id, commit))
	if err != nil {
		return err
	}
	if err := checkCommandOwners(repoRoot, home, distro{Components: map[string]component{id: c}}); err != nil {
		return err
	}
	out := newReport(os.Stdout)
	out.section(id, describeUtility(m)+pinLabel(pin))
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if text != "" {
		text += "\n"
	}
	text += fmt.Sprintf("[components.%s]\nrepo = %q\nref = %q\n", tomlKey(id), repo, ref)
	if _, err := parseTOML(text); err != nil {
		return fmt.Errorf("adding %s to %s: %w", id, config, err)
	}
	if err := writeAtomic(config, text); err != nil {
		return err
	}
	if err := setUserPin(home, id, pin); err != nil {
		return err
	}
	return installUtility(home, repoRoot, projectRoot, id, "")
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
func installUtility(home, repoRoot, projectRoot, id, prevStore string) error {
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
			if !installed(target, name) || skillDriftCount(entry.Path, target, name) == 0 {
				continue
			}
			if skillDriftCount(baseline, target, name) > 0 {
				fmt.Fprintf(os.Stderr, "%s: your edited copy of %s was kept; gap restore %s takes the new version\n", id, name, id)
				skillsOK = false
				continue
			}
			if replaced, err := replaceSkill("", repoRoot, target, entry, name); err != nil || !replaced {
				fmt.Fprintf(os.Stderr, "Updating %s: %v\n", name, err)
				skillsOK = false
			}
		}
	}
	mcpFailures, err := publishComponentMCP(home, d)
	if err != nil {
		return err
	}
	skillsOK = installSkills(repoRoot, home, entries) && skillsOK
	driftOK, err := checkDrift(repoRoot, home, entries)
	if err != nil {
		return err
	}
	if err := recordDeclarativeInstalls(home, projectRoot, d, owned, mcpFailures); err != nil {
		return err
	}
	if !hooksOK || !commandsOK || len(mcpFailures) > 0 || !skillsOK || !driftOK {
		return errSkipped
	}
	return nil
}

func updateUtilities(id string) error {
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
	lk, err := readLock(userConfigDir(home))
	if err != nil {
		return err
	}
	var ids []string
	for _, cid := range sortedIDs(d.Components) {
		if d.Components[cid].User && (id == "" || cid == id) {
			ids = append(ids, cid)
		}
	}
	if id != "" && len(ids) == 0 {
		if _, distroID := d.Components[id]; distroID {
			return fmt.Errorf("%s is a distro component; move its pin with gap pin %s@<ref>", id, id)
		}
		return fmt.Errorf("utility %s is not added; run gap add owner/repo", id)
	}
	state, err := readState(home)
	if err != nil {
		return err
	}
	out := newReport(os.Stdout)
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
			out.result(true, cid+" is up to date"+pinLabel(prev))
		} else {
			pin, err := pinCommit(home, c, c.Ref, commit, prev)
			if err != nil {
				return err
			}
			if _, err := readUtilityManifest(storePath(home, cid, commit)); err != nil {
				return fmt.Errorf("%s at %s: %w", cid, commit[:7], err)
			}
			lk.Components[cid] = pin
			if err := writeLock(userConfigDir(home), lk); err != nil {
				return err
			}
			out.result(true, cid+" pinned"+pinLabel(pin))
		}
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
	for _, cid := range ids {
		if err := installUtility(home, repoRoot, projectRoot, cid, baselines[cid]); err != nil {
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
	d, lk, err := registeredPins(repoRoot, home)
	if err != nil {
		return err
	}
	trust, err := trustList(home)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSOURCE\tREPO\tREF\tCOMMIT\tHOOKS")
	for _, id := range sortedIDs(d.Components) {
		c := d.Components[id]
		source, hooks := "distro", "run"
		if c.User {
			source, hooks = "user", "ask"
			if isTrusted(c.Repo, trust) {
				hooks = "trusted"
			}
		}
		commit := "unpinned"
		if pin, ok := lk.Components[id]; ok {
			commit = pin.Commit[:7]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", id, source, repoIdentity(c.Repo), c.Ref, commit, hooks)
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
		manifest, err := readUtilityManifest(dir)
		if err != nil {
			return err
		}
		if manifest.Hooks.Uninstall != "" {
			if !gate.approve(c, "uninstall", manifest.Hooks.Uninstall) {
				return fmt.Errorf("%s: uninstall hook not approved; nothing removed", id)
			}
			if err := runComponentCommand(dir, manifest.Hooks.Uninstall); err != nil {
				return fmt.Errorf("%s uninstall hook failed; nothing removed: %w", id, err)
			}
			out.result(true, id+" uninstall hook finished")
			if err := markUninstalled(home, id); err != nil {
				return err
			}
		} else if manifest.Hooks.Install != "" && !gate.trusted(c) {
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
	if extra, err := readManifest(repoRoot); err == nil {
		for _, e := range extra {
			for _, name := range e.Skills {
				shared[name] = true
			}
		}
	}
	backup := backupPath(home, home)
	var edited []string
	moved := 0
	moveOut := func(path string, changed bool) error {
		if _, err := os.Lstat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		rel, err := filepath.Rel(home, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("refusing to remove %s: outside home", path)
		}
		dest := filepath.Join(backup, rel)
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
	for _, name := range entry.Skills {
		if shared[name] {
			for _, owner := range sharedSkillOwners[name] {
				if err := recordOwnership(home, owner, []string{name}, nil, ""); err != nil {
					return err
				}
			}
			fmt.Printf("Kept %s: another component or global/skills.json also selects it\n", name)
			continue
		}
		changed := c.Store == "" || skillDriftCount(c.Store, home, name) > 0
		for _, dir := range skillDirs(home, name) {
			if err := moveOut(dir, changed); err != nil {
				return err
			}
		}
		if err := restoreSkillLock(skillLockSnap{path: skillLockPath(home), name: name, hadFile: fileExists(skillLockPath(home))}); err != nil {
			return err
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
		out.result(true, fmt.Sprintf("%s removed; its files moved to %s", id, backup))
	} else {
		out.result(true, id+" removed")
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
