package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type checkRecord struct {
	Pin     string    `json:"pin"`
	Tip     string    `json:"tip"`
	New     int       `json:"new"`
	Checked time.Time `json:"checked"`
}

func checkPath(home string) string {
	return filepath.Join(home, ".gap", "check.json")
}

func readChecks(home string) (map[string]checkRecord, error) {
	checks := map[string]checkRecord{}
	b, err := os.ReadFile(checkPath(home))
	if errors.Is(err, os.ErrNotExist) {
		return checks, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &checks); err != nil {
		return nil, fmt.Errorf("%s: %w", checkPath(home), err)
	}
	return checks, nil
}

func writeChecks(home string, checks map[string]checkRecord) error {
	b, err := json.MarshalIndent(checks, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(checkPath(home)), 0o755); err != nil {
		return err
	}
	return writeAtomic(checkPath(home), string(b)+"\n")
}

// pendingCommits fetches the history of ref into a blobless cache and lists the commits after from, newest first.
// diverged is true when from is not an ancestor of the tip, so the count is unknown.
func pendingCommits(home string, c component, from string) (tip string, commits []string, diverged bool, err error) {
	url, err := cloneURL(c.Repo)
	if err != nil {
		return "", nil, false, err
	}
	cache := filepath.Join(home, ".gap", "cache", c.ID+".git")
	if _, statErr := os.Stat(cache); errors.Is(statErr, os.ErrNotExist) {
		if err := os.MkdirAll(cache, 0o755); err != nil {
			return "", nil, false, err
		}
		if _, err := git(cache, "init", "-q", "--bare"); err != nil {
			return "", nil, false, err
		}
	}
	if _, err := spin("Checking "+c.ID+" @ "+c.Ref, func() (string, error) {
		return git(cache, "fetch", "-q", "--filter=blob:none", url, c.Ref)
	}); err != nil {
		return "", nil, false, fmt.Errorf("component %s: fetch %s: %w", c.ID, c.Ref, err)
	}
	tip, err = git(cache, "rev-parse", "FETCH_HEAD^{commit}")
	if err != nil {
		return "", nil, false, err
	}
	if from == "" || from == tip {
		return tip, nil, false, nil
	}
	if _, err := git(cache, "merge-base", "--is-ancestor", from, tip); err != nil {
		return tip, nil, true, nil
	}
	log, err := git(cache, "log", "--format=%h %s", from+".."+tip)
	if err != nil {
		return "", nil, false, err
	}
	if log != "" {
		commits = strings.Split(log, "\n")
	}
	return tip, commits, false, nil
}

// distroTargets lists the distro components update and check act on. A component with pin = true
// is left out unless it is named.
func distroTargets(d distro, id string) []string {
	var ids []string
	for _, cid := range sortedIDs(d.Components) {
		c := d.Components[cid]
		if c.User || (id != "" && cid != id) || (id == "" && c.Pin) {
			continue
		}
		ids = append(ids, cid)
	}
	return ids
}

// checkComponents reports the commits waiting on each component's ref without changing a pin.
// It records the result in ~/.gap/check.json so gap status can show it offline.
func checkComponents(id string) error {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return err
	}
	projectRoot, err := invokingProjectRoot()
	if err != nil {
		return err
	}
	d, lk, err := effectivePins(repoRoot, home, projectRoot, "")
	if err != nil {
		return err
	}
	var ids []string
	for _, cid := range sortedIDs(d.Components) {
		if (id == "" && !d.Components[cid].Pin) || cid == id {
			ids = append(ids, cid)
		}
	}
	if id != "" && len(ids) == 0 {
		return fmt.Errorf("component %s is not in gap.toml or added with gap add", id)
	}
	checks, err := readChecks(home)
	if err != nil {
		return err
	}
	out := newReport(os.Stdout)
	behind, failed := 0, false
	for _, cid := range ids {
		c := d.Components[cid]
		pin := lk.Components[cid]
		tip, commits, diverged, err := pendingCommits(home, c, pin.Commit)
		if err != nil {
			out.done(false, err.Error())
			failed = true
			continue
		}
		rec := checkRecord{Pin: pin.Commit, Tip: tip, New: len(commits), Checked: time.Now().UTC()}
		switch {
		case pin.Commit == "":
			rec.New = 1
			out.done(true, fmt.Sprintf("%s is not pinned; %s @ %s is available", cid, tip[:7], c.Ref))
		case diverged:
			rec.New = 1
			out.done(true, fmt.Sprintf("%s @ %s moved to %s, which does not contain the pin %s", cid, c.Ref, tip[:7], short(pin.Commit)))
		case len(commits) == 0:
			out.done(true, cid+" is up to date"+pinLabel(pin))
		default:
			out.done(true, fmt.Sprintf("%s: %d new on %s since %s", cid, len(commits), c.Ref, short(pin.Commit)))
			for _, line := range commits {
				out.println("    " + line)
			}
		}
		if rec.New > 0 {
			behind++
		}
		checks[cid] = rec
	}
	if err := writeChecks(home, checks); err != nil {
		return err
	}
	if behind > 0 {
		out.next("gap update    move the pins to the newest commits")
	}
	if failed {
		return errSkipped
	}
	return nil
}

// updateDistro moves each distro component's pin in gap.lock to the newest commit on its ref,
// the way lazy.nvim rewrites lazy-lock.json. It installs nothing: the lock change goes
// through a PR, and gap sync installs it after the merge.
func updateDistro(id string) (int, error) {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return 0, err
	}
	d, err := readDistro(repoRoot)
	if err != nil {
		return 0, err
	}
	ids := distroTargets(d, id)
	if len(ids) == 0 {
		return 0, nil
	}
	lk, err := readLock(repoRoot)
	if err != nil {
		return 0, err
	}
	checks, err := readChecks(home)
	if err != nil {
		return 0, err
	}
	out := newReport(os.Stdout)
	moved := 0
	for _, cid := range ids {
		c := d.Components[cid]
		prev, pinned := lk.Components[cid]
		tip, commits, _, err := pendingCommits(home, c, prev.Commit)
		if err != nil {
			return moved, err
		}
		if pinned && prev.Commit == tip && prev.Ref == c.Ref {
			out.done(true, cid+" is up to date"+pinLabel(prev))
			checks[cid] = checkRecord{Pin: tip, Tip: tip, Checked: time.Now().UTC()}
			continue
		}
		pin, err := pinCommit(home, c, c.Ref, tip, prev)
		if err != nil {
			return moved, err
		}
		lk.Components[cid] = pin
		if err := writeLock(repoRoot, lk); err != nil {
			return moved, err
		}
		checks[cid] = checkRecord{Pin: tip, Tip: tip, Checked: time.Now().UTC()}
		moved++
		if pinned {
			out.done(true, fmt.Sprintf("%s moved from %s to %s @ %s", cid, short(prev.Commit), tip[:7], c.Ref))
		} else {
			out.done(true, cid+" pinned"+pinLabel(pin))
		}
		for _, line := range commits {
			out.println("    " + line)
		}
	}
	if err := writeChecks(home, checks); err != nil {
		return moved, err
	}
	if moved > 0 {
		out.next("commit gap.lock in "+repoRoot+" and merge it    gap sync installs a lock only from origin/HEAD",
			"gap sync    install the new pins after the merge")
	}
	return moved, nil
}

func short(commit string) string {
	return commit[:min(7, len(commit))]
}
