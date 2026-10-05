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
- R3. A destination that exists and does not match that payload is foreign, unless its sha256 equals the last publish recorded in `~/.gap/published.json`. That file is ours and is overwritten with no prompt. Sync tells the user about a foreign file, waits for a yes, moves the file into a backup directory, prints that path, and then writes the payload. The message tells the user to review the backup and add anything they want to keep into the policy by hand.
- R4. If a foreign file needs a prompt and nobody can answer, sync skips that file, prints the path, and exits non-zero. It still publishes every other destination.
- R5. Codex's policy file is `gap/global-agent-policy.md` inside Codex home (`CODEX_HOME`, or `~/.codex` when unset). A SessionStart hook prints that file. Do not write `AGENTS.override.md`. Sync does not rewrite Codex `AGENTS.md` on a policy edit.
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

Raised by the 2026-10-04 status review. See Status Review.

- Q1. Resolved 2026-10-04. Refuse only the foreign file. R4 amended. One stale file must not block the other harnesses.
- Q2. Resolved 2026-10-04. The last-publish ledger counts as ours. R3 and KTD1 amended. Without it, every policy edit turns every copy foreign.

## Planning Contract

### System-Wide Impact

Publish writes live harness files under the home directory. Proof uses a temp `HOME`. Agents run `npm run sync` with no prompt, so a foreign file on that path stays put. Grok already reads Claude's rules file. A Grok copy would double-load the policy.

### Key Technical Decisions

- KTD1. Identity is the adapted payload, not the source file, plus the sha256 of the last publish in `~/.gap/published.json`. A file that matches that digest is ours. Publish rewrites `](docs/agents/` to an absolute docs path, so a good publish never equals `global/AGENTS.md`. Comparing to the source would prompt on every later run. (Governs R2, R3, R8)
- KTD2. Cursor adaptation is frontmatter only. The body is the same rewritten policy. A second markdown policy would break R1. (session-settled: user-directed, chosen over a second `global/AGENTS.md`: one source, TypeScript adapts per harness)
- KTD3. A non-interactive run that finds a foreign file skips that file and still writes the others. Agents are told to run `npm run sync` after edits, and that run has no one to prompt. Replacing the foreign file there would violate R3. Blocking every harness would leave policy stale wherever the file is already ours. (Governs R4)
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

### U5. Codex SessionStart hook

Goal: Codex receives the policy through a SessionStart hook. GAP owns `$CODEX_HOME/gap/global-agent-policy.md` alone. Sync does not rewrite `~/.codex/AGENTS.md` on a policy edit.

Files: `src/sync.ts`, `AGENTS.md`, `global/AGENTS.md`, `README.md`

Approach: Write the adapted payload with the normal whole-file rules and a temp-file rename. Add one SessionStart entry, matcher `startup|resume|clear|compact`, only when `gap/global-agent-policy.md` is missing from `hooks.json`. The command prints that file. Other hooks stay byte for byte, and the file is backed up first. One-time cleanup removes an unmarked prefix of Codex `AGENTS.md` only when its sha256 matches the ledger, then keeps the suffix byte for byte. A mismatch is refused and the file is left unchanged. Codex CLI 0.160 and the hooks docs say SessionStart sources are `startup`, `resume`, `clear`, and `compact`, and plain text on stdout is extra developer context.

Test scenarios:
- A copied live `AGENTS.md` whose prefix matches the ledger loses only that prefix. The blank line and both tool blocks stay byte for byte.
- An unmatched prefix is left unchanged.
- A copied live `hooks.json` gains one SessionStart entry. Every other hook stays byte for byte. A second sync does not add another entry.
- The hook command prints the policy file.

### U6. Absolute Codex home

Goal: empty or relative `CODEX_HOME` cannot write next to the repo.

Files: `src/sync.ts`

Approach: Empty is unset and means `~/.codex`. A path that is not absolute, including an unexpanded `~/.codex`, exits before any write.

