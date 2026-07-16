# Handoff: Finish cross-harness skills and commands consolidation

**Created:** 2026-07-15 16:06 EDT  
**Repository:** `/Users/ed/Developer/Atlas/Config`  
**Branch at handoff:** `feat/atlas-agent-config-updates`  
**Status:** Audit complete; implementation and tracked documentation remain  
**Tracking issue:** [edheltzel/atlas-config#18](https://github.com/edheltzel/atlas-config/issues/18)

## User intent

Finish the previously planned DRY architecture for Atlas workflows:

1. Author reusable behavior once as Agent Skills.
2. Select and deploy shared skills through the root `skills.txt` manifest.
3. Make the workflows conveniently invokable across Claude Code, Pi, and OMP.
4. Keep only thin harness-specific command adapters where a harness cannot consume the canonical surface directly.
5. Restore a **current implementation document**, not the obsolete pre-implementation plan, under tracked `docs/` after the implementation is verified.

Ed believed this might already be implemented. The audit found that the skills architecture is live, OMP compatibility covers many Claude commands, but Pi command parity and several integrity details remain unfinished.

## Safety and worktree state

At handoff creation, the checkout had one pre-existing modification:

```text
 M pi/.pi/agent/settings.json
```

This change was not made during the audit. Preserve it and do not overwrite, revert, stage, or fold it into this work without first establishing ownership. Follow the repository convention to perform implementation in an isolated git worktree where practical.

Before editing any path, re-read the applicable DOX chain:

- `AGENTS.md`
- `docs/AGENTS.md` for tracked documentation
- `pi/.pi/agent/AGENTS.md` for Pi prompt/config changes
- `packages/AGENTS.md` and any child DOX for Themis package changes
- Any upstream `AGENTS.md` in `edheltzel/Skills` before changing shared skills

## What is already implemented

### Manifest-driven shared skills

The root implementation exists and is active:

- `skills.txt`
- `install-skills.sh`
- `skills-lock.json`
- `just skills-status`
- `just skills-install`

`install-skills.sh` reads `owner/repo=skill,skill` entries and runs `skills add ... -g -a '*'`, targeting all detected agents by default.

Live shared skills are stored under `~/.agents/skills`. Claude and Pi have per-skill symlinks back to this store. OMP natively discovers `~/.agents/skills`, so it does not require its own symlink set.

Relevant implementation commit:

```text
b962a26 skills: migrate to skills.txt manifest
```

### OMP loads Claude commands directly

OMP 17.0.0 already scans `~/.claude/commands/*.md` by default. Its current source confirms:

- `commands.enableClaudeUser` defaults to `true`.
- `packages/coding-agent/src/discovery/claude.ts` recursively loads global Claude Markdown commands.
- `skills.enableAgentsUser` defaults to `true`, so `~/.agents/skills` is also loaded.
- Skill slash commands default to enabled as `/skill:<name>`.

This behavior was verified against published OMP 17.0.0. The official release is canonical; no local source worktree or downstream patch branch is required.

Therefore OMP already receives the Claude command files without duplicated OMP prompt files.

### Native Themis support exists

Repository packages exist for both harnesses:

- `packages/pi-themis/`
- `packages/omp-themis/`

Both register `/themis` and ship a `themis-pm` skill. Pi currently has `packages/pi-themis` installed. The OMP package exists but was not listed by `omp plugin list`; only `muse@0.9.3` was active at audit time. OMP can still see Claude's `/Themis` through its Claude-command compatibility provider.

## Current Claude command inventory

Tracked commands under `claude/.claude/commands/`:

```text
Continue.md
Execute.md
ProjectSetup.md
Status.md
Themis.md
design-system-bootstrap.md
```

Live `~/.claude/commands/` also contains stale files no longer tracked here:

```text
context-search.md
cs.md
pu.md
```

OMP's compatibility loader sees these stale live commands too. Reconcile them deliberately; do not blindly delete until ownership and intended replacement are confirmed.

## What remains incomplete or drifted

### 1. Pi has no equivalent prompt templates

Pi's official prompt-template locations are:

```text
~/.pi/agent/prompts/*.md
.pi/prompts/*.md
```

At audit time:

```text
~/.pi/agent/prompts/   missing
~/.pi/agent/commands/  missing
```

No Pi prompt adapters were found in current files or Git history. Pi does **not** natively scan `~/.claude/commands`, unlike OMP.

Pi can invoke loaded skills as `/skill:<name>`, but that does not preserve the current friendly Claude aliases automatically.

Pi references read during the audit:

```text
/Users/ed/.bun/install/global/node_modules/@earendil-works/pi-coding-agent/docs/prompt-templates.md
/Users/ed/.bun/install/global/node_modules/@earendil-works/pi-coding-agent/docs/skills.md
```

Read those files completely again before implementing because Pi may have changed.

### 2. Four PM commands only partially map to `pm-tools`

Claude wrappers:

- `Continue.md`
- `Execute.md`
- `ProjectSetup.md`
- `Status.md`

The canonical shared skill currently lives upstream at:

```text
edheltzel/Skills
skills/productivity/pm-tools/SKILL.md
```

`Continue` and `Execute` have corresponding references:

```text
skills/productivity/pm-tools/references/Continue.md
skills/productivity/pm-tools/references/Execute.md
```

However `pm-tools/SKILL.md` still links to several missing references, including:

```text
ProjectSetup.md
Status.md
Hierarchy.md
Labels.md
items.md
fields.md
```

The installed `~/.agents/skills/pm-tools/references/` contains only `Continue.md` and `Execute.md`. As a result, `ProjectSetup.md` and `Status.md` currently instruct agents to load nonexistent PM recipes.

Also note that Claude's `Continue.md` adds Atlas-specific plan/handoff reconciliation beyond the upstream board-only `Continue.md`. That extra behavior must either move into a canonical shared Atlas skill or remain in a clearly justified thin adapter.

### 3. Design-system command now has a canonical skill, but it is not deployed

The current `edheltzel/Skills` repository contains:

```text
skills/engineering/bootstrap-design-system/SKILL.md
```

This skill is the portable successor to much of:

```text
claude/.claude/commands/design-system-bootstrap.md
```

But `bootstrap-design-system` is not selected in this repository's `skills.txt`, and it was not present in the installed global skill inventory at audit time.

### 4. Themis is not truly code-once

`packages/pi-themis` and `packages/omp-themis` are separate implementations with substantial duplicated behavior and harness-specific deltas. Determine whether to:

- extract a shared doctrine/core file consumed by both adapters, or
- retain separate adapters but document why duplication is required and add parity verification.

Do not force source sharing if the harness APIs make the result less maintainable; the DRY goal is canonical behavior, not abstraction at any cost.

### 5. The lockfile does not match the manifest

`skills-lock.json` currently lists only `lavish` and four Recall source-command entries. It does not cover most `skills.txt` selections such as `pm-tools`, `karpathy-guidelines`, or the Matt Pocock skills.

The root docs/commit history imply that the lock pins resolved skill versions, but the current file does not provide complete reproducibility. Determine the actual role of `skills-lock.json` with the installed skills.sh version and either:

- make it authoritative and verifiably synchronized, or
- stop claiming it pins the manifest and document it as non-authoritative/generated state.

### 6. Issue #18 is still open and underspecified

Current issue body:

```text
pi prompts are similar to command with claude

need to conver these over so that workflow is the same

- note: ask pi to do it for you
```

Do not close it based only on file creation. Rewrite or comment with the final architecture, then close only after cross-harness verification passes.

## Recommended target architecture

Use this as the starting recommendation, but validate command-name collisions and harness behavior before committing:

1. **Canonical behavior:** Agent Skills in `edheltzel/Skills` or a dedicated installable Atlas package.
2. **Distribution:** select canonical skills in root `skills.txt`; install with `just skills-install`.
3. **Claude aliases:** retain minimal Markdown commands only when they add a friendly alias or small Atlas-specific composition.
4. **Pi aliases:** add minimal `pi/.pi/agent/prompts/*.md` adapters that invoke the canonical skill/workflow rather than duplicating its full body.
5. **OMP aliases:** prefer OMP's existing Claude-command compatibility where safe; use OMP-native package commands only for functionality requiring extension APIs/guards, such as Themis.
6. **Parity checks:** add automated checks proving every declared Atlas workflow has a canonical implementation and each intended harness has either native discovery or a thin adapter.
7. **Tracked documentation:** after verification, write the current contract under `docs/agents/` and update the owning DOX indexes. Do not restore the obsolete 2026-06 planning document verbatim.

## Suggested execution sequence

### Phase 1: Re-audit against current versions

- Confirm current branch/worktree safety and preserve the existing `pi/.pi/agent/settings.json` modification.
- Re-run:

```bash
skills --version
skills list -g
skills add edheltzel/skills -l --full-depth
pi list
omp plugin list
```

- Re-read current Pi and OMP command/skill discovery docs/source.
- Inventory intended aliases and check collisions with built-in commands, especially generic names such as `/status` and `/continue`.

### Phase 2: Repair canonical skills first

In `edheltzel/Skills` or the chosen canonical package:

- Restore or remove stale `pm-tools` reference links.
- Move the necessary ProjectSetup and Status behavior into canonical skills/recipes.
- Decide where Atlas-specific Continue reconciliation belongs.
- Verify `bootstrap-design-system` is the canonical replacement for the long Claude command.
- Add focused validation for missing references and frontmatter names.

Do not patch the installed `~/.agents/skills/pm-tools` copy directly; it is generated/installed state.

### Phase 3: Update manifest and adapters

- Add all required canonical skills to `skills.txt`.
- Run `just skills-status` before installation.
- Install only after reviewing the generated commands.
- Replace large Claude command bodies with thin adapters where appropriate.
- Add corresponding Pi prompt templates where friendly aliases are required.
- Decide whether OMP needs any adapters beyond its Claude compatibility provider.
- Reconcile/install `packages/omp-themis` if native OMP Themis is the desired surface.

### Phase 4: Add parity verification

Create a lightweight, dependency-minimal verification surface that checks at least:

- Every skill named in `skills.txt` resolves from its source.
- No installed/canonical SKILL.md links to missing local references.
- Every tracked Claude workflow has a documented canonical skill or an explicit harness-only justification.
- Intended Pi aliases exist and expand successfully.
- OMP loads the intended Claude commands or native extension commands without collisions.
- Stale live commands are reported.
- Lockfile/documentation claims match actual skills.sh behavior.

Prefer dry-run or loader-level tests over invoking paid models.

### Phase 5: Install and smoke-test

- Run relevant unit/static checks.
- Run `just skills-status` and `just skills-install` as appropriate.
- Run scoped stow commands, not an unrelated global mutation.
- Reload Pi and OMP if required.
- Verify the final commands in real autocomplete/runtime surfaces for Claude, Pi, and OMP.
- Confirm the same canonical skill behavior is reached from each alias.

### Phase 6: Document and close

Create a tracked document such as:

```text
docs/agents/skills-and-commands.md
```

It should document:

- canonical source ownership
- `skills.txt` manifest format
- install/update/recovery workflow
- each harness's skill and command discovery behavior
- adapter rules
- parity verification command
- lockfile semantics
- how to add or retire a workflow

Update `docs/AGENTS.md`, root `AGENTS.md`, README, and relevant child DOX only where the durable contract changes. Remove stale or contradictory claims instead of preserving historical explanations.

Update and close issue #18 only after all acceptance criteria below pass.

## Acceptance criteria

- [ ] Shared skill installation remains manifest-driven from root `skills.txt`.
- [ ] Every selected skill resolves from its declared upstream source.
- [ ] No canonical skill references missing bundled files.
- [ ] `Continue`, `Execute`, `ProjectSetup`, `Status`, Themis, and design-system bootstrap each have one documented canonical behavior source.
- [ ] Claude Code can invoke every intended workflow through the agreed command/skill UX.
- [ ] Pi can invoke every intended workflow through `/skill:<name>` and/or verified thin prompt aliases.
- [ ] OMP can invoke every intended workflow through verified Claude compatibility or native package commands.
- [ ] No large command body is duplicated across harness directories without a documented API-driven reason.
- [ ] Themis parity is either enforced from shared doctrine or verified explicitly across adapters.
- [ ] `skills-lock.json` semantics are accurate, synchronized, and tested—or its non-authoritative role is documented.
- [ ] Stale live Claude commands are reconciled without deleting user-owned files accidentally.
- [ ] Cross-harness verification avoids paid-model calls where loader/static checks suffice.
- [ ] A tracked current-state document exists under `docs/agents/` and all affected DOX indexes are current.
- [ ] GitHub issue #18 records the final architecture and is closed only after verification.
- [ ] Pre-existing unrelated worktree changes remain intact and separate.

## Useful evidence and commands from the audit

```bash
# Current tracked Claude commands
find claude/.claude/commands -maxdepth 1 -type f -print | sort

# Live harness surfaces
find ~/.claude/commands ~/.pi/agent/prompts ~/.omp/agent/prompts -maxdepth 2 -type f -print 2>/dev/null | sort

# Installed shared skills and targets
skills list -g
find ~/.claude/skills ~/.pi/agent/skills -maxdepth 1 -type l -print -exec readlink {} \;

# Available canonical skills without installing
skills add edheltzel/skills -l --full-depth

# Relevant packages
pi list
omp plugin list

# Issue
GH_TOKEN="$ATLAS_KEY_PAT" gh issue view 18 --repo edheltzel/atlas-config
```

## Historical context

The old plan was formerly tracked at:

```text
.agents/atlas/plans/2026-06-10-skills-management-workflow.md
```

It was removed from Git when `.agents/atlas/` became local-only in commit:

```text
fdc578c chore: remove codex package; untrack .agents/atlas as local-only
```

That plan described the manifest direction before the current implementation landed. Recover it from Git history only for context; do not restore it verbatim as current documentation.
