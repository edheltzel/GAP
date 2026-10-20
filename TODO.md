# To-do

Mirror of `todo.txt` (source of truth, managed with `tuxedo`). Synced 2026-10-19.

## Open

- [ ] 2026-10-14 Confirm in a live Cursor session that ~/.cursor/rules/global-agent-policy.mdc loads
- [ ] 2026-10-15 U2: revoke the Eight Sleep APP_CLIENT_SECRET from commit 98ee113c (rotate, no history rewrite) - [note](docs/plans/2026-10-15-secrets-scan.md)
- [ ] 2026-10-17 U9: offer the NM changes upstream (needs explicit go) - [note](docs/plans/2026-10-17-nm-gitbutler-safe-plan.md)
- [ ] 2026-10-19 U7: in a new Empryo tab confirm the GAP policy loads once and only do-todo/do-commit/but-axi are pinned, then /soulstack update and recheck rg -c gap:begin ~/.empryo/EMPRYO.md = 1 (PR #91 merged, gap sync clean 2026-10-19) - [note](docs/plans/2026-10-19-gap-empryo-harness-plan.md)

## Done

- [x] 2026-10-12 Commit + merge GAP todo rule, then npm run sync - [pr](https://github.com/edheltzel/agent-config/pull/70)
- [x] 2026-10-16 After feat/gap-commands merges: go install ./cmd/gap, gap sync publishes /state-docs published by PR #90 + gap sync 2026-10-19
- [x] 2026-10-13 Ship sync gitbutler/workspace gate change (src/sync.ts, README, AGENTS.md) via but branch + PR
- [x] 2026-10-14 Merge README tutorial PR - [pr](https://github.com/edheltzel/agent-config/pull/72)
- [x] 2026-10-14 Red-team 8: rollback after --remove-codex-legacy brings back the replace prompt
- [x] 2026-10-14 D6: a foreign-file backup keeps bytes, not the harness behavior it gave notice shipped on fix/gap-redteam-8; decide: splice / gap restore / refuse foreign markers - [pr](https://github.com/edheltzel/GAP/pull/88)
- [x] 2026-10-14 Rename edheltzel/agent-config to GAP; archive/rename old edheltzel/GAP Go scaffold repo so go install path resolves
- [x] 2026-10-15 Decide GitButler vs No Mistakes publish path, add phase-agent exception, publish docs/no-mistakes-autonomous - [note](docs/plans/handoffs/archive/2026-10-15-no-mistakes-autonomous-policy.md) [pr](https://github.com/edheltzel/GAP/pull/86)
- [x] 2026-10-15 Decide whether to abort stale no-mistakes run 01M4CAEW62RRW7GN17D3RZ8F8T - [note](docs/plans/handoffs/archive/2026-10-15-no-mistakes-autonomous-policy.md)
- [x] 2026-10-15 Restore the lost DoSkills --yes fix (fix/no-mistakes-yes discarded 2026-10-15 04:31) or re-apply after the merge - [note](docs/plans/2026-10-15-gap-skills-requirements.md)
- [x] 2026-10-15 Plan gap skills (A+B) with do-plan - [note](docs/plans/2026-10-15-gap-skills-requirements.md)
- [x] 2026-10-15 (C) rejected: gap does not install harnesses (user decision 2026-10-16)
- [x] 2026-10-15 U2 gitleaks scan of agent-config and DoSkills history - [note](docs/plans/2026-10-15-gap-skills-plan.md)
- [x] 2026-10-15 U3-U7 build gap skills (manifest, install, drift, docs) - [note](docs/plans/2026-10-15-gap-skills-plan.md)
- [x] 2026-10-15 U8 publish: flip agent-config public, rename to GAP, archive DoSkills (needs explicit go) - [note](docs/plans/2026-10-15-gap-skills-plan.md)
- [x] 2026-10-15 Fix untracked global/skills.json merge-gate bypass + regression test, amend into suw - [note](docs/plans/handoffs/archive/2026-10-15-gap-skills.md)
- [x] 2026-10-15 Decide U8 public history (fresh repo / filter-repo / full) and give the go - [note](docs/plans/handoffs/archive/2026-10-15-gap-skills.md)
- [x] 2026-10-16 After the checkpoint PRs merge: git pull, go install ./cmd/gap, trash the stale ~/.claude/skills/typesafe-ai, then gap sync from a terminal typesafe-ai trashed and reinstalled, gap sync clean 2026-10-19
- [x] 2026-10-16 After #79 and #87 merge: add global/commands/*.md to ignoredPathBlocks + regression test, so an ignored command can't bypass the merge gate already covered by ignoredPathBlocks (main.go:306) - [pr](https://github.com/edheltzel/GAP/pull/87)
- [x] 2026-10-16 End-to-end test from gitbutler/workspace: git push no-mistakes &lt;branch&gt;, then no-mistakes axi run --run &lt;id&gt; --yes (daemon restarted on patched NM 2026-10-17) superseded by axi run --branch (PR #89)
- [x] 2026-10-17 Confirm the GitButler command that integrates remote fix commits into a stack branch - [note](docs/plans/2026-10-17-nm-gitbutler-safe-requirements.md)
- [x] 2026-10-17 U1-U7: GitButler-safe No Mistakes built (a0f1777e) and proven on GAP PR #89; finish U5 no-upstream guidance fix, reinstall - [note](docs/plans/2026-10-17-nm-gitbutler-safe-plan.md)
- [x] 2026-10-17 U8: update GAP No Mistakes policy and do-commit for axi run --branch in GitButler repos - [note](docs/plans/2026-10-17-nm-gitbutler-safe-plan.md) [pr](https://github.com/edheltzel/GAP/pull/86)
- [x] 2026-10-17 U9: landed GitButler-safe NM on local main of ~/Developer/Contributions/NoMistakes (e9ebcc7e); remotes set no_push; nothing sent upstream - [note](docs/plans/2026-10-17-nm-gitbutler-safe-plan.md)
- [x] 2026-10-19 U1-U3: Empryo policy destination, shadow warning, harness-detection doc - [note](docs/plans/2026-10-19-gap-empryo-harness-plan.md)
- [x] 2026-10-19 U4: A/B do-adversarial-review vs empryo --headless --review, swap only if Empryo wins - [note](docs/plans/2026-10-19-gap-empryo-harness-plan.md) [report](docs/solutions/2026-10-19-empryo-review-ab.md)
- [x] 2026-10-19 U5: one-line Empryo mappings in do-delegation and do-redteam - [note](docs/plans/2026-10-19-gap-empryo-harness-plan.md)
- [x] 2026-10-19 U6: Empryo pin set (do-todo, do-commit, but-axi); no preset, gap sync prints the settings call - [note](docs/plans/2026-10-19-gap-empryo-harness-plan.md)
- [x] 2026-10-19 Decide do-git-worktree: remove, or rewrite for GitButler lanes (contradicts the no-native-worktree rule) rewritten for GitButler lanes (user 2026-10-19) - [note](docs/plans/2026-10-19-gap-empryo-harness-plan.md)
