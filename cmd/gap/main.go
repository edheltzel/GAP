// Command gap publishes global/AGENTS.md and global/commands/*.md into the files each agent harness already reads.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"
)

const (
	trustLine = "Trust both GAP hooks (SessionStart and SubagentStart) in Codex with /hooks, confirm a session and a subagent both get the policy, then run: gap sync --remove-codex-legacy"
	usage     = `gap publishes global/AGENTS.md and each global/commands/*.md into the files
those harnesses already read, and installs the skills listed in global/skills.json.

Usage:
  gap sync                         Publish the policy and commands, install missing skills, check skills for drift
  gap sync --remove-codex-legacy   Same, then strip the old policy copy from ~/.codex/AGENTS.md
  gap --help                       Show this help

gap publishes the checkout it was installed from. Reinstall with
go install ./cmd/gap after cmd/gap changes or the repo moves.
Installing skills needs Node (npx), because gap drives skills.sh.`
	// Ledger value for Codex AGENTS.md once its legacy GAP prefix is stripped; never a sha256 digest.
	cleaned = "cleaned"
)

// docsDir is the absolute global/docs/agents path that adapt writes into every policy link.
var docsDir string

// errSkipped means at least one destination was refused; the reason is already on stderr.
var errSkipped = errors.New("skipped")

var errPolicyChanged = errors.New("policy destination changed during sync")

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help" || args[0] == "help") {
		fmt.Println(usage)
		return 0
	}
	if len(args) == 0 || args[0] != "sync" {
		fmt.Fprintln(os.Stderr, usage)
		return 2
	}
	flags := flag.NewFlagSet("sync", flag.ContinueOnError)
	removeLegacy := flags.Bool("remove-codex-legacy", false, "strip the old GAP text from Codex AGENTS.md once both hooks are trusted")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() > 0 {
		fmt.Fprintln(os.Stderr, usage)
		return 2
	}
	if err := syncAll(*removeLegacy); err != nil {
		if !errors.Is(err, errSkipped) {
			fmt.Fprintln(os.Stderr, err)
		}
		return 1
	}
	return 0
}

func syncAll(removeLegacy bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	codexHome, err := resolveCodexHome(os.Getenv("CODEX_HOME"), home)
	if err != nil {
		return err
	}
	repoRoot, err := sourceRoot()
	if err != nil {
		return err
	}
	docsDir = filepath.Join(repoRoot, "global", "docs", "agents")
	co, err := readCheckout(repoRoot)
	if err != nil {
		return err
	}
	if refusal := refuseCheckout(repoRoot, co); refusal != "" {
		return errors.New(refusal)
	}

	raw, err := os.ReadFile(filepath.Join(repoRoot, "global", "AGENTS.md"))
	if err != nil {
		return err
	}
	ledgerPath := filepath.Join(home, ".gap", "published.json")
	ledger, err := readLedger(ledgerPath)
	if err != nil {
		return err
	}

	codexPolicy := filepath.Join(codexHome, "gap", "global-agent-policy.md")
	skipped, codexWrote := false, false
	for _, c := range policyDests(home, codexPolicy) {
		wrote, err := publishPolicy(string(raw), c.dest, c.cursor, ledger, ledgerPath, true, home)
		if err != nil {
			return err
		}
		if !wrote {
			skipped = true
		}
		if c.dest == codexPolicy && wrote {
			codexWrote = true
		}
	}

	commandsOK, err := publishCommands(repoRoot, home, ledger, ledgerPath)
	if err != nil {
		return err
	}
	if !commandsOK {
		skipped = true
	}

	payload := adapt(string(raw), false)
	// Cleanup runs before this run's hook install, so a hook added just now (not yet trusted in /hooks) never counts.
	if removeLegacy {
		if !codexWrote {
			fmt.Fprintf(os.Stderr, "Refusing to strip unmarked Codex policy; this run did not publish the GAP policy file: %s\n", codexPolicy)
			skipped = true
		} else if ok, err := cleanupLegacyCodex(codexHome, codexPolicy, payload, ledger, home, ledgerPath); err != nil {
			return err
		} else if !ok {
			skipped = true
		}
	}
	ok, err := afterCodexPublish(codexWrote, codexPolicy, payload, codexHome, home)
	if err != nil {
		return err
	}
	entries, err := readManifest(repoRoot)
	if err != nil {
		return err
	}
	skillsOK := installSkills(repoRoot, home, entries)
	driftOK, err := checkDrift(repoRoot, home, entries)
	if err != nil {
		return err
	}
	if advice := empryoPinAdvice(home); advice != "" {
		fmt.Println(advice)
	}
	if !ok || skipped || !skillsOK || !driftOK {
		return errSkipped
	}
	return nil
}

