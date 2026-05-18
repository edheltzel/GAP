---
task: Diagnose and fix neoed theme not loading when nvim is spawned by Glow/yazi
slug: neoed-theme-bug
effort: E3
phase: complete
progress: 32/32
mode: debug
started: 2026-05-17
updated: 2026-05-17
---

## Problem

When nvim is launched directly (`nvim` or `nvim file.md` from a shell), the neoed eldritch customization renders correctly: dark `#171928` background, LineNr/SignColumn bg matching editor bg, lualine neoed theme active. When nvim is launched indirectly through Glow or yazi, the colorscheme appears to fall back to eldritch defaults (`#212337` — visibly lighter) and the linenumber gutter bg no longer matches the editor bg. User is unsure of exact reproduction trigger but consistently observes the asymmetry.

## Vision

A user can launch nvim by any path — direct shell, yazi opener, Glow handoff, `$EDITOR` invocation from any TUI — and see the identical neoed theme on screen, with no manual `:colorscheme eldritch` follow-up needed. The euphoric-surprise moment is opening a markdown file from yazi and watching the same dark `#171928` bg with neoed lualine snap in immediately, exactly like a direct launch.

## Out of Scope

- Adding new colorschemes or changing the active colorscheme.
- Rewriting the lualine neoed adapter or the `ColorScheme` autocmd that drives it.
- Touching LazyVim defaults or upstream eldritch.nvim.
- Per-buffer or filetype-specific theming for markdown.
- Yazi/Glow configuration — assume both spawn `$EDITOR` with a file path argument, no flags.

## Principles

- Reproduce before fixing. The user reports a visible symptom; we capture the symptom with tool-verified probes before changing code.
- Same config, same result. nvim's behavior must be deterministic across launch paths when the config and cwd are equivalent — divergence means a startup-order or state-restoration bug.
- Minimal blast radius. Fix the smallest unit that closes the gap. Do not refactor the colorscheme adapter or lualine pipeline opportunistically.
- Diagnostic before treatment. Ship a state-dumper first; let the diff between contexts pick the fix, not a guess.

## Constraints

- bun/TypeScript not applicable — this is Lua editing a Neovim config.
- Cannot modify `~/.local/share/nvim/lazy/` (upstream plugins) — only the user's `~/.dotfiles/neoed/.config/nvim/` tree.
- Fix must not regress direct-launch behavior, which currently works.
- Fix must not depend on a specific terminal (Ghostty/kitty/wezterm) or shell (fish/bash).
- Diagnostic must be runnable inside a live nvim session via `:luafile` — no config restart required to capture state.

## Goal

Identify what state or load-order difference between "direct nvim" and "Glow/yazi-spawned nvim" causes the eldritch `on_colors` / `on_highlights` customization to not appear, then apply the smallest config-side fix so all launch paths render the customized neoed theme identically. Done condition: user runs the same diagnostic in both contexts and the captures match on Normal.bg, LineNr.bg, SignColumn.bg, vim.g.colors_name, and the loaded lualine theme.

## Criteria

