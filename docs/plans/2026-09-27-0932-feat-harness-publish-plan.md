---
title: Publish GAP to Codex and Cursor - Plan
type: feat
date: 2026-09-27
artifact_contract: ce-unified-plan/v1
product_contract_source: ce-plan-bootstrap
execution: code
---

# Publish GAP to Codex and Cursor - Plan

## Goal Capsule

Objective: one GAP policy reaches every harness that does not already load it, without a second copy of the policy and without silently destroying rules already on disk.

Means: `src/sync.ts` keeps `global/AGENTS.md` as the only policy source, adapts that text per destination, and treats a destination as ours only when it matches the bytes about to be written.

Stop if a foreign destination cannot be backed up, or if a prompt is required and nobody can answer.

## Product Contract

### Summary

Sync publishes one policy. Codex gets that file. Cursor gets the same body wrapped so Cursor will load it. Grok gets nothing new, because it already loads Claude's rules. The first foreign file is moved aside after a prompt. Later publishes overwrite quietly.

### Problem Frame

Claude, Pi, and OMP already receive GAP. Codex and Cursor do not. Codex already has its own global rules. A blind overwrite would delete them. Cursor will not load a raw markdown copy. Grok, with Claude compatibility at the default, already reads `~/.claude/rules/*.md`, so a second Grok file would load the policy twice.

### Requirements

- R1. `global/AGENTS.md` stays the only policy source. Sync may wrap or rewrite that text per harness. It must not add a second policy document.
- R2. A destination that is missing, or whose current bytes equal the payload about to be written, is overwritten with no prompt and no backup.
- R3. A destination that exists and does not match that payload is foreign. Sync tells the user, waits for a yes, moves the file into a backup directory, prints that path, and then writes the payload. The message tells the user to review the backup and add anything they want to keep into the policy by hand.
- R4. If a foreign file needs a prompt and nobody can answer, sync exits non-zero and writes nothing. It prints the path it would have replaced.
- R5. Codex's destination is `AGENTS.md` inside Codex home (`CODEX_HOME`, or `~/.codex` when unset). Do not write `AGENTS.override.md`.
- R6. Cursor's destination is a new `~/.cursor/rules/global-agent-policy.mdc` with `alwaysApply: true` around the same policy body. Do not modify `~/.cursor/rules/context7.mdc`.
- R7. Do not add a Grok destination while Grok's Claude rules compatibility is on. Grok already loads `~/.claude/rules/*.md`.
- R8. Claude, Pi, and OMP stay on their current destinations. A file that already matches the adapted payload is not treated as foreign.

### Scope Boundaries

In scope: the publish script, the Codex and Cursor destinations, and the short load notes in the repo contracts.

Out of scope: importing backup text into `global/AGENTS.md` for the user, account-synced Cursor settings, and a Grok copy.

#### Deferred to Follow-Up Work

- A Grok destination if Claude compatibility is turned off.
- Folding reviewed backup text into the policy. That stays a manual edit.

### Open Questions

None blocking. R4 is a planning choice, not a dialogue selection. See KTD3.

## Planning Contract

### System-Wide Impact

Publish writes live harness files under the home directory. Proof uses a temp `HOME`. Agents run `npm run sync` with no prompt, so a foreign file on that path stays put. Grok already reads Claude's rules file. A Grok copy would double-load the policy.

### Key Technical Decisions

- KTD1. Identity is the adapted payload, not the source file. Publish rewrites `](docs/agents/` to an absolute docs path, so a good publish never equals `global/AGENTS.md`. Comparing to the source would prompt on every later run. (Governs R2, R8)
- KTD2. Cursor adaptation is frontmatter only. The body is the same rewritten policy. A second markdown policy would break R1. (session-settled: user-directed, chosen over a second `global/AGENTS.md`: one source, TypeScript adapts per harness)
- KTD3. A non-interactive run that finds a foreign file exits without writing. Agents are told to run `npm run sync` after edits, and that run has no one to prompt. Replacing a foreign file there would violate R3. (Governs R4)
- KTD4. Backup is a move of the destination path into `~/.gap/backups/<timestamp>/`, preserving the tail of the path so the printed location is findable. For a symlink, move the link before any unlink. The current script unlinks a symlink before it can save the target. (Governs R3)
- KTD5. No Grok destination. Installed Grok 1.0.41 docs say home rules include `~/.claude/rules/` when `compat.claude.rules` is on, and that compatibility is the default. This machine's Grok config does not turn it off. (Governs R7)