// empryoPins is the skill set GAP recommends for Empryo's skills.autoLoad. Pinned skills load in full into every prompt.
var empryoPins = []string{"do-todo", "do-commit", "but-axi"}

// empryoPinAdvice reads Empryo's global config and, when skills.autoLoad differs from empryoPins, returns the settings call that fixes it. gap never writes Empryo's config.
func empryoPinAdvice(home string) string {
	b, err := os.ReadFile(filepath.Join(home, ".empryo", "config.json"))
	if err != nil {
		return ""
	}
	var config struct {
		Skills struct {
			AutoLoad []string `json:"autoLoad"`
		} `json:"skills"`
	}
	if json.Unmarshal(b, &config) != nil || slices.Equal(slices.Sorted(slices.Values(config.Skills.AutoLoad)), slices.Sorted(slices.Values(empryoPins))) {
		return ""
	}
	want, _ := json.Marshal(empryoPins)
	return fmt.Sprintf("Empryo pins %d skills in skills.autoLoad; GAP recommends %s. Ask an agent inside Empryo to run:\n  settings(action:\"set\", key:\"skills.autoLoad\", value:'%s', scope:\"global\")", len(config.Skills.AutoLoad), strings.Join(empryoPins, ", "), want)
}

type policyDest struct {
	dest   string
	cursor bool
}

func policyDests(home, codexPolicy string) []policyDest {
	return []policyDest{
		{filepath.Join(home, ".claude", "rules", "global-agent-policy.md"), false},
		{filepath.Join(home, ".pi", "agent", "APPEND_SYSTEM.md"), false},
		{filepath.Join(home, ".omp", "agent", "APPEND_SYSTEM.md"), false},
		{codexPolicy, false},
		{filepath.Join(home, ".cursor", "rules", "global-agent-policy.mdc"), true},
		{filepath.Join(home, ".empryo", "EMPRYO.md"), false},
	}
}

func publishCommands(repoRoot, home string, ledger map[string]string, ledgerPath string) (bool, error) {
	dir := filepath.Join(repoRoot, "global", "commands")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	ok := true
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return false, err
		}
		for _, dest := range []string{
			filepath.Join(home, ".omp", "agent", "commands", name),
			filepath.Join(home, ".claude", "commands", name),
		} {
			wrote, err := publish(string(b), dest, ledger, ledgerPath, true, home)
			if err != nil {
				return false, err
			}
			if !wrote {
				ok = false
			}
		}
	}
	return ok, nil
}

// sourceRoot is the checkout gap was built from, so `gap sync` publishes that checkout from any directory.
// ponytail: compile-time path; moving the checkout or building with -trimpath needs a fresh `go install ./cmd/gap`.
func sourceRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok || !filepath.IsAbs(file) {
		return "", errors.New("gap was built without its source path; run go install ./cmd/gap from the GAP checkout")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	if _, err := os.Stat(filepath.Join(root, "global", "AGENTS.md")); err != nil {
		return "", fmt.Errorf("gap was built from %s, which has no global/AGENTS.md; run go install ./cmd/gap from the GAP checkout", root)
	}
	return root, nil
}

// gitLocalEnv is `git rev-parse --local-env-vars`. An inherited GIT_DIR (git hooks, bare-repo dotfiles)
// would otherwise point every checkout check at a different repo.
var gitLocalEnv = []string{"GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_CONFIG", "GIT_CONFIG_PARAMETERS", "GIT_CONFIG_COUNT", "GIT_OBJECT_DIRECTORY", "GIT_DIR", "GIT_WORK_TREE", "GIT_IMPLICIT_WORK_TREE", "GIT_GRAFT_FILE", "GIT_INDEX_FILE", "GIT_NO_REPLACE_OBJECTS", "GIT_REPLACE_REF_BASE", "GIT_PREFIX", "GIT_SHALLOW_FILE", "GIT_COMMON_DIR"}

// gitEnv is the environment minus gitLocalEnv, so git acts on the repo gap names, not an inherited one.
func gitEnv() []string {
	return slices.DeleteFunc(os.Environ(), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return slices.Contains(gitLocalEnv, name)
	})
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gitEnv()
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

type checkout struct {
	branch, primary, gitDir, commonDir string
	unmerged                           bool
}

