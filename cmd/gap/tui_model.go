package main

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

const (
	tabItems = iota
	tabStatus
	tabProjects
)

var tabNames = []string{"Items", "Status", "Projects"}

type tuiMode int

const (
	modeBrowse tuiMode = iota
	modeConfirm
	modeRun
	modeAsk
	modeSummary
	modeDetail
	modeForm
	modeSearch
)

type jobState int

const (
	jobPending jobState = iota
	jobRunning
	jobOK
	jobFailed
)

type jobView struct {
	tuiJob
	state     jobState
	began     time.Time
	ended     time.Time
	lines     []string
	code      int
	cancelled bool
}

type loadedMsg struct {
	gen          int
	quick        bool
	projectsOnly bool
	items        []itemRow
	projects     []projectView
	err          error
	projErr      error
}

type tuiKeys struct {
	Up, Down, Top, Bottom, Tab, Help, Mark, Check, CheckAll, Update, UpdateAll, Sync, Follow, Find, Add, Edit, Editor, Delete, Disable, Forget, Refresh, Quit, Confirm, Open, Back key.Binding
}

func newTUIKeys() tuiKeys {
	b := func(help, desc string, keys ...string) key.Binding {
		return key.NewBinding(key.WithKeys(keys...), key.WithHelp(help, desc))
	}
	return tuiKeys{
		Up:        b("k/\u2191", "up", "k", "up"),
		Down:      b("j/\u2193", "down", "j", "down"),
		Top:       b("gg", "top", "home"),
		Bottom:    b("G", "bottom", "G", "end"),
		Tab:       b("tab/1-3", "view", "tab", "shift+tab", "1", "2", "3"),
		Help:      b("?", "all keys", "?"),
		Mark:      b("space", "mark", "space"),
		Check:     b("c", "check", "c"),
		CheckAll:  b("C", "check all", "C"),
		Update:    b("u", "update", "u"),
		UpdateAll: b("U", "update all", "U"),
		Sync:      b("s", "sync", "s"),
		Follow:    b("f", "follow", "f"),
		Find:      b("/", "find skills", "/"),
		Add:       b("a", "add", "a"),
		Edit:      b("e", "edit", "e"),
		Editor:    b("E", "open gap.toml", "E"),
		Delete:    b("d", "remove", "d"),
		Disable:   b("x", "disable", "x"),
		Forget:    b("f", "forget", "f"),
		Refresh:   b("r", "refresh", "r"),
		Quit:      b("q", "quit", "q", "ctrl+c"),
		Confirm:   b("enter", "confirm", "enter"),
		Open:      b("enter", "details", "enter"),
		Back:      b("esc", "back", "esc"),
	}
}

type tuiModel struct {
	home, homeReal, repoRoot, projectRoot string
	st                                    tuiStyles
	keys                                  tuiKeys
	help                                  help.Model
	spin                                  spinner.Model
	vp                                    viewport.Model
	width, height                         int

	tab          int
	sel          [3]int
	items        []itemRow
	projects     []projectView
	loadErr      error
	projErr      error
	loading      bool
	checking     bool
	projChecking bool
	projAsked    bool
	projDone     bool
	statusAsked  bool
	statusReq    int
	gen          int
	stores       *storeCache
	marked       map[string]bool

	status        string
	statusErr     error
	statusLoading bool
	statusCols    int
	statusAt      int

	mode         tuiMode
	confirmTitle string
	confirmIDs   []string
	confirmNote  []string
	confirmTop   int
	pending      []tuiJob
	jobs         []jobView
	jobSel       int
	follow       bool
	askChoice    int
	note         string
	chord        string
	showHelp     bool
	top          [3]int
	quitting     bool
	pendingExec  []string
	formState    *formState
	search       *searchState
	searchGen    int
	handOff      func(dir string, args []string) tea.Cmd
	findSkills   func(context.Context, string, int) ([]skillHit, error)
	frame        int
	start, now   time.Time
	clock        func() time.Time

	runner       jobRunner
	children     *childGroup
	send         func(tea.Msg)
	loadQuick    func() loadedMsg
	loadFull     func(*storeCache) loadedMsg
	loadProjects func(*storeCache) loadedMsg
	fetchStatus  func(context.Context) tea.Cmd
	statusStop   context.CancelFunc
}

