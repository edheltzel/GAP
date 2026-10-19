# Handoff: autonomous No Mistakes policy line

Date: 2026-10-15

## State

- `global/AGENTS.md:39`: new rule. Projects that use or require No Mistakes publish with `no-mistakes axi run --yes --intent "..."`, and the policy is standing consent for `--yes`. Committed as `lsu` on local GitButler branch `docs/no-mistakes-autonomous`. **Not pushed, no PR.**
- `no-mistakes init` ran in this repo (user approved). Gate: `~/.no-mistakes/repos/3daa1fd09cda.git` (machine-local).
- Blocker: `no-mistakes axi` sees `current_branch: gitbutler/workspace`, so `axi run` cannot target the feature branch.
- Machine-local: a stale run `01M4CAEW62RRW7GN17D3RZ8F8T` on `gitbutler/workspace`, parked for 7 days with 3 findings awaiting a response. Untouched.
- Merged on master: #74 Go `gap` CLI, #72 cleanup, #75 Global Agent Policies name, #76 merge-gate bypass fixes. `gap sync` has not been run against the real home since #74, so harness copies still say `npm run sync`.

## Decisions

- User: autonomous `--yes` applies globally to every project that uses or requires No Mistakes.
- User: run `no-mistakes init` here.
- Mine: the policy line is the only global switch. `~/.no-mistakes/config.yaml` has no auto-`--yes` setting.
- Mine: stop rather than work around the `gitbutler/workspace` view (do-commit procedure, No Mistakes section).

## Open questions

- GitButler vs No Mistakes: (1) allow `git push no-mistakes <branch>` as the one exception (my recommendation), (2) GitButler repos skip the gate, or (3) hold pushes in GitButler repos.
- Abort the stale run `01M4CAEW62RRW7GN17D3RZ8F8T`, or not?
- Required edit before pushing (reviewer advisory, valid): line 39 must exempt No Mistakes **phase agents**. The installed skill's "Active validation-step boundary" forbids them from starting, reattaching, or responding to a pipeline (otherwise `nested_gate_context`). Scope the rule to the outer publishing agent.
- ATL-613 asked for a Codex-in-Herdr review. Two generic reviewer agents ran instead (#76), so the task stays open.

## Next step

Get the user's GitButler/No Mistakes choice. Then amend `lsu` with that choice and the phase-agent exception, and publish this branch through the gate.

## Links

- `global/AGENTS.md:38-39`: the GitButler rule and the new No Mistakes rule.
- `~/.agents/skills/no-mistakes/SKILL.md:322-345`: `--yes` consent and refusal rules (machine-local).
- `~/.agents/skills/do-commit/references/procedure.md:13-23`: when No Mistakes applies, and the stop on `gitbutler/workspace` (machine-local).
- `regroup/2026-10-15-regroup-gap.md`: status at handoff.
- `docs/plans/2026-10-14-002-fix-gap-merge-gate-bypass-plan.md`: ATL-613 fix plan.

## todo.txt

- Decide the GitButler vs No Mistakes publish path, then add the phase-agent exception and publish docs/no-mistakes-autonomous
- Decide whether to abort the stale no-mistakes run 01M4CAEW62RRW7GN17D3RZ8F8T
