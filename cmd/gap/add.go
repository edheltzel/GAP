package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"charm.land/huh/v2"
)

var errAddCancelled = errors.New("installation cancelled")

var errPromptNeedsTTY = errors.New("Interactive prompt required but stdin is not a TTY. Nothing was installed. Use --agent <name> (or --agent '*') and -y to run non-interactively.")

type addOptions struct {
	Source                                                 string
	Global, Project, Yes, List, All, Copy, FullDepth, JSON bool
	Agents, Skills, Subagents, Explicit                    []string
	Rename, ID                                             string
}

type addFlag struct{ name, short, kind, value string }

var addFlags = []addFlag{
	{"global", "g", "bool", ""},
	{"project", "p", "bool", ""},
	{"local", "", "bool", ""},
	{"yes", "y", "bool", ""},
	{"list", "l", "bool", ""},
	{"all", "", "bool", ""},
	{"agent", "a", "list", "an agent name"},
	{"skill", "s", "list", "a skill name"},
	{"metadata", "", "string", "a JSON value"},
	{"full-depth", "", "bool", ""},
	{"json", "", "bool", ""},
	{"copy", "", "bool", ""},
	{"subagent", "", "list", "a subagent name"},
	{"rename", "", "string", "a skill name"},
	{"id", "", "string", "an id"},
}

func addFlagNamed(name string) (addFlag, bool) {
	i := slices.IndexFunc(addFlags, func(f addFlag) bool { return f.name == name })
	if i < 0 {
		return addFlag{}, false
	}
	return addFlags[i], true
}

func addFlagShort(short string) string {
	if i := slices.IndexFunc(addFlags, func(f addFlag) bool { return f.short == short }); i >= 0 {
		return addFlags[i].name
	}
	return short
}

type argToken struct {
	option, terminator bool
	name, raw, value   string
	hasValue, inline   bool
}

func addTokens(args []string) []argToken {
	takesValue := func(name string) bool {
		f, ok := addFlagNamed(name)
		return ok && f.kind != "bool"
	}
	var tokens []argToken
	rest := slices.Clone(args)
	for len(rest) > 0 {
		arg := rest[0]
		rest = rest[1:]
		lone := func(name string) {
			t := argToken{option: true, name: name, raw: arg}
			if takesValue(name) && len(rest) > 0 {
				t.value, t.hasValue, rest = rest[0], true, rest[1:]
			}
			tokens = append(tokens, t)
		}
		short := len(arg) > 1 && arg[0] == '-' && arg[1] != '-'
		switch {
		case arg == "--":
			tokens = append(tokens, argToken{terminator: true})
			for _, p := range rest {
				tokens = append(tokens, argToken{value: p})
			}
			rest = nil
		case short && len(arg) == 2:
			lone(addFlagShort(arg[1:]))
		case short && !takesValue(addFlagShort(arg[1:2])):
			var expanded []string
			for i := 1; i < len(arg); i++ {
				if !takesValue(addFlagShort(arg[i:i+1])) || i == len(arg)-1 {
					expanded = append(expanded, "-"+arg[i:i+1])
					continue
				}
				expanded = append(expanded, "-"+arg[i:])
				break
			}
			rest = append(expanded, rest...)
		case short:
			tokens = append(tokens, argToken{option: true, name: addFlagShort(arg[1:2]), raw: arg[:2], value: arg[2:], hasValue: true, inline: true})
		case len(arg) > 2 && strings.HasPrefix(arg, "--") && !strings.Contains(arg[3:], "="):
			lone(arg[2:])
		case len(arg) > 2 && strings.HasPrefix(arg, "--"):
			i := strings.Index(arg, "=")
			tokens = append(tokens, argToken{option: true, name: arg[2:i], raw: "--" + arg[2:i], value: arg[i+1:], hasValue: true, inline: true})
		default:
			tokens = append(tokens, argToken{value: arg})
		}
	}
	return tokens
}

func parseAddArgs(args []string) (addOptions, error) {
	var opts addOptions
	var errs []error
	var positionals []string
	values := map[string][]string{}
	tokens := addTokens(args)
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if t.terminator {
			continue
		}
		if !t.option {
			if t.value != "" {
				positionals = append(positionals, t.value)
			}
			continue
		}
		f, known := addFlagNamed(t.name)
		if !known {
			continue
		}
		if f.kind == "bool" {
			if t.hasValue {
				errs = append(errs, fmt.Errorf("%s does not take a value", t.raw))
				continue
			}
			switch f.name {
			case "global":
				opts.Global = true
			case "project", "local":
				opts.Project = true
			case "yes":
				opts.Yes = true
			case "list":
				opts.List = true
			case "all":
				opts.All = true
			case "full-depth":
				opts.FullDepth = true
			case "json":
				opts.JSON = true
			case "copy":
				opts.Copy = true
			}
			continue
		}
		value := t.value
		if t.inline && !strings.HasPrefix(t.raw, "--") {
			value = strings.TrimPrefix(value, "=")
		}
		if value == "" || !t.inline && strings.HasPrefix(value, "-") {
			errs = append(errs, fmt.Errorf("%s requires %s", t.raw, f.value))
			continue
		}
		values[f.name] = append(values[f.name], value)
		if f.kind == "list" && !t.inline {
			for i+1 < len(tokens) && !tokens[i+1].option && !tokens[i+1].terminator {
				i++
				if tokens[i].value != "" {
					values[f.name] = append(values[f.name], tokens[i].value)
				}
			}
		}
	}
	opts.Agents, opts.Skills, opts.Subagents = values["agent"], values["skill"], values["subagent"]
	if v := values["rename"]; len(v) > 0 {
		if name := v[len(v)-1]; sanitizeName(name) != name {
			errs = append(errs, fmt.Errorf(`--rename must be a skill name of lowercase letters, digits, ".", "_" and "-" (e.g. %s)`, sanitizeName(name)))
		} else {
			opts.Rename = name
		}
	}
	if v := values["metadata"]; len(v) > 0 && !json.Valid([]byte(v[len(v)-1])) {
		errs = append(errs, errors.New("--metadata must be valid JSON"))
	}
	if v := values["id"]; len(v) > 0 {
		opts.ID = v[len(v)-1]
	}
	if len(positionals) > 0 {
		opts.Source = positionals[0]
	}
	if opts.Global && opts.Project {
		errs = append(errs, errors.New("--global and --project cannot be used together"))
	}
	if !slices.Contains(opts.Agents, "*") {
		opts.Explicit = slices.Clone(opts.Agents)
	}
	if opts.All {
		opts.Skills, opts.Agents, opts.Yes = []string{"*"}, []string{"*"}, true
	}
	return opts, errors.Join(errs...)
}

