# GAP Bubble Tea v2 Vim-First UX, Agent-Agnostic GAP AXI, and Future MCP Server

## Outcome
- **Human UX (Track 1 — Bubble Tea v2 + Vim-first):** Bare `gap` becomes a low-friction, visually responsive, Vim-native terminal UI with unmistakable pane focus, modal status indicators (`NORMAL` / `FILTER` / `CMD` / `SEARCH` / `RUN`), color-coded feedback toasts, instant local `/` filtering, `:` command palette, true Vim chords (`gg`, `gt`/`gT`, `Ctrl+d`/`Ctrl+u`, `H`/`M`/`L`, `h`/`l`, `v`/`V`, `y`), and a `?` grouped keybinding overlay — layered directly on top of the Bubble Tea v2 `gap sync` live progress work (`2026-11-27-gap-sync-progress-and-tui-plan.md`).
- **Agent UX (Track 2 — GAP AXI):** GAP stays 100% agent-agnostic and harness-agnostic while giving any agent a direct, token-efficient, self-describing CLI interface built into `gap` itself: `gap axi` (live compact dashboard + contextual `next_steps`), `gap describe [cmd]` (live Cobra-generated command/flag/safety contract), universal `--format toon|json` with structured errors (`code`, `message`, `hint`, `retry_cmd`, `mutated`), strict non-interactive safety, and a minimal `agentskills.io` discovery stub (`skills/gap-axi/SKILL.md`).
- **Cloud / Remote Agents (Track 3 — Future `gap mcp`):** A future `gap mcp` server (`stdio` + authenticated Streamable HTTP) enables cloud/remote agents without local shell access (Grokbot, Dot, Muse, remote Hermes) to operate GAP through a small, coarse-grained tool surface backed by the exact same typed Go core as GAP AXI — avoiding eager-schema token bloat and enforcing strict read-only / hook-trust safety boundaries.

## Prerequisite (Phase 0)
- `docs/plans/2026-11-27-gap-sync-progress-and-tui-plan.md` (todos `#130`, `#131`): replaces the raw ANSI scroll-region progress bar with Bubble Tea v2 inline progress for `gap sync` and verifies bare `gap` TUI entry via PTY. Track 1 builds directly on that Bubble Tea v2 foundation.

## Architecture Principle: One Typed Core, Three Surfaces
1. **Typed Go domain layer** (`cmd/gap/`): returns structured Go structs and typed errors instead of writing human-formatted strings directly to stdout.
2. **Human surface**: Bubble Tea v2 TUI + Lipgloss CLI rendering (visual hierarchy, live spinners, color-coded pills, Vim ergonomics).
3. **Agent CLI surface (GAP AXI)**: compact TOON / JSON encoding, live Cobra introspection, structured errors, and contextual next steps — zero harness lock-in, matching the `gh-axi` (`axi.md`) pattern where AXI outperforms both raw CLIs and eager-schema MCP servers in token cost and task success.
4. **Remote MCP surface (`gap mcp`, future)**: thin MCP adapter over the same typed Go functions for cloud/sandboxed agents that cannot shell out locally.

## Implementation Units

### Track 1: Bubble Tea v2 Visual/Sensory Polish & Vim-First UX

#### T1. Vim navigation, chords, pane focus, local filter, and `:` command palette
- Depends on: Phase 0 (`2026-11-27-gap-sync-progress-and-tui-plan.md` U1–U4)
- Files: `cmd/gap/tui_model.go`, `cmd/gap/tui_view.go`, `cmd/gap/tui_search.go`, `cmd/gap/tui_vim.go` (new)
- Change:
  - Add a lightweight Vim key-state handler (`pendingKey` for two-key chords like `gg`, `gt`, `gT`, with a visible pending-key indicator in the statusline so chords feel crisp and predictable):
    - Motions: `j`/`k`, `gg`/`G`, `Ctrl+d`/`Ctrl+u` (half-page), `Ctrl+f`/`Ctrl+b` (full-page), `H`/`M`/`L` (top/middle/bottom of visible window).
    - Pane & tab focus: `h`/`l` switches focus between list pane and scrollable detail viewport (so detail scrolls directly without an `Enter` modal hop); `gt`/`gT` and `1`/`2`/`3`/`Tab`/`Shift+Tab` switch tabs.
    - Selection & yank: `v` toggles mark (alongside `space`), `V` starts visual-line range mark, `Esc` clears marks/filter; `y` yanks selected item ID/repo/commit (OSC 52 clipboard sequence + toast confirmation).
  - Separate **instant local filter** from **remote catalog search**:
    - `/` enters `modeFilter` on `Items` and `Projects` for live incremental filtering with match highlighting, `Enter` locks filter, `n`/`N` jumps matches, `Esc` clears.
    - `S` (and `:search`) opens remote `skills.sh` catalog search (`modeSearch`), where `Esc` toggles between query typing and Vim normal navigation (`j`/`k`/`Enter`) in results.
  - Add `:` ex-command palette (`modeCmd`) with live prefix completion for common actions (`:sync`, `:update [id]`, `:check [id]`, `:doctor`, `:restore [id]`, `:search <q>`, `:q`).