func newTUIModel(home, repoRoot, projectRoot string, p palette) *tuiModel {
	m := &tuiModel{
		home: home, homeReal: canonicalRoot(home), repoRoot: repoRoot, projectRoot: projectRoot,
		stores:     newStoreCache(),
		st:         newTUIStyles(p),
		keys:       newTUIKeys(),
		help:       help.New(),
		vp:         viewport.New(),
		width:      100,
		height:     30,
		marked:     map[string]bool{},
		runner:     &execRunner{},
		children:   newChildGroup(),
		send:       func(tea.Msg) {},
		clock:      time.Now,
		findSkills: searchSkills,
	}
	m.spin = spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(m.st.accent))
	m.loadQuick = func() loadedMsg { return quickTUIData(home, repoRoot) }
	m.loadFull = func(stores *storeCache) loadedMsg { return loadTUIItems(home, repoRoot, projectRoot, stores) }
	m.loadProjects = func(stores *storeCache) loadedMsg { return loadTUIProjects(home, repoRoot, stores) }
	m.fetchStatus = func(ctx context.Context) tea.Cmd { return statusText(m.children, ctx, max(m.width-4, 1)) }
	return m
}

func quickTUIData(home, repoRoot string) loadedMsg {
	items, err := registeredRows(home, repoRoot)
	list, perr := readProjects(home)
	var projects []projectView
	for _, p := range list {
		v := projectView{Root: p.Root, Added: p.Added}
		if _, serr := os.Stat(p.Root); errors.Is(serr, os.ErrNotExist) {
			v.Missing = true
		}
		projects = append(projects, v)
	}
	return loadedMsg{items: items, projects: projects, err: err, projErr: perr}
}

func loadTUIItems(home, repoRoot, projectRoot string, stores *storeCache) loadedMsg {
	items, ierr := inventory(home, repoRoot, projectRoot, stores)
	if ierr == nil {
		return loadedMsg{items: items}
	}
	items, err := registeredRows(home, repoRoot)
	if err != nil {
		return loadedMsg{err: errors.Join(ierr, err)}
	}
	return loadedMsg{items: items, err: fmt.Errorf("status unavailable: %w", ierr)}
}

func loadTUIProjects(home, repoRoot string, stores *storeCache) loadedMsg {
	list, err := readProjects(home)
	var projects []projectView
	for _, p := range list {
		projects = append(projects, projectInventory(home, repoRoot, p, stores))
	}
	return loadedMsg{projects: projects, projErr: err}
}

func (m *tuiModel) reload() tea.Cmd {
	m.gen++
	m.stores = newStoreCache()
	m.loading = true
	m.projAsked, m.projDone = false, false
	m.stopStatus()
	m.statusAsked, m.statusCols = false, 0
	return tea.Batch(m.loadCmd(m.loadQuick, true), m.statusCmd())
}

func (m *tuiModel) stopStatus() {
	if m.statusStop != nil {
		m.statusStop()
		m.statusStop = nil
		m.statusAsked, m.statusLoading = false, false
		m.statusReq++
	}
}

func (m *tuiModel) statusCmd() tea.Cmd {
	if m.tab != tabStatus || m.statusAsked {
		return nil
	}
	m.statusAsked, m.statusCols = true, max(m.width-4, 1)
	ctx, stop := context.WithCancel(context.Background())
	fetch := m.fetchStatus(ctx)
	if fetch == nil {
		stop()
		return nil
	}
	m.statusReq++
	m.statusStop, m.statusLoading = stop, true
	gen, req, cols := m.gen, m.statusReq, m.statusCols
	return func() tea.Msg {
		msg := fetch()
		if s, ok := msg.(statusTextMsg); ok {
			s.gen, s.req = gen, req
			if s.cols == 0 {
				s.cols = cols
			}
			return s
		}
		return msg
	}
}

func (m *tuiModel) itemsCmd() tea.Cmd {
	load, stores := m.loadFull, m.stores
	return m.loadCmd(func() loadedMsg { return load(stores) }, false)
}

