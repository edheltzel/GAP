# gap sync Bubble Tea live progress and bare-gap TUI verification

## Outcome
- `gap sync` on a terminal lists what it is syncing and displays a live working indicator, progress bar, per-item detail, elapsed time, and estimated time remaining, powered by Bubble Tea v2 (`charm.land/bubbletea/v2`) instead of a raw ANSI scroll-region bar.
- Interactive prompts (`confirmReplace`, `confirmSplice`, hook trust approval, skill drift diff/prompt) and component hook commands suspend and resume the Bubble Tea progress UI cleanly without cursor or stderr races; piped and non-TTY output stays unchanged.
- Running bare `gap` with no commands or flags opens the existing Bubble Tea v2 TUI on an interactive terminal (`cmd/gap/cli.go`, `cmd/gap/tui_model.go`), verified by direct PTY end-to-end coverage.
- Current open PR #162 (`feat/sync-live-progress`, raw ANSI scroll-region pass) is updated or superseded cleanly once this plan is approved and built.

## Implementation Units

### U1. Replace raw ANSI progress bar with a Bubble Tea v2 inline progress model
- Depends on: none
- Files: `cmd/gap/ui_progress.go`, `cmd/gap/ui_progress_unix.go`, `cmd/gap/ui_progress_test.go`, `cmd/gap/ui_progress_unix_test.go`
- Change:
  - Replace the raw ANSI scroll-region renderer (`\x1b[1;...r`) in `cmd/gap/ui_progress.go` with an inline Bubble Tea v2 model (`v.AltScreen = false`) run via `tea.NewProgram(model, tea.WithInput(nil), tea.WithOutput(os.Stderr), tea.WithoutSignalHandler())`.
  - Keep `progressBar` / `activeProgress` nil-safe off a terminal (`!isTerminal(os.Stdout) || !isTerminal(os.Stderr) || os.Getenv("TERM") == "dumb"`), and expose `startProgress`, `plan(steps []progressStep)`, `step(label, hue)`, `item(done, total, name)`, `detail(text)`, `add(n)`, `println(line)`, `suspended(fn)`, and `stop()`.
  - Render a live multi-line inline view using `bubbles/v2/spinner` (MiniDot) and GAP's `palette`:
    - Step checklist showing completed steps (`✓ <label>  <duration>`), the active step (`<spinner> <label>  <done>/<total> <name>` and optional `<detail>`), and upcoming planned steps (`· <label>`), windowed if taller than the terminal.
    - Support both pre-seeded steps (`plan(steps)`, where `step(label, hue)` advances the matching planned entry) and dynamic steps (when `plan` was not called, as in `gap update` / `check.go`, where `step(label, hue)` appends a new entry).
    - Keep `detail(text)` orthogonal to `subDone`/`subTotal` so transient status text (such as `spin("Fetching ...")`) never resets sub-item counts or ETA.
    - Progress bar line below the list with GAP gradient fill, interpolated percent (incorporating sub-item progress), `done/total`, elapsed `MM:SS`, and ETA (`~Xs left` / `~XmYYs left`, shown once >= 1s of work has elapsed and remaining > 0).
  - Implement `suspended(fn func())` around `Program.ReleaseTerminal()` and `Program.RestoreTerminal()` so stdin prompts and subprocess output never race with the inline renderer; implement `println(line)` via `Program.Println(strings.TrimRight(line, "\n"))` so log lines print cleanly above the live view without double newlines.
  - Keep `restoreOnSignal` in `cmd/gap/ui_progress_unix.go` calling `p.stop()` (ending the Bubble Tea program cleanly so terminal state is restored) before re-raising `SIGINT`/`SIGTERM`/`SIGHUP` (`cmd/gap/ui_progress_other.go` keeps its no-op signature unchanged).
  - Update `cmd/gap/ui_progress_test.go` (`TestProgressBarModel`, `TestProgressBarOffWhenPiped`) to test model state transitions, pre-seeded and dynamic steps, live checklist rendering, spinner/clock tick, sub-item percentage interpolation, `detail()` preserving ETA, narrow-width truncation, nil safety, and piped no-op; update `cmd/gap/ui_progress_unix_test.go` (`TestProgressBarResetsTerminalOnInterrupt`) to verify `SIGINT` terminates cleanly with restored terminal state.
- Done when: `ui_progress.go` uses Bubble Tea v2 with no raw scroll-region escape codes, renders the live step list + spinner + progress bar + elapsed + ETA, suspends/resumes cleanly around prompts, and stays nil when piped.
- Verify: `go test -v ./cmd/gap -run 'TestProgressBarModel|TestProgressBarOffWhenPiped|TestProgressBarResetsTerminalOnInterrupt'`

