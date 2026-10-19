# Regroup: GAP

Date: 2026-10-18
Source: `todo.txt` (no `done.txt` exists), `+gap` lines in `$HOME/todo.txt`, previous check-in `regroup/2026-10-18-regroup-gap.md`, `gh pr view` for #76, #79, #87. Resume point: `docs/plans/handoffs/2026-10-15-no-mistakes-autonomous-policy.md`.

No task line changed since the earlier check-in today.

## Last completed

- 2026-10-16 D6: a foreign-file backup keeps bytes, not the harness behavior it gave (PR #88).
- 2026-10-16 Rename edheltzel/agent-config to GAP; archive the old scaffold repo.
- 2026-10-16 U8 publish and the public-history decision.
- 2026-10-16 (C) rejected: gap does not install harnesses.

## Up next

- (C) ATL-613 Codex-in-Herdr review then Red-Team (global list). Its PR #76 merged 2026-10-15; the task is still open, so it likely only needs closing.
- U1-U7 GitButler-safe No Mistakes (plan step): blocked, `@waiting` on the codex agent working `feat/axi-run-selector`; U7 install + GAP proof follow it.
- U8 GAP policy and do-commit for `axi run --branch`: blocked on U7.

## Remaining

- U9 offer the NM changes upstream: blocked on U7 and your explicit go.
- Decide GitButler vs No Mistakes publish path, publish docs/no-mistakes-autonomous: blocked on U8 (2026-10-17 requirements chose `axi run --branch`).
- Confirm the GitButler command that brings remote fix commits into a stack branch.
- End-to-end `git push no-mistakes <branch>` test from gitbutler/workspace (likely superseded by U7).
- After #79 and #87 merge: `global/commands/*.md` in ignoredPathBlocks + regression test: blocked, both PRs still open.
- After feat/gap-commands merges: `go install ./cmd/gap`, `gap sync` publishes /state-docs: blocked, #87 still open.
- After the checkpoint PRs merge: pull, reinstall, trash stale `typesafe-ai`, `gap sync`: blocked on merges.
- U2 sign-off: remove or rotate the real-looking secrets-scan findings.
- U2: revoke the Eight Sleep `APP_CLIENT_SECRET` from commit 98ee113c.
- Confirm `~/.cursor/rules/global-agent-policy.mdc` loads in a live Cursor session.

## Raw lines

```
(C) 2026-09-28 ATL-613 gap: Codex-in-Herdr review then Red-Team +gap @intern @todo id:ATL-613 pr:https://github.com/edheltzel/agent-config/pull/76
2026-10-14 Confirm in a live Cursor session that ~/.cursor/rules/global-agent-policy.mdc loads +gap
2026-10-15 Decide GitButler vs No Mistakes publish path, add phase-agent exception, publish docs/no-mistakes-autonomous +gap note:docs/plans/handoffs/2026-10-15-no-mistakes-autonomous-policy.md
2026-10-15 U2 sign-off: remove or rotate the real-looking secrets-scan findings before U8 +gap note:docs/plans/2026-10-15-secrets-scan.md
2026-10-16 After #79 and #87 merge: add global/commands/*.md to ignoredPathBlocks + regression test, so an ignored command can't bypass the merge gate +gap pr:https://github.com/edheltzel/GAP/pull/87
2026-10-16 After feat/gap-commands merges: go install ./cmd/gap, gap sync publishes /state-docs +gap
2026-10-16 After the checkpoint PRs merge: git pull, go install ./cmd/gap, trash the stale ~/.claude/skills/typesafe-ai, then gap sync from a terminal +gap
2026-10-16 End-to-end test from gitbutler/workspace: git push no-mistakes <branch>, then no-mistakes axi run --run <id> --yes (daemon restarted on patched NM 2026-10-17) +gap
2026-10-16 U2: revoke the Eight Sleep APP_CLIENT_SECRET from commit 98ee113c (rotate, no history rewrite) +gap note:docs/plans/2026-10-15-secrets-scan.md
2026-10-17 Confirm the GitButler command that integrates remote fix commits into a stack branch +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-17-nm-gitbutler-safe-requirements.md
2026-10-17 U1-U7: build GitButler-safe No Mistakes in ~/Developer/Contributions/NoMistakes and prove it on GAP +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-17-nm-gitbutler-safe-plan.md @waiting handed off to the codex agent working feat/axi-run-selector; U7 install + GAP proof after it finishes
2026-10-17 U8: update GAP No Mistakes policy and do-commit for axi run --branch in GitButler repos +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-17-nm-gitbutler-safe-plan.md
2026-10-17 U9: offer the NM changes upstream (needs explicit go) +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-17-nm-gitbutler-safe-plan.md
```