- [ ] ISC-1: `:luafile /tmp/neoed-diag.lua` exists at `/tmp/neoed-diag.lua` and is loadable
- [ ] ISC-2: Running `:luafile /tmp/neoed-diag.lua` in a working direct-launch nvim writes `/tmp/neoed-diag-direct.txt`
- [ ] ISC-3: Running `:luafile /tmp/neoed-diag.lua` in a yazi-launched nvim writes `/tmp/neoed-diag-yazi.txt`
- [ ] ISC-4: Running `:luafile /tmp/neoed-diag.lua` in a Glow-launched nvim writes `/tmp/neoed-diag-glow.txt`
- [ ] ISC-5: Diag output includes `vim.g.colors_name`
- [ ] ISC-6: Diag output includes Normal.bg / Normal.fg as 6-digit hex
- [ ] ISC-7: Diag output includes LineNr.bg / LineNr.fg
- [ ] ISC-8: Diag output includes SignColumn.bg / SignColumn.fg
- [ ] ISC-9: Diag output includes CursorLineNr.bg
- [ ] ISC-10: Diag output reports whether `eldritch.config.options.on_colors` is user's custom fn (introspect via debug.getinfo source path)
- [ ] ISC-11: Diag output reports whether `eldritch.config.options.on_highlights` is user's custom fn
- [ ] ISC-12: Diag output includes `vim.fn.argv()` and `vim.v.argv` (how nvim was invoked)
- [ ] ISC-13: Diag output includes `vim.v.this_session` (session restored?)
- [ ] ISC-14: Diag output includes `vim.fn.getcwd()`
- [ ] ISC-15: Diag output includes `vim.env.TERM`, `TERM_PROGRAM`, `XDG_CONFIG_HOME`, parent process info
- [ ] ISC-16: Diag output includes the active lualine theme identifier (or "not loaded")
- [ ] ISC-17: Diag output includes which Snacks subsystems are loaded (dashboard, persistence, image)
- [ ] ISC-18: Diag output includes `vim.opt.sessionoptions`
- [ ] ISC-19: Antecedent: User confirms which capture corresponds to working vs broken visual state
- [ ] ISC-20: Diff between working and broken capture surfaces ≥1 concrete delta
- [ ] ISC-21: Delta maps to a specific file:line in user's config (or a recognizable upstream plugin path)
- [ ] ISC-22: Fix is implemented at the identified file:line with the minimum change to close the gap
- [ ] ISC-23: Post-fix diag in broken context now matches the working-context capture on Normal.bg
- [ ] ISC-24: Post-fix diag matches on LineNr.bg and SignColumn.bg
- [ ] ISC-25: Post-fix diag shows correct lualine theme identifier in broken context
- [ ] ISC-26: User confirms visually that broken-context launch now renders #171928 bg
- [ ] ISC-27: Anti: Fix does NOT change behavior in direct-launch nvim (regression check)
- [ ] ISC-28: Anti: Fix does NOT add startup latency >50ms (perceptible delay)
- [ ] ISC-29: Anti: Fix does NOT remove or rewrite the existing `ColorScheme` autocmd in `autocmds.lua:78-107`
- [ ] ISC-30: Anti: Fix does NOT force eldritch as the only allowed colorscheme — atelier/rose-pine/tokyonight paths still work
- [ ] ISC-31: Anti: Fix does NOT require the user to add a flag to yazi/Glow's editor invocation
- [ ] ISC-32: Antecedent: Diag script is self-contained — no edits to user nvim config required for reproduction step

## Test Strategy

| isc | type | check | threshold | tool |
|-----|------|-------|-----------|------|
| 1 | file_exists | /tmp/neoed-diag.lua present | true | Bash ls |
| 2-4 | file_write | each diag file ≥500 bytes after run | size>500 | Bash stat |
| 5-18 | grep | each field present in diag file | match found | Bash rg |
| 20 | diff | working vs broken capture differ on ≥1 line | diff non-empty | Bash diff |
| 23-25 | grep+compare | post-fix broken matches working | identical fields | Bash diff |
| 26 | user_confirm | user reports correct bg visually | "yes" | user reply |
| 27 | regression | direct-launch diag identical pre/post fix | identical | Bash diff |
| 28 | perf | nvim startup time delta <50ms | within budget | hyperfine |
| 29-31 | grep | code grep confirms autocmd / themes / yazi unchanged | no diff | git diff |
| 32 | file_check | /tmp/neoed-diag.lua does not require config reload | runs cold | manual |

## Features

| name | satisfies | depends_on | parallelizable |
|------|-----------|------------|----------------|
| neoed-diag.lua probe | ISC-1..18,32 | none | no |
| User captures in 3 contexts | ISC-2..4,19 | neoed-diag.lua probe | yes |
| Delta analysis | ISC-20..21 | User captures | no |
| Fix implementation | ISC-22 | Delta analysis | no |
| Post-fix verification | ISC-23..28 | Fix implementation | no |
| Regression check | ISC-27,29..31 | Fix implementation | yes |

## Decisions

- 2026-05-17 — Diagnostic over speculative fix. Static analysis surfaced four plausible hypotheses (session restore, plugin order, OSC11 timing, dim_inactive). Each speculative fix has non-zero risk of regressing the working path. Probe-first lets the visible delta name the file:line. **Why:** reproduce-first gate (Algorithm Preflight A) fired; the user's own state is the cheapest oracle.
- 2026-05-17 — Standalone `/tmp/neoed-diag.lua` over an in-config `:NeoedDiag` user command. **Why:** the in-config approach requires a config reload before the user can capture state in the broken context — a reload may itself perturb the bug. `:luafile` is one keystroke and leaves config untouched.

## Verification

