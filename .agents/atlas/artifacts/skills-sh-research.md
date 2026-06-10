# skills.sh — Global vs Project Skills, Lockfiles & Reinstall

**Sourcing caveat up front:** The skills.sh first-party docs pages (`/docs`, `/docs/cli`, `/docs/faq`, `/docs/api`, `/docs/customize`) are thin and do **not** document lockfiles, scopes, or restore workflows at all — I verified each. The detailed, accurate information lives in the repo's `AGENTS.md`, the auto-generated DeepWiki reference, the project's GitHub issues, and a community write-up. Where a claim rests only on a non-first-party source, I say so.

## 1. Global vs. project-level skills

The CLI has two install scopes, controlled by the `-g` / `--global` flag. **Project is the default.**

| Scope | Flag | Location | Intended use |
|---|---|---|---|
| **Project** (default) | *(none)* | `./<agent>/skills/` | Committed with the project, shared with the team |
| **Global** | `-g` / `--global` | `~/<agent>/skills/` | Available across all projects |

This table is quoted from the repo README's "Installation Scope" section ([github.com/vercel-labs/skills](https://github.com/vercel-labs/skills)). The `-g`/`--global` flag is accepted by `add`, `remove`, `list`, and `update` ([DeepWiki CLI reference](https://deepwiki.com/vercel-labs/skills/4-cli-commands-reference)). `update` additionally has `-p`/`--project` to target project scope explicitly, and `-y`/`--yes` to auto-detect scope and skip prompts (README, via [GitHub](https://github.com/vercel-labs/skills)).

## 2. Lockfiles — what file, what it tracks, when written