- Done when: All Vim motions, chords, pane focus, local filter, and `:` commands work smoothly without conflicting with existing `modeForm` / `modeConfirm` / `modeRun` interactions.
- Verify: `go test -v ./cmd/gap -run 'TestTUI'`

#### T2. Visual & sensory feedback polish and `?` grouped help overlay
- Depends on: T1
- Files: `cmd/gap/tui_style.go`, `cmd/gap/tui_view.go`, `cmd/gap/tui_run.go`, `cmd/gap/tui_model.go`
- Change:
  - **Active-pane visual contrast:** Render the focused pane border in `pal.Accent` (and bold pane title) and unfocused pane border in `pal.Border` so focus state is immediately obvious.
  - **Modal Vim statusline & color-coded toast pills:** Replace plain unstyled `m.note` with a structured status bar showing:
    - Color-coded mode pill (`NORMAL`, `FILTER`, `CMD`, `SEARCH`, `RUN`, `CONFIRM`),
    - Position/filter/mark telemetry (`3/24`, `filtered: 4`, `2 marked`, pending chord like `g`),
    - Color-coded transient feedback pill (`ok` / `warn` / `bad`) with subtle spinner cue when background refresh/check is active.
  - **Live run telemetry:** In `modeRun`, display per-job elapsed duration badges and estimated remaining time alongside the existing progress meter and live output pane.
  - **Clean footer + `?` help overlay:** Keep the bottom bar concise (5–6 highest-signal keys for the current mode + `? help`) instead of truncating 18 bindings with `…`, and add `modeHelp` toggled by `?` showing grouped bindings (Navigation, Filter & Search, Actions, Ex Commands).
- Done when: Focus state, active mode, background activity, and action feedback are immediately legible at a glance on both dark and light palettes, and narrow terminals degrade cleanly.
- Verify: `go test -v ./cmd/gap -run 'TestTUI'`

#### T3. TUI unit/PTY verification and documentation
- Depends on: T2
- Files: `cmd/gap/tui_test.go`, `cmd/gap/tui_e2e_test.go`, `README.md`, `cmd/gap/AGENTS.md`
- Change:
  - Add unit tests covering `gg`/`G`, `Ctrl+d`/`Ctrl+u`, `H`/`M`/`L`, `h`/`l` pane focus & detail scrolling, local `/` filter + `n`/`N`, `:` command dispatch, `v`/`V` visual marking, `?` help overlay, and statusline mode pills.
  - Extend `cmd/gap/tui_e2e_test.go` with PTY coverage for entering `/` filter, `?` help, and `:q` clean exit.
  - Update `README.md` and `cmd/gap/AGENTS.md` with the Vim-first keymap and updated TUI screenshots/notes as appropriate.
- Done when: Full TUI unit + PTY test suite passes and docs accurately reflect all bindings.
- Verify: `go test ./... && go build ./cmd/gap`

---

### Track 2: Agent-Agnostic CLI Understanding & GAP AXI

