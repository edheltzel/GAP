package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
)

type itemRow struct {
	ID, Source, Scope, Repo, Ref, Commit, Hooks string
	Pinned, Held, Disabled, Inactive            bool
	Store                                       string
	Drift, New                                  int
	Skills                                      [][]string
}

func (r itemRow) user() bool { return r.Source == "user" }

type storeResult struct {
	state string
	err   error
}

type storeCache struct {
	mu     sync.Mutex
	states map[string]storeResult
	misses int
}

func newStoreCache() *storeCache {
	return &storeCache{states: map[string]storeResult{}}
}

func (c *storeCache) state(dir, tree string) (string, error) {
	if c == nil {
		return storeState(dir, tree)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	key := dir + "\x00" + tree
	if r, ok := c.states[key]; ok {
		return r.state, r.err
	}
	c.misses++
	state, err := storeState(dir, tree)
	c.states[key] = storeResult{state, err}
	return state, err
}

func registeredRows(home, repoRoot string) ([]itemRow, error) {
	rows, _, err := registeredRowsLock(home, repoRoot)
	return rows, err
}

func registeredRowsLock(home, repoRoot string) ([]itemRow, lockFile, error) {
	d, lk, err := registeredPins(repoRoot, home)
	if err != nil {
		return nil, lockFile{}, err
	}
	trust, err := trustList(home)
	if err != nil {
		return nil, lockFile{}, err
	}
	var rows []itemRow
	for _, id := range sortedIDs(d.Components) {
		c := d.Components[id]
		row := itemRow{ID: id, Source: "distro", Scope: "global", Repo: repoIdentity(c.Repo), Ref: c.Ref, Commit: "unpinned", Hooks: "run", Held: c.Pin, Disabled: c.Disabled, Store: storeUnpinned}
		if c.User {
			row.Source, row.Hooks = "user", "ask"
			if isTrusted(c.Repo, trust) {
				row.Hooks = "trusted"
			}
		}
		if c.ProjectOnly {
			row.Scope = "project"
		}
		if pin, ok := lk.Components[id]; ok {
			row.Pinned, row.Commit = true, pin.Commit[:7]
		}
		rows = append(rows, row)
	}
	return rows, lk, nil
}

func componentStatus(home, projectRoot, id string, c component, e lockEntryM1, pinned bool, checks map[string]checkRecord, stores *storeCache) (statusRow, error) {
	row := statusRow{id: id, ref: c.Ref, commit: "unpinned", store: storeUnpinned}
	if rec, found := checks[id]; found && !pinned && rec.Pin == "" {
		row.pending = rec.New
	}
	if pinned {
		row.ref, row.commit = e.Ref, e.Commit
		if len(row.commit) > 7 {
			row.commit = row.commit[:7]
		}
		store, err := stores.state(storePath(home, id, e.Commit), e.Tree)
		if err != nil {
			return statusRow{}, err
		}
		row.store = store
		if rec, found := checks[id]; found && rec.Pin == e.Commit {
			row.pending = rec.New
		}
	}
	if row.store != storeOK {
		return row, nil
	}
	dir := storePath(home, id, e.Commit)
	for _, entry := range selectedEntries(c, dir, projectRoot) {
		for _, name := range entry.Skills {
			n := entryDriftCount(entry, entryHome(home, entry), name)
			row.skills = append(row.skills, []string{name, fmt.Sprintf("drift=%d", n)})
			if n > 0 {
				row.drifted++
			}
		}
	}
	return row, nil
}

func inventory(home, repoRoot, projectRoot string, stores *storeCache) ([]itemRow, error) {
	rows, machine, err := registeredRowsLock(home, repoRoot)
	if err != nil {
		return nil, err
	}
	d, lk, err := effectivePinsWith(repoRoot, home, projectRoot, "", stores)
	if err != nil {
		return nil, err
	}
	checks, err := readChecks(home)
	if err != nil {
		return nil, err
	}
	for i, row := range rows {
		c, active := d.Components[row.ID]
		if !active {
			rows[i].Inactive = !row.Disabled
			if e, ok := machine.Components[row.ID]; ok {
				state, err := stores.state(storePath(home, row.ID, e.Commit), e.Tree)
				if err != nil {
					state = storeUnreadable
				}
				rows[i].Store = state
			}
			continue
		}
		e, pinned := lk.Components[row.ID]
		st, err := componentStatus(home, projectRoot, row.ID, c, e, pinned, checks, stores)
		if err != nil {
			return nil, err
		}
		rows[i].Ref, rows[i].Commit, rows[i].Pinned = st.ref, st.commit, pinned
		rows[i].Store, rows[i].Drift, rows[i].New, rows[i].Skills = st.store, st.drifted, st.pending, st.skills
	}
	return rows, nil
}

type projectView struct {
	Root, Profile, Added string
	Missing              bool
	Err                  error
	Items                []projectItem
}

type projectItem struct {
	ID     string
	Store  string
	Skills [][]string
	Drift  int
}

const storeUnreadable = "unreadable"

func projectApplies(c component, root string) bool {
	if c.ProjectOnly {
		return c.InProject
	}
	for name, spec := range c.SkillSet {
		if spec.Load == "project" && (!c.Named || slices.Contains(c.Skills, name)) && projectMatches(root, spec.WhenFiles) {
			return true
		}
	}
	return false
}

func (v projectView) unchecked() bool {
	return slices.ContainsFunc(v.Items, func(it projectItem) bool { return it.Store != storeOK })
}

func projectInventory(home, repoRoot string, p projectRecord, stores *storeCache) projectView {
	v := projectView{Root: p.Root, Added: p.Added, Profile: "project"}
	if _, err := os.Stat(p.Root); errors.Is(err, os.ErrNotExist) {
		v.Missing = true
		return v
	}
	d, lk, err := effectivePinsWith(repoRoot, home, p.Root, "", stores)
	if err != nil {
		v.Err = err
		return v
	}
	if d.Profile != "" {
		v.Profile = d.Profile
	}
	if d.Profile == "crew" {
		return v
	}
	state, err := readState(home)
	if err != nil {
		v.Err = err
		return v
	}
	for _, id := range sortedIDs(d.Components) {
		c := d.Components[id]
		e, pinned := lk.Components[id]
		if !pinned {
			continue
		}
		dir := storePath(home, id, e.Commit)
		store, err := stores.state(dir, e.Tree)
		if err != nil {
			store = storeUnreadable
		}
		if store != storeOK {
			if projectApplies(c, p.Root) || len(state.Components[id].ProjectSkills[p.Root]) > 0 {
				v.Items = append(v.Items, projectItem{ID: id, Store: store})
			}
			continue
		}
		item := projectItem{ID: id, Store: storeOK}
		for _, entry := range selectedEntries(c, dir, p.Root) {
			if entry.Project == "" {
				continue
			}
			for _, name := range entry.Skills {
				n := entryDriftCount(entry, entry.Project, name)
				item.Skills = append(item.Skills, []string{name, fmt.Sprintf("drift=%d", n)})
				if n > 0 {
					item.Drift++
				}
			}
		}
		if len(item.Skills) > 0 {
			v.Items = append(v.Items, item)
		}
	}
	return v
}