There are **two** lockfiles, one per scope ([DeepWiki: Skill Lock File System](https://deepwiki.com/vercel-labs/skills/5.9-skill-lock-file-system)):

- **Project lockfile:** `skills-lock.json` in the project root — **intended to be checked into version control.**
- **Global lockfile:** `~/.agents/.skill-lock.json` (or `$XDG_STATE_HOME/skills/.skill-lock.json`).

**Tracked fields** (global lockfile, per DeepWiki): `source` (normalized identifier, e.g. `owner/repo`), `sourceType` (e.g. `github`, `local`), `skillFolderHash` (for GitHub sources this is the **GitHub tree SHA** of the skill folder; for local sources a recursive **SHA-256** of sorted relative-path + file content, skipping `.git`/`node_modules`), and `installedAt` / `updatedAt` ISO timestamps. The **project** lockfile "intentionally omits timestamps to minimize merge conflicts" but keeps the content hash ([DeepWiki](https://deepwiki.com/vercel-labs/skills/5.9-skill-lock-file-system)).

**When written/updated** ([DeepWiki](https://deepwiki.com/vercel-labs/skills/5.9-skill-lock-file-system)):
- Auto-generated "when you install your first skill" ([maier.tech, Mar 19 2026](https://maier.tech/notes/a-lockfile-for-agent-skills)).
- Updated on `add` (via `addSkillToLock` / `addSkillToLocalLock`), on `remove` (via `removeSkillFromLock` / `removeSkillFromLocalLock`), and during `experimental_sync` (rescans `node_modules`, recomputes hashes).

> ⚠️ **Known rough edge / source conflict:** The community note states "`npx skills remove` does not update the lockfile" ([maier.tech, Mar 19 2026](https://maier.tech/notes/a-lockfile-for-agent-skills)), whereas DeepWiki lists a `removeSkillFromLocalLock` path. This looks like version drift — treat lockfile-on-remove as not fully reliable, and verify against your installed CLI version.

## 3. Reinstalling from a lockfile — exact process

This is the area with the most nuance, so read carefully.

**The experimental command that exists today:**
```
npx skills experimental_install
```
Description (verbatim, repo `AGENTS.md`): *"Restore skills from skills-lock.json."* It reads `skills-lock.json` and (per DeepWiki) calls `runInstallFromLock`, reinstalling remote skills via the `add` path and `node_modules` skills via `sync`. It is **explicitly marked experimental** ([DeepWiki CLI reference](https://deepwiki.com/vercel-labs/skills/4-cli-commands-reference); [repo AGENTS.md](https://github.com/vercel-labs/skills/blob/main/AGENTS.md)). A companion experimental command, `npx skills experimental_sync`, syncs skills packaged in `node_modules` into agent dirs.

**Do not confuse this with `skills install`:** `install` (and `i`, `a`) are **aliases of `add`** — i.e. they install a *named* skill from a source, they do **not** restore from the lockfile ([DeepWiki CLI reference](https://deepwiki.com/vercel-labs/skills/4-cli-commands-reference)).

**A stable, non-experimental restore is still an open feature request:** Issue [#549](https://github.com/vercel-labs/skills/issues/549) (opened Mar 8 2026, **Open**, no maintainer resolution) asks for `npx skills install` (no-args, restore from `skills-lock.json`, an `npm ci` equivalent), `npx skills install -g` (restore global), and `npx skills ci` (strict mode). It states plainly that "there is no command to restore skills from it" and that "running `npx skills update` when no skills are installed does nothing." A related earlier request is [#283](https://github.com/vercel-labs/skills/issues/283).

**Bottom line for question 3:** As of June 2026, the only working restore-from-lockfile path is the **experimental** `npx skills experimental_install`. A stable command is requested but not yet shipped. If you need reproducible installs on a fresh clone / in CI, either rely on the experimental command (accepting instability) or script `npx skills add <source>` per skill.

## 4. Related workflow notes (sync, update, CI, committing)

- **Commit the project lockfile.** The community write-up's explicit guidance: *"Add this to version control"* so skill changes surface through git ([maier.tech](https://maier.tech/notes/a-lockfile-for-agent-skills)). The repo describes `skills-lock.json` as "checked in" ([repo AGENTS.md](https://github.com/vercel-labs/skills/blob/main/AGENTS.md)). The global `~/.agents/.skill-lock.json` is machine-level state, not committed.
- **Check for updates:**
  ```
  npx skills check
  ```
  *"Checks for available updates to installed skills."* It compares stored hashes against sources ([DeepWiki CLI reference](https://deepwiki.com/vercel-labs/skills/4-cli-commands-reference)).
- **Update:**
  ```
  npx skills update [skills...]
  npx skills update -g      # global only
  npx skills update -p      # project only
  npx skills update -y      # skip prompts, auto-detect scope
  ```
  ([repo README via GitHub](https://github.com/vercel-labs/skills); [DeepWiki](https://deepwiki.com/vercel-labs/skills/4-cli-commands-reference)). Note the caveat from #549: `update` does nothing when no skills are installed, so it is **not** a substitute for a restore command.
- **CI:** Telemetry is **automatically disabled in CI environments** (repo README). You can also opt out anywhere with the env var `DISABLE_TELEMETRY=1` ([skills.sh/docs/cli](https://www.skills.sh/docs/cli)). Beyond telemetry, there is **no documented first-party CI recipe** for restoring skills — issue [#549](https://github.com/vercel-labs/skills/issues/549) exists precisely because the `npm ci`-style flow is missing.
- **Sync:** `npx skills experimental_sync` pulls skills from `node_modules` into agent directories (experimental) ([DeepWiki](https://deepwiki.com/vercel-labs/skills/4-cli-commands-reference)).

> ⚠️ **Open bug to be aware of:** Issue [#542](https://github.com/vercel-labs/skills/issues/542) reports that `skills check`/`update` ignore the project-level `skills-lock.json` and that a global lock migration can "silently wipe tracked skills." Confirm behavior on your version before depending on it.

---

**Sources:**
- [skills.sh/docs](https://www.skills.sh/docs) · [/docs/cli](https://www.skills.sh/docs/cli) · [/docs/faq](https://www.skills.sh/docs/faq) · [/docs/api](https://www.skills.sh/docs/api) · [/docs/customize](https://www.skills.sh/docs/customize) — (confirm lockfiles/scopes are *not* covered here)
- [github.com/vercel-labs/skills (README)](https://github.com/vercel-labs/skills) · [AGENTS.md](https://github.com/vercel-labs/skills/blob/main/AGENTS.md)
- [DeepWiki: CLI Commands Reference](https://deepwiki.com/vercel-labs/skills/4-cli-commands-reference) · [DeepWiki: Skill Lock File System](https://deepwiki.com/vercel-labs/skills/5.9-skill-lock-file-system)
- GitHub issues: [#549 (restore/CI)](https://github.com/vercel-labs/skills/issues/549) · [#283 (install/sync)](https://github.com/vercel-labs/skills/issues/283) · [#337 (project-scoped update)](https://github.com/vercel-labs/skills/issues/337) · [#542 (lockfile bug)](https://github.com/vercel-labs/skills/issues/542)
- [maier.tech — "A lockfile for agent skills" (Mar 19 2026)](https://maier.tech/notes/a-lockfile-for-agent-skills)

---

## 5. GitHub issues — skills not updating (deep dive)

Investigated via the `gh` CLI against `vercel-labs/skills` (June 2026). **Headline finding: every issue in the "skills won't update" cluster is still OPEN.** No fixes have shipped for the core lockfile/update bugs — the only closed items are unrelated feature additions (non-GitHub source support, #386/#411) and one narrow SSH-overwrite fix (#567). Treat `update` and `experimental_install` as unreliable; see the safety verdict at the end.

### 5a. The central bug — #542 (the one you started with)

- **#542 — `skills check`/`update` ignore project-level `skills-lock.json`; global lock migration silently wipes tracked skills** — **OPEN**, CLI v1.4.4. ([#542](https://github.com/vercel-labs/skills/issues/542))
- **Root cause (two bugs, well-diagnosed by the reporter):**
  1. The CLI keeps **two lock files with different schemas** — global `~/.agents/.skill-lock.json` (lock `version: 3`) used by `check`/`update`, vs. project `./skills-lock.json` (lock `version: 1`) used by `experimental_install`. `check`/`update` are **hardcoded to read the global lock only**; there is no flag to point them at the project lock. So default (project-scope) installs report `"No skills tracked in lock file."`
  2. `readSkillLock()` has `CURRENT_LOCK_VERSION = 3`; when it reads an older lock it returns `{ version: 3, skills: {} }` — **silently discarding all entries**, no migration, no warning. A CLI upgrade can therefore wipe your tracked skills.
- **Scope affected:** Bug 1 = **project scope** (the default). Bug 2 = **global scope** after a CLI version bump.
- **Maintainer response:** None on the thread. **Five independent users confirmed reproduction** (pierresisson, gillesperrin, albertlieyingadrian, The-Best-Codes, sylvaindupuy — "can't update project level skills"). No linked fix/PR.
- **Workaround:** `npx skills update -p` forces project scope (per #283 comments) — but only updates skills **already installed**; it cannot restore from a lockfile. Re-`add` skills to repopulate a wiped global lock.

### 5b. Closely related "won't update / false success" reports

| # | Title | State | Root cause | Scope | Maintainer / fix | Workaround |
|---|---|---|---|---|---|---|
| [#1143](https://github.com/vercel-labs/skills/issues/1143) | `update` "No global skills tracked in lock file" | OPEN (v1.5.6) | Same as #542 — `update -g` finds nothing because installs never populated the global lock (or it was wiped). User's lock had only 1 of N skills. | Global | Collaborator **quuu** engaged (asked for lock contents); no fix | Re-`add` each skill so it re-registers in `.skill-lock.json` |
| [#923](https://github.com/vercel-labs/skills/issues/923) | `update` reports success without updating contents | OPEN (v1.2.3) | `runUpdate()` spawns `npx -y skills <url> --skill <name> -g -y` **missing the `add` subcommand**; the unknown-command branch prints an error but **exits 0**, so `update` prints `✓ Updated <skill>` while files stay stale | Global (shown) | None | Verify files on disk vs repo manually; don't trust the success line |
| [#954](https://github.com/vercel-labs/skills/issues/954) | `check` reinstalls outdated skills instead of just reporting | OPEN | `check` is supposed to be read-only (per AGENTS.md) but mutates state — it reinstalls outdated skills as a side effect | Global (shown) | None | Be aware `check` is **not** side-effect-free |
| [#1114](https://github.com/vercel-labs/skills/issues/1114) | Peer skills (`peers.yaml`) not tracked in lock, can't be updated | OPEN | Auto-installed peer skills get no entry in `.skill-lock.json`, so `update` silently skips them | Both | None | Manually `npx skills add <repo> --skill <peer>` per peer |
| [#999](https://github.com/vercel-labs/skills/issues/999) | well-known source truncated in `skills-lock.json` prevents updates | OPEN | `handleWellKnownSkills()` writes `source: sourceIdentifier` (truncated) instead of full URL; updater can't reconstruct source → update fails. One-line fix proposed (`source: url`) | Project | None merged | Reinstall from full URL |
| [#1220](https://github.com/vercel-labs/skills/issues/1220) | `update` silently up-to-date when lockfile `skillPath` casing ≠ repo casing | OPEN (v1.5.7) | Case-insensitive FS (macOS/Windows) lets a casing mismatch persist; `fetchSkillFolderHash` misses it → never fetches changes, no warning | Both (global shown) | None | Match casing exactly in the repo/lockfile |
| [#415](https://github.com/vercel-labs/skills/issues/415) | Stale skills from removed/renamed source repos never pruned | OPEN | `fetchSkillFolderHash()` returns `null` for vanished `skillPath`; `runCheck`/`runUpdate` silently skip → renamed/deleted skills persist forever, new ones never auto-install. `add` is purely additive, never reconciles | Both | None (proposes `--prune` flag) | Manually `remove` stale + `add` repo again |
| [#808](https://github.com/vercel-labs/skills/issues/808) | `skills-lock.json` not updated after `remove` | OPEN (v1.4.6) | `remove` deletes files but leaves the lock entry → lock drifts from disk, breaks later installs | Project | None | Hand-edit `skills-lock.json` after removing |
| [#155](https://github.com/vercel-labs/skills/issues/155) | Project installs don't detect lock; lock only tracks global | OPEN | Early form of the two-lock split behind #542 | Project | None | — |

**Secondary "won't update" causes (same silent-skip family):**
- **Private repos:** [#162](https://github.com/vercel-labs/skills/issues/162) (`skillFolderHash` empty for private repos breaks check/update), [#1318](https://github.com/vercel-labs/skills/issues/1318) / [#1331](https://github.com/vercel-labs/skills/issues/1331) (tree fetch never uses token on 404) — all OPEN.
- **Non-`main` default branch:** [#373](https://github.com/vercel-labs/skills/issues/373) / [#809](https://github.com/vercel-labs/skills/issues/809) — `update` hardcodes `/tree/main/`, so repos using `master` never update. OPEN.
- **Self-hosted GitLab / well-known:** [#997](https://github.com/vercel-labs/skills/issues/997) — source normalized to short slug breaks update. OPEN.
- **`--help` side effects:** [#960](https://github.com/vercel-labs/skills/issues/960) / [#1030](https://github.com/vercel-labs/skills/issues/1030) — `npx skills update --help` (or `-h`) **actually runs the update** instead of printing help. OPEN.

### 5c. `experimental_install` has its own correctness bugs

The restore command isn't a safe harbor either:
- [#1005](https://github.com/vercel-labs/skills/issues/1005) — `experimental_install` strips the subpath from an `owner/repo/<subpath>` source in `skills-lock.json`. **OPEN.**
- [#666](https://github.com/vercel-labs/skills/issues/666) — `experimental_install` does not respect / replaces `https://` in the skill source. **OPEN.**
- [#567](https://github.com/vercel-labs/skills/issues/567) — `experimental_install` overwrote SSH source URLs with HTTPS shorthand. **CLOSED** (the one update/lock-adjacent fix that landed).
- [#781](https://github.com/vercel-labs/skills/issues/781) — `skills-lock.json` hash differs between Windows and Linux (line-ending sensitivity), so cross-platform restores see false drift. **OPEN.**

### 5d. The stable restore/CI story is still vaporware

- [#283](https://github.com/vercel-labs/skills/issues/283) (`skills install`/`sync` from lock) — **OPEN**; collaborator **quuu** replied **"great idea! WIP"** but nothing shipped. Users report committing lockfiles to chezmoi/dotfiles and wanting machine sync. A commenter notes `npx skills update -p` is a partial workaround **but only for already-installed skills**.
- [#549](https://github.com/vercel-labs/skills/issues/549) (`skills install` no-args + `skills ci`) — **OPEN**, unaddressed.
- [#500](https://github.com/vercel-labs/skills/issues/500) (lockfile enhancements: `ci`, `verify`, commit-SHA pinning) — **OPEN.**
- [#683](https://github.com/vercel-labs/skills/issues/683) (global restore/relink from `~/.agents/.skill-lock.json`), [#729](https://github.com/vercel-labs/skills/issues/729) (declarative Skillfile) — **OPEN.**
- A frustrated user has published a third-party CLI, **`bmsuisse/skillup`**, with a migrate command, rather than wait (linked in #283).

### 5e. Safety verdict — can you rely on `update` / `experimental_install`?

**No — not for unattended or reproducible use, as of June 2026.**

- **`skills update`:** Unreliable. It silently no-ops or reports false success across the most common setups — **project-scope installs (the default) aren't seen at all** (#542, #1143, #155), it can **print success while leaving files stale** (#923), and it **silently skips** peers (#1114), renamed/deleted skills (#415), `master`-branch repos (#373/#809), private repos (#162), casing mismatches (#1220), and well-known/GitLab sources (#999/#997). Best-effort workaround for project skills: `npx skills update -p`, then **verify files on disk against the source**.
- **`skills check`:** Not read-only despite the docs — it can mutate/reinstall (#954). Don't use it as a dry-run.
- **`experimental_install`:** Experimental and buggy — mangles subpaths (#1005), `https://` sources (#666), and produces cross-platform hash drift (#781). Usable for a quick single-machine restore of simple `owner/repo` skills, but **not** dependable for CI or portable/multi-machine setups.
- **Lockfile trust:** The lockfile can silently drift from reality — entries survive `remove` (#808), peers never get entries (#1114), and a CLI upgrade can wipe the global lock (#542 Bug 2). Commit it, but don't treat it as ground truth without verifying installed files.

**Bottom line:** For atlas-config's purposes, treat skills.sh skills as **manually managed** — re-`add` from source to update, verify on disk, and don't depend on `update`/`check`/`experimental_install` for correctness or reproducibility until these OPEN bugs are fixed.