func readCheckout(root string) (checkout, error) {
	originHead, err := git(root, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return checkout{}, errors.New("refusing to publish: origin/HEAD is not set; run git remote set-head origin --auto")
	}
	c := checkout{primary: strings.TrimPrefix(originHead, "origin/")}
	_, diffErr := git(root, "diff", "--quiet", "refs/remotes/origin/HEAD", "--", "global/", "skills/")
	untracked, untrackedErr := git(root, "ls-files", "--others", "--exclude-standard", "--", "global/", "skills/")
	ignored, ignoredErr := git(root, "ls-files", "-z", "--others", "--ignored", "--exclude-standard", "--", "global/", "skills/")
	c.unmerged = diffErr != nil || untrackedErr != nil || ignoredErr != nil || untracked != "" || ignoredBlocks(ignored)
	if c.branch, err = git(root, "rev-parse", "--abbrev-ref", "HEAD"); err != nil {
		return c, err
	}
	if c.gitDir, err = git(root, "rev-parse", "--git-dir"); err != nil {
		return c, err
	}
	c.commonDir, err = git(root, "rev-parse", "--git-common-dir")
	return c, err
}

// refuseCheckout returns why this checkout must not publish, or "" when it may.
func refuseCheckout(root string, c checkout) string {
	if c.branch != c.primary && c.branch != "gitbutler/workspace" {
		return fmt.Sprintf("refusing to publish from %s; checkout %s", c.branch, c.primary)
	}
	if resolveIn(root, c.gitDir) != resolveIn(root, c.commonDir) {
		return "refusing to publish from a worktree"
	}
	if c.unmerged {
		return fmt.Sprintf("refusing to publish: global/ or skills/ differs from origin/%s; commit, merge and pull first", c.primary)
	}
	return ""
}

func resolveIn(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, path)
}

// ignoredBlocks reports whether an ignored path would change what gap publishes or what skills.sh installs.
// node_modules and .DS_Store do not. An ignored global/skills.json still does.
func ignoredBlocks(list string) bool {
	list = strings.TrimRight(list, "\x00")
	if list == "" {
		return false
	}
	for _, path := range strings.Split(list, "\x00") {
		if path != "" && ignoredPathBlocks(path) {
			return true
		}
	}
	return false
}

func ignoredPathBlocks(path string) bool {
	path = filepath.ToSlash(path)
	switch path {
	case "global/AGENTS.md", "global/skills.json":
		return true
	}
	if filepath.Dir(path) == "global/commands" && strings.HasSuffix(path, ".md") {
		return true
	}
	if strings.HasPrefix(path, "skills/") {
		return !indexPathExcluded(strings.TrimPrefix(path, "skills/")) && filepath.Base(path) != ".DS_Store"
	}
	return false
}

func resolveCodexHome(env, home string) (string, error) {
	if env == "" {
		return filepath.Join(home, ".codex"), nil
	}
	if !filepath.IsAbs(env) {
		return "", fmt.Errorf("CODEX_HOME must be an absolute path: %s", env)
	}
	return env, nil
}

const (
	gapBegin    = "<!-- gap:begin -->"
	gapEnd      = "<!-- gap:end -->"
	cursorFront = "---\nalwaysApply: true\n---\n\n"
)

func adapt(source string, cursor bool) string {
	text := strings.ReplaceAll(source, "](docs/agents/", "]("+docsDir+"/")
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if cursor {
		return cursorFront + text
	}
	return text
}

func digest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// ours reports whether a destination may be overwritten without asking:
// it is missing, already the payload, or byte-for-byte the last publish.
func ours(existing string, exists bool, payload, last string) bool {
	return !exists || existing == payload || (last != "" && digest(existing) == last)
}

