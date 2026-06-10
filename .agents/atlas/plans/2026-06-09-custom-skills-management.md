# Repo-Backed Skills Store for atlas-config

**Date:** 2026-06-09
**Status:** Plan — awaiting Ed's go + small decisions (§7). No implementation yet.
**Research:** `.agents/atlas/artifacts/skills-sh-research.md`

> **Direction (Ed):** Don't reinvent skills.sh. Keep using it for what it's good at —
> fetching a skill once and symlinking it into any agent you select. Just move its store
> into this repo so git gives us reproducibility, and drive the workflow from the `justfile`.

---

## 1. How skills.sh actually works on this machine (verified)

- **Central store:** `~/.agents/skills/` holds the real skill files (`agent-browser`,
  `caveman`, `herdr`, `tdd`, …) plus `node_modules/`, `package.json`, `pnpm-lock.yaml`.
- **Global lockfile:** `~/.agents/.skill-lock.json` (sibling of the store, not inside it).
- **Per-agent symlinks:** skills.sh symlinks store entries into each selected agent dir, e.g.
  - `~/.claude/skills/agent-browser -> ../../.agents/skills/agent-browser`
  - `~/.pi/agent/skills/caveman -> ../../../.agents/skills/caveman`
- Agent dirs are a **mix**: PAI-authored skills are real dirs (`~/.claude/skills/Agents`, …);
  skills.sh skills are symlinks back to the store.

So skills.sh already gives us **install-once → symlink-to-many**. The only thing it does
badly is restore/update reproducibility (every core bug OPEN — see research §5). **git fixes
exactly that** if the store lives in the repo.

## 2. The whole idea (one sentence)

Make `~/.agents/skills/` a **stow-managed package in atlas-config** so the store's skill
files are committed; let skills.sh keep fetching + symlinking unchanged; use the `justfile`
to wrap add/update/relink, with **`git diff` as the safety net** over skills.sh's unreliable
update/lock behavior.

No custom fetcher. No manifest format. No new CLI. Just relocate + thin justfile glue.

## 3. Design

### 3.1 New stow package `agents`

Map `agents/.agents/skills/` (repo) → `~/.agents/skills/` (disk), as a **folded symlink**
(whole-dir link, like the `claude`/`codex` packages — not `--no-folding`). Then:

- skills.sh writes a new skill into `~/.agents/skills/<name>/` → it lands **inside the repo**
  → `git status` shows it. Commit it.
- Restore on a fresh machine: `git clone && just link` recreates `~/.agents/skills` from the
  repo — all skill files present, **no network, no `experimental_install`**.

Only `~/.agents/skills` is mapped, so the siblings `~/.agents/Recall/` (out-of-band per
AGENTS.md) and `~/.agents/atlas/` are untouched.

### 3.2 What's committed vs ignored (in the store)

| Path | Tracked? | Why |
|---|---|---|
| `agents/.agents/skills/<skill>/**` | **commit** | the reproducibility payload |
| `node_modules/` | gitignore | regenerable by `npx skills` |
| `.DS_Store` | gitignore | noise |
| `package.json`, `pnpm-lock.yaml` | commit | tiny; lets `npx skills` rebuild deterministically |
| `~/.agents/.skill-lock.json` | **not tracked** (default) | research §5 shows it drifts / gets wiped; skill files are the real source of truth. Optional record-only snapshot via a just target if Ed wants it. |

### 3.3 justfile targets (the workflow Ed wants)

Add a `skills` group mirroring the existing justfile style:

- **`just skills-add <source>`** → runs `npx skills add <source>` (interactive agent pick).
  Files land in the repo-backed store; review `git diff`; `just push` to commit.
- **`just skills-update [names…]`** → runs `npx skills update …`. Because the store is in git,
  the result shows up as a **diff we can inspect and revert** — this neutralizes the
  false-success (#923) and silent-skip/wipe (#542) bugs without fixing skills.sh.
- **`just skills-relink`** → small shell loop: for each skill in the store, ensure the symlink
  exists in each selected agent dir. This is the **global relink** skills.sh lacks (#683) and
  is what re-creates per-agent links on a fresh machine after `just link`.
- **`just skills-list`** → `npx skills list` (or list the store dir).

Everything else (the fetch, the per-agent symlink creation on first install) stays
skills.sh's job. The only custom code is the `skills-relink` loop — a few lines of shell.

### 3.4 Global vs per-project

- **Global skills** → repo-backed `~/.agents/skills` store, symlinked into `~/<agent>/skills`.
  Managed here by atlas-config's justfile. ✅ This is the bulk of the win.
- **Per-project skills** → skills.sh installs into *that project's* `./<agent>/skills/`; those
  belong in **that project's** git, not atlas-config. atlas-config owns only the global store.
  (`skills-relink` can take a `--project` path later if a project wants the same relink helper.)

## 4. Restore flow (fresh machine)

```
git clone … && cd atlas-config
just link            # stow → ~/.agents/skills now points at the repo store (all files back)
just skills-relink   # recreate per-agent symlinks (~/.claude/skills/* -> store, etc.)
# optional: npx skills (rebuild node_modules) if you'll run add/update on this machine
```

No reliance on `skills experimental_install` / `update` / `check`.

## 5. Why this beats both alternatives

- **vs. plain skills.sh:** identical daily UX (still `npx skills add`, still symlinks to chosen
  agents), but reproducible and revertable because the store is in git. Sidesteps every OPEN
  update/restore bug.
- **vs. my first draft (custom `skillsctl` + manifest):** that reinvented skills.sh's fetch +
  pinning. Dropped — it's more code for a problem git already solves once the store is repo-backed.

## 6. Rollout

1. Decide §7.
2. Create the `agents` package; move current `~/.agents/skills/<skill>` dirs into
   `agents/.agents/skills/`; add `agents` to `pkgs` in the justfile; gitignore
   `node_modules`/`.DS_Store` under it.
3. `rm -rf ~/.agents/skills` (after the move) → `just link` → confirm `~/.agents/skills` is now
   the repo symlink and agent symlinks still resolve.
4. Add the `skills-add/update/relink/list` targets; test `skills-relink` recreates a link.
5. Commit. Document a short "Skills" section in AGENTS.md next to the stow conventions.
6. Pilot: `just skills-add <something>`, confirm it lands in the repo and symlinks to agents.

## 7. Decisions for Ed

1. **Confirm the folded-symlink store** (§3.1 — `~/.agents/skills` becomes a symlink into the
   repo) vs keeping it real and mirroring into the repo via rsync? (Recommend folded symlink —
   zero copy step, skills.sh writes straight to git.)
2. **Which agents does `skills-relink` target?** claude, codex, pi, omp — all four, or a subset?
3. **Track `.skill-lock.json`?** Default no (it drifts). Want a record-only snapshot target?
4. Anything in the current store to **exclude** from git (e.g. half-installed/experimental
   skills like `setup-matt-pocock-skills`, `source-command-recall-*`)?