### U2. Wire step plan, sub-item progress, prompt suspension, and spinner delegation into `gap sync` and `gap update`
- Depends on: U1
- Files: `cmd/gap/main.go`, `cmd/gap/components.go`, `cmd/gap/skills.go`, `cmd/gap/trust.go`, `cmd/gap/provision.go`, `cmd/gap/style.go`, `cmd/gap/ui_spinner.go`
- Change:
  - In `syncAll` (`cmd/gap/main.go`), construct the ordered step list up front from selected groups (`fetch sources` when components are selected; `policy` when selected; one step per selected component id with its group hue; `commands and MCP`, `skills: upgrade unedited`, `skills: install missing`, `skills: check drift` only when components are selected so `--only policy` omits empty skill steps) and seed `startProgress("sync", ...)` via `plan(...)`.
  - Report per-item progress via `activeProgress.item(done, total, name)` in:
    - `syncAll` (`cmd/gap/main.go`) for policy destinations and global commands,
    - `prepareComponentSources` (`cmd/gap/components.go`) for each component source,
    - `upgradeUntouched`, `installSkills`, and `checkDrift` (`cmd/gap/skills.go`, using a `skillCount(entries)` helper) for every skill processed.
  - In `cmd/gap/ui_spinner.go`, when `activeProgress != nil`, delegate `spin(title, fn)` to `activeProgress.detail(title)` (clearing detail on return) instead of spawning a competing stderr spinner goroutine.
  - Wrap interactive prompts and hook execution in `activeProgress.suspended(...)`:
    - `confirmReplace` and `confirmSplice` (`cmd/gap/main.go`),
    - `hookGate.approve` (`cmd/gap/trust.go`),
    - skill drift diff + `[k]eep/[t]ake/[s]kip` prompt in `checkDrift` (`cmd/gap/skills.go`),
    - `runComponentCommand` (`cmd/gap/provision.go`).
  - In `cmd/gap/style.go`, when `r.tty && activeProgress != nil`, route terminal `report` output (`section`, `wrote`, `result`, `done`, `warn`) through `activeProgress.println(...)` so written paths and warnings appear cleanly above the Bubble Tea view without interleaving, while piped/non-TTY output remains byte-for-byte unchanged.
- Done when: `gap sync` on a terminal lists every phase and active item live with working indicator, progress bar, elapsed time, and ETA; `gap update` continues to work with the new progress model; prompts and hooks pause/resume cleanly; non-TTY stdout is unchanged.
- Verify: `go test ./cmd/gap -run 'TestSync|TestProgress|TestSpinner|TestSkill|TestHook|TestDrift|TestUpdate'`

### U3. Verify bare `gap` launches the Bubble Tea TUI with direct PTY coverage
- Depends on: none
- Files: `cmd/gap/tui_e2e_test.go`
- Change:
  - Add `TestBareGapOpensBubbleTeaTUI` in `cmd/gap/tui_e2e_test.go` launching the compiled test binary with zero arguments on a real PTY (with `TERM=xterm-256color`, `NO_COLOR` unset), asserting the Bubble Tea v2 TUI renders (`Items`, `Status`, `Projects`), sending `q`, and asserting exit code 0.
  - Confirm existing `TestTUIEntryNeedsATerminal` continues to verify piped/`NO_COLOR` fallback to help (exit 2).
- Done when: bare `gap` -> Bubble Tea TUI entry and clean quit are directly covered by a passing PTY test alongside existing piped/`NO_COLOR` fallback checks.
- Verify: `go test -v ./cmd/gap -run 'TestBareGapOpensBubbleTeaTUI|TestTUIEntryNeedsATerminal'`

### U4. Update docs, refresh PR #162, and run full verification
- Depends on: U1, U2, U3
- Files: `README.md`, `cmd/gap/AGENTS.md`
- Change:
  - Update `README.md` and `cmd/gap/AGENTS.md` to document the Bubble Tea v2 inline progress UI for `gap sync` (live step checklist, active item, spinner, elapsed time, ETA, prompt suspension) and remove outdated references to the raw ANSI scroll-region bar.
  - Run `go test ./...` and `go build ./cmd/gap`.
  - Update branch `feat/sync-live-progress` (PR #162) with these changes (or open a clean replacement PR if preferred).
- Done when: docs match the Bubble Tea v2 progress implementation, the full test suite passes, and `go build ./cmd/gap` succeeds.
- Verify: `go test ./... && go build ./cmd/gap`

## Out of scope
- Embedding `gap sync`'s picker, trust prompts, and drift diffs as custom modals inside `tuiModel` (pressing `s` in `gap` TUI continues to hand over to `gap sync`, which now renders both its `huh` picker and inline progress in Bubble Tea v2).
- Changing `NO_COLOR=1 gap` on a terminal from help fallback (exit 2) to uncolored TUI (covered by existing `TestTUIEntryNeedsATerminal`).
- Changing piped or non-TTY stdout format for `gap sync` or `gap update`.
- Rebuilding `gap tui` itself (it already uses `charm.land/bubbletea/v2 v2.1.1`).

## todo.txt
- Build Bubble Tea v2 live progress for gap sync and verify bare gap TUI entry (U1-U4) +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-sync-progress-and-tui-plan.md
- After PR #162 updates and merges: run go install ./cmd/gap and verify bare gap and gap sync live in a terminal +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-sync-progress-and-tui-plan.md

Review complete
