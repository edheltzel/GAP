# GAP regroup, 2026-11-27

| # | Action | Ref | Priority | Task | Status / Note |
|---|---|---|---|---|---|
| 1 | Do | `id:gap-tui-vim` | B | T1-T3: Upgrade Bubble Tea v2 TUI with Vim-first keybindings and visual/sensory feedback polish | Not started |
| 2 | Do | `id:gap-axi` | B | A1-A3: Build agent-agnostic GAP AXI interface (`gap axi`, `gap describe`, `--format toon\|json`) | Not started |
| 3 | Open | `gap#131` | — | After PR #162 updates and merges: `go install ./cmd/gap` and verify bare `gap` and `gap sync` | Not started |
| 4 | Wait | `id:gap-mcp` | C | M1-M2: Design and build future `gap mcp` server over GAP AXI core | Blocked on `id:gap-axi` |

**GAP**: last done "Build Bubble Tea v2 live progress for gap sync and verify bare gap TUI entry (U1-U4)" (2026-11-27), "File GitHub issues for GAP AXI, Bubble Tea Vim UX, and future MCP server roadmap after plan approval" (2026-11-27), and "Refresh site TUI screenshots and landing copy (scripts/tui-shots.sh)" (2026-11-27). Resume point: `docs/plans/handoffs/2026-11-05-gap-plans-sync.md`.

1. **Do** `id:gap-tui-vim` (B) T1-T3: Upgrade Bubble Tea v2 TUI with Vim-first keybindings (gg/Ctrl+d/u/h/l/:/local filter/? help) and visual/sensory feedback polish
2. **Do** `id:gap-axi` (B) A1-A3: Build agent-agnostic GAP AXI interface (gap axi dashboard, gap describe Cobra introspection, --format toon|json, structured errors, skills/gap-axi stub)
3. **Open** `gap#131` After PR #162 updates and merges: run go install ./cmd/gap and verify bare gap and gap sync live in a terminal
4. **Wait** `id:gap-mcp` (C) M1-M2: Design and build future gap mcp server (stdio + authenticated Streamable HTTP) over GAP AXI core for cloud agents (Grokbot, Dot, Muse, Hermes). Blocked on `id:gap-axi`
