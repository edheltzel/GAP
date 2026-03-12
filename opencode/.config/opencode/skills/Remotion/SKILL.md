---
name: "Remotion"
description: "Programmatic video creation with React. USE WHEN video, animation, motion graphics, video rendering, React video, intro video, YouTube video, TikTok video, video production, render video."
---
# Remotion

Create professional videos programmatically with React.

## Customization

**Before executing, check for user customizations at:**
`~/.config/opencode/skills/PAI/USER/SKILLCUSTOMIZATIONS/Remotion/`

## Workflow Routing

| Trigger | Workflow |
|---------|----------|
| "animate this", "create animations for", "video overlay" | `Workflows/ContentToAnimation.md` |

## Quick Reference

- **Theme:** Always use PAI_THEME from `Tools/Theme.ts`
- **Art Integration:** Load Art preferences before creating content
- **Critical:** NO CSS animations - use `useCurrentFrame()` only
- **Output:** Always to `~/Downloads/` first

**Render command:**
```bash
npx remotion render {composition-id} ~/Downloads/{name}.mp4
```

## Full Documentation

- **Art integration:** `ArtIntegration.md` - theme constants, color mapping
- **Common patterns:** `Patterns.md` - code examples, presets
- **Critical rules:** `CriticalRules.md` - what NOT to do
- **Detailed reference:** `Tools/Ref-*.md` - 28 pattern files from Remotion

## Tools

| Tool | Purpose |
|------|---------|
| `Tools/Render.ts` | Render, list compositions, create projects |
| `Tools/Theme.ts` | PAI theme constants derived from Art |

## Links

- Remotion Docs: https://remotion.dev/docs
- GitHub: https://github.com/remotion-dev/remotion
