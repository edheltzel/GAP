# FeedbackAgent — the $0 feedback-intake scaffold

The `feedback-agent` overlay (interview Q7) has two halves: a board footprint (the "Feedback Intake" view over `label:needs:triage`, merged by `templates/overlays.ts`) and a repo-side `.github/` scaffold installed by copying the `FEEDBACK_AGENT_SCAFFOLD` manifest entries (`templates/overlays.ts`) into the target repo. Everything runs at $0 on the free tier: Actions are free and unlimited on public repos; on private repos each job bills against the 2,000 free min/mo (Free) / 3,000 (Pro) **rounded up to a whole minute per job** — these jobs run in seconds, so a busy repo still stays comfortably inside the allowance, but "seconds" ≠ "free" arithmetic on private repos. GitHub Models has a genuine free tier (low-tier models like `openai/gpt-4o-mini`: 150 requests/day; the id was verified against the live catalog `https://models.github.ai/catalog/models` on 2026-07-02).

## Scaffold contents

| Template (under `templates/feedback-agent/`) | Installs to | Purpose |
|---|---|---|
| `ISSUE_TEMPLATE/feedback.yml` | `.github/ISSUE_TEMPLATE/feedback.yml` | Feedback Issue Form — required dropdown + textarea, auto-applies `needs:triage`, best-effort board auto-add via `projects:` |
| `ISSUE_TEMPLATE/bug.yml` | `.github/ISSUE_TEMPLATE/bug.yml` | Bug Issue Form — auto-applies `type:bug` + `needs:triage` |
| `workflows/feedback-router.yml` | `.github/workflows/feedback-router.yml` | On `issues: [opened, labeled]` with `needs:triage`: mints a 1-hour App token, `actions/add-to-project` onto the central board, ack comment authored as Atlas, optional GitHub Models triage |
| `workflows/pm-commands.yml` | `.github/workflows/pm-commands.yml` | `/pm:triage` `/pm:needs-info` `/pm:ready` maintainer slash commands on the free `GITHUB_TOKEN` |

Label strings follow the canon taxonomy ([Labels.md](Labels.md), `docs/agents/triage-labels.md` — which defines the strings only). The transition semantics are defined **here**: the three workflow-state labels are mutually exclusive, and each command swaps the issue to exactly one of them.

| Command | Adds | Removes |
|---|---|---|
| `/pm:triage` | `needs:triage` | `needs:info`, `agent:ready` |
| `/pm:needs-info` | `needs:info` | `needs:triage`, `agent:ready` |
| `/pm:ready` | `agent:ready` | `needs:triage`, `needs:info` |

Commands are matched case-insensitively and require **write or admin** repo permission — checked live via `getCollaboratorPermissionLevel`, not `author_association` (which would let read-only members flip `agent:ready`). The `pm:` namespace avoids colliding with other agents' `/triage` commands.

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
# Optional AI triage (GitHub Models free tier; gpt-4o-mini is low tier: 150 requests/day):
gh variable set PM_AI_TRIAGE --repo "$REPO" --body "true"
```

The App needs `Issues: RW` + org `Projects: RW` (it has them — proven by the §8.7 capstone). The private key's canonical backup is the Bitwarden note "GAT atlas-bot (GitHub App)".

## Install steps

1. Copy each `FEEDBACK_AGENT_SCAFFOLD` entry from `templates/feedback-agent/` to its target path in the repo.
2. Replace the `ORG/PROJECT_NUMBER` placeholders (both Issue Forms' `projects:` key and the router's `project-url`) with the central board, e.g. `Gentle-Art-Technologies/3`.
3. Set the secrets above.
4. Ensure **every label the workflows can apply** pre-exists in the repo (applying a missing label fails the API call): the forms use `needs:triage`/`type:bug`, the slash commands add `needs:info`/`agent:ready`, and the AI triage allowlist can apply `type:feature`/`type:docs`/`type:chore`. Use the Phase-2 label-sync target, or:

```bash
gh label create "needs:triage" --repo "$REPO" --color FBCA04 --description "maintainer needs to evaluate" --force
gh label create "needs:info"   --repo "$REPO" --color c73aa4 --description "waiting on reporter for info" --force
gh label create "agent:ready"  --repo "$REPO" --color bfd4f2 --description "ready for an AFK agent" --force
gh label create "type:bug"     --repo "$REPO" --color d4c5f9 --description "work kind: bug" --force
gh label create "type:feature" --repo "$REPO" --color d4c5f9 --description "work kind: feature" --force
gh label create "type:docs"    --repo "$REPO" --color d4c5f9 --description "work kind: docs" --force
gh label create "type:chore"   --repo "$REPO" --color d4c5f9 --description "work kind: chore" --force
```

5. GitHub Models on private repos may require org-level enablement (org **Settings → Models**). If it isn't enabled, the AI triage step degrades gracefully — nothing else breaks.

## E2E verification runbook ($0 proof)

Proves the Phase 5 gate: *an Issue-Form submission lands on the board with an ack comment, at $0.* Needs a test repo in the org (the App is installed org-wide).

**Runbook prerequisites:**

- The **Atlas-Key PAT's repository grant must include the test repo** — a fine-grained PAT scoped to selected repositories does not see a newly created repo until the grant is updated (or is set to "All repositories"). Otherwise the ack step fails with 403 and the run half-passes.
- Atlas-Key needs read access to the fresh **private** repo (org base role or an invite).
- Cleanup needs the `delete_repo` scope on the default gh auth: `gh auth refresh -s delete_repo`.
- Source `~/.env` into the shell before `gh secret set` — an unset `$ATLAS_KEY_PAT` silently stores an **empty** secret.

**Ordering note:** the scripted probe below creates the issue with labels **atomically**, a *stronger* ordering than a real Issue-Form submission, where the label can arrive as a follow-up `labeled` event after `opened`. The router handles that race (the ack runs on either event, deduped by the `<!-- pm-feedback-ack -->` marker), but the **manual form variant at the end is the true proof of the ack path** — run it at least once.

```zsh
ORG=Gentle-Art-Technologies
REPO=$ORG/pm-feedback-e2e
BOARD=PROJECT_NUMBER   # the central board's number