### Assumptions

- Codex home is `~/.codex` on this machine because `CODEX_HOME` is unset. The script must still honor `CODEX_HOME` when set.
- Cursor loads `~/.cursor/rules/*.mdc` with `alwaysApply: true` as a machine-local user rule. The help page names that directory. A live Cursor session was not run. U3's check is the proof.

## Implementation Units

### U1. Foreign-file backup and quiet overwrite

Goal: publish distinguishes ours from foreign, and only a foreign file is backed up.

Files: `src/sync.ts`

Approach: Build the payload first. If the dest is missing or byte-matches that payload, write it. If it exists and differs, require a yes on a TTY, move it under `~/.gap/backups/<timestamp>/`, print the path and the review reminder, then write. No TTY means exit non-zero and write nothing. Move a symlink before unlinking it. Add `node src/sync.ts --check` that asserts the identity and backup decision with in-memory fixtures. No test framework.

Execution note: Prove `--check` fails before the branch exists, then make it pass.

Test scenarios:
- A payload that differs from the source only by the docs-link rewrite counts as ours when the dest matches it.
- Other dest text is foreign.
- No TTY and a foreign dest writes nothing.

### U2. Codex destination

Goal: Codex receives the same adapted policy at the file it loads.

Files: `src/sync.ts`

Approach: Add one destination, `AGENTS.md` under Codex home. Reuse U1. Do not add `AGENTS.override.md`.

### U3. Cursor destination

Goal: Cursor receives the policy in a file it will load, without a second policy body and without touching the existing Context7 rule.

Files: `src/sync.ts`

Approach: Add `~/.cursor/rules/global-agent-policy.mdc`. Frontmatter is `alwaysApply: true` and nothing else. Body is the U1 payload. Identity includes the wrapper, so a later sync matches the whole file.

Test scenarios:
- The written file starts with `alwaysApply: true` and its body equals the shared payload.
- `context7.mdc` is not a destination.

### U4. Load notes

Goal: the repo contracts name the new destinations and the Grok non-destination.

Files: `AGENTS.md`, `global/AGENTS.md`, `README.md`

Approach: Add the Codex and Cursor paths next to the existing load list. State that Grok already loads Claude's rules file, so sync does not write a Grok copy. Do not paste the policy into those notes.

## Verification Contract

- `node src/sync.ts --check` passes.
- With `HOME` set to an empty temp directory, `npm run sync` writes the Claude, Pi, OMP, Codex, and Cursor destinations and does not prompt.
- A second sync in that temp home overwrites quietly and does not create a backup.
- A temp Codex file with other text, and a TTY yes, moves that file under `~/.gap/backups/` and writes the policy. A non-TTY run on that file exits non-zero and leaves the file in place.
- Do not run the proof against the real home.

## Definition of Done

- R1 through R8 hold in the temp-home proof.
- `--check` covers the ours-versus-foreign decision, including a payload that differs from the source only by the docs-link rewrite.
- `context7.mdc` is not a destination.
- No Grok path is written.

## Sources

- `src/sync.ts` writes three destinations from `global/AGENTS.md` and rewrites docs links. It does not read destination text.
- Codex discovery: global scope is `AGENTS.override.md` if present, otherwise `AGENTS.md`, under Codex home. https://learn.chatgpt.com/docs/agent-configuration/agents-md
- Cursor user rule files: `~/.cursor/rules`. Plain `.md` in `.cursor/rules` is ignored. https://cursor.com/help/customization/rules and https://cursor.com/docs/rules
- Grok home rules include `~/.claude/rules/` when Claude compatibility is on. `~/.grok/docs/user-guide/12-project-rules.md`