func (m *tuiModel) projectsCmd() tea.Cmd {
	if m.tab != tabProjects || m.projDone || m.projAsked {
		return nil
	}
	m.projAsked = true
	load, stores, gen := m.loadProjects, m.stores, m.gen
	return func() tea.Msg {
		msg := load(stores)
		msg.gen, msg.projectsOnly = gen, true
		return msg
	}
}

func (m *tuiModel) loadCmd(load func() loadedMsg, quick bool) tea.Cmd {
	gen := m.gen
	return func() tea.Msg {
		msg := load()
		msg.gen, msg.quick = gen, quick
		return msg
	}
}

func (m *tuiModel) Init() tea.Cmd {
	return tea.Batch(m.reload(), m.spin.Tick)
}

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		var resized tea.Cmd
		cols := max(m.width-4, 1)
		if m.tab == tabStatus && ((m.statusCols != 0 && m.statusCols != cols) || (m.statusAt != 0 && m.statusAt != cols)) {
			m.stopStatus()
			m.status, m.statusErr, m.statusAsked, m.statusAt = "", nil, false, 0
			resized = m.statusCmd()
		}
		m.refreshPane()
		m.sizeForm()
		if m.search != nil {
			m.search.input.SetWidth(max(m.width-8, 10))
		}
		return m, resized
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		if m.running() {
			m.frame++
			m.now = m.clock()
		}
		return m, cmd
	case loadedMsg:
		if msg.gen != m.gen {
			return m, nil
		}
		var next tea.Cmd
		switch {
		case msg.projectsOnly:
			m.projects, m.projErr = msg.projects, msg.projErr
			m.projChecking, m.projDone = false, true
		case msg.quick:
			m.items, m.loadErr = msg.items, msg.err
			m.loading, m.checking = true, true
			if !m.projDone {
				m.projects, m.projErr, m.projChecking = msg.projects, msg.projErr, true
			}
			next = tea.Batch(m.itemsCmd(), m.projectsCmd())
		default:
			m.items, m.loadErr = msg.items, msg.err
			m.loading, m.checking = false, false
			if msg.projects != nil {
				m.projects, m.projErr, m.projChecking, m.projDone = msg.projects, msg.projErr, false, true
			}
		}
		for id := range m.marked {
			if !slices.ContainsFunc(m.items, func(r itemRow) bool { return r.ID == id && r.user() }) {
				delete(m.marked, id)
			}
		}
		for t := range m.sel {
			m.sel[t] = min(m.sel[t], max(m.count(t)-1, 0))
		}
		m.refreshPane()
		return m, next
	case statusTextMsg:
		if msg.gen != m.gen || msg.req != m.statusReq {
			return m, nil
		}
		if m.statusStop != nil {
			m.statusStop()
			m.statusStop = nil
		}
		m.statusLoading = false
		if msg.cols != 0 && msg.cols != max(m.width-4, 1) {
			m.status, m.statusErr, m.statusAsked, m.statusAt = "", nil, false, 0
			m.refreshPane()
			return m, m.statusCmd()
		}
		m.status, m.statusErr, m.statusAt = msg.text, msg.err, msg.cols
		m.refreshPane()
	case jobStartedMsg:
		if msg.i < len(m.jobs) {
			m.jobs[msg.i].state, m.jobs[msg.i].began = jobRunning, m.clock()
			if m.follow {
				m.jobSel = msg.i
			}
			m.refreshPane()
		}
	case jobOutputMsg:
		if msg.i < len(m.jobs) {
			m.jobs[msg.i].lines = append(m.jobs[msg.i].lines, msg.line)
			m.refreshPane()
		}
	case jobDoneMsg:
		if msg.i < len(m.jobs) {
			j := &m.jobs[msg.i]
			j.code, j.cancelled, j.ended = msg.code, msg.cancelled, m.clock()
			j.state = jobOK
			if msg.code != 0 || msg.err != nil {
				j.state = jobFailed
				if msg.err != nil {
					j.lines = append(j.lines, msg.err.Error())
				}
			}
			m.refreshPane()
		}
	case handedBackMsg:
		m.note = "Back from gap."
		if msg.edited != "" {
			m.note = "Saved " + m.short(msg.edited) + ". s syncs it."
		}
		if msg.err != nil {
			m.note = "gap exited: " + msg.err.Error()
		}
		if msg.after != nil {
			msg.note = msg.after(msg.err)
		}
		if msg.note != "" {
			m.note = msg.note
		}
		return m, m.reload()
	case jobsFinishedMsg:
		m.mode, m.follow = modeSummary, false
		if i := slices.IndexFunc(m.jobs, func(j jobView) bool { return j.state == jobFailed }); i >= 0 {
			m.jobSel = i
		}
		m.now = m.clock()
		m.refreshPane()
		m.vp.GotoBottom()
	case searchTickMsg, searchDoneMsg:
		if m.mode == modeSearch {
			return m, m.searchUpdate(msg)
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.mode == modeSearch {
			return m, m.searchUpdate(msg)
		}
		if m.mode == modeForm {
			return m, m.formUpdate(msg)
		}
		return m, m.key(msg)
	default:
		switch m.mode {
		case modeForm:
			return m, m.formUpdate(msg)
		case modeSearch:
			return m, m.searchUpdate(msg)
		}
	}
	return m, nil
}

