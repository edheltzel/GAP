---
title: PAI Upstream Sync Report — 2026-04-17
status: In Review
percent_complete: 90
last_updated: 2026-04-19
phase: Analysis complete — sync state update pending
blockers: awaiting merge of pai-upgrade branch before updating upstream-sync.json
next_action: after merging worktree-pai-upgrade-2026-04-17, update upstream-sync.json to record dc05fe9 as new last_sync_ref
---

# PAI Upstream Sync Report

**Generated:** 2026-04-17T22:12:39-04:00
**Last sync:** v4.0.3 @ fb07141 (2026-03-16)
**Current upstream HEAD:** dc05fe9
**New commits:** 5 (2026-03-22 → 2026-04-11)
**New release tags:** None (still on v4.0.3)
**Releases changed:** Releases/Pi/** (new Pi scaffold — separate release, NOT the Claude Code release)

---

## Change Categories

### NEW RELEASE — Releases/Pi/ (LOW for Atlas direct action)

**PAI on Pi v1.0.0 (Alpha)** — A model-agnostic scaffold for running PAI's core methodology in the Pi coding agent (`@mariozechner/pi-coding-agent`). Targets users who want local/open-source models (Ollama, OpenRouter, etc.) instead of Claude Code.

**What's in it:**
- `Releases/Pi/config/SYSTEM.md` — Pi-specific system prompt with Algorithm v3.6.0 (note: older than our v3.7.0-atlas)
- `Releases/Pi/extensions/pai-core/index.ts` — Full TypeScript extension implementing:
  - Voice notification tool (via env var config, same POST /notify pattern as our VoiceServer)
  - Security event blocking (dangerous bash patterns — subset of our SecurityValidator patterns)
  - PRD work tracking (`prd_create`, `prd_update` tools)
  - Learning signal capture (`capture_learning` tool)
  - Session start/stop lifecycle hooks
  - Slash commands: `/algorithm`, `/status`, `/voice`
- `Releases/Pi/skills/` — 9 lightweight SKILL.md files (agents, content-analysis, investigation, media, research, scraping, security, telos, thinking)
- `Releases/Pi/memory/` — Standard MEMORY/WORK, MEMORY/LEARNING, MEMORY/STATE layout

**Atlas relevance:** Not directly mergeable — Pi uses a completely different extension API vs Claude Code hooks. But contains interesting patterns (see Merge Plan below).

**Files changed:**
- `Releases/Pi/INSTALL.md` (added, 2 URL fixes applied)
- `Releases/Pi/README.md` (added, 2 URL fixes applied)
- `Releases/Pi/VERSION` (1.0.0)
- `Releases/Pi/config/AGENTS.md`, `SYSTEM.md`, `models.json`, `settings.json`
- `Releases/Pi/extensions/pai-core/index.ts`
- `Releases/Pi/skills/**` (9 SKILL.md files)
- `Releases/Pi/memory/**` (.gitkeep placeholders)

**Protected file conflicts:** None — entirely new directory, no conflict with Atlas.

---

### SECURITY — .pai-protected.json update (MEDIUM for Atlas)

**Commit:** 6e0bcc3 (2026-04-06)

Upstream added 4 new security pattern categories to `.pai-protected.json` for PAI 5.0.0 release preparation:
1. `private_repos` — Private GitHub repo references (`danielmiessler/arbol`, `danielmiessler/feed`, `danielmiessler/substrate`, `danielmiessler/telos`, `danielmiessler/ladder`)
2. `personal_project_urls` — Personal project URLs (`thesurface.ai`)
3. `da_names` — Digital Assistant name patterns (`\bDevi\b`)
4. `imessage_bot` — iMessage bot references

**Atlas relevance:** The `private_repos` and `da_names` patterns are upstream-specific to Daniel Miessler's setup. However, the **pattern categories themselves** are generalizable — Atlas's `patterns.yaml` has no content-leakage detection (no section for blocking output containing secrets or private repo names). This is a model difference: upstream uses `.pai-protected.json` as an output/content filter, while Atlas uses `patterns.yaml` as an input/command filter.

**Atlas currently has:** Input-level security (what commands can run). Not output-level security (what content can be exposed).

---

### DOCS — README framing update (SKIP for Atlas)

**Commit:** dc05fe9 (2026-04-11)

Upstream README: "Your personal AI stack." → "Your Life Operating System." + 2 URL canonicalizations for the Real Internet of Things blog post.

**Atlas relevance:** None — Atlas has its own README. No action needed.

---

## Merge Plan

| # | Action | Category | Files | Risk |
|---|--------|----------|-------|------|
| 1 | Review Pi extension for `prd_create`/`prd_update` tool patterns | Pi scaffold | Informational only | None |
| 2 | Consider adding content-leakage section to patterns.yaml | Security | `~/.claude/skills/PAI/USER/PAISECURITYSYSTEM/patterns.yaml` | Low |
| 3 | Update upstream-sync.json state | Meta | `~/.claude/skills/PAIUpgrade/State/upstream-sync.json` | None |
| 4 | Skip README, Pi scaffold files | Docs/Pi | None | None |

---

## Protected File Conflicts

| Protected File | Upstream Change | Atlas Action |
|----------------|-----------------|--------------|
| None | No protected files touched | No action needed |

---

## Recommendations

### SKIP — PAI on Pi scaffold (Releases/Pi/)

Not applicable to Atlas. Pi and Claude Code use entirely different extension APIs. The Pi scaffold's Algorithm is v3.6.0 — older than our v3.7.0-atlas. Nothing to merge.

**Exception — awareness item:** The Pi `extensions/pai-core/index.ts` implements `prd_create` and `prd_update` as explicit tools (not implicit markdown). This is an interesting architectural reference showing how PRD tracking could be made explicit. Atlas currently relies on the AI following PRDFORMAT.md instructions — no tool enforcement. This is a FUTURE consideration if we want to enforce PRD creation.

### CONSIDER — Content-leakage patterns in patterns.yaml

The upstream `.pai-protected.json` update reveals a security model Atlas doesn't have: patterns that detect when the AI might be exposing sensitive content in output (private repo names, personal URLs, internal names). Atlas's `patterns.yaml` only covers command/file-operation security.

Adding an `output_checks` or `content_leakage` section to Atlas patterns would be LOW effort and add meaningful defense-in-depth. Suggested additions:

```yaml
# Content leakage prevention (what the AI should not expose in output)
content_leakage:
  - category: private_repos
    description: "Ed's private repository references"
    patterns:
      - "rainyday-media/[a-z-]+"
      # add actual private repo names here
  - category: sensitive_urls
    description: "Internal/private URLs"
    patterns: []  # add as needed
```

However, this requires Ed to populate it with his actual private repo names — it's a personal-data section.

### UPDATE — Sync state to HEAD

Update `upstream-sync.json` to record dc05fe9 as the new sync point.

---

## Decision: Apply Changes?

**Recommended actions (all LOW risk, no protected file conflicts):**

1. **Update sync state** → Record dc05fe9 as new last_sync_ref, date 2026-04-17
2. **Skip all file merges** → No v4.0.3 release content changed; Pi scaffold is separate and not applicable
3. **Optional:** Add content-leakage section skeleton to patterns.yaml (requires Ed input on actual private names)

---

## Sources Processed

**Commits analyzed:** 5 (fb07141..dc05fe9)
- `a544ea2` — Add PAI on Pi v1.0.0 (22 files, 1309 insertions)
- `e921aa3` — Fix Pi repo URL (badlogic/pi not mariozechner), add alpha notice
- `4560c95` — Fix Pi repo URL to badlogic/pi-mono (from npm package metadata)
- `6e0bcc3` — Update .pai-protected.json: add 4 new security pattern categories
- `dc05fe9` — README: subtle Life OS framing + canonical RIoT URL

**Atlas release dir checked:** `Releases/v4.0.3/.claude/` — NO changes in this release since last sync.
