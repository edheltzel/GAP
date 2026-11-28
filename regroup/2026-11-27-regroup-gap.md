# Regroup: GAP

Date: 2026-11-27
Scope: GAP
Source: `/Users/ed/Developer/Atlas/GAP/todo.txt` (`tuxedo lsa --json`; `done.txt` does not exist), `/Users/ed/todo.txt` and `/Users/ed/done.txt` (`tuxedo lsa --json`; route only, no other `+gap` tasks), previous file `regroup/2026-11-26-regroup-gap.md`, newest handoff `docs/plans/handoffs/2026-11-05-gap-plans-sync.md`.

**GAP**: last done "Build Bubble Tea v2 live progress for gap sync and verify bare gap TUI entry (U1-U4)" (2026-11-27), "File GitHub issues for GAP AXI, Bubble Tea Vim UX, and future MCP server roadmap after plan approval" (2026-11-27), and "Refresh site TUI screenshots and landing copy (scripts/tui-shots.sh)" (2026-11-27). Resume point: `docs/plans/handoffs/2026-11-05-gap-plans-sync.md`.

Note: `#<n>` refs (`gap#131`) are valid only until `todo.txt` changes.

1. **Do** `id:gap-tui-vim` (B) T1-T3: Upgrade Bubble Tea v2 TUI with Vim-first keybindings (gg/Ctrl+d/u/h/l/:/local filter/? help) and visual/sensory feedback polish. Not started
2. **Do** `id:gap-axi` (B) A1-A3: Build agent-agnostic GAP AXI interface (gap axi dashboard, gap describe Cobra introspection, --format toon|json, structured errors, skills/gap-axi stub). Not started
3. **Open** `gap#131` After PR #162 updates and merges: run go install ./cmd/gap and verify bare gap and gap sync live in a terminal. Not started
4. **Wait** `id:gap-mcp` (C) M1-M2: Design and build future gap mcp server (stdio + authenticated Streamable HTTP) over GAP AXI core for cloud agents (Grokbot, Dot, Muse, Hermes). Blocked on `id:gap-axi`

## Raw lines

### Open

```text
(B) 2026-11-27 T1-T3: Upgrade Bubble Tea v2 TUI with Vim-first keybindings (gg/Ctrl+d/u/h/l/:/local filter/? help) and visual/sensory feedback polish +gap id:gap-tui-vim note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-tui-vim-axi-mcp-plan.md
(B) 2026-11-27 A1-A3: Build agent-agnostic GAP AXI interface (gap axi dashboard, gap describe Cobra introspection, --format toon|json, structured errors, skills/gap-axi stub) +gap id:gap-axi note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-tui-vim-axi-mcp-plan.md
2026-11-27 After PR #162 updates and merges: run go install ./cmd/gap and verify bare gap and gap sync live in a terminal +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-sync-progress-and-tui-plan.md
(C) 2026-11-27 M1-M2: Design and build future gap mcp server (stdio + authenticated Streamable HTTP) over GAP AXI core for cloud agents (Grokbot, Dot, Muse, Hermes) +gap id:gap-mcp dep:gap-axi note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-tui-vim-axi-mcp-plan.md
```

### Completed (2026-11-27)

```text
x 2026-11-27 2026-11-27 Build Bubble Tea v2 live progress for gap sync and verify bare gap TUI entry (U1-U4) +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-sync-progress-and-tui-plan.md pr:https://github.com/edheltzel/GAP/pull/162
x 2026-11-27 2026-11-27 File GitHub issues for GAP AXI, Bubble Tea Vim UX, and future MCP server roadmap after plan approval +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-27-gap-tui-vim-axi-mcp-plan.md issues:https://github.com/edheltzel/GAP/issues/163,164,165 — filed #166 (GAP AXI), #167 (Vim TUI), #168 (MCP roadmap)
x 2026-11-27 2026-11-27 Refresh site TUI screenshots and landing copy (scripts/tui-shots.sh) +gap
```