func (m *tuiModel) count(tab int) int {
	switch tab {
	case tabItems:
		return len(m.items)
	case tabProjects:
		return len(m.projects)
	}
	return 0
}

func (m *tuiModel) item() (itemRow, bool) {
	if m.sel[tabItems] < len(m.items) {
		return m.items[m.sel[tabItems]], true
	}
	return itemRow{}, false
}

func (m *tuiModel) project() (projectView, bool) {
	if m.sel[tabProjects] < len(m.projects) {
		return m.projects[m.sel[tabProjects]], true
	}
	return projectView{}, false
}

func (m *tuiModel) key(msg tea.KeyPressMsg) tea.Cmd {
	if m.showHelp {
		if key.Matches(msg, m.keys.Help, m.keys.Back, m.keys.Quit) {
			m.showHelp = false
		}
		return nil
	}
	if key.Matches(msg, m.keys.Help) && (m.mode == modeBrowse || m.mode == modeDetail) {
		m.showHelp, m.chord = true, ""
		return nil
	}
	if m.mode == modeBrowse || m.mode == modeDetail || m.mode == modeSummary {
		if mv := m.vimMotion(msg); mv != vimNone {
			return m.applyVim(mv)
		}
	} else {
		m.chord = ""
	}
	switch m.mode {
	case modeConfirm:
		_, rows, _, shown := m.confirmParts()
		switch {
		case key.Matches(msg, m.keys.Confirm):
			if args := m.pendingExec; args != nil {
				m.pendingExec, m.mode = nil, modeBrowse
				return m.handOver(args...)
			}
			if len(rows) > 0 && shown == 0 {
				m.note = "Enlarge the terminal to see what this removes."
				return nil
			}
			jobs := m.pending
			m.pending = nil
			return m.run(jobs)
		case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit), msg.String() == "n":
			m.mode, m.pending, m.pendingExec, m.note = modeBrowse, nil, nil, "Nothing changed."
		case key.Matches(msg, m.keys.Down):
			m.confirmTop = max(min(m.confirmTop+1, len(rows)-shown), 0)
		case key.Matches(msg, m.keys.Up):
			m.confirmTop = max(min(m.confirmTop, len(rows)-shown)-1, 0)
		}
		return nil
	case modeDetail:
		switch {
		case key.Matches(msg, m.keys.Back, m.keys.Confirm, m.keys.Quit):
			m.mode = modeBrowse
			m.refreshPane()
		case key.Matches(msg, m.keys.Up, m.keys.Down, m.keys.Top, m.keys.Bottom):
			m.scroll(msg)
		default:
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return cmd
		}
		return nil
	case modeRun:
		switch {
		case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit):
			m.mode, m.askChoice = modeAsk, 0
		case key.Matches(msg, m.keys.Up):
			m.follow, m.jobSel = false, max(m.jobSel-1, 0)
			m.refreshPane()
		case key.Matches(msg, m.keys.Down):
			m.follow, m.jobSel = false, min(m.jobSel+1, len(m.jobs)-1)
			m.refreshPane()
		case key.Matches(msg, m.keys.Follow):
			m.follow = true
			for i, j := range m.jobs {
				if j.state == jobRunning {
					m.jobSel = i
				}
			}
			m.refreshPane()
		default:
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return cmd
		}
		return nil
	case modeAsk:
		switch {
		case msg.String() == "ctrl+c":
			m.runner.cancel()
			m.quitting = true
			return tea.Quit
		case key.Matches(msg, m.keys.Up), key.Matches(msg, m.keys.Down):
			m.askChoice = 1 - m.askChoice
		case key.Matches(msg, m.keys.Confirm):
			if m.askChoice == 1 {
				m.runner.cancel()
				m.note = "Stopping: the current step is cancelled and the rest are skipped."
			}
			m.mode = modeRun
		case key.Matches(msg, m.keys.Back):
			m.mode = modeRun
		}
		return nil
	case modeSummary:
		switch {
		case key.Matches(msg, m.keys.Confirm, m.keys.Back, m.keys.Quit):
			m.mode, m.jobs = modeBrowse, nil
			return m.reload()
		case key.Matches(msg, m.keys.Up, m.keys.Down, m.keys.Top, m.keys.Bottom):
			m.jobSel = m.step(msg, m.jobSel, len(m.jobs))
			m.refreshPane()
			m.vp.GotoBottom()
		default:
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return cmd
		}
		return nil
	}
	m.note = ""
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return tea.Quit
	case key.Matches(msg, m.keys.Tab):
		next := (m.tab + 1) % len(tabNames)
		switch msg.String() {
		case "1", "2", "3":
			next = int(msg.String()[0] - '1')
		case "shift+tab":
			next = (m.tab + len(tabNames) - 1) % len(tabNames)
		}
		return m.switchTab(next)
	case m.tab == tabStatus && key.Matches(msg, m.keys.Up, m.keys.Down, m.keys.Top, m.keys.Bottom):
		m.scroll(msg)
	case key.Matches(msg, m.keys.Up):
		m.move(-1)
	case key.Matches(msg, m.keys.Down):
		m.move(1)
	case key.Matches(msg, m.keys.Top):
		m.move(-m.count(m.tab))
	case key.Matches(msg, m.keys.Bottom):
		m.move(m.count(m.tab))
	case key.Matches(msg, m.keys.Refresh):
		m.note = "Refreshing."
		return m.reload()
	case m.tab != tabStatus && key.Matches(msg, m.keys.Open):
		if m.count(m.tab) > 0 {
			m.mode = modeDetail
			m.refreshPane()
			m.vp.GotoTop()
		}
	case key.Matches(msg, m.keys.Sync):
		return m.handOver("sync")
	case m.tab == tabItems && key.Matches(msg, m.keys.UpdateAll):
		ids := m.updateIDs()
		if len(ids) == 0 {
			m.note = "Nothing to update: every item is held."
			break
		}
		m.confirmTitle = fmt.Sprintf("Update %s?", plural(len(ids), "item"))
		m.confirmIDs, m.confirmTop = ids, 0
		m.confirmNote = []string{"Runs gap update in this terminal. Items held with pin = true, here or in gap.toml, are skipped; disabled items still move. Trust prompts show as usual."}
		m.pendingExec, m.mode = []string{"update"}, modeConfirm
	case key.Matches(msg, m.keys.Find):
		return m.openSearch()
	case m.tab == tabItems && key.Matches(msg, m.keys.Add):
		return m.addForm(m.here())
	case key.Matches(msg, m.keys.Editor), m.tab == tabProjects && key.Matches(msg, m.keys.Edit):
		if p, ok := m.project(); ok && m.tab == tabProjects {
			return m.editorForm(p.Root)
		}
		return m.editorForm(m.here())
	case m.tab == tabProjects && key.Matches(msg, m.keys.Add):
		if p, ok := m.project(); ok {
			return m.addForm(p.Root)
		}
		return m.addForm("")
	case m.tab == tabItems:
		return m.itemKey(msg)
	case m.tab == tabProjects && key.Matches(msg, m.keys.Forget):
		p, ok := m.project()
		if !ok {
			return nil
		}
		if err := forgetProject(m.home, p.Root); err != nil {
			m.note = err.Error()
			return nil
		}
		m.note = "Forgot " + p.Root + ". gap init or gap sync there records it again."
		return m.reload()
	case m.tab == tabStatus:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return cmd
	}
	return nil
}

