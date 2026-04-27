---
title: External Skill Adoption — karpathy-guidelines
date: 2026-04-27
status: completed (claude-code) · deferred (opencode + pi)
revision_history:
  - v1 — stow + custom origin:external frontmatter (rejected)
  - v2 — adopt third-party skills CLI for cross-agent symlinks (rejected)
  - v3 — repo-recommended Claude Code plugin marketplace install (shipped)
---

# karpathy-guidelines — Final Plan and Outcome

## Final decision (v3)

Use Claude Code's built-in plugin marketplace — the install path the upstream README explicitly recommends. Cross-agent parity (OpenCode, Pi) deferred.

## What shipped (Phase 1)

1. ✅ Marketplace registered — `/plugin marketplace add forrestchang/andrej-karpathy-skills` → `karpathy-skills`
2. ✅ Plugin installed — `/plugin install andrej-karpathy-skills@karpathy-skills` (pinned to commit `2c606141936f1eeef17fa3043a72095b4765b9c2`, version 1.0.0)
3. ✅ `/reload-plugins` run — 12 plugins loaded, plugin became active
4. ✅ Legacy install cleaned up — `skills remove karpathy-guidelines -g -y` removed the duplicate from `~/.agents/skills/karpathy-guidelines/` and the symlink at `~/.claude/skills/karpathy-guidelines`
5. ✅ ISC subsumption note appended to `claudecode/.claude/CLAUDE.md` — establishes that karpathy is supplemental to PAI ISC, not parallel
6. ✅ External Skills section appended to `docs/pai-update-guide.md` — documents the plugin marketplace path, deferred cross-agent parity, and update/remove commands

## Deferred

- **OpenCode parity** — Claude Code plugins live in `~/.claude/plugins/` and do NOT auto-propagate to OpenCode (which reads its own `~/.config/opencode/skills/` tree). Revisit if karpathy-guidelines becomes important during OpenCode sessions; options at that time: (a) append principles to `opencode/.config/opencode/AGENTS.md`, or (b) stow a copy of the SKILL.md into the OpenCode skills tree.
- **Pi parity** — pi-agent currently in `oldSetup/pi-agent/`; revisit on reactivation.
- **EXAMPLES.md snapshot** — 522 lines of worked examples in the upstream repo are not pulled by the plugin install. Skipped for now. If useful later, snapshot to `docs/karpathy-examples.md` with a provenance header (source URL + pinned SHA + fetch date).

## Decision history (the iteration that mattered)

| Round | Approach | Why rejected/accepted |
|---|---|---|
| v1 | Stow karpathy SKILL.md into atlas-config; invent `origin: external` frontmatter; register in PAI protected-files lists; add LICENSE files; manual three-package propagation | **Red-team caught factual errors:** the install was via the third-party `skills` CLI (not Claude marketplace as I'd written); `context: fork` was cited as existing precedent but had zero current uses; PAI's "Nothing escapes the Algorithm" rule invalidated the original ISC-vs-karpathy disambiguation framing. Required substantial rewrite. |
| v2 | Use the `skills` CLI for multi-agent symlinks | **Ed correctly rejected:** "skills.sh" was not actually the repo's recommended install — that's `/plugin install`. The CLI is a third-party tool, not what the README endorses. |
| v3 | Repo-recommended Claude Code plugin marketplace install | **Shipped.** Simplest possible path; closest to upstream's published guidance; clean separation from PAI sync surface (different directory tree). Cross-agent parity intentionally deferred. |

## Lessons captured

1. **Read the upstream README before designing infrastructure around it.** v1 was full of assumptions about what the install path actually was. Verifying ground truth first would have saved two iterations.
2. **PAI's "Nothing escapes the Algorithm" is load-bearing.** Any framing that posits an "Algorithm-off" mode is wrong by construction in this setup. Future external-skill-adoption framings must position third-party additions as supplemental to ISC, not parallel.
3. **Ed's instinct to use a unified install tool was right; the specific tool name was off.** "use skills.sh" actually meant "use whatever tool unifies installs" — and the repo's own recommended `/plugin install` IS that tool for the Claude Code half.
4. **Cross-agent parity is genuinely separate work.** The repo-recommended install is Claude Code-only by design. If OpenCode/Pi parity is required, that's a deliberate additional task — not something a single install can satisfy.

## Verification (post-shipment)

```fish
# Plugin install present
ls ~/.claude/plugins/cache/karpathy-skills/andrej-karpathy-skills/1.0.0/

# Legacy install gone
ls ~/.agents/skills/karpathy-guidelines/         # → No such file or directory
ls ~/.claude/skills/karpathy-guidelines           # → No such file or directory
jq '.skills["karpathy-guidelines"]' ~/.agents/.skill-lock.json  # → null

# Doc updates landed
rg "karpathy-guidelines & PAI ISC" claudecode/.claude/CLAUDE.md
rg "External Skills"               docs/pai-update-guide.md
```

## Files committed in this round

- `.atlas/plans/archive/2026-04-27-karpathy-guidelines-plugin-install.md` (this file)
- `claudecode/.claude/CLAUDE.md` (ISC subsumption note appended)
- `docs/pai-update-guide.md` (External Skills section appended)
- `CHANGELOG.md` (`[Unreleased]` entry)

## Files NOT committed (intentionally)

The visual at `~/.agent/diagrams/atlas-karpathy-merge-plan.html` lives outside the atlas-config repo and is excluded from the commit.