### U7. Check covers a partial refuse

Goal: `--check` covers R4.

Files: `src/sync.ts`

Approach: Assert a foreign decision alongside write decisions still counts as a publish of the other files. The temp-home proof shows the foreign file stays and the other destinations change.

### U8. Primary checkout only

Goal: refuse to publish from a checkout that is not the primary branch, or from a worktree.

Files: `src/sync.ts`

Approach: The primary branch is `refs/remotes/origin/HEAD`, which is `master` here, not `main`. Exit before any write when `HEAD` is not that branch, or when `git rev-parse --git-dir` and `--git-common-dir` differ. `--check` does not publish, so it still runs.

### U9. Atomic ledger write

Goal: a killed write cannot leave a truncated `~/.gap/published.json`.

Files: `src/sync.ts`

Approach: Write a temp file in the same directory, then rename it over the ledger.

## Verification Contract

- `node src/sync.ts --check` passes.
- Proof uses a temp `HOME` and a temp clone on `master`. Do not run `npm run sync` and do not use the real home.
- A non-TTY foreign file stays in place. The other destinations are still written. The process exits non-zero.
- A copied Codex `AGENTS.md` loses only a ledger-matched prefix. The suffix stays byte for byte. `hooks.json` gains one SessionStart entry and keeps every other hook byte for byte.
- Empty `CODEX_HOME` uses `~/.codex`. A relative `CODEX_HOME` writes nothing.
- A branch other than `origin/HEAD`, or a worktree, writes nothing.
- The ledger after publish is valid JSON, written by rename.

## Known limits

- Cursor load is not proven. No live Cursor session has confirmed it loads `global-agent-policy.mdc`.
- D6 stays open. A backup keeps the bytes, not the harness behavior. Once a foreign file moves, that harness stops loading those rules until a person puts the kept text back by hand.
- The one-time `hooks.json` insert and the one-time `AGENTS.md` cleanup can race another writer at setup. A later policy edit does not rewrite either file. Add a lock if two syncs overlap during that first setup.

## Definition of Done

- R1 through R8 hold in the temp-home proof.
- `--check` covers the ours-versus-foreign decision, including a payload that differs from the source only by the docs-link rewrite, and R4's partial refuse.
- `context7.mdc` is not a destination.
- No Grok path is written.


## Status Review (2026-10-04)

Checked against `src/sync.ts` after PRs #55 and #56. The proof ran in a temp `HOME`, not the real home.

### Met

- U2 and R5. Codex writes `AGENTS.md` under Codex home. No `AGENTS.override.md`.
- U3 and R6. Cursor gets `global-agent-policy.mdc` starting with `alwaysApply: true`, with the shared body. `context7.mdc` is not a destination.
- U4. `README.md`, `AGENTS.md`, and `global/AGENTS.md` list all five destinations and the Grok non-destination.
- R1, R2, R7, R8. One source. A missing or matching file is written quietly. No Grok path is written. A second sync made no backup.
- `node src/sync.ts --check` prints `ok`.

### Not met or not proven

- R4, not met as written. A non-TTY run with a foreign Codex file exits 1 and leaves that file in place. The same run still writes Claude, Pi, OMP, Cursor, and `~/.gap/published.json`. See Q1.
- R3, code departs from the text. `decide()` treats a file whose sha256 equals the last publish recorded in `~/.gap/published.json` as ours and overwrites it with no prompt. R3 calls any bytes that differ from the new payload foreign. See Q2.
- R3 TTY path, not proven. The yes path that moves a foreign file under `~/.gap/backups/` and then writes was not exercised. It needs a real TTY.
- Cursor load, not proven. No live Cursor session has confirmed it loads the `.mdc` file. This is still the Assumption above.

### Defects found

