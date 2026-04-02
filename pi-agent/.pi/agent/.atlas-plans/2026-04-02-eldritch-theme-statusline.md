# Eldritch Theme + Statusline for pi

## Context

Ed uses Eldritch theme across his terminal (Neovim, WezTerm, etc.) and wants pi's TUI to match.
The atlas-statusline.sh in `claudecode/.claude/` shows exactly what he wants in a status bar.

## Decisions

1. **Aesthetic**: Eldritch as a color system only — no cosmic horror terminology in labels
2. **Scope**: Both a theme (colors) + extension (custom footer behavior)
3. **Data points**: Match atlas-statusline.sh feature set
   - pi version
   - model name + version
   - git branch
   - pwd (basename only)
   - session label
   - visual context bar (⛁ gradient blocks)
   - token stats (↑↓, R/W, $)
   - rate limits from Anthropic API (5h + weekly)
   - extra usage / Max plan overage
4. **Color mapping**:
   - `accent` → `#37f499` (Great Old One Green) — primary
   - `success` → `#37f499` (green)
   - `error` → `#f16c75` (R'lyeh Red)
   - `warning` → `#f1fc79` (Gold of Yuggoth)
   - `thinkingHigh` → `#a48cf2` (Lovecraft Purple)
   - All syntax/semantic tokens follow Eldritch's own highlight mappings
5. **Terminal**: pi-only, no terminal sync
6. **Icons**: Nerd Font (matching atlas-statusline.sh)

## Deliverables

### 1. Theme: `~/.pi/agent/themes/eldritch.json`
- All 51 required color tokens mapped to Eldritch palette
- vars: 11 named colors from Eldritch palette
- colors: all 51 tokens referencing vars or direct hex
- `export` section for HTML output

### 2. Extension: `eldritch-statusline.ts`
Location: `~/.pi/agent/extensions/eldritch-statusline.ts`

**Line 1**: `▲ Atlas v{version} · {model} · /{basename} · ⎇ {branch} · {SESSION}`
- pi version: read from `~/.pi/agent/package.json` at runtime
- Nerd font icons: `▲` (atlas logo), `⎇` (branch)
- Uses `ctx.ui.theme` with Eldritch colors

**Line 2**: Usage + context bar + cache + rate limits
- Rate limits: fetched from Anthropic OAuth API with 5-minute cache
  - `GAUGE 5H:{pct}% ↻{reset_time} · WK:{pct}% ↻{reset_time}`
  - Color: green→yellow→orange→red based on percentage
- Context bar: 24 `⛁` blocks with gradient fill
  - Colors: green→yellow→orange→red based on fill %
  - `⚠` prefix if exceeds 200k
- Cache hit rate: `Cache {pct}%` with color gradient
- Extra usage (Max plan): `$used/$limit` in pink

**API integration**:
- OAuth token: `security find-generic-password -s "Claude Code-credentials"` → keychain
- API endpoint: `https://api.anthropic.com/api/oauth/usage`
- Cache file: `~/.pi/agent/memory/STATE/eldritch-usage-cache.json`
- TTL: 300s

## Technical Approach

- TypeScript extension using `ctx.ui.setFooter()` 
- Theme colors via `ctx.ui.theme.fg(colorToken, text)`
- Footer data: `footerData.getGitBranch()`, `ctx.sessionManager`, `ctx.model`
- Async API calls with background refresh (non-blocking)
- Fallback: graceful degradation if keychain/API unavailable
- Re-render on git branch changes via `footerData.onBranchChange()`
- Timer: refresh every 30s for rate limit reset countdown

## Color Mapping Reference

| Token | Value | Eldritch Name |
|-------|-------|-------------|
| accent | #37f499 | Great Old One Green |
| success | #37f499 | Great Old One Green |
| error | #f16c75 | R'lyeh Red |
| warning | #f1fc79 | Gold of Yuggoth |
| muted | #7081d0 | The Old One Purple |
| dim | #323449 | Shallow Depths Grey |
| text | "" | (terminal default) |
| thinkingText | #7081d0 | The Old One Purple |
| ...all 51 tokens | see theme file | |
