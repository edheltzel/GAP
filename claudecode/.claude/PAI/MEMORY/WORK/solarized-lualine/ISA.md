---
task: Add solarized palettes (dark+light) to NEO.ED lualine theme dispatcher
slug: solarized-lualine
effort: E2
phase: complete
progress: 16/16
mode: ALGORITHM
started: 2026-05-18
updated: 2026-05-18
---

## Problem

NEO.ED lualine theme system supports eldritch and rose-pine variants, but the user has now installed `maxmx03/solarized.nvim` and configured solarized as the active LazyVim colorscheme. With no solarized palette in the dispatcher, NEO.ED falls back to eldritch — wrong colors for the active theme, and especially wrong for solarized-light where eldritch's dark palette is unreadable.

## Goal

Extend NEO.ED so that when `vim.g.colors_name == "solarized"` it dispatches to a new `plugins.ui.lualine.solarized` palette module, picks the dark or light variant based on `vim.o.background`, and exposes the standard 10-color NEO.ED palette keys so all existing lualine sections render correctly.

## Criteria

- [ ] ISC-1: `lua/plugins/ui/lualine/solarized.lua` exists with `M.palettes["solarized-dark"]` and `M.palettes["solarized-light"]`.
- [ ] ISC-2: Each palette defines all 10 NEO.ED keys: `darker`, `bg`, `darkGray`, `fg`, `gray`, `green`, `blue`, `purple`, `red`, `magenta`.
- [ ] ISC-3: `solarized.lua` exposes `M.get_colors(variant)` returning `(colors, overrides)` matching the rose-pine signature.
- [ ] ISC-4: `get_colors` detects background variant via `vim.o.background == "light"` when called with `"solarized"`.
- [ ] ISC-5: `neoed.lua` `get_colors()` dispatches `colorscheme == "solarized"` to the new module.
- [ ] ISC-6: `neoed.lua` header comment SUPPORTED COLORSCHEMES lists solarized variants.
- [ ] ISC-7: `autocmds.lua` ColorScheme handler clears `package.loaded["plugins.ui.lualine.solarized"]`.
- [ ] ISC-8: `:LualineRefresh` user command also clears the solarized cache key.
- [ ] ISC-9: README.md Supported Colorschemes table includes solarized dark and light.
- [ ] ISC-10: Dark palette uses solarized base03/base02/base0/base1 for bg/text roles (verified vs upstream palette.lua).
- [ ] ISC-11: Light palette uses solarized base3/base2/base00/base01 for bg/text roles (light-on-light readable).
- [ ] ISC-12: Accent colors (red, green, blue, magenta, violet=purple) match canonical Ethan Schoonover solarized hex values across both variants.
- [ ] ISC-13: `nvim --headless --noplugin -c "luafile lua/plugins/ui/lualine/solarized.lua" -c "qa"` returns exit code 0 (no syntax errors).
- [ ] ISC-14: `nvim --headless -c "lua require('plugins.ui.lualine.neoed').setup('solarized')" -c "qa"` returns exit code 0 (dispatcher works end-to-end with lazy-loaded modules).
- [ ] ISC-15: Anti: rose-pine and eldritch dispatch paths in `neoed.lua` are unchanged (grep confirms both branches intact).
- [ ] ISC-16: Anti: no hardcoded paths, no XML tags, no Python — pure Lua additions only.

## Test Strategy

| isc | type | check | threshold | tool |
|-----|------|-------|-----------|------|
| 1-2 | file | new palette file with both variant tables | both keys present | Read |
| 3-4 | code | get_colors signature + background detection | matches rose-pine pattern | Read |
| 5-6 | code | neoed.lua dispatch + header | new elseif branch + doc | Read |
| 7-8 | code | autocmds cache clear in both places | two new lines | Read |
| 9 | doc | README table row | new row exists | Read |
| 10-12 | color | hex values match upstream palette.solarized | exact match | Read+grep upstream |
| 13-14 | syntax | nvim headless luafile + setup() | exit 0 | Bash |
| 15-16 | regression | grep eldritch/rose-pine branches still present | found in neoed.lua | Grep |

## Decisions

- 2026-05-18: Solarized variant detection uses `vim.o.background` because solarized.nvim sets `vim.g.colors_name = "solarized"` regardless of dark/light. This mirrors upstream's own lualine theme at `lua/lualine/themes/solarized.lua`.
- 2026-05-18: Map NEO.ED `purple` slot to solarized `violet` (#6C71C4) — solarized's "purple" name is reserved for magenta-family; violet is the visual-mode accent in upstream's theme.
- 2026-05-18: Light variant uses base2 as `bg` (active surface) and base3 as `darker` (inactive) so the inactive section reads lighter than the active one, matching the dark variant's hierarchy.

## Verification

(populated at VERIFY)
