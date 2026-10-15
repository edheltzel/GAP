package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// manifestEntry is one source in global/skills.json. Source "." is the checkout gap was built from.
type manifestEntry struct {
	Source string   `json:"source"`
	Skills []string `json:"skills"`
}

func readManifest(repoRoot string) ([]manifestEntry, error) {
	b, err := os.ReadFile(filepath.Join(repoRoot, "global", "skills.json"))
	if err != nil {
		return nil, err
	}
	var entries []manifestEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, fmt.Errorf("global/skills.json: %w", err)
	}
	return entries, nil
}

// installSkills installs every manifest skill missing from ~/.agents/skills or from Claude's mirror
// through skills.sh. Skills the manifest does not list are never touched. It returns the names it installed.
func installSkills(repoRoot, home string, entries []manifestEntry) (map[string]bool, bool) {
	fresh := map[string]bool{}
	var npx string
	var err error
	ok := true
	for _, e := range entries {
		var missing []string
		for _, name := range e.Skills {
			if !installed(home, name) {
				missing = append(missing, name)
			}
		}
		if len(missing) == 0 {
			continue
		}
		if npx == "" {
			if npx, err = exec.LookPath("npx"); err != nil {
				fmt.Fprintln(os.Stderr, "Skipping skills: npx not found; install Node to let gap install skills")
				return fresh, false
			}
		}
		if !skillsAdd(npx, repoRoot, home, e.Source, missing) {
			ok = false
			continue
		}
		for _, name := range missing {
			fresh[name] = true
			fmt.Println(filepath.Join(home, ".agents", "skills", name))
		}
	}
	return fresh, ok
}

func skillsAdd(npx, repoRoot, home, source string, names []string) bool {
	src := source
	if src == "." {
		src = repoRoot
	}
	// skills.sh installs into ~/.agents/skills (read by Codex, Cursor, Pi, OMP) and links Claude Code to it.
	args := []string{"-y", "skills", "add", src, "-g", "-y", "-a", "claude-code"}
	for _, name := range names {
		args = append(args, "-s", name)
	}
	cmd := exec.Command(npx, args...)
	cmd.Dir = home
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Skills install failed for %s (%s): %v\n%s", source, strings.Join(names, ", "), err, out)
		return false
	}
	return true
}

// lockEntry is the part of skills.sh's ~/.agents/.skill-lock.json drift needs.
type lockEntry struct {
	SourceURL       string `json:"sourceUrl"`
	SkillPath       string `json:"skillPath"`
	SkillFolderHash string `json:"skillFolderHash"`
}

// checkDrift compares each installed managed skill with its source: this repo's skill folder, or the
// tree hash skills.sh recorded at install. Each drifted skill shows its diff and asks keep, take
// source, or skip; with no terminal it is only listed. "Keep" is remembered in the ledger.
func checkDrift(repoRoot, home string, entries []manifestEntry, fresh map[string]bool, ledger map[string]string, ledgerFile string) (bool, error) {
	var lock struct {
		Skills map[string]lockEntry `json:"skills"`
	}
	if b, err := os.ReadFile(filepath.Join(home, ".agents", ".skill-lock.json")); err == nil {
		_ = json.Unmarshal(b, &lock)
	}
	tty := stdinIsTerminal()
	in := bufio.NewReader(os.Stdin)
	ok := true
	for _, e := range entries {
		for _, name := range e.Skills {
			installedDir := filepath.Join(home, ".agents", "skills", name)
			if fresh[name] || !installed(home, name) {
				continue
			}
			got, err := treeHash(installedDir)
			if err != nil {
				return false, err
			}
			var want, sourceDir string
			var external lockEntry
			if e.Source == "." {
				if sourceDir = localSkillDir(repoRoot, name); sourceDir == "" {
					continue
				}
				if want, err = treeHash(sourceDir); err != nil {
					return false, err
				}
			} else {
				l, found := lock.Skills[name]
				if !found || l.SkillFolderHash == "" {
					continue
				}
				want, external = l.SkillFolderHash, l
			}
			if got == want || ledger["skill:"+name] == got {
				continue
			}
			fmt.Printf("Skill drifted from its source: %s\n", name)
			if !tty {
				ok = false
				continue
			}
			showDiff(sourceDir, external, installedDir)
			fmt.Printf("%s: [k]eep installed, [t]ake source, [s]kip? ", name)
			line, _ := in.ReadString('\n')
			switch strings.ToLower(strings.TrimSpace(line)) {
			case "k":
				ledger["skill:"+name] = got
				if err := writeLedger(ledgerFile, ledger); err != nil {
					return false, err
				}
			case "t":
				npx, err := exec.LookPath("npx")
				if err != nil {
					fmt.Fprintln(os.Stderr, "Cannot take source: npx not found")
					ok = false
					continue
				}
				backup := backupPath(installedDir, home)
				if err := copyDir(installedDir, backup); err != nil {
					return false, err
				}
				fmt.Println("Backed up to " + backup)
				if err := os.RemoveAll(installedDir); err != nil {
					return false, err
				}
				delete(ledger, "skill:"+name)
				if !skillsAdd(npx, repoRoot, home, e.Source, []string{name}) {
					ok = false
				}
			default:
				ok = false
			}
		}
	}
	return ok, nil
}

// localSkillDir finds skills/<scope>/<group>/<name> in this repo.
func localSkillDir(repoRoot, name string) string {
	matches, _ := filepath.Glob(filepath.Join(repoRoot, "skills", "*", "*", name, "SKILL.md"))
	if len(matches) != 1 {
		return ""
	}
	return filepath.Dir(matches[0])
}

// treeHash is the git tree id of dir's files, the same value skills.sh records as skillFolderHash.
func treeHash(dir string) (string, error) {
	tmp, err := os.MkdirTemp("", "gap-tree-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	base := gitEnv()
	run := func(env []string, args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir, cmd.Env = dir, env
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("hashing %s: git %s: %w", dir, args[len(args)-1], err)
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err := run(base, "init", "-q", "--bare", tmp); err != nil {
		return "", err
	}
	env := append(base, "GIT_INDEX_FILE="+filepath.Join(tmp, "index"))
	if _, err := run(env, "--git-dir="+tmp, "--work-tree="+dir, "add", "-A", "."); err != nil {
		return "", err
	}
	return run(env, "--git-dir="+tmp, "--work-tree="+dir, "write-tree")
}

// showDiff prints source vs installed. An external source is cloned shallowly from its lock entry.
func showDiff(sourceDir string, external lockEntry, installedDir string) {
	if sourceDir == "" {
		if external.SourceURL == "" {
			return
		}
		tmp, err := os.MkdirTemp("", "gap-src-")
		if err != nil {
			return
		}
		defer os.RemoveAll(tmp)
		if exec.Command("git", "clone", "-q", "--depth", "1", external.SourceURL, tmp).Run() != nil {
			fmt.Fprintln(os.Stderr, "Could not fetch "+external.SourceURL+" to show the diff")
			return
		}
		sourceDir = filepath.Join(tmp, filepath.Dir(external.SkillPath))
	}
	cmd := exec.Command("git", "--no-pager", "diff", "--no-index", "--", sourceDir, installedDir)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	_ = cmd.Run()
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case d.Type().IsRegular():
			return copyFile(path, target)
		}
		return errors.New("unsupported file type: " + path)
	})
}

func installed(home, name string) bool {
	for _, dir := range []string{filepath.Join(home, ".agents", "skills", name), filepath.Join(home, ".claude", "skills", name)} {
		if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
			return false
		}
	}
	return true
}