func runAdd(args []string) error {
	opts, parseErr := parseAddArgs(args)
	if !opts.JSON {
		if parseErr != nil {
			return parseErr
		}
		_, err := addSkills(opts)
		return err
	}
	stdout := os.Stdout
	os.Stdout = os.Stderr
	results, err := []any{}, parseErr
	if err == nil {
		results, err = addSkills(opts)
	}
	os.Stdout = stdout
	if err != nil && parseErr == nil && !errors.Is(err, errSkipped) && len(results) == 0 {
		results = append(results, addFailed{Status: "failed", Error: err.Error()})
	}
	b, jerr := json.MarshalIndent(results, "", "  ")
	if jerr != nil {
		return jerr
	}
	fmt.Fprintln(stdout, string(b))
	return err
}

type addInstalled struct {
	Name     string    `json:"name"`
	Status   string    `json:"status"`
	Source   string    `json:"source"`
	Ref      *string   `json:"ref"`
	Hash     *string   `json:"hash"`
	Path     string    `json:"path"`
	Scope    string    `json:"scope"`
	Agents   []string  `json:"agents"`
	Mode     string    `json:"mode"`
	Security *struct{} `json:"security"`
}

type addSkipped struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type addFailed struct {
	Name   string `json:"name,omitempty"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

type addTarget struct{ agent, subagent string }

func (t addTarget) display() string {
	if t.subagent != "" {
		return agentDisplay(t.agent) + " (" + t.subagent + ")"
	}
	return agentDisplay(t.agent)
}

func addTargets(agents, subagents []string) []addTarget {
	var targets []addTarget
	for _, id := range agents {
		if id != "eve" {
			targets = append(targets, addTarget{agent: id})
			continue
		}
		for _, sub := range eveSubagents(subagents) {
			targets = append(targets, addTarget{agent: id, subagent: sub})
		}
	}
	return targets
}

func shortenPath(path, cwd string) string {
	home, _ := os.UserHomeDir()
	sep := string(filepath.Separator)
	if home != "" && (path == home || strings.HasPrefix(path, home+sep)) {
		return "~" + path[len(home):]
	}
	if path == cwd || strings.HasPrefix(path, cwd+sep) {
		return "." + path[len(cwd):]
	}
	return path
}

func formatList(items []string, limit int) string {
	if len(items) <= limit {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s +%d more", strings.Join(items[:limit], ", "), len(items)-limit)
}

func kebabTitle(s string) string {
	words := strings.Split(s, "-")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

type promptChoice struct{ label, value, hint string }

func runPrompt(field huh.Field) error {
	if !termReady() {
		return errPromptNeedsTTY
	}
	err := huh.NewForm(huh.NewGroup(field)).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return errAddCancelled
	}
	return err
}

func promptMany(title, description string, choices []promptChoice, initial []string, required bool) ([]string, error) {
	options := make([]huh.Option[string], len(choices))
	for i, c := range choices {
		label := c.label
		if c.hint != "" {
			label += " (" + c.hint + ")"
		}
		options[i] = huh.NewOption(label, c.value).Selected(slices.Contains(initial, c.value))
	}
	var picked []string
	field := huh.NewMultiSelect[string]().Title(title).Description(description).Options(options...).Value(&picked).Filterable(true)
	if required {
		field = field.Validate(func(v []string) error {
			if len(v) == 0 {
				return errors.New("select at least one")
			}
			return nil
		})
	}
	err := runPrompt(field)
	return picked, err
}

func promptOne(title string, choices []promptChoice) (string, error) {
	options := make([]huh.Option[string], len(choices))
	for i, c := range choices {
		options[i] = huh.NewOption(c.label+"  "+c.hint, c.value)
	}
	var picked string
	err := runPrompt(huh.NewSelect[string]().Title(title).Options(options...).Value(&picked))
	return picked, err
}

func promptConfirm(title string, initial bool) (bool, error) {
	answer := initial
	err := runPrompt(huh.NewConfirm().Title(title).Value(&answer))
	return answer, err
}

func printMissingSource(out *report) {
	out.println()
	out.println(out.tone(" ERROR ", colorBad) + " " + out.tone("Missing required argument: source", colorBad))
	out.println()
	out.println(out.tone("  Usage:", colorDim))
	out.println("    " + out.tone("gap add", colorAccent) + " <source> " + out.tone("[options]", colorDim))
	out.println()
	out.println(out.tone("  Example:", colorDim))
	out.println("    " + out.tone("gap add", colorAccent) + " vercel-labs/agent-skills")
	out.println()
}

type localSource struct{ root, sub, ref, commit string }

func resolveLocalSource(path string) (localSource, error) {
	if !fileExists(path) {
		return localSource{}, fmt.Errorf("Local path does not exist: %s", path)
	}
	top, err := git(path, "rev-parse", "--show-toplevel")
	if err != nil {
		return localSource{}, fmt.Errorf("%s is not inside a git repository; commit it to git first, since gap add pins a commit", path)
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return localSource{}, err
	}
	sub, err := filepath.Rel(top, real)
	if err != nil {
		return localSource{}, err
	}
	l := localSource{root: top, sub: filepath.ToSlash(sub)}
	if l.sub == "." {
		l.sub, l.root = "", path
	} else if trimmed, ok := strings.CutSuffix(path, string(filepath.Separator)+sub); ok {
		l.root = trimmed
	}
	if l.commit, err = git(top, "rev-parse", "HEAD"); err != nil {
		return localSource{}, fmt.Errorf("%s has no commit yet; commit it to git first, since gap add pins a commit", path)
	}
	if l.ref, err = git(top, "symbolic-ref", "--short", "HEAD"); err != nil {
		l.ref = l.commit
	}
	if status, _ := git(top, "status", "--porcelain", "--", cmp.Or(l.sub, ".")); status != "" {
		fmt.Printf("Note: %s has uncommitted changes; gap installs commit %s\n", path, l.commit[:7])
	}
	return l, nil
}

func resolveRemote(url, ref string) (string, string, string, error) {
	resolve := func(url string) (string, string, error) {
		r := ref
		if r == "" {
			var err error
			if r, err = defaultBranch(url); err != nil {
				return "", "", err
			}
		}
		commit, err := resolveRef(url, r)
		return r, commit, err
	}
	r, commit, err := resolve(url)
	if err == nil {
		return url, r, commit, nil
	}
	ssh, host := githubSSHFallback(url)
	if ssh == "" || !gitAuthFailure(err.Error()) {
		return "", "", "", err
	}
	if os.Getenv("GIT_SSH_COMMAND") == "" {
		os.Setenv("GIT_SSH_COMMAND", "ssh -o BatchMode=yes")
		defer os.Unsetenv("GIT_SSH_COMMAND")
	}
	if r, commit, err := resolve(ssh); err == nil {
		return ssh, r, commit, nil
	}
	return "", "", "", fmt.Errorf("Authentication failed for %s.\n  - For private repos, ensure you have access\n  - Retry with SSH: gap add %s\n  - Check access with: gh auth status -h %s or ssh -T git@%s", url, ssh, host, host)
}

func eveSubagentDirs(root string) []string {
	entries, err := os.ReadDir(filepath.Join(root, "agent", "subagents"))
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

type addSkill struct {
	discoveredSkill
	install string
}

type addRun struct {
	opts                             addOptions
	out                              *report
	results                          []any
	home, repoRoot, projectRoot, cwd string
	hostName, scope                  string
	src                              skillSource
	explicitRef                      string
	id, repo, url, ref, commit       string
	store                            string
	pin, prev                        lockEntryM1
	d                                distro
	lk                               lockFile
	readd                            bool
	manifest                         *utilityManifest
	skills                           []discoveredSkill
	chosen                           []addSkill
	agents, subagents, skipped       []string
	explicit                         map[string]bool
	copyMode                         bool
	state                            stateFile
	env                              agentEnv
}

func addSkills(opts addOptions) ([]any, error) {
	r := &addRun{opts: opts, out: newReport(os.Stdout), results: []any{}, explicit: map[string]bool{}}
	if opts.Source == "" {
		printMissingSource(r.out)
		if opts.JSON {
			return r.results, errors.New("Missing required argument: source")
		}
		return r.results, errSkipped
	}
	for _, step := range []func() error{r.start, r.resolve, r.discover} {
		if err := step(); err != nil {
			return r.results, err
		}
	}
	if opts.List {
		printAvailableSkills(r.out, r.skills)
		return r.results, nil
	}
	for _, step := range []func() error{r.chooseSkills, r.chooseAgents, r.guard, r.confirm, r.persist} {
		if err := step(); err != nil {
			return r.results, err
		}
	}
	err := r.install()
	return r.results, err
}

func (r *addRun) start() error {
	opts, out := &r.opts, r.out
	name, id := hostAgent(os.Getenv)
	r.hostName = name
	if name != "" {
		opts.Yes = true
		if len(opts.Agents) == 0 && id != "" {
			opts.Agents = ensureUniversalAgents([]string{id})
		}
	}
	if opts.JSON && !opts.Yes {
		return errors.New("The --json flag requires --yes (or --all) to run non-interactively.")
	}
	if opts.JSON && opts.List {
		return errors.New("The --json flag cannot be combined with --list.")
	}
	if name != "" {
		out.println(out.bold(" "+name+" ", colorAccent) + " Agent detected — installing non-interactively")
	} else if !termReady() {
		out.println(out.tone("Tip: use the --yes (-y) and --global (-g) flags to install without prompts.", colorDim))
	}
	if len(opts.Agents) > 0 && !slices.Contains(opts.Agents, "*") {
		invalid := slices.DeleteFunc(slices.Clone(opts.Agents), func(id string) bool {
			_, known := agentByID(id)
			return known
		})
		if len(invalid) > 0 {
			return fmt.Errorf("Invalid agents: %s\nValid agents: %s", strings.Join(invalid, ", "), strings.Join(agentIDs(), ", "))
		}
	}
	var err error
	if r.cwd, err = os.Getwd(); err != nil {
		return err
	}
	if r.src, err = parseSkillSource(opts.Source, r.cwd); err != nil {
		return err
	}
	line := "Source: " + cmp.Or(r.src.Local, r.src.URL)
	if r.src.Ref != "" {
		line += " @ " + out.tone(r.src.Ref, colorWarn)
	}
	if r.src.Subpath != "" {
		line += " (" + r.src.Subpath + ")"
	}
	if r.src.Skill != "" {
		line += " " + out.tone("@", colorDim) + out.tone(r.src.Skill, colorAccent)
	}
	out.println(line)
	if r.src.Skill != "" {
		opts.Skills = appendMissing(opts.Skills, r.src.Skill)
	}
	if r.home, r.repoRoot, err = gapHomes(); err != nil {
		return err
	}
	if r.projectRoot, err = invokingProjectRoot(); err != nil {
		return err
	}
	switch {
	case opts.Global:
		r.scope = "global"
	case opts.Project:
		r.scope = "project"
	}
	markExplicit(r.explicit, opts.Explicit)
	return nil
}

func sameRepo(recorded, repo string) bool {
	if recorded == repo {
		return true
	}
	url, err := cloneURL(repo)
	if err != nil {
		return false
	}
	ssh, _ := githubSSHFallback(url)
	return ssh != "" && ssh == recorded
}

func (r *addRun) resolve() error {
	var err error
	r.explicitRef, r.repo, r.ref = r.src.Ref, sourceRepo(r.src), r.src.Ref
	if r.src.Kind == "local" {
		local, err := resolveLocalSource(r.src.Local)
		if err != nil {
			return err
		}
		r.repo, r.url, r.ref, r.commit, r.src.Subpath, r.explicitRef = local.root, local.root, local.ref, local.commit, local.sub, ""
	} else if r.url, err = cloneURL(r.repo); err != nil {
		return err
	}
	r.id = cmp.Or(r.opts.ID, sourceID(r.repo))
	if !validID(r.id) {
		return fmt.Errorf("id %q must use lowercase letters, digits, dot, dash, or underscore; policy and manifest are reserved; pass --id", r.id)
	}
	if r.d, r.lk, err = registeredPins(r.repoRoot, r.home); err != nil {
		return err
	}
	_, r.readd = r.d.Components[r.id]
	if !r.opts.List {
		if err := r.checkExisting(); err != nil {
			return err
		}
	}
	r.prev = r.lk.Components[r.id]
	existing := r.d.Components[r.id]
	if !r.opts.List && r.readd && existing.ProjectOnly && r.prev.Commit != "" {
		if r.prev.Ref != existing.Ref {
			return fmt.Errorf("utility %s tracks %s but is pinned on %s; run gap update %s first", r.id, existing.Ref, r.prev.Ref, r.id)
		}
		r.ref, r.commit = existing.Ref, r.prev.Commit
	} else if r.src.Kind != "local" {
		if r.url, r.ref, r.commit, err = resolveRemote(r.url, r.ref); err != nil {
			return err
		}
		if strings.HasPrefix(r.url, "git@") {
			r.repo = r.url
		}
	}
	c := component{ID: r.id, Repo: r.repo, Ref: r.ref, User: true}
	if r.pin, err = pinCommit(r.home, c, r.ref, r.commit, r.prev); err != nil {
		return err
	}
	r.store, err = ensureStore(r.home, c, r.pin)
	return err
}

func (r *addRun) checkExisting() error {
	if !r.readd {
		text, err := readOptional(userConfigFile(r.home))
		if err != nil {
			return err
		}
		if cc, declared, err := readComponentConfig(text, r.id); err != nil {
			return err
		} else if declared && cc.Repo == "" {
			return fmt.Errorf("%s already declares components.%s; pass --id to use another id", userConfigFile(r.home), tomlKey(r.id))
		}
		return nil
	}
	e := r.d.Components[r.id]
	if !e.User {
		return fmt.Errorf("%s is already a distro component; pass --id to add this repo under another id", r.id)
	}
	existingScope := "global"
	if e.ProjectOnly {
		existingScope = "project"
	}
	switch {
	case !sameRepo(e.Repo, r.repo):
		return fmt.Errorf("utility %s is already added from %s; pass --id to add %s under another id", r.id, e.Repo, r.opts.Source)
	case e.SkillPath != cleanSubpath(r.src.Subpath):
		return fmt.Errorf("utility %s is already added with path %s; pass --id to add another path", r.id, cmp.Or(e.SkillPath, "."))
	case r.explicitRef != "" && r.explicitRef != e.Ref:
		return fmt.Errorf("utility %s tracks %s; run gap update %s, or pass --id to add %s separately", r.id, e.Ref, r.id, r.opts.Source)
	case r.scope != "" && r.scope != existingScope:
		return scopeMismatch(r.id, e.ProjectOnly)
	}
	r.scope, r.repo = existingScope, e.Repo
	if r.src.Kind != "local" {
		r.ref = e.Ref
		r.url, _ = cloneURL(e.Repo)
	}
	return nil
}

func scopeMismatch(id string, projectOnly bool) error {
	if projectOnly {
		return fmt.Errorf("utility %s is added per project; pass --id to add it globally as well", id)
	}
	return fmt.Errorf("utility %s is already added globally; pass --id to add it per project as well", id)
}

func (r *addRun) discover() error {
	warn := func(s string) { fmt.Fprintln(os.Stderr, s) }
	if hasUtilityManifest(r.store) {
		m, err := readUtilityManifest(r.store)
		if err != nil {
			return fmt.Errorf("%s: %w", r.repo, err)
		}
		r.manifest = &m
		for _, name := range m.Skills {
			dir := entrySkillDir(manifestEntry{Path: r.store, SkillPath: cleanSubpath(r.src.Subpath)}, name)
			if dir == "" {
				continue
			}
			s := discoveredSkill{Name: name, Dir: dir}
			if b, err := os.ReadFile(filepath.Join(dir, "SKILL.md")); err == nil {
				if data, _, err := parseFrontmatter(string(b)); err == nil {
					if description, ok := data["description"].(string); ok {
						s.Description = sanitizeMetadata(description)
					}
				}
			}
			r.skills = append(r.skills, s)
		}
	} else {
		includeInternal := len(r.opts.Skills) > 0 && !slices.Contains(r.opts.Skills, "*")
		var err error
		if r.skills, err = discoverSkills(r.store, r.src.Subpath, r.opts.FullDepth, includeInternal, warn); err != nil {
			return err
		}
		missing := slices.ContainsFunc(r.opts.Skills, func(name string) bool {
			return name != "*" && len(filterSkills(r.skills, []string{name})) == 0
		})
		if missing && !r.opts.FullDepth {
			if r.skills, err = discoverSkills(r.store, r.src.Subpath, true, includeInternal, warn); err != nil {
				return err
			}
		}
		if len(r.skills) == 0 {
			return errors.New("No valid skills found. Skills require a SKILL.md with name and description.")
		}
	}
	r.out.println("Found " + plural(len(r.skills), "skill"))
	return nil
}

func (r *addRun) chooseSkills() error {
	opts, out := r.opts, r.out
	var selected []discoveredSkill
	switch {
	case len(r.skills) == 0 && len(opts.Skills) == 0:
	case slices.Contains(opts.Skills, "*"):
		selected = r.skills
		logChosenSkills(out, selected)
	case len(opts.Skills) > 0:
		selected = filterSkills(r.skills, opts.Skills)
		if opts.JSON {
			for _, requested := range opts.Skills {
				if len(filterSkills(r.skills, []string{requested})) == 0 {
					r.results = append(r.results, addSkipped{Name: requested, Status: "skipped", Reason: "No matching skill found in source"})
				}
			}
		}
		if len(selected) == 0 {
			var names []string
			for _, s := range r.skills {
				names = append(names, "  - "+s.Name)
			}
			return fmt.Errorf("No matching skills found for: %s\nAvailable skills:\n%s", strings.Join(opts.Skills, ", "), strings.Join(names, "\n"))
		}
		var names []string
		for _, s := range selected {
			names = append(names, out.tone(s.Name, colorAccent))
		}
		out.println(fmt.Sprintf("Selected %s: %s", plural(len(selected), "skill"), strings.Join(names, ", ")))
	case len(r.skills) == 1 || opts.Yes:
		selected = r.skills
		logChosenSkills(out, selected)
	default:
		var choices []promptChoice
		for _, s := range r.skills {
			hint := s.Description
			if runes := []rune(hint); len(runes) > 60 {
				hint = string(runes[:57]) + "…"
			}
			choices = append(choices, promptChoice{label: s.Name, value: s.Dir, hint: hint})
		}
		picked, err := promptMany("Select skills to install", "", choices, nil, true)
		if err != nil {
			return err
		}
		selected = slices.DeleteFunc(slices.Clone(r.skills), func(s discoveredSkill) bool { return !slices.Contains(picked, s.Dir) })
	}
	if opts.Rename != "" && len(selected) != 1 {
		return fmt.Errorf("--rename installs one skill under a new name, but %d skills were selected.\nPick one skill with --skill <name> or <source>@<skill>.", len(selected))
	}
	for _, s := range selected {
		name := s.Name
		if r.manifest == nil {
			name = sanitizeName(s.Name)
		}
		install := cmp.Or(opts.Rename, name)
		if i := slices.IndexFunc(r.chosen, func(c addSkill) bool { return c.install == install }); i >= 0 {
			return fmt.Errorf("skills %q and %q both install as %s; select one of them, or install one under another name with --rename", r.chosen[i].Name, s.Name, install)
		}
		r.chosen = append(r.chosen, addSkill{discoveredSkill: s, install: install})
	}
	return nil
}

func (r *addRun) chooseAgents() error {
	opts, out := r.opts, r.out
	r.env = currentAgentEnv()
	r.env.Cwd = r.projectRoot
	var err error
	if r.state, err = readState(r.home); err != nil {
		return err
	}
	if len(r.chosen) == 0 {
		return r.settleScope(false)
	}
	switch {
	case slices.Contains(opts.Agents, "*"):
		r.agents = agentIDs()
		out.println(fmt.Sprintf("Installing to all %d agents", len(r.agents)))
	case len(opts.Agents) > 0:
		r.agents = opts.Agents
	default:
		if err := r.detectAgents(); err != nil {
			return err
		}
	}
	if err := r.chooseSubagents(); err != nil {
		return err
	}
	supportsGlobal := slices.ContainsFunc(r.agents, func(id string) bool {
		a, _ := agentByID(id)
		return a.GlobalDir != nil
	})
	if err := r.settleScope(supportsGlobal); err != nil {
		return err
	}
	global := r.scope == "global"
	if global {
		var skipped []string
		for _, id := range r.agents {
			if a, _ := agentByID(id); a.GlobalDir == nil {
				skipped = append(skipped, id)
			}
		}
		named := len(opts.Agents) > 0 && !slices.Contains(opts.Agents, "*")
		if !named && len(skipped) > 0 && len(skipped) < len(r.agents) {
			var names []string
			for _, id := range skipped {
				names = append(names, agentDisplay(id))
			}
			out.println(fmt.Sprintf("Skipping %s: no global skills directory", strings.Join(names, ", ")))
			r.agents = slices.DeleteFunc(r.agents, func(id string) bool { return slices.Contains(skipped, id) })
		} else if len(skipped) > 0 {
			return fmt.Errorf("%s does not support global skill installation", agentDisplay(skipped[0]))
		}
	}
	var recorded componentConfig
	if r.readd {
		if recorded, err = r.recorded(); err != nil {
			return err
		}
		markExplicit(r.explicit, recorded.Agents)
		r.subagents = mergeSubagents(recorded.Agents, recorded.Subagents, r.agents, r.subagents)
		r.agents = appendMissing(slices.Clone(recorded.Agents), r.agents...)
	}
	r.copyMode = opts.Copy || defaultCopyMode(r.agents, r.subagents)
	if !r.copyMode && !opts.Yes {
		picked, err := promptOne("Installation method", []promptChoice{
			{label: "Symlink (Recommended)", value: "symlink", hint: "Single source of truth, easy updates"},
			{label: "Copy to all agents", value: "copy", hint: "Independent copies for each agent"},
		})
		if err != nil {
			return err
		}
		r.copyMode = picked == "copy"
	} else if !r.copyMode {
		r.copyMode = recorded.Copy
	}
	if !global && !r.copyMode {
		for _, id := range r.agents {
			a, _ := agentByID(id)
			first, _, _ := strings.Cut(a.Dir, "/")
			if !isUniversal(id) && !r.explicit[id] && !a.MakeProjectDir && !fileExists(filepath.Join(r.projectRoot, first)) {
				r.skipped = append(r.skipped, id)
			}
		}
		r.agents = slices.DeleteFunc(r.agents, func(id string) bool { return slices.Contains(r.skipped, id) })
	}
	if len(r.agents) == 0 {
		return errors.New("no agents left to install to; pass --agent")
	}
	return nil
}

func (r *addRun) recorded() (componentConfig, error) {
	path := userConfigFile(r.home)
	if r.scope == "project" {
		path = projectConfigFile(r.projectRoot)
	}
	text, err := readOptional(path)
	if err != nil {
		return componentConfig{}, err
	}
	cc, _, err := readComponentConfig(text, r.id)
	return cc, err
}

func (r *addRun) detectAgents() error {
	opts, out := r.opts, r.out
	detected := detectInstalledAgents(r.env)
	var err error
	switch {
	case slices.Contains(detected, "eve") && (opts.Yes || r.hostName == ""):
		useEve := opts.Yes
		if !opts.Yes {
			var labels []string
			for _, c := range r.chosen {
				labels = append(labels, c.install)
			}
			if useEve, err = promptConfirm(fmt.Sprintf("Detected an eve project. Install %s for your eve agent to use?", formatList(labels, 3)), true); err != nil {
				return err
			}
		}
		if !useEve {
			r.agents, err = selectAgents(&r.state, r.scope == "global")
			markExplicit(r.explicit, r.agents)
			return err
		}
		r.agents = []string{"eve"}
		if !opts.Yes {
			r.explicit["eve"] = true
		}
		out.println("Installing to: eve agent")
	case len(detected) == 0 && opts.Yes:
		r.agents = agentIDs()
		out.println("Installing to all agents")
	case len(detected) == 0:
		out.println("Select agents to install skills to")
		r.agents, err = promptAgents(&r.state)
		markExplicit(r.explicit, r.agents)
	case len(detected) == 1 || opts.Yes:
		r.agents = ensureUniversalAgents(detected)
		var names []string
		for _, id := range detected {
			names = append(names, out.tone(agentDisplay(id), colorAccent))
		}
		out.println("Installing to: " + strings.Join(names, ", "))
	default:
		r.agents, err = selectAgents(&r.state, r.scope == "global")
		markExplicit(r.explicit, r.agents)
	}
	return err
}

func (r *addRun) chooseSubagents() error {
	if len(r.opts.Subagents) > 0 {
		r.explicit["eve"] = true
		r.agents = appendMissing(r.agents, "eve")
	}
	if !slices.Contains(r.agents, "eve") {
		return nil
	}
	switch available := eveSubagentDirs(r.projectRoot); {
	case len(r.opts.Subagents) > 0:
		for _, s := range r.opts.Subagents {
			if s == "root" || s == "." {
				s = ""
			}
			r.subagents = appendMissing(r.subagents, s)
		}
	case len(available) > 0 && !r.opts.Yes:
		choices := []promptChoice{{label: "Root agent", value: "", hint: "agent/skills"}}
		for _, name := range available {
			choices = append(choices, promptChoice{label: name, value: name, hint: "agent/subagents/" + name + "/skills"})
		}
		var err error
		if r.subagents, err = promptMany("Where should Eve skills be installed?", "", choices, []string{""}, true); err != nil {
			return err
		}
	}
	if len(r.subagents) == 1 && r.subagents[0] == "" {
		r.subagents = nil
	}
	return nil
}

func mergeSubagents(recordedAgents, recorded, agents, subagents []string) []string {
	var out []string
	if slices.Contains(recordedAgents, "eve") {
		out = appendMissing(out, eveSubagents(recorded)...)
	}
	if slices.Contains(agents, "eve") {
		out = appendMissing(out, eveSubagents(subagents)...)
	}
	if len(out) == 1 && out[0] == "" {
		return nil
	}
	return out
}

func (r *addRun) settleScope(supportsGlobal bool) error {
	if r.scope == "" {
		switch {
		case len(r.chosen) == 0:
			r.scope = "global"
		case r.opts.Yes || !supportsGlobal:
			r.scope = "project"
		default:
			picked, err := promptOne("Installation scope", []promptChoice{
				{label: "Project", value: "project", hint: "Install in current directory (committed with your project)"},
				{label: "Global", value: "global", hint: "Install in home directory (available across all projects)"},
			})
			if err != nil {
				return err
			}
			r.scope = picked
		}
	}
	if r.scope == "project" && filepath.Clean(r.projectRoot) == filepath.Clean(r.home) {
		return errors.New("gap add installs into the current project by default; run it inside a project, or pass -g to install globally")
	}
	return nil
}

func (r *addRun) root() string {
	if r.scope == "global" {
		return r.home
	}
	return r.projectRoot
}

func (r *addRun) guard() error {
	global := r.scope == "global"
	if err := loadLocalManifests(r.home, r.d, r.lk); err != nil {
		return err
	}
	for _, c := range r.chosen {
		owner := ""
		for _, oid := range slices.Sorted(maps.Keys(r.state.Components)) {
			entry := r.state.Components[oid]
			if oid != r.id && owner == "" && (global && slices.Contains(entry.Skills, c.install) || !global && slices.Contains(entry.ProjectSkills[r.root()], c.install)) {
				owner = oid
			}
		}
		for _, oid := range sortedIDs(r.d.Components) {
			other := r.d.Components[oid]
			if owner == "" && global && oid != r.id && !other.Disabled && !other.ProjectOnly && slices.Contains(other.Skills, c.install) {
				owner = oid
			}
		}
		if owner != "" {
			return fmt.Errorf("skill %s is already installed by %s; pass --rename to install it under another name, or run gap remove %s", c.install, owner, owner)
		}
	}
	return nil
}

func (r *addRun) confirm() error {
	if len(r.chosen) == 0 {
		return nil
	}
	out, global, root := r.out, r.scope == "global", r.root()
	targets := addTargets(r.agents, r.subagents)
	preview := manifestEntry{Agents: r.agents, Copy: r.copyMode, Subagents: r.subagents}
	method := "symlink →"
	if r.copyMode {
		method = "copy →"
	}
	var summary []string
	for _, c := range r.chosen {
		canonical := filepath.Join(root, ".agents", "skills", c.install)
		if len(targets) == 1 && targets[0].agent == "eve" {
			canonical = filepath.Join(agentBaseDir("eve", global, root, targets[0].subagent, r.env), c.install)
		}
		if len(summary) > 0 {
			summary = append(summary, "")
		}
		summary = append(summary, out.tone(shortenPath(canonical, r.cwd), colorAccent))
		var universal, linked, subs, overwrites []string
		for _, t := range targets {
			switch {
			case t.subagent != "":
				subs = append(subs, t.display())
			case usesCanonical(t.agent, global):
				universal = append(universal, t.display())
			default:
				linked = append(linked, t.display())
			}
			if fileExists(filepath.Join(agentBaseDir(t.agent, global, root, t.subagent, r.env), c.install)) {
				overwrites = append(overwrites, t.display())
			}
		}
		for _, line := range []struct {
			label, tone string
			names       []string
		}{{"universal:", colorOK, universal}, {method, colorDim, linked}, {"copy →", colorDim, subs}, {"overwrites:", colorWarn, overwrites}} {
			if len(line.names) > 0 {
				summary = append(summary, "  "+out.tone(line.label, line.tone)+" "+formatList(line.names, 5))
			}
		}
		for _, dest := range skillDests(root, global, c.install, preview, r.env) {
			if names := categoryConflict(dest.Dir); len(names) > 0 {
				shown := strings.Join(names[:min(3, len(names))], ", ")
				if len(names) > 3 {
					shown += ", …"
				}
				summary = append(summary, "  "+out.tone("conflicts with category:", colorBad)+" "+out.tone("("+filepath.Base(dest.Dir)+"/ contains "+shown+")", colorDim))
			}
		}
	}
	out.println()
	out.heading("Installation Summary")
	for _, line := range summary {
		out.println("  " + line)
	}
	out.println()
	if r.opts.Yes {
		return nil
	}
	proceed, err := promptConfirm("Proceed with installation?", true)
	if err != nil {
		return err
	}
	if !proceed {
		return errAddCancelled
	}
	return nil
}

func (r *addRun) persist() error {
	global := r.scope == "global"
	if r.manifest != nil {
		m := *r.manifest
		r.out.section(r.id, describeUtility(m)+pinLabel(r.pin))
		if !global && (len(m.Commands) > 0 || len(m.MCP) > 0 || m.Hooks.Install != "" || m.Hooks.Update != "" || m.Hooks.Uninstall != "") {
			r.out.println("Its skills install into " + r.projectRoot + "; its commands, hooks, and MCP servers are machine-wide.")
		}
	}
	sel := addSelection{agents: r.agents, copy: r.copyMode, subagents: r.subagents, from: map[string]string{}}
	skillSet := map[string]skillSpec{}
	for _, c := range r.chosen {
		sel.names = append(sel.names, c.install)
		if c.install != c.Name {
			sel.from[c.install] = c.Name
			skillSet[c.install] = skillSpec{From: c.Name}
		}
	}
	if r.manifest != nil {
		sel.manifest = r.manifest.Skills
	}
	probe, err := applyManifest(component{ID: r.id, Repo: r.repo, Ref: r.ref, User: true, ProjectOnly: !global, Named: true, Skills: sel.names, SkillSet: skillSet, SkillPath: cleanSubpath(r.src.Subpath)}, r.store)
	if err != nil {
		return err
	}
	probe.Store = r.store
	if err := checkCommandOwners(r.repoRoot, r.home, distro{Components: map[string]component{r.id: probe}}); err != nil {
		return err
	}
	if err := writeAddConfig(r.home, r.projectRoot, r.id, global, componentConfig{Repo: r.repo, Ref: r.ref, Path: cleanSubpath(r.src.Subpath)}, sel); err != nil {
		return err
	}
	if err := setUserPin(r.home, r.id, r.pin); err != nil {
		return err
	}
	if r.state.LastSelectedAgents == nil {
		return nil
	}
	fresh, err := readState(r.home)
	if err != nil {
		return err
	}
	fresh.LastSelectedAgents = r.state.LastSelectedAgents
	return writeState(r.home, fresh)
}

func (r *addRun) install() error {
	effective, _, err := effectivePins(r.repoRoot, r.home, r.projectRoot, "")
	if err != nil {
		return err
	}
	c, err := applyManifest(effective.Components[r.id], r.store)
	if err != nil {
		return err
	}
	installed, failed := map[string][]destResult{}, map[string]error{}
	for _, entry := range selectedEntries(c, r.store, r.projectRoot) {
		target := entryHome(r.home, entry)
		for _, name := range entry.Skills {
			if !slices.ContainsFunc(r.chosen, func(s addSkill) bool { return s.install == name }) {
				if incompleteUnedited(entry, target, name) {
					if _, err := replaceSkill(target, entry, name); err != nil {
						fmt.Fprintf(os.Stderr, "Installing %s: %v\n", name, err)
					}
				}
				continue
			}
			if dests, err := replaceSkill(target, entry, name); err != nil {
				failed[name] = err
			} else {
				installed[name] = dests
			}
		}
	}
	prevStore := ""
	if r.prev.Commit != "" && r.prev.Commit != r.commit {
		prevStore = storePath(r.home, r.id, r.prev.Commit)
	}
	installErr := installUtility(r.home, r.repoRoot, r.projectRoot, r.id, prevStore, mcpInventory{})
	if installErr != nil && !errors.Is(installErr, errSkipped) {
		return installErr
	}
	r.report(installed, failed)
	where := ""
	if r.scope == "project" {
		where = " to " + r.projectRoot
	}
	if installErr == nil && len(failed) == 0 {
		r.out.done(true, r.id+" added"+where+pinLabel(r.pin))
	} else {
		r.out.done(false, r.id+" added"+where+pinLabel(r.pin)+"; some install steps were left over, see above")
	}
	if installErr != nil || len(failed) > 0 || slices.ContainsFunc(r.results, func(res any) bool { _, skipped := res.(addSkipped); return skipped }) {
		return errSkipped
	}
	return nil
}

func (r *addRun) report(installed map[string][]destResult, failed map[string]error) {
	out, global, root := r.out, r.scope == "global", r.root()
	targets := addTargets(r.agents, r.subagents)
	var displays, skipped []string
	for _, t := range targets {
		displays = append(displays, t.display())
	}
	for _, id := range r.skipped {
		skipped = append(skipped, agentDisplay(id))
	}
	mode := "symlink"
	if r.copyMode {
		mode = "copy"
	}
	var sourceRef *string
	if r.explicitRef != "" {
		sourceRef = &r.explicitRef
	}
	source := sourceLabel(r.src)
	if r.src.Kind == "local" {
		source = filepath.Join(r.repo, filepath.FromSlash(r.src.Subpath))
	}
	var lines []string
	for _, c := range r.chosen {
		dests, ok := installed[c.install]
		if !ok {
			err := cmp.Or(failed[c.install], errors.New("Installation failed"))
			if strings.HasPrefix(err.Error(), "Refusing to replace ") {
				r.results = append(r.results, addSkipped{Name: c.install, Status: "skipped", Reason: "Conflicts with a category directory"})
			} else {
				r.results = append(r.results, addFailed{Name: c.install, Status: "failed", Error: err.Error()})
			}
			continue
		}
		path := dests[0].Dir
		if canonical := filepath.Join(root, ".agents", "skills", c.install); !r.copyMode && slices.ContainsFunc(dests, func(d destResult) bool { return d.Dir == canonical }) {
			path = canonical
		}
		var hash *string
		if h, err := skillsComputedHash(path); err == nil {
			hash = &h
		}
		r.results = append(r.results, addInstalled{Name: c.install, Status: "installed", Source: source, Ref: sourceRef, Hash: hash, Path: path, Scope: r.scope, Agents: displays, Mode: mode})
		if r.copyMode {
			lines = append(lines, out.tone("✓", colorOK)+" "+c.install+" "+out.tone("(copied)", colorDim))
			var seen []string
			for _, d := range dests {
				if short := shortenPath(d.Dir, r.cwd); !slices.Contains(seen, short) {
					seen = append(seen, short)
					lines = append(lines, "  "+out.tone("→", colorDim)+" "+short)
				}
			}
			continue
		}
		lines = append(lines, out.tone("✓", colorOK)+" "+shortenPath(path, r.cwd))
		var universal, linked, copied []string
		for _, t := range targets {
			if t.subagent == "" && usesCanonical(t.agent, global) {
				universal = append(universal, t.display())
			}
		}
		for _, d := range dests {
			for _, a := range d.Agents {
				if d.Linked {
					linked = append(linked, agentDisplay(a))
				} else if d.LinkFailed {
					copied = append(copied, agentDisplay(a))
				}
			}
		}
		for _, line := range []struct {
			label, tone, suffix string
			names               []string
		}{{"universal:", colorOK, "", universal}, {"symlinked:", colorDim, "", linked}, {"copied:", colorWarn, "", copied}, {"skipped:", colorWarn, " " + out.tone("(project directory not found)", colorDim), skipped}} {
			if len(line.names) > 0 {
				lines = append(lines, "  "+out.tone(line.label, line.tone)+" "+formatList(line.names, 5)+line.suffix)
			}
		}
	}
	if len(installed) > 0 {
		out.println()
		out.heading("Installed " + plural(len(installed), "skill"))
		for _, line := range lines {
			out.println("  " + line)
		}
	}
	if len(failed) > 0 {
		out.println()
		out.println(out.tone(fmt.Sprintf("Failed to install %d", len(failed)), colorBad))
		for _, c := range r.chosen {
			if err, ok := failed[c.install]; ok {
				out.println(fmt.Sprintf("  %s %s → %s: %s", out.tone("✗", colorBad), c.install, strings.Join(displays, ", "), out.tone(err.Error(), colorDim)))
			}
		}
	}
	if len(r.chosen) > 0 {
		out.println()
		out.println(out.tone("Done!", colorOK) + out.tone("  Review skills before use; they run with full agent permissions.", colorDim))
	}
}

func cleanSubpath(sub string) string {
	if clean := filepath.ToSlash(filepath.Clean(sub)); sub != "" && clean != "." {
		return clean
	}
	return ""
}

func markExplicit(explicit map[string]bool, agents []string) {
	for _, id := range agents {
		explicit[id] = true
	}
}

func logChosenSkills(out *report, skills []discoveredSkill) {
	if len(skills) != 1 {
		out.println(fmt.Sprintf("Installing all %d skills", len(skills)))
		return
	}
	out.println("Skill: " + out.tone(skills[0].Name, colorAccent))
	if skills[0].Description != "" {
		out.println(out.tone(skills[0].Description, colorDim))
	}
}

func printAvailableSkills(out *report, skills []discoveredSkill) {
	out.println()
	out.heading("Available Skills")
	groups := map[string][]discoveredSkill{}
	var general []discoveredSkill
	for _, s := range skills {
		if s.Plugin != "" {
			groups[s.Plugin] = append(groups[s.Plugin], s)
		} else {
			general = append(general, s)
		}
	}
	show := func(list []discoveredSkill) {
		for _, s := range list {
			out.println("  " + out.tone(s.Name, colorAccent))
			out.println("    " + out.tone(s.Description, colorDim))
		}
	}
	titles := slices.Sorted(maps.Keys(groups))
	for _, group := range titles {
		out.println(out.bold(kebabTitle(group), colorAccent))
		show(groups[group])
		out.println()
	}
	if len(general) > 0 {
		if len(titles) > 0 {
			out.println(out.bold("General", colorAccent))
		}
		show(general)
	}
	out.println()
	out.println("Use --skill <name> to install specific skills")
}

func promptAgents(state *stateFile) ([]string, error) {
	var choices []promptChoice
	var valid []string
	for _, a := range agentTable {
		if a.ID != "eve" {
			choices = append(choices, promptChoice{label: a.Display, value: a.ID})
			valid = append(valid, a.ID)
		}
	}
	initial := slices.DeleteFunc(slices.Clone(state.LastSelectedAgents), func(id string) bool { return !slices.Contains(valid, id) })
	if len(initial) == 0 {
		initial = []string{"claude-code", "opencode", "codex", "grok"}
	}
	picked, err := promptMany("Which agents do you want to install to?", "", choices, initial, true)
	if err != nil {
		return nil, err
	}
	state.LastSelectedAgents = picked
	return picked, nil
}

func selectAgents(state *stateFile, global bool) ([]string, error) {
	env := currentAgentEnv()
	var locked, lockedNames, others []string
	var choices []promptChoice
	hidden := 0
	for _, a := range agentTable {
		if global && a.GlobalDir == nil {
			continue
		}
		switch {
		case isUniversal(a.ID) && !a.HideInUniversalList && a.HideInUniversalPrompt:
			hidden++
		case isUniversal(a.ID) && !a.HideInUniversalList:
			locked = append(locked, a.ID)
			lockedNames = append(lockedNames, a.Display)
		case !isUniversal(a.ID) && a.ID != "eve":
			hint := a.Dir
			if global {
				hint = a.GlobalDir(env)
			}
			choices = append(choices, promptChoice{label: a.Display, value: a.ID, hint: hint})
			others = append(others, a.ID)
		}
	}
	description := "Universal (.agents/skills), always included: " + strings.Join(lockedNames, ", ")
	if hidden > 0 {
		description += fmt.Sprintf(", and %d more", hidden)
	}
	initial := slices.DeleteFunc(slices.Clone(state.LastSelectedAgents), func(id string) bool { return !slices.Contains(others, id) })
	picked, err := promptMany("Which agents do you want to install to?", description, choices, initial, false)
	if err != nil {
		return nil, err
	}
	selected := append(locked, picked...)
	state.LastSelectedAgents = selected
	return selected, nil
}
