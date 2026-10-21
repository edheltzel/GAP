# Plan: make Gap the LazyVim of agents (ATL-GAPENV)

Tue Oct 20, 2026, ET. Plan only; nothing has been built yet. The captain answered the open questions on Oct 20; they are folded in below (see Decisions at the end).
Backing evidence, sources and the full component notes are in the appendix `ATL-GAPENV.md` (kept outside this repo, in `/Users/ed/Documents/Atlas/Gap/`). Section numbers like "R§3.3" point there.

**In one breath:** Gap becomes two things in one CLI. First, a **distribution**: our opinionated, batteries-included setup, meaning Forge, Recall, Echo, Do-Skills and the shared policy, wired into every harness. Second, a **manager**: it fetches each component at a pinned version, renders it into each harness's own folders, records it in a lockfile, checks drift, and rolls back. Components stay in their own repos. Gap configures them and doesn't own them. You change behavior through override files, never by editing a component.

---

## 1. Architecture: distribution + manager

| LazyVim | Gap |
|---|---|
| lazy.nvim (manager) | `gap` CLI + `~/.gap/store` + `gap.lock` |
| LazyVim specs (distro defaults) | `GAP/gap.toml` + `GAP/profiles/*.toml` |
| a plugin repo | a component repo (Forge, Recall, Echo, Do-Skills) with its own `gap-component.toml` |
| extras (`lazyvim.plugins.extras.*`) | optional component groups, e.g. `extras.marketing`, `extras.swift` |
| your `lua/plugins/*.lua` overrides | `~/.config/gap/`, `<repo>/.gap/` override files |
| `lazy-lock.json` | `gap.lock` |
| `event` / `ft` / `cmd` triggers | `load = always / listed / project / command / off` + `when` |

Rules that make the split work (the same reasons it works for LazyVim, R§7):
1. The **component** owns its files and its own installer and health check. Gap never edits them.
2. The **distro** (Gap repo) only chooses components, pins and default options.
3. **You** override in your own files, and Gap merges the layers at sync time.
4. The **manager** writes only into harness paths, records every byte it wrote, and can restore from the lock.

What Gap still owns itself: the shared policy (`global/AGENTS.md`, `global/docs/agents/`) and the slash commands (`global/commands/`). Those are Gap's own content, exactly as today.

Layout:
```
/Users/ed/Developer/Atlas/GAP/          (repo edheltzel/GAP)
  gap.toml                    distro: components, pins, defaults
  gap.lock                    generated, committed: exact commit per component
  profiles/machine.toml       what a whole machine gets
  profiles/project.toml       what a repo gets
  profiles/crew.toml          box / cloud: committed files only
  global/                     policy + commands (Gap-owned, unchanged)
  cmd/gap/                    the CLI (Go stdlib)

/Users/ed/.gap/               (machine state; exists today)
  store/<id>/<commit>/        read-only checkouts, one per pin
  state.json                  every rendered file + hash (grows from today's published.json)
  backups/                    exists today

/Users/ed/.config/gap/        your overrides (user + machine)
<repo>/.gap/                  project overrides (gap.toml committed, local.toml ignored)
```

## 2. Manifest and lockfile

**Distro manifest** (`GAP/gap.toml`). It extends the TOML design already locked in `/home/box/docs/gap-manifest-schema.md` (TOML only, `when` conditions, `~/.agents` root).
```toml
schema = "2"
name   = "edheltzel/GAP"

[defaults]
profile   = "machine"
harnesses = ["claude", "omp", "pi", "codex", "cursor", "empryo", "grok"]

[components.do-skills]
repo = "github.com/edheltzel/Do-Skills"
ref  = "master"                 # branch, tag or commit; the lock records the commit
load = "listed"                 # default: description listed, body on demand
[components.do-skills.skills]
do-todo   = { load = "always", harness = ["empryo"] }   # Empryo autoLoad pin
do-commit = { load = "always", harness = ["empryo"] }
do-astro  = { load = "project", when.files = ["astro.config.*"] }
do-review-ios = { load = "project", when.files = ["*.xcodeproj", "Package.swift"] }

[components.recall]
repo = "github.com/edheltzel/Recall"
ref  = "v0.10.2"
required = true                 # ATL-645: Gap relies on Recall
when.profile = ["machine"]

[components.echo]
repo = "github.com/edheltzel/Echo"
ref  = "vX.Y.Z"                 # first tagged Echo release; never master (decision 9)
when.os = "darwin"
when.profile = ["machine"]
[components.echo.opts]
adapters = { claude = "claudecode", pi = "pi", omp = "omp", codex = "codex", grok = "grok" }

[components.forge]
repo = "github.com/edheltzel/Forge"
ref  = "master"
when.profile = ["machine", "project"]

[extras.marketing]               # opt-in, like LazyVim extras
skills = ["do-marketing", "keyword-research", "site-audit", "seo-foundations", "seo-action-plan"]
```