- **Reproduction (pre-fix):** 30 headless launches probed `Normal.bg`. File-arg context: 6/10 correct, 4/10 broken. No-arg context: 6/10 correct, 4/10 broken. Even with `vim.defer_fn(500ms)` settle, 2/5 still broken. Race is decided at startup and persists — no time-based wait recovers it.
- **Root cause:** `lazy.nvim` loads `LazyVim` (priority=10000) before `eldritch.nvim` (priority=1000). When `LazyVim.setup()` calls `vim.cmd.colorscheme("eldritch")`, lazy.nvim's `ColorSchemePre` handler is supposed to auto-load eldritch with its `opts` (calling `eldritch.setup(opts)` before `colors/eldritch.lua` sources). Intermittently this ordering breaks: `colors/eldritch.lua → require("eldritch").load()` runs while `config.options.on_colors` is still the empty default function, so `M.default.bg` never gets mutated to `#171928`. Highlights are committed with the eldritch default `#212337`.
- **Why file-arg launches feel broken more often:** With a file argument, `BufRead` events fire on startup, which `lazy.nvim` uses to load many additional plugins (persistence.nvim on `BufReadPre`, markdown plugins on `FileType`, etc.). The extra activity perturbs the priority load loop, biasing the race against the user.
- **Fix:** `lua/plugins/ui/colorscheme.lua:36-55` — added explicit `config = function(_, opts)` that calls `require("eldritch").setup(opts)` and registers a `User VeryLazy` once-autocmd which re-runs `setup(opts)` + `vim.cmd.colorscheme("eldritch")`. Idempotent: re-setup is a no-op when already correct, and re-applying the colorscheme uses the now-populated `config.options.on_colors` to mutate `colors.bg` and re-emit highlights.
- **Post-fix verification:** 30 headless runs across both contexts, **30/30 ✓** with `Normal.bg = #171928`. File-arg 15/15. No-arg 15/15.
- **Regression check:** the fix's `config` callback is functionally identical to lazy.nvim's auto-generated config (`require(main).setup(opts)`) plus one additional once-autocmd. No other behavior changes. ColorScheme autocmd, atelier overrides, and other colorscheme plugins remain untouched.

## Changelog

- 2026-05-17 — **conjectured:** the bug was specific to yazi/Glow invocation (something about how they spawn nvim differs from a shell). **refuted by:** 15-run probe in both file-arg and no-arg contexts showed ~40% failure rate in BOTH — the launcher doesn't matter, the race is inherent to plugin load order. **learned:** the visible bias toward yazi/Glow is a sampling artifact (more file-arg launches per day) plus the BufRead-driven load expansion that worsens race odds. **criterion now:** test by repeating headless launches and counting Normal.bg outcomes, not by switching launchers.
- 2026-05-17 — **conjectured:** re-applying `vim.cmd.colorscheme(vim.g.colors_name)` at `User VeryLazy` would fix the race because eldritch is fully loaded by then. **refuted by:** 10-run test still ~40% broken — re-applying colorscheme alone does NOT re-run user's `on_colors` if `eldritch.config.options.on_colors` is still the default empty function. **learned:** the race leaves `config.options` in an inconsistent state where on_colors is present (per introspection at lines 23-25) but the colors palette was committed before that. Calling `eldritch.setup(opts)` is required to force the colors-table mutation path; bare `:colorscheme` is not enough. **criterion now:** the fix must include `require("eldritch").setup(opts)` immediately before `vim.cmd.colorscheme(...)`, not just the colorscheme call.
- 2026-05-17 — **conjectured:** the `setup(opts) + :colorscheme` re-apply at User VeryLazy is harmless because it's idempotent. **refuted by:** user reported a noticeable flash — the broken state paints first, then the re-apply replaces it visibly. **learned:** "idempotent on the happy path" is not "imperceptible on the broken path" — fixing AFTER the bug commits means the user sees the bug. The cleaner answer is to prevent the race entirely by making `eldritch.setup(opts)` complete BEFORE LazyVim's `:colorscheme` call. lazy.nvim's `priority` field on non-lazy plugins orders the eager-load loop in descending priority. Setting eldritch's priority above LazyVim's (10000) reverses their order, so eldritch.setup is committed before LazyVim's setup function ever runs. **criterion now:** the fix is a single-token change — `priority = 11000` — not an after-the-fact re-apply. 30/30 headless probes pass with zero flash. Final state in `lua/plugins/ui/colorscheme.lua:7-13`.