func (m *tuiModel) itemKey(msg tea.KeyPressMsg) tea.Cmd {
	r, ok := m.item()
	if !ok {
		return nil
	}
	switch {
	case key.Matches(msg, m.keys.Mark):
		if !r.user() {
			m.note = r.ID + " is a distro component: it cannot be removed, x disables it."
			return nil
		}
		m.marked[r.ID] = !m.marked[r.ID]
		if !m.marked[r.ID] {
			delete(m.marked, r.ID)
		}
		m.move(1)
	case key.Matches(msg, m.keys.Check):
		return m.run([]tuiJob{{label: "check " + r.ID, name: r.ID, color: m.groupColor(r.ID), args: []string{"check", "--", r.ID}}})
	case key.Matches(msg, m.keys.CheckAll):
		return m.run([]tuiJob{{label: "check all", args: []string{"check"}}})
	case key.Matches(msg, m.keys.Update):
		return m.run([]tuiJob{{label: "update " + r.ID, name: r.ID, color: m.groupColor(r.ID), args: []string{"update", "--", r.ID}}})
	case key.Matches(msg, m.keys.Delete):
		ids := m.markedIDs()
		if len(ids) == 0 {
			if !r.user() {
				m.note = r.ID + " is a distro component: it cannot be removed, x disables it."
				return nil
			}
			ids = []string{r.ID}
		}
		var jobs []tuiJob
		for _, id := range ids {
			jobs = append(jobs, tuiJob{label: "remove " + id, name: id, color: m.groupColor(id), args: []string{"remove", "--", id}, removes: id})
		}
		m.confirmTitle = "Remove 1 utility?"
		if len(ids) > 1 {
			m.confirmTitle = fmt.Sprintf("Remove %d utilities?", len(ids))
		}
		m.confirmIDs, m.confirmTop = ids, 0
		m.confirmNote = []string{"Each one runs its uninstall hook and its files move to ~/.gap/backups/."}
		m.pending, m.mode = jobs, modeConfirm
	case key.Matches(msg, m.keys.Edit):
		return m.editForm(r)
	case key.Matches(msg, m.keys.Disable):
		if r.user() {
			m.note = r.ID + " is a utility: d removes it."
			return nil
		}
		verb, enable := "Disable", false
		if r.Disabled {
			verb, enable = "Enable", true
		}
		home, repoRoot, id := m.home, m.repoRoot, r.ID
		m.confirmTitle = verb + " " + id + "?"
		m.confirmIDs, m.confirmTop = nil, 0
		m.confirmNote = []string{"Writes enabled = " + fmt.Sprint(enable) + " under [components." + id + "] in " + userConfigFile(home) + ".", "gap sync applies it."}
		m.pending = []tuiJob{{label: strings.ToLower(verb) + " " + id, fn: func() error { return setComponentEnabled(home, repoRoot, id, enable) }}}
		m.mode = modeConfirm
	}
	return nil
}