// ledgerPrefixEnd finds the line boundary where the leading text hashes to last.
func ledgerPrefixEnd(existing, last string) (int, bool) {
	for cut := 0; cut < len(existing); {
		next := len(existing)
		if nl := strings.IndexByte(existing[cut:], '\n'); nl >= 0 {
			next = cut + nl + 1
		}
		if digest(existing[:next]) == last {
			return next, true
		}
		cut = next
	}
	return 0, false
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func hookCommand(policyPath string) string {
	return "cat " + shellQuote(policyPath)
}

type hookResult int

const (
	hookChanged hookResult = iota
	hookPresent
	hookMalformed
)

// SessionStart covers root sessions; SubagentStart is the only event Codex runs for subagents.
// An empty matcher means the group carries no matcher key.
var gapEvents = []struct{ event, matcher string }{
	{"SessionStart", "startup|resume|clear|compact"},
	{"SubagentStart", ""},
}

// insertGapHook appends the GAP hook to each event that lacks it. Other entries keep their values;
// key order is not preserved because hooks.json is rewritten from a decoded map.
func insertGapHook(text, policyPath string) (string, hookResult) {
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	var root any
	if dec.Decode(&root) != nil {
		return "", hookMalformed
	}
	if _, err := dec.Token(); err != io.EOF {
		return "", hookMalformed
	}
	doc, ok := root.(map[string]any)
	if !ok {
		return "", hookMalformed
	}
	hooks, ok := doc["hooks"].(map[string]any)
	if !ok {
		return "", hookMalformed
	}
	command := hookCommand(policyPath)
	changed := false
	for _, e := range gapEvents {
		current, exists := hooks[e.event]
		groups, isList := current.([]any)
		if exists && !isList {
			return "", hookMalformed
		}
		if hasGapHook(groups, e.matcher, command) {
			continue
		}
		group := map[string]any{"hooks": []any{map[string]any{"type": "command", "command": command, "timeout": 10, "additionalContextLimit": 0}}}
		if e.matcher != "" {
			group["matcher"] = e.matcher
		}
		hooks[e.event] = append(groups, group)
		changed = true
	}
	if !changed {
		return "", hookPresent
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if enc.Encode(doc) != nil {
		return "", hookMalformed
	}
	return buf.String(), hookChanged
}

func hasGapHook(groups []any, matcher, command string) bool {
	for _, g := range groups {
		group, ok := g.(map[string]any)
		if !ok {
			continue
		}
		if m, has := group["matcher"]; (matcher == "" && has) || (matcher != "" && m != matcher) {
			continue
		}
		list, _ := group["hooks"].([]any)
		for _, h := range list {
			hook, ok := h.(map[string]any)
			if ok && hook["type"] == "command" && hook["command"] == command && isZero(hook["additionalContextLimit"]) && hook["async"] != true {
				return true
			}
		}
	}
	return false
}

func isZero(v any) bool {
	n, ok := v.(json.Number)
	if !ok {
		return false
	}
	f, err := n.Float64()
	return err == nil && f == 0
}

func writeAtomic(path, text string) error {
	return writeStaged(path, text, func(tmp string) error { return os.Rename(tmp, path) })
}

func writeStaged(path, text string, commit func(string) error) error {
	info, err := os.Lstat(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	mode := fs.FileMode(0o644)
	preserveMode := info != nil && info.Mode().IsRegular()
	if preserveMode {
		mode = info.Mode().Perm()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	if preserveMode {
		if err := file.Chmod(mode); err != nil {
			file.Close()
			return err
		}
	}
	_, writeErr := file.WriteString(text)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return commit(tmp)
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, info.Mode().Perm())
}

func readLedger(path string) (map[string]string, error) {
	ledger := map[string]string{}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return ledger, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &ledger); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if ledger == nil {
		ledger = map[string]string{}
	}
	return ledger, nil
}

func writeLedger(path string, ledger map[string]string) error {
	b, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, string(b)+"\n")
}

func backupPath(dest, root string) string {
	stamp := strings.NewReplacer(":", "-", ".", "-").Replace(time.Now().UTC().Format("2006-01-02T15:04:05.000Z"))
	tail, err := filepath.Rel(root, dest)
	if err != nil || strings.HasPrefix(tail, "..") {
		tail = dest[1:]
	}
	return filepath.Join(root, ".gap", "backups", stamp, tail)
}

// recoveryKeep is the number of timestamped recovery copies retained per destination.
const recoveryKeep = 3

func recoveryID(dest, home string) string {
	rel, err := filepath.Rel(home, dest)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return strings.TrimPrefix(dest, string(filepath.Separator))
	}
	return rel
}

func recoveryDir(dest, home string) string {
	return filepath.Join(home, ".gap", "recovery", recoveryID(dest, home))
}

func makeRecovery(dest, home string) (string, error) {
	stamp := strings.NewReplacer(":", "-", ".", "-").Replace(time.Now().UTC().Format("2006-01-02T15:04:05.000Z"))
	parent := recoveryDir(dest, home)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(parent, stamp+"-")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, filepath.Base(dest)), nil
}

var pruneRecovery = pruneRecoveryDir

func recoveryHoldsSymlink(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Type()&fs.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

func removeLegacyRecovery(dest string) {
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(dest), "."+filepath.Base(dest)+".gap-recovery-*"))
	if err != nil {
		return
	}
	for _, p := range matches {
		info, err := os.Lstat(p)
		if err != nil || !info.IsDir() {
			continue
		}
		if err := os.RemoveAll(p); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: removing leftover recovery dir %s: %v\n", p, err)
		}
	}
}

func pruneRecoveryDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	regular := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if recoveryHoldsSymlink(filepath.Join(dir, name)) {
			continue
		}
		regular = append(regular, name)
	}
	slices.Sort(regular)
	if len(regular) <= recoveryKeep {
		return nil
	}
	for _, name := range regular[:len(regular)-recoveryKeep] {
		if err := os.RemoveAll(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	return nil
}

const replaceWarning = "This harness will no longer load that file. The backup keeps the bytes only. Add to global/AGENTS.md only rules you want every harness to follow."

func replacePrompt(dest, backup string) string {
	return fmt.Sprintf("%s is not the GAP publish. Move it to %s and replace it with GAP? %s [y/N] ", dest, backup, replaceWarning)
}

// termReady and confirmLine are the terminal boundary. Tests replace them.
var (
	termReady   = stdinIsTerminal
	confirmLine = func() string {
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		return line
	}
)

func confirmReplace(dest, backup string) bool {
	fmt.Print(replacePrompt(dest, backup))
	return strings.ToLower(strings.TrimSpace(confirmLine())) == "y"
}

func splicePrompt(dest, backup string) string {
	return fmt.Sprintf("%s is not the GAP publish. Back it up to %s and append the GAP block? The rest of the file stays. [y/N] ", dest, backup)
}

func confirmSplice(dest, backup string) bool {
	fmt.Print(splicePrompt(dest, backup))
	return strings.ToLower(strings.TrimSpace(confirmLine())) == "y"
}

func refuse(format string, args ...any) (bool, error) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	return false, nil
}

func isSymlink(info fs.FileInfo) bool { return info.Mode()&fs.ModeSymlink != 0 }

func marked(body string) string {
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return gapBegin + "\n" + body + gapEnd + "\n"
}

func canonical(body string, cursor bool) string {
	if cursor {
		return cursorFront + marked(body)
	}
	return marked(body)
}

func findMarkerLine(text, marker string, from int) (start, end int, ok bool) {
	for from < len(text) {
		i := strings.Index(text[from:], marker)
		if i < 0 {
			return 0, 0, false
		}
		i += from
		if i > 0 && text[i-1] != '\n' {
			from = i + 1
			continue
		}
		after := i + len(marker)
		switch {
		case after == len(text):
			return i, after, true
		case text[after] == '\n':
			return i, after + 1, true
		case text[after] == '\r' && after+1 < len(text) && text[after+1] == '\n':
			return i, after + 2, true
		default:
			from = i + 1
		}
	}
	return 0, 0, false
}

// gapSpan is the last complete marker pair. An appended block stays last, so a stray begin line above it is outside.
func gapSpan(text string) (start, innerStart, innerEnd, end int, ok bool) {
	for from := 0; ; {
		beginStart, beginEnd, found := findMarkerLine(text, gapBegin, from)
		if !found {
			return
		}
		endStart, endLine, found := findMarkerLine(text, gapEnd, beginEnd)
		if !found {
			return
		}
		for {
			s, e, found := findMarkerLine(text, gapBegin, beginEnd)
			if !found || s >= endStart {
				break
			}
			beginStart, beginEnd = s, e
		}
		start, innerStart, innerEnd, end, ok = beginStart, beginEnd, endStart, endLine, true
		from = endLine
	}
}

func replaceBlock(text, body string) (string, bool) {
	start, _, _, end, ok := gapSpan(text)
	if !ok {
		return "", false
	}
	return text[:start] + marked(body) + text[end:], true
}

func appendBlock(existing, body string) string {
	block := marked(body)
	if existing == "" || strings.HasSuffix(existing, "\n") {
		return existing + block
	}
	return existing + "\n" + block
}

func blockInner(text string) (string, bool) {
	_, innerStart, innerEnd, _, ok := gapSpan(text)
	if !ok {
		return "", false
	}
	return text[innerStart:innerEnd], true
}

func matchesPublish(onDisk, payload string) bool {
	inner, ok := blockInner(onDisk)
	return ok && inner == payload
}

func policyUnchanged(dest, existing, linkTarget string, before fs.FileInfo) error {
	current, err := os.Lstat(dest)
	if errors.Is(err, fs.ErrNotExist) {
		if before == nil {
			return nil
		}
		return errPolicyChanged
	}
	if err != nil {
		return err
	}
	if before == nil || !os.SameFile(before, current) || before.Mode() != current.Mode() {
		return errPolicyChanged
	}
	if isSymlink(current) {
		got, err := os.Readlink(dest)
		if err != nil {
			return err
		}
		if got != linkTarget {
			return errPolicyChanged
		}
		return nil
	}
	if current.Mode().IsRegular() {
		b, err := os.ReadFile(dest)
		if err != nil {
			return err
		}
		if string(b) != existing {
			return errPolicyChanged
		}
	}
	return nil
}

