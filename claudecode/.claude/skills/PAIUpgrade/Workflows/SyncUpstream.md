# SyncUpstream Workflow

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the SyncUpstream workflow to check for upstream PAI changes"}' \
  > /dev/null 2>&1 &
```

Running the **SyncUpstream** workflow in the **PAIUpgrade** skill to sync with upstream PAI...

**Syncs Atlas config with the upstream PAI repo.** Diffs upstream changes since last sync, categorizes by relevance, identifies conflicts with Atlas customizations, and produces a merge plan.

**Trigger:** "sync upstream", "check PAI repo", "sync PAI", "upstream changes", "PAI repo updates", "pull PAI", "merge upstream"

---

## Overview

Atlas is a customized fork of PAI. The upstream repo at `~/Developer/AI/PAI` is the source of truth for new features, bug fixes, and releases. This workflow bridges upstream changes into our atlas-config while preserving Atlas-specific customizations.

```
Upstream PAI Repo              Atlas Config (atlas-config)
~/Developer/AI/PAI             claudecode/.claude/
┌──────────────────┐           ┌──────────────────────┐
│ Releases/v4.0.3/ │           │ skills/PAI/           │
│   .claude/       │  ──diff── │ hooks/                │
│     PAI/         │  ──merge─ │ settings.json         │
│     skills/      │           │ ...                   │
│     hooks/       │           └──────────────────────┘
│     settings.json│
└──────────────────┘
```

The workflow:
1. Pulls latest from upstream (with user approval)
2. Diffs against the last sync point
3. Categorizes changes by type and relevance
4. Cross-references Atlas protected files
5. Produces a merge plan
6. Executes with user approval per category
7. Stores the new sync point

---

## State

**Sync state file:** `~/.claude/skills/PAIUpgrade/State/upstream-sync.json`

```json
{
  "upstream_repo": "~/Developer/AI/PAI",
  "release_dir": "Releases/v4.0.3/.claude",
  "last_sync_ref": "52e7ad8",
  "last_sync_tag": "v4.0.3",
  "last_sync_date": "2026-03-16T08:30:00-04:00",
  "algorithm_version": "v3.7.0-atlas"
}
```

If this file doesn't exist, create it during first run using current known state.

**Protected files** — these are Atlas customizations that must NEVER be overwritten wholesale from upstream:

| File | Reason |
|------|--------|
| `skills/PAI/Components/Algorithm/*.md` | Atlas fork with custom voice, paths, extensions |
| `skills/PAI/Tools/Banner.ts` | Atlas branding |
| `settings.json` | Atlas identity, hooks, permissions — merge field-by-field only |
| `hooks/VoiceGreeting.hook.ts` | Atlas startup behavior |
| `hooks/VoiceGate.hook.ts` | Atlas voice routing |

---

## Execution

### Step 1: Read Sync State

```
1. Read ~/.claude/skills/PAIUpgrade/State/upstream-sync.json
2. If missing, initialize with:
   - upstream_repo: ~/Developer/AI/PAI
   - release_dir: Releases/v4.0.3/.claude
   - last_sync_ref: the git ref of the v4.0.3 tag
   - last_sync_tag: v4.0.3
   - algorithm_version: current from LATEST file
3. Report: "Last sync: {tag} @ {ref} on {date}"
```

### Step 2: Check for Upstream Changes

```bash
cd ~/Developer/AI/PAI

# Check for remote changes without pulling
git fetch origin --dry-run 2>&1

# Show commits since last sync
git log --oneline {last_sync_ref}..origin/main | head -40

# Count changes
git log --oneline {last_sync_ref}..origin/main | wc -l
```

**If no new commits:** Report "Already up to date" and exit.

**If new commits found:** Ask user whether to pull.

```
AskUserQuestion: "Found {N} new commits upstream since {last_sync_tag}.
Pull latest and analyze changes?"
Options: ["Pull and analyze", "Just show commit log", "Skip"]
```

### Step 3: Pull and Diff

If user approves:

```bash
cd ~/Developer/AI/PAI && git pull

# Get the diff stats
git diff --stat {last_sync_ref}..HEAD -- {release_dir}/

# Check for new tags
git tag --sort=-creatordate | head -5
```

### Step 4: Categorize Changes

Spawn 1 agent to analyze the diff and categorize:

```
Use Task tool with subagent_type=Intern:

"Analyze upstream PAI changes and categorize them.

Run these commands:
  cd ~/Developer/AI/PAI
  git diff --stat {last_sync_ref}..HEAD -- {release_dir}/
  git log --oneline {last_sync_ref}..HEAD

Categorize every changed file into one of these buckets:

ALGORITHM — Changes to PAI/Algorithm/ (version bumps, new algorithm files, rule changes)
SKILLS — Changes to skills/ (new skills, skill fixes, workflow updates)
HOOKS — Changes to hooks/ (new hooks, hook fixes, lib changes)
INSTALLER — Changes to PAI-Install/ or install.sh (not relevant to Atlas — we use stow)
SETTINGS — Changes to settings.json (need field-by-field merge)
DOCS — Changes to PAI/*.md, CLAUDE.md, README.md (documentation updates)
TOOLS — Changes to PAI/Tools/ or lib/ (utility scripts)
OTHER — Anything else

For each category, report:
1. Files changed (with brief description of what changed)
2. Whether any PROTECTED files are affected
3. Relevance to Atlas: HIGH / MEDIUM / LOW / SKIP

Protected files (NEVER overwrite wholesale):
- skills/PAI/Components/Algorithm/*.md (Atlas fork)
- skills/PAI/Tools/Banner.ts (Atlas branding)
- settings.json (Atlas identity — merge fields only)
- hooks/VoiceGreeting.hook.ts (Atlas startup)
- hooks/VoiceGate.hook.ts (Atlas voice routing)

Return structured report with categories, file lists, and relevance ratings."
```

### Step 5: Produce Merge Plan

Based on the categorization, produce a merge plan:

```markdown
# Upstream Sync Report

**Upstream:** ~/Developer/AI/PAI
**Commits:** {N} new since {last_sync_tag} ({last_sync_ref})
**New tags:** {tags if any}
**Date range:** {earliest commit} to {latest commit}

---

## Change Categories

### ALGORITHM {relevance}
{files and descriptions}
⚠️ Protected: Atlas maintains its own algorithm fork. Cherry-pick specific improvements only.

### SKILLS {relevance}
{files and descriptions}

### HOOKS {relevance}
{files and descriptions}

### SETTINGS {relevance}
{files and descriptions}
⚠️ Protected: Field-by-field merge only.

### INSTALLER — SKIP
{files — not relevant, Atlas uses stow}

### DOCS {relevance}
{files and descriptions}

---

## Merge Plan

| # | Action | Category | Files | Risk |
|---|--------|----------|-------|------|
| 1 | [specific action] | [cat] | [files] | Low/Med/High |

## Protected File Conflicts

| Protected File | Upstream Change | Atlas Action |
|----------------|-----------------|--------------|
| [file] | [what changed upstream] | [merge strategy] |
```

### Step 6: Execute Merge (with approval)

Present the merge plan and ask for approval per category:

```
AskUserQuestion: "Here's the merge plan. Which categories should I apply?"
Options (multiSelect):
- "Algorithm changes (cherry-pick into atlas fork)"
- "Skill fixes (direct apply)"
- "Hook changes (review and apply)"
- "Settings updates (field-by-field merge)"
- "Doc updates (direct apply)"
- "Skip all — just save the report"
```

For each approved category:
1. Apply the changes using appropriate strategy (direct copy, merge, cherry-pick)
2. For Algorithm changes: read upstream diff, apply relevant parts to the `-atlas` file, preserve Atlas customizations
3. For settings.json: read both versions, merge specific fields, preserve Atlas identity/hooks/permissions
4. For protected files: show the diff and ask before each change
5. Run `bun skills/PAI/Tools/RebuildPAI.ts` if Algorithm or SKILL.md components changed

### Step 6b: Post-Merge Sanitization

After applying upstream changes, run these hygiene checks on `CLAUDE.md`:

1. **Remove "This file does nothing"** — upstream PAI ships CLAUDE.md with this placeholder line. If present, replace with `# Atlas — Global Agent Rules` header. This line causes agents to deprioritize the mandatory rules that follow it.
2. **Verify mandatory sections exist** — check that these sections are present (they're Atlas additions, not upstream):
   - `## Plans Directory (MANDATORY)` — `.atlas/plans/` rule
   - `## Git Worktrees (MANDATORY)` — worktree-first rule
   - `## MEMORY` — Recall integration
3. **If any mandatory section was removed by the merge**, restore it from git (`git show HEAD:claudecode/.claude/CLAUDE.md`).

### Step 7: Update State

After successful merge:

```json
{
  "upstream_repo": "~/Developer/AI/PAI",
  "release_dir": "Releases/v4.0.3/.claude",
  "last_sync_ref": "{new HEAD ref}",
  "last_sync_tag": "{new tag if any, else previous}",
  "last_sync_date": "{ISO timestamp}",
  "algorithm_version": "{current from LATEST}"
}
```

Write updated state to `~/.claude/skills/PAIUpgrade/State/upstream-sync.json`.

---

## Quick Mode

If user says "just check upstream" or "any PAI updates":
- Fetch only (no pull)
- Show commit log and change summary
- Don't produce full merge plan
- Report: "{N} commits since {tag} — run 'sync upstream' for full analysis"

---

## New Release Mode

When upstream has a new release tag (e.g., v5.0.0):

1. **Detect:** Compare tags — if new tag exists beyond `last_sync_tag`
2. **Alert:** "New PAI release detected: {tag}. This is a major/minor/patch release."
3. **Compare release dirs:** Check if a new `Releases/vX.Y.Z/` directory exists
4. **If new release dir:** Update `release_dir` in state to point to the new release
5. **Full diff:** Run full categorization against the new release
6. **Migration guide:** Check for `Releases/vX.Y.Z/MIGRATION.md` or `CHANGELOG.md`

```
AskUserQuestion: "New release {tag} detected with a new release directory.
Update sync target from {old_release_dir} to {new_release_dir}?"
Options: ["Yes, switch to new release", "No, stay on current release", "Show me the changelog first"]
```

---

## Error Handling

- **No upstream repo found:** Report path and ask user to verify `~/Developer/AI/PAI` exists
- **Dirty upstream working tree:** Warn and ask whether to proceed
- **Merge conflicts:** Never auto-resolve — show the conflict and ask user
- **Protected file changed upstream:** Always show diff and ask before applying
- **RebuildPAI fails:** Report error, don't update state

---

## Example

```
User: "sync upstream"

Atlas: Last sync: v4.0.3 @ 52e7ad8 on 2026-03-05

[fetches upstream]

Atlas: Found 22 new commits since v4.0.3. No new release tags.

Categories:
- ALGORITHM (HIGH): New v3.7.0 algorithm file, LATEST pointer updated
- SKILLS (MEDIUM): Telos path fix, Art dead refs removed, Apify dead ref removed
- INSTALLER (SKIP): bun dedup, unzip check — not relevant (we use stow)
- SETTINGS (LOW): Version string bumps only
- DOCS (LOW): README, CLI.md, DOCUMENTATIONINDEX.md dead ref cleanup

Protected file conflicts:
- Algorithm: Upstream has v3.7.0, we have v3.5.0-atlas → cherry-pick into v3.7.0-atlas
- settings.json: algorithmVersion "3.5.0" → "3.7.0" (field update only)

[User approves Algorithm + Skills + Settings]

[Applies changes, rebuilds SKILL.md]

Atlas: Sync complete. Updated to 22 commits past v4.0.3. State saved.
```