# 0) Env: make ~/.env vars visible to this shell (empty-var guard)
set -a; . ~/.env; set +a
[ -n "$ATLAS_KEY_PAT" ] || { echo "ATLAS_KEY_PAT is empty"; exit 1; }

# 1) Test repo + scaffold
gh repo create "$REPO" --private --add-readme
#    ...copy the 4 scaffold files in, replace ORG/PROJECT_NUMBER, commit, push.
#    ...set the secrets + labels per the sections above (all 7 labels).

# 2) Submit intake (scripted approximation — labels atomic at create)
gh issue create --repo "$REPO" --title "[Bug]: e2e probe" \
  --body "What happened: e2e probe" --label "type:bug" --label "needs:triage"

# 3) Wait for the router run (poll — a bare `gh run list` right after issue
#    creation races event delivery)
run_id=""
for i in {1..12}; do
  run_id=$(gh run list --repo "$REPO" --workflow feedback-router.yml --limit 1 \
    --json databaseId --jq '.[0].databaseId')
  [ -n "$run_id" ] && break
  sleep 5
done
[ -n "$run_id" ] || { echo "router run never appeared"; exit 1; }
gh run watch --repo "$REPO" "$run_id" --exit-status

# 4) ASSERT: the issue is on the board (jq -e sets exit status; --limit high
#    enough that a busy board doesn't truncate the probe out of view)
gh project item-list "$BOARD" --owner "$ORG" --format json --limit 500 \
  | jq -e '.items[] | select(.content.title == "[Bug]: e2e probe")'

# 5) ASSERT: ack comment exists, authored by Atlas-Key, carrying the dedupe marker
gh issue view 1 --repo "$REPO" --json comments \
  | jq -e '.comments[] | select(.author.login == "Atlas-Key" and (.body | contains("pm-feedback-ack")))'

# 6) Slash command round-trip (as a maintainer; poll instead of a blind sleep)
gh issue comment 1 --repo "$REPO" --body "/pm:needs-info"
for i in {1..12}; do
  gh issue view 1 --repo "$REPO" --json labels --jq '.labels[].name' \
    | grep -qx "needs:info" && break
  sleep 5
done
labels=$(gh issue view 1 --repo "$REPO" --json labels --jq '.labels[].name')
echo "$labels" | grep -qx "needs:info"    || { echo "needs:info never applied"; exit 1; }
echo "$labels" | grep -qx "needs:triage"  && { echo "needs:triage not removed"; exit 1; }

# 7) Cost check — honest version. Private-repo Actions bills each job rounded
#    UP to a whole minute against the 2,000 free min/mo; run conclusions do
#    not prove $0. Check actual metered usage (needs org admin):
gh api "orgs/$ORG/settings/billing/actions" \
  --jq '{total_minutes_used, included_minutes}'
#    Expect total_minutes_used well under included_minutes. (Public repos are
#    free and unlimited — transferring the target repo public moots this.)

# 8) Cleanup (needs delete_repo scope)
gh repo delete "$REPO" --yes
```

Manual variant of step 2 (full-fidelity form check — **the true ack-path proof**): open `https://github.com/$REPO/issues/new/choose`, pick "Bug Report", verify the required fields block empty submission, submit, then continue from step 3. This exercises the real `opened`-possibly-without-label ordering that the marker-dedupe ack was built for.