- D1. `CODEX_HOME` that is empty or relative resolves the Codex destination against the current directory. `??` keeps `""`, and nothing makes the path absolute. Empty gives `./AGENTS.md`, which is this repo's `AGENTS.md` when run from the root. Only the foreign-file refusal stopped an overwrite. A relative value such as `codex`, or an unexpanded `~/.codex`, creates a new file under the current directory with no prompt. Fix: treat empty as unset and reject a path that is not absolute.
- D2. `--check` does not cover R4. Nothing asserts what a non-TTY foreign file does to the other destinations.
- D3. Codex has co-writers. The live `~/.codex/AGENTS.md` is 149 lines: the GAP body, then a `soulstack` managed block (lines 47-121), then a `gitbutler-agent-setup` managed block (lines 123-149). Its sha256 `281bcb37…` matches neither the payload nor the ledger entry `8aafbf61…`. Every agent sync refuses Codex and exits 1. A TTY yes strips both blocks until their installers run again. `~/.gap/backups/` already holds five backups from 2026-09-27, two of them Codex copies with the GitButler block. The whole-file ownership model in KTD1 cannot hold for a file other tools also write.
- D4. Sync publishes whichever checkout runs it. `repoRoot` and `docsDir` come from `import.meta.url`, and `adapt()` writes that absolute path into every docs link. `global/AGENTS.md` tells agents to use git worktrees and to run `npm run sync` after edits. A sync from a worktree publishes unmerged policy to all five harnesses, the ledger marks it ours, and every Details link breaks when the worktree is removed.
- D5. The ledger is not crash-safe or concurrency-safe. `writeLedger` is a plain `writeFile` written after the destination, the ledger is read once and rewritten whole after each file, and `readLedger` rethrows a parse error before any write. A killed run or two overlapping agent syncs can leave a GAP-written file that counts as foreign, or a truncated ledger that stops every later sync.
- D6. A backup keeps bytes, not behavior. Once a foreign file moves, the harness stops loading those rules. The backup message is printed once, and pasting the backup into `global/AGENTS.md` would copy tool-specific rules, such as GitButler's, into every harness.

### Red Team (2026-10-04)

Eight adversarial reviewers: edge cases, failure modes, trade-offs, second-order effects, assumption breaking, threat modeling, naive questions, simplicity. D3 to D6 came from this pass. D3 was confirmed on the live files.

- Q1, consensus: refuse only the foreign file and amend R4 to match. With all-or-nothing, one stale Codex file would block policy updates to every harness. D3 makes that the normal state today.
- Q2, consensus: keep the ledger and amend R3 and KTD1 to say a file matching the last recorded publish is ours. Without it, every policy edit turns all five copies foreign.
- Refuted: Grok loading the Cursor `.mdc` a second time. `grok inspect` lists `~/.claude/rules/global-agent-policy.md` and no `~/.cursor/rules` entry.
- Set aside: forging the ledger or faking a TTY. An agent with that much shell access can write the destination files directly, so the ledger is not the boundary that matters.
- Core flaw: KTD1 treats "GAP owns the whole destination file" as a hard fact. That is an assumption, and it is already false for Codex. A fix for D3 needs GAP-owned markers inside the shared file, a Codex path GAP owns alone, or excluding Codex.

### Tracking

No GitHub issue tracks this plan. It shipped through PRs #55 and #56. The open issues cover atlas-flow, Forge phases, and triage-label docs, not this work.

## Sources

- `src/sync.ts` writes five destinations from `global/AGENTS.md`, rewrites docs links, and reads a destination before deciding whether it is ours. Codex policy is `$CODEX_HOME/gap/global-agent-policy.md`.
- Codex SessionStart hook output is plain text on stdout, added as developer context. Matcher values are `startup`, `resume`, `clear`, and `compact`. https://developers.openai.com/codex/hooks
- Cursor user rule files: `~/.cursor/rules`. Plain `.md` in `.cursor/rules` is ignored. https://cursor.com/help/customization/rules and https://cursor.com/docs/rules
- Grok home rules include `~/.claude/rules/` when Claude compatibility is on. `~/.grok/docs/user-guide/12-project-rules.md`