func policyContentUnchanged(path, existing, linkTarget string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if isSymlink(info) {
		got, err := os.Readlink(path)
		if err != nil {
			return err
		}
		if got != linkTarget {
			return errPolicyChanged
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return errPolicyChanged
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if string(b) != existing {
		return errPolicyChanged
	}
	return nil
}

var (
	renamePolicy = os.Rename
	linkPolicy   = os.Link
)

func isCrossDevice(err error) bool {
	return errors.Is(err, syscall.EXDEV)
}

var (
	copyPolicy     = copyPolicyFile
	afterCopyWrite func(src, dst string)
)

func copyRace(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, errPolicyChanged) {
		return errPolicyChanged
	}
	return err
}

func copyPolicyFile(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return copyRace(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if isSymlink(info) {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.Symlink(target, dst); err != nil {
			return err
		}
		got, err := os.Readlink(dst)
		if err != nil || got != target {
			os.Remove(dst)
			if err != nil {
				return err
			}
			return errPolicyChanged
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a file", src)
	}
	perm := info.Mode().Perm()
	in, err := os.Open(src)
	if err != nil {
		return copyRace(err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	cleanup := func() { out.Close(); os.Remove(dst) }
	if err := out.Chmod(perm); err != nil {
		cleanup()
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		cleanup()
		return err
	}
	if err := out.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}
	if afterCopyWrite != nil {
		afterCopyWrite(src, dst)
	}
	want, err := os.ReadFile(src)
	if err != nil {
		os.Remove(dst)
		return copyRace(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		os.Remove(dst)
		return err
	}
	if string(got) != string(want) {
		os.Remove(dst)
		return errPolicyChanged
	}
	return nil
}

func restorePolicy(backup, dest string) error {
	info, err := os.Lstat(backup)
	if err != nil {
		return err
	}
	if isSymlink(info) {
		target, err := os.Readlink(backup)
		if err != nil {
			return err
		}
		return os.Symlink(target, dest)
	}
	if err := os.Link(backup, dest); err == nil || !isCrossDevice(err) {
		return err
	}
	return copyPolicy(backup, dest)
}

func recoveryDiverged(backup, existing, linkTarget string) bool {
	info, err := os.Lstat(backup)
	if err != nil {
		return false
	}
	if isSymlink(info) {
		got, err := os.Readlink(backup)
		return err == nil && got != linkTarget
	}
	if !info.Mode().IsRegular() {
		return false
	}
	b, err := os.ReadFile(backup)
	return err == nil && string(b) != existing
}

func refuseChanged(dest, backup string, restoreErr error) (bool, error) {
	switch {
	case backup != "" && restoreErr != nil:
		return refuse("Refusing to publish policy changed during sync: %s (previous policy retained at %s; restore: %v)", dest, backup, restoreErr)
	case backup != "":
		return refuse("Refusing to publish policy changed during sync: %s (previous policy retained at %s)", dest, backup)
	default:
		return refuse("Refusing to publish policy changed during sync: %s", dest)
	}
}

func commitPublish(dest, text, body string, ledger map[string]string, ledgerFile, existing, linkTarget, backupRoot, preparedBackup string, before fs.FileInfo) (bool, error) {
	var backup string
	var restoreErr error
	err := writeStaged(dest, text, func(tmp string) (err error) {
		if err := policyUnchanged(dest, existing, linkTarget, before); err != nil {
			return err
		}
		if before != nil && before.Mode().IsRegular() && existing == text {
			return nil
		}
		if before != nil {
			backup = preparedBackup
			if backup == "" {
				var makeErr error
				backup, makeErr = makeRecovery(dest, backupRoot)
				if makeErr != nil {
					return makeErr
				}
			}
			copied := false
			if err := renamePolicy(dest, backup); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					os.RemoveAll(filepath.Dir(backup))
					backup = ""
					return errPolicyChanged
				}
				if !isCrossDevice(err) {
					os.RemoveAll(filepath.Dir(backup))
					backup = ""
					return err
				}
				if err := copyPolicy(dest, backup); err != nil {
					os.RemoveAll(filepath.Dir(backup))
					backup = ""
					return copyRace(err)
				}
				if err := policyContentUnchanged(backup, existing, linkTarget); err != nil {
					os.RemoveAll(filepath.Dir(backup))
					backup = ""
					return err
				}
				if err := policyUnchanged(dest, existing, linkTarget, before); err != nil {
					os.RemoveAll(filepath.Dir(backup))
					backup = ""
					return err
				}
				if err := os.Remove(dest); err != nil {
					os.RemoveAll(filepath.Dir(backup))
					backup = ""
					return err
				}
				copied = true
			}
			defer func() {
				if err != nil && backup != "" {
					restoreErr = restorePolicy(backup, dest)
				}
			}()
			if !copied {
				if err := policyUnchanged(backup, existing, linkTarget, before); err != nil {
					return err
				}
			}
		}
		if err := linkPolicy(tmp, dest); err != nil {
			if errors.Is(err, fs.ErrExist) {
				return errPolicyChanged
			}
			return err
		}
		return nil
	})
	if errors.Is(err, errPolicyChanged) {
		return refuseChanged(dest, backup, restoreErr)
	}
	if err != nil {
		if restoreErr != nil {
			return false, fmt.Errorf("restore %s from %s: %w (publish failed: %v)", dest, backup, restoreErr, err)
		}
		return false, err
	}
	if backup != "" {
		if recoveryDiverged(backup, existing, linkTarget) {
			fmt.Fprintf(os.Stderr, "Warning: live policy edit retained only at %s\n", backup)
		} else {
			fmt.Println("Previous policy retained at " + backup)
		}
	}
	ledger[dest] = digest(body)
	if err := writeLedger(ledgerFile, ledger); err != nil {
		return false, err
	}
	if err := pruneRecovery(recoveryDir(dest, backupRoot)); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: pruning recovery copies at %s: %v\n", recoveryDir(dest, backupRoot), err)
	}
	fmt.Println(dest)
	return true, nil
}

func publish(payload, dest string, ledger map[string]string, ledgerFile string, allowPrompt bool, backupRoot string) (bool, error) {
	var existing string
	exists, symlink := false, false
	info, err := os.Lstat(dest)
	switch {
	case err == nil && isSymlink(info):
		symlink = true
	case err == nil && info.Mode().IsRegular():
		b, err := os.ReadFile(dest)
		if err != nil {
			return false, err
		}
		existing, exists = string(b), true
	case err == nil:
		return false, fmt.Errorf("%s is not a file", dest)
	case !errors.Is(err, fs.ErrNotExist):
		return false, err
	}
	if symlink || !ours(existing, exists, payload, ledger[dest]) {
		if !allowPrompt || !stdinIsTerminal() {
			return refuse("Refusing to replace foreign file (no prompt): %s", dest)
		}
		backup := backupPath(dest, backupRoot)
		if !confirmReplace(dest, backup) {
			return false, nil
		}
		if symlink {
			if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
				return false, err
			}
			err = os.Rename(dest, backup)
		} else {
			err = copyFile(dest, backup)
		}
		if err != nil {
			return false, err
		}
		fmt.Println("Backed up to " + backup)
		fmt.Println(replaceWarning)
	}
	if err := writeAtomic(dest, payload); err != nil {
		return false, err
	}
	ledger[dest] = digest(payload)
	if err := writeLedger(ledgerFile, ledger); err != nil {
		return false, err
	}
	fmt.Println(dest)
	return true, nil
}

func publishPolicy(source, dest string, cursor bool, ledger map[string]string, ledgerFile string, allowPrompt bool, backupRoot string) (bool, error) {
	removeLegacyRecovery(dest)
	body := adapt(source, false)
	unmarked := adapt(source, cursor)
	var existing, linkTarget string
	exists, symlink := false, false
	info, err := os.Lstat(dest)
	switch {
	case err == nil && isSymlink(info):
		symlink = true
		linkTarget, err = os.Readlink(dest)
		if err != nil {
			return false, err
		}
	case err == nil && info.Mode().IsRegular():
		b, err := os.ReadFile(dest)
		if err != nil {
			return false, err
		}
		existing, exists = string(b), true
	case err == nil:
		return false, fmt.Errorf("%s is not a file", dest)
	case !errors.Is(err, fs.ErrNotExist):
		return false, err
	}
	if symlink {
		if !allowPrompt || !termReady() {
			return refuse("Refusing to replace foreign file (no prompt): %s", dest)
		}
		backup, err := makeRecovery(dest, backupRoot)
		if err != nil {
			return false, err
		}
		if !confirmReplace(dest, backup) {
			os.RemoveAll(filepath.Dir(backup))
			return false, nil
		}
		fmt.Println(replaceWarning)
		return commitPublish(dest, canonical(body, cursor), body, ledger, ledgerFile, existing, linkTarget, backupRoot, backup, info)
	}
	if !exists {
		return commitPublish(dest, canonical(body, cursor), body, ledger, ledgerFile, existing, linkTarget, backupRoot, "", info)
	}
	if next, ok := replaceBlock(existing, body); ok {
		return commitPublish(dest, next, body, ledger, ledgerFile, existing, linkTarget, backupRoot, "", info)
	}
	if ours(existing, true, unmarked, ledger[dest]) {
		return commitPublish(dest, canonical(body, cursor), body, ledger, ledgerFile, existing, linkTarget, backupRoot, "", info)
	}
	if !allowPrompt || !termReady() {
		return refuse("Refusing to replace foreign file (no prompt): %s", dest)
	}
	backup := backupPath(dest, backupRoot)
	if !confirmSplice(dest, backup) {
		return false, nil
	}
	info, err = os.Lstat(dest)
	if errors.Is(err, fs.ErrNotExist) || err == nil && !info.Mode().IsRegular() {
		return refuse("Refusing to splice policy changed during sync: %s", dest)
	}
	if err != nil {
		return false, err
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		return false, err
	}
	existing = string(b)
	if err := copyFile(dest, backup); err != nil {
		return false, err
	}
	fmt.Println("Backed up to " + backup)
	next, ok := replaceBlock(existing, body)
	if !ok {
		next = appendBlock(existing, body)
	}
	return commitPublish(dest, next, body, ledger, ledgerFile, existing, linkTarget, backupRoot, "", info)
}

func ensureGapHook(codexDir, policyPath, backupRoot string) (bool, error) {
	hooksPath := filepath.Join(codexDir, "hooks.json")
	text := "{\n  \"hooks\": {}\n}\n"
	existed := false
	info, err := os.Lstat(hooksPath)
	switch {
	case err == nil && isSymlink(info):
		return refuse("Refusing to edit hooks.json; it is a symlink: %s", hooksPath)
	case err == nil:
		b, err := os.ReadFile(hooksPath)
		if err != nil {
			return false, err
		}
		text, existed = string(b), true
	case !errors.Is(err, fs.ErrNotExist):
		return false, err
	}
	next, res := insertGapHook(text, policyPath)
	switch res {
	case hookPresent:
		return true, nil
	case hookMalformed:
		return refuse("Refusing to edit hooks.json; it is not JSON: %s", hooksPath)
	}
	if existed {
		backup := backupPath(hooksPath, backupRoot)
		if err := copyFile(hooksPath, backup); err != nil {
			return false, err
		}
		fmt.Println("Backed up to " + backup)
	}
	if err := writeAtomic(hooksPath, next); err != nil {
		return false, err
	}
	fmt.Println(hooksPath)
	fmt.Println(trustLine)
	return true, nil
}

func cleanupLegacyCodex(codexDir, policyPath, payload string, ledger map[string]string, backupRoot, ledgerFile string) (bool, error) {
	dest := filepath.Join(codexDir, "AGENTS.md")
	if policy, err := os.ReadFile(policyPath); err != nil || !matchesPublish(string(policy), payload) {
		return refuse("Refusing to strip unmarked Codex policy; the GAP policy file is missing or not this publish: %s", policyPath)
	}
	hooksPath := filepath.Join(codexDir, "hooks.json")
	info, err := os.Lstat(hooksPath)
	if err == nil && isSymlink(info) {
		return refuse("Refusing to strip unmarked Codex policy; hooks.json is a symlink: %s", hooksPath)
	}
	var hooksText []byte
	if err == nil {
		hooksText, err = os.ReadFile(hooksPath)
	}
	if err != nil {
		return refuse("Refusing to strip unmarked Codex policy; the GAP hook is not installed: %s", hooksPath)
	}
	if _, res := insertGapHook(string(hooksText), policyPath); res != hookPresent {
		return refuse("Refusing to strip unmarked Codex policy; the GAP hook is not installed: %s", hooksPath)
	}
	last, recorded := ledger[dest]
	if last == cleaned {
		fmt.Printf("Legacy GAP text already stripped from %s; nothing to strip\n", dest)
		return true, nil
	}
	info, err = os.Lstat(dest)
	if errors.Is(err, fs.ErrNotExist) {
		delete(ledger, dest)
		return true, writeLedger(ledgerFile, ledger)
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return refuse("Refusing to strip unmarked Codex policy; not a regular file: %s", dest)
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		return false, err
	}
	cut, ok := ledgerPrefixEnd(string(b), last)
	if !ok {
		why := "prefix does not match the ledger"
		if !recorded {
			why = "no ledger entry records the GAP text"
		}
		return refuse("Refusing to strip unmarked Codex policy; %s: %s", why, dest)
	}
	backup := backupPath(dest, backupRoot)
	if err := copyFile(dest, backup); err != nil {
		return false, err
	}
	if err := writeAtomic(dest, string(b[cut:])); err != nil {
		return false, err
	}
	ledger[dest] = cleaned
	if err := writeLedger(ledgerFile, ledger); err != nil {
		return false, err
	}
	fmt.Println("Backed up to " + backup)
	fmt.Println(dest)
	return true, nil
}

// afterCodexPublish installs the Codex hook only when this run wrote the policy file and it still matches.
func afterCodexPublish(wrote bool, policyPath, payload, codexDir, backupRoot string) (bool, error) {
	if !wrote {
		return true, nil
	}
	onDisk, err := os.ReadFile(policyPath)
	if err != nil {
		return refuse("Refusing to install the Codex hook; policy file could not be reread: %s", policyPath)
	}
	if !matchesPublish(string(onDisk), payload) {
		return refuse("Refusing to install the Codex hook; policy file does not match the payload: %s", policyPath)
	}
	return ensureGapHook(codexDir, policyPath, backupRoot)
}
