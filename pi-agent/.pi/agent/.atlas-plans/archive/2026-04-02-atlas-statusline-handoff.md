---
slug: 2026-04-02-atlas-statusline-handoff
title: "Atlas Statusline Extension for pi"
effort: medium
phase: completed ✅
started: 2026-04-02T12:00:00-05:00
updated: 2026-04-02T12:30:00-05:00
mode: implement
---

## Context

Goal: Create a custom footer extension for pi's TUI that renders an Atlas-style two-line status bar with the Eldritch color theme. This complements the `eldritch.json` color theme.

### Deliverables

1. **Theme**: `~/.pi/agent/themes/eldritch.json` — full 51-token Eldritch color theme (✅ done, 51/51 tokens)
2. **Extension**: `~/.pi/agent/extensions/atlas-footer.ts` — custom footer with Atlas-style layout (⚠️ partially done, issues remain)

---

## What Was Built

### Theme: `eldritch.json`

Full 51-token Eldritch color mapping. Key tokens:
- `accent` → `#37f499` (Lovecraft Green — Great Old One green)
- `error` → `#ff4d6d` (R'lyeh Red)
- `warning` → `#ffd60a` (Eldritch Gold)
- `success` → `#37f499` (Lovecraft Green)
- `comment` → `#667085` (Slate)
- `loveCraft` → `#a48cf2` (Lovecraft Purple)
- `cyan` → `#04d1f9` (Shoggoth Cyan)
- `green` → `#37f499` (Lovecraft Green)
- `pink` → `#f26597` (Pustule Pink)
- `orange` → `#f7c67f` (Dreaming Orange)
- `yellow` → `#ffd60a` (Eldritch Gold)
- `red` → `#ff4d6d` (R'lyeh Red)

### Extension: `atlas-footer.ts`

**Line 1 layout:**
```
▲ Atlas v{version} · {provider} {model} · π v{piVersion} /{dir} · git:{branch} · {SESSION}
```

**Line 2 layout (current order):**
```
⚡ 5H:{pct}% ↻{reset} · WK:{pct}% ↻{reset} · ↑{input} ↓{output} ${cost} · ⛁⛁...⛁ {pct}%/{ctxWindow} · Cache {cacheHitPct}% · $${used}/${limit}
```

**Features implemented:**
- Rate limits from Anthropic OAuth API (5H + WK, with reset timestamps)
- Token stats (↑ input, ↓ output, $ cost)
- Visual context bar: 24 ⛁ padlock blocks with gradient color
- Cache hit rate
- Extra usage (Max plan overage)
- Version resolution: pi from `~/.bun/install/global/node_modules/@mariozechner/pi-coding-agent/package.json`, Atlas from `~/.pi/agent/settings.json`
- Auto-refresh every 30s + on branch change

**Known issues (see below)**

---

## Root Cause Found (2026-04-02)

### Root Cause: `custom-footer.ts` Overwrites Custom Footer

The extension `~/.pi/agent/extensions/custom-footer.ts` (pi's built-in example) also registers a `session_start` handler that calls `setFooter()`. Both extensions listen to `session_start` and both call `setFooter()`. Extensions load alphabetically, so `custom-footer.ts` loads AFTER `atlas-footer.ts` and overwrites the custom footer.

**Fix**: Renamed to `z-atlas-footer.ts` — alphabetical 'z' ensures it loads LAST, so it wins the `setFooter` battle.

## Known Issues (pre-fix — now resolved)

### 1. Extension Loading — Footer Not Rendering

~~The extension file (`atlas-footer.ts`) IS being loaded (confirmed via `console.error` marker). The `ctx.ui.setFooter()` callback is registered on `session_start`. However, the custom footer lines are NOT showing up in the TUI — only the built-in footer appears.~~

**RESOLVED**: Renamed to `z-atlas-footer.ts`.

**Potential causes:**
- The footer is registered inside `pi.on("session_start")` which may fire too late
- Should use `ctx.on("ready")` or register directly (not inside an event handler)
- Check how `custom-footer.ts` example does it — it calls `ctx.ui.setFooter()` directly

**Reference:** `examples/extensions/custom-footer.ts` line 24 — no event wrapping, just calls `ctx.ui.setFooter()` directly.

### 2. `ctx.model?.provider` — Type Issue

The `model` object on `ctx` is typed as `Model<any> | undefined`. The `Model` interface from `@mariozechner/pi-ai` does have a `provider: Provider` field. But accessing `model?.provider` in the extension may return `undefined` at runtime even though the field exists in the type.

**Current code (broken):**
```typescript
const modelProvider = model?.provider ?? "";
```

**Should try:** Cast through `as any` or check if the field is accessible:
```typescript
const modelProvider = (model as any)?.provider ?? "";
```

### 3. Git Branch Icon

The `⎇` (U+2395) and `⇄` (U+21C4) Unicode characters don't render in your terminal (WezTerm). Currently using text label `git:` instead.

**User prefers:** text label `git:` in pink (current)

### 4. `modelProvider` Not Showing

Even after fix attempts, `opencode-go` provider name is not showing in the footer. This may be the same root cause as issue #1 (footer not rendering at all).

---

## Files

| File | Status | Notes |
|------|--------|-------|
| `~/.pi/agent/themes/eldritch.json` | ✅ Done | All 51 tokens mapped |
| `~/.pi/agent/extensions/atlas-footer.ts` | ⚠️ Partial | Loads but footer doesn't render |
| `~/.atlas-plans/2026-04-02-eldritch-theme-statusline.md` | 📋 Archive | Original design plan |

---

## Next Steps

### High Priority

1. **Fix footer registration** — Move `ctx.ui.setFooter()` call OUTSIDE of `pi.on("session_start")`. Register it directly after the cache setup, or use `pi.on("ready")` if available.

2. **Test `model?.provider` access** — Add a `console.error` to check if `model?.provider` is accessible at runtime. If not, find an alternative way to get the provider name (e.g., from session config or env).

3. **Verify footer renders** — After fixing #1, restart pi and confirm the custom footer lines appear with `▲ Atlas`, `opencode-go/minimax-m2.7`, `π v0.64.0`, `git:master`, etc.

### Medium Priority

4. **Remove debug marker** — The `console.error("[atlas-footer] loading extension...")` at line 2 of `atlas-footer.ts` should be removed once verified working.

5. **Test with real usage** — Send a few messages to see token stats, cache hit rate, and context bar update correctly.

6. **Test rate limits** — Rate limit data should come from Anthropic OAuth API. Verify with real API credentials.

---

## Reference: How Extensions Work in pi

Extensions in `~/.pi/agent/extensions/*.ts` are auto-discovered and loaded via jiti (TypeScript JIT compiler).

**Registration pattern (from `examples/extensions/custom-footer.ts`):**
```typescript
pi.on("ready", (ctx) => {
    ctx.ui.setFooter((tui, theme, footerData) => {
        // Return string[] for footer lines
        return ["line1", "line2"];
    });
});
```

**Key APIs available in ExtensionContext:**
- `ctx.ui.setFooter(fn)` — set custom footer renderer
- `ctx.ui.setStatus(key, text)` — set status bar text
- `ctx.model` — current model (has `id`, `provider`, `api`, etc.)
- `ctx.sessionManager.getBranch()` — session message history
- `ctx.sessionManager.getSessionName()` — current session name
- `ctx.getContextUsage()` — context window usage
- `footerData.getGitBranch()` — git branch name
- `footerData.onBranchChange(fn)` — subscribe to branch changes

---

## Design Decisions (Final)

| Decision | Value |
|----------|-------|
| Atlas icon | `▲` (U+25B2) |
| pi icon | `π` (U+03C0) in orange `#f7c67f` |
| Git branch | `git:` text label in pink |
| Context bar | 24 `⛁` blocks, gradient color by usage level |
| Rate limit icon | `⚡` (U+26A1) |
| Cache icon | `󿛛` (Nerd Font nf-oct-cache — may not render, fallback to text) |
| Line 2 order | Rate limits → Token stats → Context bar → Cache → Extra usage |

---

## To Restart pi After Changes

```bash
pkill -9 -f "pi " 2>/dev/null; sleep 1; pi --theme eldritch
```

The jiti cache does NOT need to be cleared — renaming the file to `atlas-footer.ts` (from `atlas-statusline.ts`) was sufficient.
