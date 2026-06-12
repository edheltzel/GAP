# Skills manifest + install script (skills.sh wrapper)

**Date:** 2026-06-10 · **Status:** Plan — simplified per Ed's direction (supersedes the manifest+PATH-command design and the 2026-06-09 stow-store plan).

## Context

Ed installs agent skills with skills.sh, but atlas-config has no record of what's installed and no way to reinstall on a fresh machine. The fix is deliberately minimal: commit a **manifest of skill sources** (`<repoOwner>/<repo>`, one per line) and a **script** that replays them through the skills.sh CLI non-interactively. Nothing else — no PATH commands, no stow changes, no git-tracking of skill files.

## Design

### 1. `skills.manifest` (repo root, committed)

One source per line as `<repoOwner>/<repo>`; `#` comments and blank lines allowed. Bootstrapped from the distinct sources in `~/.agents/.skill-lock.json` (verified 2026-06-10):

```
# Skill sources installed via skills.sh — install with scripts/install-skills.ts
manavarya09/design-extract
mattpocock/skills
ogulcancelik/herdr
pbakaus/impeccable
vercel-labs/agent-browser
vercel-labs/opensrc
```

### 2. `scripts/install-skills.ts` (Bun TypeScript, executable)

Reads the manifest and runs the skills.sh CLI per source, fully non-interactive, **global scope**, targeting **Claude Code, Codex, and Pi**:

```
bunx -y skills add <owner/repo> --skill '*' -g -a claude-code -a codex -a pi -y
```

Behavior:

- Resolve the manifest relative to the script's own location (`<repo>/skills.manifest`) — no hardcoded paths, works from any cwd.
- Skip `#` comments and blank lines.
- Run sources sequentially; report per-source success/failure and exit non-zero if any source fails.
- Idempotent: `skills add` re-fetches and overwrites, so the script doubles as update-all and fresh-machine restore.
- Contains `// TODO: add oh-my-pi conversion for skills` (OMP isn't a skills.sh-supported agent; needs a conversion step into OMP-native skills).

Why only `add`: skills.sh's `update`/`check`/`experimental_install` have open correctness bugs (false success #923, lockfile wipe #542, state mutation #954, no stable restore #549). `add` is the one reliable command, and replaying it is the whole restore strategy.

## Files to change

| File | Change |
|---|---|
| `skills.manifest` | **New** — the 6 sources above |
| `scripts/install-skills.ts` | **New** — Bun script as specified, `chmod +x` |
| `AGENTS.md` | One short "Skills" note under Agent skills pointing at the manifest + script |
| `.agents/atlas/plans/2026-06-09-custom-skills-management.md` | Move to `plans/archive/` — superseded |

## Verification

1. `bun scripts/install-skills.ts` runs clean end-to-end; each source reports success.
2. Spot-check one skill per agent: `ls -la ~/.claude/skills/<skill> ~/.codex/skills/<skill> ~/.pi/agent/skills/<skill>` resolve to `~/.agents/skills/<skill>`.
3. Re-run the script — still exits 0 (idempotent).
4. `git status` shows only the files listed above.