#### A1. Shared structured output (`--format toon|json`), error envelope, and non-interactive safety
- Depends on: none (can proceed in parallel with Track 1 after Phase 0, or immediately after Track 1)
- Files: `cmd/gap/axi.go` (new), `cmd/gap/axi_toon.go` (new), `cmd/gap/cli.go`, `cmd/gap/main.go`, `cmd/gap/status.go`, `cmd/gap/doctor.go`, `cmd/gap/check.go`, `cmd/gap/search.go`, `cmd/gap/list.go` (or existing files housing those handlers)
- Change:
  - Introduce persistent root flags `--format <human|toon|json>` (default `human`) and `--axi` (shorthand for `--format toon` + non-interactive agent mode; also honored via `GAP_FORMAT=toon|json`).
  - Implement a compact TOON encoder (`cmd/gap/axi_toon.go`) aligned with `gh-axi` conventions (compact key-value headers, tabular arrays, bounded string truncation with explicit truncation notice, `--full` escape hatch).
  - Define a uniform structured error envelope for `toon`/`json` modes:
    - `code` (e.g., `INTERACTION_REQUIRED`, `VALIDATION_ERROR`, `UNTRUSTED_HOOK`, `DRIFT_DETECTED`, `NOT_FOUND`, `IO_ERROR`),
    - `message`, `hint`, `retry_cmd` (exact copy-pasteable non-interactive command), and `mutated` (`true`/`false` so agents know whether a mutation landed before retrying).
  - Guarantee **strict non-interactive safety** in `toon`/`json`/`--axi` mode: never block on stdin prompts (`huh`, `confirmReplace`, `confirmSplice`, hook trust, drift prompt); fail immediately with `INTERACTION_REQUIRED` / `UNTRUSTED_HOOK` and the exact non-interactive flags/commands.
  - Wire typed structured output into core commands first: `status`, `list`, `doctor`, `check`, `search`, `source`, `add`, `sync`, `update`, `remove`, `restore`, `pin`, `init`, attaching contextual `next_steps` suggestions (e.g., when drift or pending updates exist).
- Done when: Agents can invoke core `gap` commands with `--axi` / `--format toon|json`, receive compact deterministic output and actionable errors, and never hang on a prompt.
- Verify: `go test -v ./cmd/gap -run 'TestAXI|TestFormat'`

#### A2. Self-describing CLI interface: `gap axi` dashboard & `gap describe [command]`
- Depends on: A1
- Files: `cmd/gap/axi_describe.go` (new), `cmd/gap/cli.go`, `cmd/gap/axi_test.go` (new)
- Change:
  - Add `gap axi` (zero-arg agent entry point): returns a live, compact dashboard in TOON (or `--json`) containing GAP version, source checkout, active project root, detected harnesses, component/utility counts, drift & pending-update counts, a one-line command index, and contextual `next_steps`. Read failures in individual sections degrade gracefully (reporting `unknown` for that section while preserving other sections, matching `gh-axi`).
  - Add `gap describe [command]` (and `gap axi schema [command]` alias): introspects the live Cobra command tree at runtime and outputs a structured machine-readable spec:
    - Command path, purpose, arguments, flags (name, shorthand, type, default, required, mutually exclusive groups),
    - Safety metadata (`read_only: true|false`, `mutates_lock`, `may_prompt`, `non_interactive_flags`),
    - Canonical agent examples.
  - Add a crisp one-line hint in non-TTY bare `gap` help output pointing agents to `gap axi` and `gap describe` (preserving existing exit code `2` so nothing breaks).
  - Add contract tests verifying every registered Cobra subcommand is represented in `gap describe` with valid safety metadata.
- Done when: Any agent from any harness can learn GAP's full live capabilities and state in 1–2 low-token calls without reading static prose docs.
- Verify: `go test -v ./cmd/gap -run 'TestAXIDashboard|TestDescribe'`

#### A3. Harness-agnostic Agent Skill discovery stub & docs
- Depends on: A2
- Files: `skills/gap-axi/SKILL.md` (new), `README.md`, `cmd/gap/AGENTS.md`
- Change:
  - Create a minimal `agentskills.io`-compliant discovery stub (`skills/gap-axi/SKILL.md`, `user-invocable: false`, optional `metadata.hermes` tags, zero harness-specific assumptions) modeled on `gh-axi`: instead of duplicating stale command docs, it instructs any agent to run `gap axi` for live state and `gap describe <cmd>` for exact syntax.
  - Add a test ensuring `skills/gap-axi/SKILL.md` stays minimal and never references removed subcommands.
  - Document GAP AXI in `README.md` and `cmd/gap/AGENTS.md`.
- Done when: Any agent harness supporting Agent Skills or raw shell execution can discover and operate GAP accurately with zero doc drift.
- Verify: `go test ./... && go build ./cmd/gap`

---

### Track 3: Future MCP Server for Cloud & Remote Agents