func (m *tuiModel) markedIDs() []string {
	var ids []string
	for _, r := range m.items {
		if m.marked[r.ID] {
			ids = append(ids, r.ID)
		}
	}
	return ids
}

// updateIDs is what an unnamed gap update moves: every registered item that is not held.
func (m *tuiModel) updateIDs() []string {
	var ids []string
	for _, r := range m.items {
		if !r.Held {
			ids = append(ids, r.ID)
		}
	}
	return ids
}

func (m *tuiModel) groupColor(id string) color.Color {
	for _, r := range m.items {
		if r.ID == id {
			return m.st.pal.group(r.Color, id)
		}
	}
	return m.st.pal.Accent
}

func (m *tuiModel) handOver(args ...string) tea.Cmd { return m.handOverIn("", args...) }

func (m *tuiModel) handOverIn(dir string, args ...string) tea.Cmd {
	return m.handOverThen(dir, args, nil)
}

// handOverThen runs gap with args on the real terminal. after runs as soon as gap exits, or at once
// when gap cannot be started, and its note is shown back in the TUI.
func (m *tuiModel) handOverThen(dir string, args []string, after func(error) string) tea.Cmd {
	if m.handOff != nil {
		cmd := m.handOff(dir, args)
		return func() tea.Msg {
			if cmd != nil {
				cmd()
			}
			msg := handedBackMsg{}
			if after != nil {
				msg.note = after(nil)
			}
			return msg
		}
	}
	exe, err := tuiExecutable()
	if err != nil {
		m.note = "Cannot find the gap binary: " + err.Error()
		if after != nil {
			if note := after(err); note != "" {
				m.note += ". " + note
			}
		}
		return nil
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	h := &handoff{cmd: cmd, after: after}
	return tea.Exec(h, func(err error) tea.Msg {
		msg := handedBackMsg{err: err, note: h.note}
		if !h.ran {
			msg.after = after
		}
		return msg
	})
}

func (m *tuiModel) run(jobs []tuiJob) tea.Cmd {
	m.jobs = nil
	for _, j := range jobs {
		m.jobs = append(m.jobs, jobView{tuiJob: j})
	}
	for _, j := range jobs {
		if j.removes != "" {
			delete(m.marked, j.removes)
		}
	}
	m.mode, m.jobSel, m.follow, m.note = modeRun, 0, true, ""
	m.start, m.frame = m.clock(), 0
	m.now = m.start
	m.refreshPane()
	m.runner.start(jobs, m.send)
	return nil
}

func (m *tuiModel) step(msg tea.KeyPressMsg, sel, n int) int {
	switch {
	case key.Matches(msg, m.keys.Up):
		sel--
	case key.Matches(msg, m.keys.Down):
		sel++
	case key.Matches(msg, m.keys.Top):
		sel = 0
	case key.Matches(msg, m.keys.Bottom):
		sel = n - 1
	}
	return min(max(sel, 0), max(n-1, 0))
}

func (m *tuiModel) scroll(msg tea.KeyPressMsg) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.vp.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.vp.ScrollDown(1)
	case key.Matches(msg, m.keys.Top):
		m.vp.GotoTop()
	case key.Matches(msg, m.keys.Bottom):
		m.vp.GotoBottom()
	}
}

