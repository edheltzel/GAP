# ProjectSetup — interview-driven board scaffold

Bootstrap a GitHub Projects v2 board from eight explicit answers. The agent conducts the interview with `AskUserQuestion`; `Tools/Scaffold.ts` deterministically maps those answers to an archetype, overlays, visibility, and Priority scheme, then creates or copies a board, links the repository, runs the reconcile engine in plan and safe-apply modes, and prints the API-only residual checklist.

The command is dry-run by default. Never add `--apply` until the user has reviewed the displayed plan.

## 0. Resolve the target

Derive defaults from the current checkout instead of asking for information GitHub can provide:

```bash
gh auth status
gh --version                         # must be >= 2.94
gh repo view --json nameWithOwner
git rev-parse --show-toplevel
```

Resolve:

- `OWNER`: project owner; defaults to the repository owner.
- `OWNER/REPO`: target repository.
- `TITLE`: proposed project title; default to the repository name.
- `REPO_PATH`: local checkout root; required only when feedback intake is enabled.
- Optional golden project number and owner. Without one, the command creates an empty board and reports the views/workflows that must be configured manually.

The authenticated account needs the `project` scope. Run `gh auth refresh -s project` only when `gh auth status` shows it is missing.

## 1. Conduct the eight-question interview

Use `AskUserQuestion`. Ask related questions together where the harness supports it, but record one explicit answer for every row. Do not infer answers from unrelated repository details.

| # | Question | Options | CLI value |
|---|---|---|---|
| 1 | What kind of work will this board track? | Software; Writing/content; Research; Personal/Life OS; Open source | `--work-kind software|writing|research|personal|open-source` |
| 2 | Should the project be public or private? | Public; Private | `--visibility public|private` |
| 3 | Who contributes? | Solo; External contributors | `--contributors solo|external` |
| 4 | How is work scheduled? | Continuous flow; Time-boxed | `--cadence continuous|time-boxed` |
| 5 | Track effort estimation? | Yes; No | `--estimation yes|no` |
| 6 | Track a dated roadmap? | Yes; No | `--roadmap yes|no` |
| 7 | Install external feedback intake? | Yes; No | `--feedback yes|no` |
| 8 | Which Priority vocabulary? | P0–P3; High/Medium/Low; None | `--priority p0-p3|high-medium-low|none` |

Deterministic mapping:

| Work kind | Archetype |
|---|---|
| `software` | `solo-software` |
| `writing` | `content-pipeline` |
| `research` | `research-tracker` |
| `personal` | `life-os-tasks` |
| `open-source` | `open-source-repo` |

Overlay mapping, in stable order:

1. External contributors → `triage`.
2. Time-boxed work → `iterations`.
3. Estimation → `estimation`.
4. Dated roadmap → `roadmap`.
5. Feedback intake → `feedback-agent`.

Restate the resolved configuration before running the command.

## 2. Review the scaffold plan

Run from the target repository checkout, not the installed skill directory. Invoke the tools by absolute path and pass that checkout as `--repo-path`. Reconcile writes `.agents/atlas/artifacts/` under the git root of its working directory; running it inside the installed skill dirties that copy.

```bash
bun "$SKILL/Tools/Scaffold.ts" \
  --owner OWNER \
  --repo OWNER/REPO \
  --title "TITLE" \
  --work-kind software \
  --visibility private \
  --contributors solo \
  --cadence continuous \
  --estimation no \
  --roadmap no \
  --feedback no \
  --priority p0-p3 \
  --repo-path "$PWD"
```

`$SKILL` is the installed skill directory (`~/.agents/skills/do-gh-pm`, or this repo's `skills/global/workflow/do-gh-pm`). When Q7 is yes, `--repo-path` is already required. When a golden project exists, add both `--source-project NUMBER` and `--source-owner OWNER`.

Plan mode performs no shell command and no file write. Review:

- Resolved archetype and overlays.
- Create-vs-copy operation.
- Project visibility and repository link.
- Reconcile plan and safe-apply commands.
- Feedback scaffold destination files.
- Manual views and automations that remain after API work.

Ask for confirmation before continuing. A changed answer means rerun plan mode; do not patch the displayed command ad hoc.

## 3. Apply the reviewed plan

Rerun the identical command with `--apply`.

The command performs this order:

1. Verify `gh >= 2.94`.
2. Preflight feedback-template conflicts before any GitHub mutation.
3. Copy the supplied golden project or create an empty project.
4. Set visibility.
5. Link the target repository.
6. Run `Tools/Reconcile.ts` in plan mode.
7. Run it again with `--apply`, which executes safe field, option, and label changes only.
8. Install the four feedback-agent files when selected, refusing to overwrite differing files.
9. Print residual manual steps.

The scaffold command never runs `--prune` and never deletes live options, fields, or labels. If work fails after project creation, the error names the created project number; inspect and resume deliberately rather than silently creating another board.

## 4. Complete API-only residuals

Views and workflows cannot be created through the Projects v2 API. Follow the exact `manualSteps` returned by the command:

- Create every declared view when no golden project supplied it.
- Re-enable auto-add after a project copy; GitHub does not copy that workflow.
- For an empty project, enable auto-add, closed → Done, and Done-after-14-days auto-archive.
- After a golden copy, verify that its live Priority options match Q8. Safe apply preserves extra options; removing a source board's old scheme requires a separately reviewed `--prune --confirm` reconcile run.
- Create an Iteration field in the web UI when the interview enabled time-boxed work.
- Configure feedback placeholders and `ATLAS_APP_ID`, `ATLAS_APP_PRIVATE_KEY`, and `ATLAS_KEY_PAT` when feedback intake is enabled. See [FeedbackAgent.md](FeedbackAgent.md).

## 5. Verify

Use the returned project number:

```bash
gh project view PROJECT_NUM --owner OWNER --format json
gh project field-list PROJECT_NUM --owner OWNER --format json
gh project list --owner OWNER --format json
gh label list --repo OWNER/REPO --limit 200
```

Then rerun the exact reconcile command from the scaffold output without `--apply`, from the same checkout. A converged board reports no safe changes. Extra live options are preserved by design. A fresh board's default `Todo` can be renamed assignment-safely only when `Backlog` does not already exist — a rename onto an existing option is rejected:

```bash
bun "$SKILL/Tools/Reconcile.ts" --project PROJECT_NUM --owner OWNER \
  --archetype ARCHETYPE --priority PRIORITY \
  --rename "Status:Todo=Backlog" --apply
```

Do not claim views or workflows are complete until they are checked in the web UI or inherited from a verified golden project.
