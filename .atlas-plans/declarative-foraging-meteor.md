# Plan: Implement InstructionsLoaded Hook + Commit PAIUpgrade Changes

## Context

Two pending items from the PAIUpgrade findings session:

1. **PAIUpgrade/SKILL.md** already has source verification edits (uncommitted) — needs committing
2. **ISC-11 & ISC-12**: Ed confirmed implementing the `InstructionsLoaded` hook event. Per `~/.claude/cache/changelog.md` line 17, this event "fires when CLAUDE.md or `.claude/rules/*.md` files are loaded into context." It's in the changelog but not yet in official docs — a changelog-only feature. Ed explicitly said "yes, implement ISC-11 and ISC-12."

Additional uncommitted changes visible in git status: `atlas-statusline.sh`, `settings.json`, `THEHOOKSYSTEM.md` — these appear to be from prior work and should be reviewed before committing.

## Steps

### 1. Commit PAIUpgrade source verification changes
- Stage and commit `claudecode/.claude/skills/PAIUpgrade/SKILL.md`
- Message: `feat(paiupgrade): add source verification requirements to findings`

### 2. Create `InstructionsLoaded.hook.ts` (ISC-12)
- **File:** `claudecode/.claude/hooks/InstructionsLoaded.hook.ts`
- **Purpose:** Log which instruction files were loaded into context
- **Behavior:**
  - Read stdin for hook event JSON (like all hooks)
  - Log to stderr: which file(s) loaded (CLAUDE.md path or rules file path)
  - Output a `<system-reminder>` to stdout confirming context load (e.g., "PAI instructions loaded from {path}")
  - Non-blocking: always exit 0
- **Pattern:** Follow existing hook conventions (shebang, JSDoc header, stdin reader, typed interfaces)
- **Note:** Since this event type is changelog-only, the exact input schema is unknown. The handler should be defensive — log whatever it receives and gracefully handle unexpected shapes.

### 3. Register `InstructionsLoaded` in settings.json (ISC-11)
- Add new top-level hook event entry in the `hooks` section:
```json
"InstructionsLoaded": [
  {
    "hooks": [
      {
        "type": "command",
        "command": "${PAI_DIR}/hooks/InstructionsLoaded.hook.ts",
        "timeout": 5000
      }
    ]
  }
]
```

### 4. Update PRD
- Revert ISC-11 and ISC-12 from "N/A" to proper completion text
- Update progress counter

### 5. Commit InstructionsLoaded implementation
- Stage: `hooks/InstructionsLoaded.hook.ts`, `settings.json`
- Message: `feat(hooks): add InstructionsLoaded hook for context load confirmation`

### 6. Review other uncommitted changes
- `atlas-statusline.sh` and `THEHOOKSYSTEM.md` — check if these are from prior sessions and should be committed separately or left as-is

## Critical Files
- `claudecode/.claude/hooks/InstructionsLoaded.hook.ts` (NEW)
- `claudecode/.claude/settings.json` (MODIFY — add hook registration)
- `claudecode/.claude/skills/PAIUpgrade/SKILL.md` (COMMIT existing edits)
- `~/.claude/MEMORY/WORK/20260305-180500_paiupgrade-remaining-findings/PRD.md` (UPDATE ISC-11/12)

## Verification
- `bun build --no-bundle claudecode/.claude/hooks/InstructionsLoaded.hook.ts` compiles without errors
- settings.json is valid JSON after edit
- Hook is registered under `InstructionsLoaded` event key
- PRD ISC-11 and ISC-12 marked complete with accurate descriptions
