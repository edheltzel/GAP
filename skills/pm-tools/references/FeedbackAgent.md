# FeedbackAgent — the $0 feedback-intake scaffold

The `feedback-agent` overlay (interview Q7) has two halves: a board footprint (the "Feedback Intake" view over `label:needs:triage`, merged by `templates/overlays.ts`) and a repo-side `.github/` scaffold installed by copying the `FEEDBACK_AGENT_SCAFFOLD` manifest entries (`templates/overlays.ts`) into the target repo. Everything runs at $0: Actions are free on public repos (2,000–3,000 min/mo on private — bot runs are seconds), and GitHub Models has a genuine free tier.

## Scaffold contents

| Template (under `templates/feedback-agent/`) | Installs to | Purpose |
|---|---|---|
| `ISSUE_TEMPLATE/feedback.yml` | `.github/ISSUE_TEMPLATE/feedback.yml` | Feedback Issue Form — required dropdown + textarea, auto-applies `needs:triage`, best-effort board auto-add via `projects:` |
| `ISSUE_TEMPLATE/bug.yml` | `.github/ISSUE_TEMPLATE/bug.yml` | Bug Issue Form — auto-applies `type:bug` + `needs:triage` |
| `workflows/feedback-router.yml` | `.github/workflows/feedback-router.yml` | On `issues: [opened, labeled]` with `needs:triage`: mints a 1-hour App token, `actions/add-to-project` onto the central board, ack comment authored as Atlas, optional GitHub Models triage |
| `workflows/pm-commands.yml` | `.github/workflows/pm-commands.yml` | `/pm:triage` `/pm:needs-info` `/pm:ready` maintainer slash commands on the free `GITHUB_TOKEN` |

Label transitions follow the canon taxonomy ([Labels.md](Labels.md), `docs/agents/triage-labels.md`): `/pm:triage` → `needs:triage`, `/pm:needs-info` → `needs:info`, `/pm:ready` → `agent:ready`. The `pm:` namespace avoids colliding with other agents' `/triage` commands.

## Why three tokens

- **`GITHUB_TOKEN`** (free, automatic) — same-repo labels, comments, reactions, and GitHub Models (`models: read`). It **cannot write Projects v2**; that limitation is the whole reason the App exists.
- **GitHub App installation token** — minted per run via `actions/create-github-app-token`, 1-hour lifetime, does the board write.
- **Atlas-Key PAT** — the ack comment's author, so reporters see a reply from Atlas the teammate, not `github-actions[bot]`.

## GitHub App setup (once per org)

Reuse the existing **gat-atlas-bot** App — **App id `4127929`**, GAT installation `142206211`, installed org-wide (see `docs/agents/github-identities.md`). Do **not** register a new App. Per target repo (or once as org secrets):

```bash
REPO=ORG/REPO   # the repo receiving the scaffold
gh secret set ATLAS_APP_ID          --repo "$REPO" --body "4127929"
gh secret set ATLAS_APP_PRIVATE_KEY --repo "$REPO" < ~/.config/atlas-bot/atlas-bot.private-key.pem
gh secret set ATLAS_KEY_PAT         --repo "$REPO" --body "$ATLAS_KEY_PAT"
# Optional AI triage (GitHub Models free tier, ~50 requests/day):
gh variable set PM_AI_TRIAGE --repo "$REPO" --body "true"
```

The App needs `Issues: RW` + org `Projects: RW` (it has them — proven by the §8.7 capstone). The private key's canonical backup is the Bitwarden note "GAT atlas-bot (GitHub App)".

## Install steps

1. Copy each `FEEDBACK_AGENT_SCAFFOLD` entry from `templates/feedback-agent/` to its target path in the repo.
2. Replace the `ORG/PROJECT_NUMBER` placeholders (both Issue Forms' `projects:` key and the router's `project-url`) with the central board, e.g. `Gentle-Art-Technologies/3`.
3. Set the secrets above.
4. Ensure the canon labels exist in the repo (`needs:triage`, `needs:info`, `agent:ready`, `type:bug` at minimum) — the Phase-2 label-sync target, or:

```bash
gh label create "needs:triage" --repo "$REPO" --color FBCA04 --description "maintainer needs to evaluate" --force
gh label create "needs:info"   --repo "$REPO" --color c73aa4 --description "waiting on reporter for info" --force
gh label create "agent:ready"  --repo "$REPO" --color bfd4f2 --description "ready for an AFK agent" --force
gh label create "type:bug"     --repo "$REPO" --color d4c5f9 --description "work kind: bug" --force
```

## E2E verification runbook ($0 proof)

Proves the Phase 5 gate: *an Issue-Form submission lands on the board with an ack comment, at $0.* Needs a test repo in the org (the App is installed org-wide).

```bash
ORG=Gentle-Art-Technologies
REPO=$ORG/pm-feedback-e2e
BOARD=PROJECT_NUMBER   # the central board's number

# 1) Test repo + scaffold
gh repo create "$REPO" --private --add-readme
#    ...copy the 4 scaffold files in, replace ORG/PROJECT_NUMBER, commit, push.
#    ...set the secrets + labels per the sections above.

# 2) Submit intake. Issue Forms only render in the web UI; for a scripted run,
#    approximate the form's output (same labels the form auto-applies):
gh issue create --repo "$REPO" --title "[Bug]: e2e probe" \
  --body "What happened: e2e probe" --label "type:bug" --label "needs:triage"

# 3) Watch the router run complete
gh run list --repo "$REPO" --workflow feedback-router.yml
gh run watch --repo "$REPO" "$(gh run list --repo "$REPO" --workflow feedback-router.yml --limit 1 --json databaseId --jq '.[0].databaseId')"

# 4) Assert: the issue is on the board
gh project item-list "$BOARD" --owner "$ORG" --format json \
  | jq '.items[] | select(.content.title == "[Bug]: e2e probe")'

# 5) Assert: ack comment exists and is authored by Atlas-Key
gh issue view 1 --repo "$REPO" --json comments \
  --jq '.comments[] | {author: .author.login, body: .body}'

# 6) Slash command round-trip (as a maintainer)
gh issue comment 1 --repo "$REPO" --body "/pm:needs-info"
sleep 30
gh issue view 1 --repo "$REPO" --json labels --jq '.labels[].name'
# expect: needs:info present, needs:triage gone

# 7) $0 assertion: runs consumed seconds of the free tier
gh api "repos/$REPO/actions/runs" --jq '.workflow_runs[] | {name, run_started_at, conclusion}'

# 8) Cleanup
gh repo delete "$REPO" --yes
```

Manual variant of step 2 (full-fidelity form check): open `https://github.com/$REPO/issues/new/choose`, pick "Bug Report", verify the required fields block empty submission, submit, then continue from step 3.