func (m *tuiModel) switchTab(next int) tea.Cmd {
	if m.tab == tabStatus && next != tabStatus {
		m.stopStatus()
	}
	m.tab = next
	if m.tab == tabStatus && m.statusAt != 0 && m.statusAt != max(m.width-4, 1) {
		m.status, m.statusErr, m.statusAsked, m.statusAt = "", nil, false, 0
	}
	m.refreshPane()
	return tea.Batch(m.projectsCmd(), m.statusCmd())
}

func (m *tuiModel) move(d int) {
	n := m.count(m.tab)
	if n == 0 {
		return
	}
	m.sel[m.tab] = min(max(m.sel[m.tab]+d, 0), n-1)
	m.top[m.tab], _ = m.listWindow(m.tab)
	m.refreshPane()
}

func runTUI() error {
	home, repoRoot, err := gapHomes()
	if err != nil {
		return err
	}
	projectRoot, err := invokingProjectRoot()
	if err != nil {
		return err
	}
	pal := darkPalette
	if !terminalIsDark() {
		pal = lightPalette
	}
	m := newTUIModel(home, repoRoot, projectRoot, pal)
	p := tea.NewProgram(m)
	m.send = p.Send
	defer func() {
		m.runner.cancel()
		if w, ok := m.runner.(interface{ wait() }); ok {
			w.wait()
		}
		m.children.close()
	}()
	if _, err := p.Run(); err != nil && !errors.Is(err, tea.ErrInterrupted) {
		return err
	}
	return nil
}

func tuiReadyDefault() bool {
	return stdinIsTerminal() && stdoutIsTerminal() && !plainEnv()
}

var tuiReady = tuiReadyDefault