#### M1. Design & safety spec for `gap mcp` (Grokbot, Dot, Muse, remote Hermes)
- Depends on: A2
- Files: `docs/plans/YYYY-MM-DD-gap-mcp-spec.md` (future spec when Track 2 ships)
- Change:
  - Specify the thin MCP server design layered over A1–A2's typed Go core:
    - **Transports:** `stdio` (local MCP clients) and **Streamable HTTP** (remote/cloud agents like Grokbot, Dot, Muse, Hermes) with explicit bind/auth rules (loopback/tailnet + bearer token; never unauthenticated public bind).
    - **Coarse-grained tool surface (token-efficient):** Keep the tool set intentionally small (6–8 tools: `gap_dashboard`, `gap_describe`, `gap_status`, `gap_list`, `gap_search`, `gap_doctor`, plus opt-in mutating tools `gap_sync`, `gap_update`, `gap_add`, `gap_restore`) so eager MCP tool schemas don't bloat context windows.
    - **Security & mutation boundaries:** Read-only by default (`--read-only` / explicit `--allow-mutations`), refuse untrusted component hooks over remote MCP unless pre-approved in `gap.toml`, never expose arbitrary shell passthrough, and return the same structured `mutated` + `hint` fields as GAP AXI.
- Done when: Security model, transport auth, and tool schemas are reviewed and approved before writing server code.

#### M2. Implement `gap mcp` server over the shared Go core
- Depends on: M1
- Files: `cmd/gap/mcp.go`, `cmd/gap/mcp_test.go`, `go.mod`
- Change:
  - Implement `gap mcp [--transport stdio|http] [--listen addr] [--read-only]` using the official Go MCP SDK (`github.com/modelcontextprotocol/go-sdk/mcp`), delegating directly to the A1–A2 typed Go functions.
  - Add end-to-end MCP client-server tests covering read tools, read-only mutation refusal, untrusted hook rejection, and structured error responses.
- Done when: Local and remote MCP clients can inspect and (when explicitly permitted) manage GAP safely with zero duplicated business logic.
- Verify: `go test -v ./cmd/gap -run 'TestMCP'`

---

## Proposed GitHub Issues (to file after plan review)
Because `edheltzel/GAP` is public, these are drafted here first and ready to create via `gh issue create` on approval:

1. **`feat: agent-agnostic CLI introspection and GAP AXI interface (gap axi, gap describe, --format toon|json)`** (`enhancement`, `type:feature`)
   - Covers Track 2 (A1–A3): harness-agnostic agent CLI understanding, live Cobra schema introspection, token-efficient TOON/JSON output, structured error envelopes, non-interactive safety, and minimal `agentskills.io` discovery stub.
2. **`feat: Vim-first UX and visual/sensory polish for Bubble Tea v2 TUI`** (`enhancement`, `type:feature`)
   - Covers Track 1 (T1–T3): Vim chords/motions (`gg`, `Ctrl+d`/`u`, `H`/`M`/`L`, `h`/`l` pane focus), local `/` filter vs. remote `S` search, `:` command palette, active-pane border contrast, modal statusline pills, and `?` help overlay.
3. **`roadmap: future gap mcp server for cloud and remote agents`** (`roadmap`, `type:feature`, `safety`)
   - Covers Track 3 (M1–M2): future thin MCP server (`stdio` + authenticated Streamable HTTP) over the GAP AXI Go core for cloud/remote agents (Grokbot, Dot, Muse, remote Hermes) with coarse-grained tools and strict read-only/trust boundaries.

## Out of scope
- Coupling GAP to any single agent harness or vendor-specific protocol (all agent features stay harness-agnostic).
- Building a separate Node/TypeScript `gap-axi` wrapper repository that duplicates GAP CLI flags and risks drifting from the Go binary.
- Exposing arbitrary shell execution or auto-approving untrusted component hooks over remote MCP.
- Replacing the CLI with MCP for local shell agents (CLI/AXI remains the primary, fastest, most token-efficient interface for local agents).

## todo.txt
- (B) T1-T3: Upgrade Bubble Tea v2 TUI with Vim-first keybindings (gg/Ctrl+d/u/h/l/:/local filter/? help) and visual/sensory feedback polish +gap id:gap-tui-vim note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-tui-vim-axi-mcp-plan.md
- (B) A1-A3: Build agent-agnostic GAP AXI interface (gap axi dashboard, gap describe Cobra introspection, --format toon|json, structured errors, skills/gap-axi stub) +gap id:gap-axi note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-tui-vim-axi-mcp-plan.md
- (C) M1-M2: Design and build future gap mcp server (stdio + authenticated Streamable HTTP) over GAP AXI core for cloud agents (Grokbot, Dot, Muse, Hermes) +gap id:gap-mcp dep:gap-axi note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-tui-vim-axi-mcp-plan.md
- File GitHub issues for GAP AXI, Bubble Tea Vim UX, and future MCP server roadmap after plan approval +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-tui-vim-axi-mcp-plan.md