**Component self-description** (lives in each component's repo, e.g. `Recall/gap-component.toml`; the component owns it):
```toml
schema   = "2"
id       = "recall"
requires = ["bun"]
install  = "./packaging/install.sh"          # its own idempotent installer
update   = "./packaging/update.sh"
doctor   = "recall doctor"
restore  = "./packaging/install.sh restore"
skills   = ["agent-skills/*"]                # do-recall-* (Recall-owned, Gap won't drift-fix them)
[harness.cursor]
project_files = { "templates/cursor/rule.md" = ".cursor/rules/recall.mdc" }
```
Echo's would say `install = "cli/echo install --adapter {adapter}"`, `doctor = "cli/echo doctor"`, `uninstall = "cli/echo uninstall"`. Forge's would say `skills = ["skills/forge-interview"]`, `commands = ["commands/forge-interview.md"]`. Do-Skills' would say `skills = ["skills/*/*/*"]`.

**Per-project override** (`<repo>/.gap/gap.toml`, committed):
```toml
profile = "project"
slug    = "rosetta2"                      # same slug in Recall, +tags, factory board
[components.do-skills.skills]
do-astro = { load = "always" }            # this repo is all Astro
[components.echo]
enabled = false                           # quiet repo
```

**Lockfile** (`GAP/gap.lock`, generated, committed, like `lazy-lock.json`):
```toml
schema = "2"
[components.do-skills]
ref    = "master"
commit = "a4eece7…"            # full 40-hex in the real file
tree   = "…"                   # git tree id, used to verify the store copy
[components.recall]
ref    = "v0.10.2"
commit = "4319176…"
[components.echo]
ref    = "vX.Y.Z"                # set once Echo tags a release
commit = "…"
[components.forge]
ref    = "master"
commit = "…"
```
Per machine, `~/.gap/state.json` records what was rendered where: path, hash, which component, which override files were applied. That's today's `published.json` ledger, extended.

## 3. Gap configures harnesses; it doesn't own them

Gap never installs or runs a harness (decision on record: "(C) rejected: gap does not install harnesses", 2026-10-16). `gap doctor` reports which harnesses are present and prints install commands for missing ones. It only checks and prints (decision 1). Targets found on BigMac:

| Harness | Policy (exists) | Skills | Commands | Notes |
|---|---|---|---|---|
| Claude CLI | `/Users/ed/.claude/rules/global-agent-policy.md` | `/Users/ed/.claude/skills/<name>/` | `/Users/ed/.claude/commands/` | never create `~/CLAUDE.md` |
| omp | `/Users/ed/.omp/agent/APPEND_SYSTEM.md` | `/Users/ed/.agents/skills/` | `/Users/ed/.omp/agent/commands/` | Recall/Echo add their own extension and skills |
| Pi | `/Users/ed/.pi/agent/APPEND_SYSTEM.md` | `/Users/ed/.agents/skills/` | — | Recall via `pi install` |
| Codex | `/Users/ed/.codex/gap/global-agent-policy.md` + hooks | `/Users/ed/.agents/skills/` | — | trust hooks once via `/hooks` |
| Cursor (local) | `/Users/ed/.cursor/rules/global-agent-policy.mdc` | `/Users/ed/.agents/skills/` | — | `/Users/ed/.cursor/skills/` also exists |
| Cursor cloud | committed repo files only | committed `<repo>/.agents/skills/` (verify) | — | short pointer note in each repo's `AGENTS.md`, no committed policy copy (decision 4) |
| Empryo | `/Users/ed/.empryo/EMPRYO.md` (GAP block next to SoulStack) | `skills.autoLoad` pins; `~/.empryo/skills/` shadows | — | Gap only reads `config.json` and prints the `settings` call |
| Grok CLI | reads the Claude rules copy | `/Users/ed/.grok/skills/` (shared: Gap and other tools write here) | — | Gap skips a skill that already exists there, so nothing loads twice (decision 7) |
| Herdr | — | — | — | not a Gap dependency; Gap doesn't write its config or register projects (decision 6) |
| Grok Bot crew (box) | task brief + committed repo files | `/home/box/agent-data/workflows/` | — | no Recall, no `~/` files |

Component wiring in short (full detail in R§3):
- **Forge**: installs `forge-interview` skill + command everywhere (not installed today). `gap init` hands new projects to `/forge-interview` and never writes `forge/` itself.
- **Recall** (ATL-645): required on machines. Gap runs Recall's own installer and doctor, leaves `do-recall-*` to Recall, expands the policy rule, and owns one slug per project. When Gap provisions a project it writes a setup breadcrumb to Recall (decision 5).
- **Echo**: macOS machines only. Waits for a tagged Echo release; no `master` pin (decision 9). Core plus one adapter per present harness through `cli/echo install --adapter …`, and `cli/echo doctor`. Gap records the adapters because Echo's uninstall leaves them behind.
- **Do-Skills**: the first external component (section 6).
- **Empryo / codegraph**: check and print. No writes outside the files Gap already owns.
- **Herdr**: no dependency. `gap init` does not register projects with Herdr (decision 6).

## 4. Install flow

**New machine**
1. Install the prerequisites `gap doctor` asks for: git, Go, Bun, and the harnesses you use. Gap prints the install commands; it never installs a harness.
2. `git clone https://github.com/edheltzel/GAP && cd GAP && go install ./cmd/gap`
3. `gap sync --profile machine` → fetch pins from `gap.lock` into `~/.gap/store` → publish policy and commands → render skills → run Recall's and Echo's installers → write `state.json`.
4. Do the steps Gap prints but can't do for you: Codex `/hooks` trust, the `y` for the `EMPRYO.md` splice, the Empryo `settings` call.
5. `gap doctor` → all green.

**Existing project** (`gap init --adopt`)
1. Read only: detect `AGENTS.md`, `CLAUDE.md`, `todo.txt`, `forge/`, `.codegraph/`, `.empryo/`, the stack files.
2. Show a dry-run list of proposed writes. Every write is a marker-guarded block or a new file. Foreign files get a backup plus an append only after a yes, the same rule as `EMPRYO.md` today.
3. Write `.gap/gap.toml` (slug, profile), the AGENTS.md policy pointer note, `.cursor/rules/forge.mdc`, `CLAUDE.md` = `@AGENTS.md` (if you agree), and the `.gitignore` lines for `.empryo/`, `.codegraph/` and `.gap/local.toml`.
4. If `forge/` is missing, suggest a brownfield `/forge-interview`.
5. `gap sync` installs this project's `load = "project"` skills into `<repo>/.agents/skills/`.
6. Gap writes a setup breadcrumb to Recall for the project slug.

**New project**: `gap init` → `/forge-interview` → `gap sync` → commit.

## 5. Updates and versions

| Verb | What it does | LazyVim equivalent |
|---|---|---|
| `gap sync` | install missing, render, publish, clean what the manifest dropped; **never moves pins** | `:Lazy install` + `clean` |
| `gap check` | fetch and show what's new upstream per component | `:Lazy check` |
| `gap update [id]` | move pins to the newest allowed commit, rewrite `gap.lock`, re-render | `:Lazy update` |
| `gap restore [id]` | put everything back to the lock (the **rollback**) | `:Lazy restore` |
| `gap pin id@ref` | change a request (tag/commit) | `version=` / `pin=true` |
| `gap status` / `gap diff` | pin, commit, drift per component; show diffs | `:Lazy` UI / log |
| `gap doctor` | prerequisites, harnesses, each component's own doctor | `:checkhealth lazy` |

- Drift check (exists today for skills): compare each rendered file with what `state.json` says it should be. The keep / take / skip prompt stays. Without a terminal it lists the drift and exits 1.
- Rollback of a bad update: `git revert` the `gap.lock` change, then `gap restore`. The store still holds the old commit, so no network is needed. Components with their own rollback (Recall's `install.sh restore`) get it called.
- Two Gap-side safety rules stay: the merge gate (publish only from master/GitButler workspace) and regular files only.

## 6. Migration from today's swallowed skills (depends on ATL-GAPDEUCE)

Today: 66 Do skills were **copied without history** into GAP (PR #78 on Oct 16, PR #80 on Oct 19; 0 shared commits with Do-Skills; GAP-side edits since then in #81, #82, #83, #84, #86, `4a95564`). Details in R§8.

**ATL-GAPDEUCE (Intern, in progress) already covers:** unarchiving Do-Skills (done), porting GAP-side edits back, removing the README warning, and making Gap pull Do-Skills at a pinned ref. **This plan doesn't redo that.** Decided (decision 2): part 2 **lands the pin in the M1 file shapes** (`gap.toml` `[components.do-skills]` + `gap.lock`), not as a new field in `global/skills.json`.

Order that keeps the 66 installed skills and drift checks working:
1. Do-Skills master = GAP's current skill bytes (ATL-GAPDEUCE part 1).
2. Gap points at Do-Skills at exactly that commit, so content hashes match.
3. `gap sync` on a throwaway HOME, then on BigMac. Gap's existing source-identity migration sees matching hashes and switches the source without prompting (it backs up first).
4. Then remove `skills/` and the copied `docs/global`, `docs/project` from GAP.
5. Retarget the install docs (section 9).

## 7. Milestones

| # | Milestone | Done when |
|---|---|---|
| **M1 (recommended first)** | **Components + lock, Do-Skills as the only component.** `gap.toml` `[components]`, `gap.lock`, `~/.gap/store`, `gap status`, `gap restore`. Still installs through today's skills step. | In a temp HOME: `gap sync` then `gap restore` give byte-identical `~/.agents/skills` and `~/.claude/skills` for all 66; a second sync changes nothing; on BigMac `gap status` shows zero drift. |
| M2 | **Gap's own skill installer** (replaces `npx skills add`) + `load` levels + project skills | No `npx` call; same bytes as M1; `do-astro` appears only in Astro repos |
| M3 | **Recall** component + `gap doctor` (ATL-645) | `gap doctor` runs `recall doctor`; slug set by `gap init`; setup breadcrumb written to Recall |
| M4 | **Echo** component (macOS), after Echo's first tagged release | adapters recorded in `state.json`; `cli/echo doctor` green via `gap doctor` |
| M5 | **Forge** component + `gap init` / `gap init --adopt`. Forge goes public first, after a clean secrets scan and the captain's go | forge-interview installed everywhere; adopt dry-run on Recall/Echo/GAP repos makes no unmarked writes |
| M6 | **Override layer** (section 8) | an override survives `gap update` and re-renders clean |
| M7 | **Crew/cloud profile** + new-machine bootstrap | a cloud agent in a fresh repo reads AGENTS.md → forge/status.md; fresh-HOME `gap sync --profile machine` → doctor green |

Why M1 first: it's small, testable with the throwaway-HOME harness Gap already has, builds directly on ATL-GAPDEUCE, and gives every later component the same lock/store/restore path.

## 8. Override layer (change behavior without touching a component)

**Where overrides live** (later wins):
1. component defaults (`gap-component.toml` in the component repo)
2. distro (`GAP/gap.toml`, `GAP/profiles/*.toml`)
3. user: `/Users/ed/.config/gap/gap.toml` + `/Users/ed/.config/gap/overrides/` (dotfiles may symlink `~/.config/gap`; Gap reads through the link)
4. machine: `/Users/ed/.config/gap/machines/<hostname>.toml`
5. project: `<repo>/.gap/gap.toml` + `<repo>/.gap/overrides/` (committed)
6. project-local: `<repo>/.gap/local.toml` (gitignored, like mise's `*.local.toml`)

**Merge rules** (copied from lazy.nvim, R§7): tables like `opts` **merge**; lists like `harness`, `when.files`, `exclude` **extend**; scalars like `ref`, `load` **replace**; `enabled = false` removes the component or skill. To remove an item from a list, use `exclude = [...]`.

**Overriding a skill's text or frontmatter** (applied at render time into the harness copy, never in `~/.gap/store` or the component checkout):
```
/Users/ed/.config/gap/overrides/do-skills/do-commit/
  frontmatter.toml     # [frontmatter] description = "…"   (field-level replace)
  append.md            # appended to the SKILL.md body
  SKILL.md.patch       # optional unified diff for surgical edits
```
Render = pinned source file → frontmatter merge → patch → append → write to `~/.agents/skills/do-commit/SKILL.md` and the Claude copy → hash into `state.json` with the list of override files used.

**Overriding a harness config or component setting:**
- Component options: `[components.echo.opts] adapters = { pi = "pi" }` (merge), or `[components.echo] enabled = false` on the office Mac's machine file.
- Harness files Gap already owns (policy, commands): a user `append.md` per destination, rendered inside the GAP block.
- Harness files Gap doesn't own (`~/.empryo/config.json`): Gap prints the change and doesn't write it (today's Empryo rule).

**Updates stay clean:** every `gap update` re-renders from the new pinned source plus the same override files. If a patch no longer applies, Gap keeps the last good render, marks it `override-conflict` in `gap status`, and shows the rejected hunk. Hand edits in harness paths show up as drift (keep / take / skip). The fix is to move the edit into an override file.

## 9. Replace skills.sh with Gap's own skill handling

**skills.sh today** (sources in R§9): `npx skills add <source>` puts the skill in `.agents/skills/` and symlinks or copies it into each agent folder. Lock `~/.agents/.skill-lock.json` (global, v3) or `./skills-lock.json` (project). Updates compare GitHub tree SHAs, and old lock versions get wiped.

| | skills.sh (today) | Gap (proposed) |
|---|---|---|
| Runtime | Node + `npx` per install | none beyond the `gap` binary (Go stdlib) |
| Unit | one skill name per `-s` flag | a component (repo@commit) with many skills |
| Source of truth | its own lock (source strings can differ from Gap's manifest, which forced Gap's "source-identity migration") | `gap.toml` + `gap.lock` only |
| Placement | canonical dir + symlinks into agent dirs | regular-file copies per harness (Gap's rule) |
| Update check | GitHub Trees API per skill | one `git fetch` per component |
| Restore | `skills experimental_install` (project only) | `gap restore` (machine and project) |
| Lazy loading | none; everything listed everywhere | `always / listed / project / command / off` + `when` |
| Overrides | none (edits show as drift) | override layer (section 8) |
| Per-harness targeting | `-a claude-code -a codex …` | from `defaults.harnesses` + per-skill `harness` |

**How Gap does it:** fetch component → read its `skills` globs → apply `load`/`when`/`enabled` → render with overrides → copy to `~/.agents/skills/<name>/` (Codex, Cursor, Pi, omp read this) and `~/.claude/skills/<name>/` (Claude), or `<repo>/.agents/skills/` for project skills → hash into `state.json`. For `~/.grok/skills/`, Gap skips any skill name that already exists there and reports it, so nothing loads twice. `load = "always"` for Empryo stays a printed `settings` call. Gap never writes `~/.empryo/skills/` and keeps the shadow warning.

**Migration off `npx skills add`:**
1. M2 reads `~/.agents/.skill-lock.json` once, adopts entries whose bytes match the pinned source (no reinstall), and flags the rest as drift.
2. Gap stops writing that lock. It leaves it in place for skills Gap doesn't manage.
3. Replace the symlinks in `~/.claude/skills` with regular copies (backup first), one prompt for the batch.
4. Do-Skills README install section changes from `npx skills add edheltzel/Do-Skills …` to: "Install with Gap: `gap add do-skills` (or it's already in the default distro). Project skills install automatically when the repo matches." Keep a short "without Gap" note pointing at skills.sh for outside users, if you want one. External skills (gitbutler, plannotator, codegraph, …) stay in the default distro (decision 8).
5. GAP README "Manage skills" section and the "you need Node" note go away.

---

## Decisions (captain, Oct 20 2026)
1. **Harness installs**: check and print only. Gap never installs a harness.
2. **Do-Skills pin**: use the new `gap.toml` + `gap.lock` format now (ATL-GAPDEUCE part 2).
3. **Forge access**: make `edheltzel/Forge` public, after a secrets scan and the captain's confirmation before the flip.
4. **Cloud policy**: a short pointer note in each repo's `AGENTS.md`. No committed full copy.
5. **Recall writes**: yes. Gap leaves a setup breadcrumb in Recall when it provisions a project.
6. **Herdr**: no Herdr dependency. `gap init` does not register projects with Herdr.
7. **Grok skills** (`~/.grok/skills/`): shared. Gap and other tools may write there; Gap checks for existing skills and skips duplicates.
8. **External skills** (gitbutler, plannotator, codegraph, …): stay in the default distro.
9. **Echo pin**: wait for a tagged release. Don't pin `master`.
10. **Overrides**: `~/.config/gap/` (personal) and `<repo>/.gap/` (project). Dotfiles may link to `~/.config/gap`.
11. **Eight Sleep `APP_CLIENT_SECRET`** (commit `98ee113c`, public repo): replace it now, take it out of the repo, and load it from a secret.
