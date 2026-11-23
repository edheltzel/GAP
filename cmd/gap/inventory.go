package main

import (
	"errors"
	"fmt"
	"os"
)

type itemRow struct {
	ID, Source, Scope, Repo, Ref, Commit, Hooks string
	Pinned, Held, Disabled                      bool
	Store                                       string
	Drift, New                                  int
	Skills                                      [][]string
}

func (r itemRow) user() bool { return r.Source == "user" }

func registeredRows(home, repoRoot string) ([]itemRow, error) {
	d, lk, err := registeredPins(repoRoot, home)
	if err != nil {
		return nil, err
	}
	trust, err := trustList(home)
	if err != nil {
		return nil, err
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
	return rows, nil
}

func componentStatus(home, projectRoot, id string, c component, e lockEntryM1, pinned bool, checks map[string]checkRecord) (statusRow, error) {
	row := statusRow{id: id, ref: c.Ref, commit: "unpinned", store: storeUnpinned}
	if pinned {
		row.ref, row.commit = e.Ref, e.Commit
		if len(row.commit) > 7 {
			row.commit = row.commit[:7]
		}
		store, err := storeState(storePath(home, id, e.Commit), e.Tree)
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

func inventory(home, repoRoot, projectRoot string) ([]itemRow, error) {
	rows, err := registeredRows(home, repoRoot)
	if err != nil {
		return nil, err
	}
	d, lk, err := effectivePins(repoRoot, home, projectRoot, "")
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
			continue
		}
		e, pinned := lk.Components[row.ID]
		st, err := componentStatus(home, projectRoot, row.ID, c, e, pinned, checks)
		if err != nil {
			return nil, err
		}
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
	Skills [][]string
	Drift  int
}

func projectInventory(home, repoRoot string, p projectRecord) projectView {
	v := projectView{Root: p.Root, Added: p.Added, Profile: "project"}
	if _, err := os.Stat(p.Root); errors.Is(err, os.ErrNotExist) {
		v.Missing = true
		return v
	}
	d, lk, err := effectivePins(repoRoot, home, p.Root, "")
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
	for _, id := range sortedIDs(d.Components) {
		c := d.Components[id]
		e, pinned := lk.Components[id]
		if !pinned {
			continue
		}
		dir := storePath(home, id, e.Commit)
		if state, err := storeState(dir, e.Tree); err != nil || state != storeOK {
			continue
		}
		item := projectItem{ID: id}
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
