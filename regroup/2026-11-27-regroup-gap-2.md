# Regroup: GAP

Date: 2026-11-27 (second check-in today, refreshed after #170/#171 merged)
Scope: GAP
Source: `/Users/ed/Developer/Atlas/GAP/todo.txt` and `done.txt`, previous file `regroup/2026-11-27-regroup-gap.md`, open PRs on edheltzel/GAP (`gh pr list`), newest handoff `docs/plans/handoffs/2026-11-05-gap-plans-sync.md`.

Resume point: `docs/plans/handoffs/2026-11-05-gap-plans-sync.md`.

## Last completed

- 2026-11-27: Build Bubble Tea v2 live progress for gap sync and verify bare gap TUI entry (U1-U4), PR #162.
- 2026-11-27: File GitHub issues for GAP AXI, Vim TUI, and MCP roadmap (#166, #167, #168).
- 2026-11-27: Refresh site TUI screenshots and landing copy, PR #169.
- Not tracked in todo.txt, merged 2026-11-27: #170 Bubble Tea v2 progress follow-ups, #171 the two 2026-11-27 plan docs.

## Up next

- (B) `id:gap-tui-vim` T1-T3: Vim-first keybindings and feedback polish for the Bubble Tea TUI. Not started; this tab takes it (agreed with the site tab).
- (B) `id:gap-axi` A1-A3: agent-agnostic GAP AXI interface. Not started.

## Remaining

- After PR #162 merges: `go install ./cmd/gap` and verify bare `gap` and `gap sync` live in a terminal. Not started. Unblocked now that #162 and #170 merged; best done after #172.
- (C) `id:gap-mcp` M1-M2: future `gap mcp` server. Blocked on `id:gap-axi`.

## Open PRs (not tracked in todo.txt)

- #172 fix(sync): Pullfrog's #170 findings (torn lines under the bar, 500ms prompt pause, bar with redirected stdout). Full `go test ./...` green locally; waiting on CI and Ed's merge.
- Site lane: `fix/site-hero-start` (hero-start CI check flaky on PRs since b68082b5), owned by the site tab.

## Changes since the previous file

- No todo.txt line changed. done.txt now exists (site tab archived done tasks).
- #170 and #171 merged; #172 opened for the review fixes #170 merged without.

## Raw lines

```text
2026-11-27 After PR #162 updates and merges: run go install ./cmd/gap and verify bare gap and gap sync live in a terminal +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-sync-progress-and-tui-plan.md
(B) 2026-11-27 T1-T3: Upgrade Bubble Tea v2 TUI with Vim-first keybindings (gg/Ctrl+d/u/h/l/:/local filter/? help) and visual/sensory feedback polish +gap id:gap-tui-vim note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-tui-vim-axi-mcp-plan.md
(B) 2026-11-27 A1-A3: Build agent-agnostic GAP AXI interface (gap axi dashboard, gap describe Cobra introspection, --format toon|json, structured errors, skills/gap-axi stub) +gap id:gap-axi note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-tui-vim-axi-mcp-plan.md
(C) 2026-11-27 M1-M2: Design and build future gap mcp server (stdio + authenticated Streamable HTTP) over GAP AXI core for cloud agents (Grokbot, Dot, Muse, Hermes) +gap id:gap-mcp dep:gap-axi note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-tui-vim-axi-mcp-plan.md
```
