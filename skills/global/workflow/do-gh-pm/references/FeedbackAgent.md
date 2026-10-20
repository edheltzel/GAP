# FeedbackAgent — the $0 feedback-intake scaffold

The `feedback-agent` overlay (interview Q7) has two halves: a board footprint (the "Feedback Intake" view over `label:needs:triage`, merged by `templates/overlays.ts`) and a repo-side `.github/` scaffold installed by copying the `FEEDBACK_AGENT_SCAFFOLD` manifest entries (`templates/overlays.ts`) into the target repo. Everything runs at $0 on the free tier: Actions are free and unlimited on public repos; on private repos each job bills against the 2,000 free min/mo (Free) / 3,000 (Pro) **rounded up to a whole minute per job** — these jobs run in seconds, so a busy repo still stays comfortably inside the allowance, but "seconds" ≠ "free" arithmetic on private repos. GitHub Models has a genuine free tier (low-tier models like `openai/gpt-4o-mini`: 150 requests/day; the id was verified against the live catalog `https://models.github.ai/catalog/models` on 2026-07-02).

## Scaffold contents

| Template (under `templates/feedback-agent/`) | Installs to | Purpose |
|---|---|---|
| `ISSUE_TEMPLATE/feedback.yml` | `.github/ISSUE_TEMPLATE/feedback.yml` | Feedback Issue Form — required dropdown + textarea, auto-applies `needs:triage`, best-effort board auto-add via `projects:` |
| `ISSUE_TEMPLATE/bug.yml` | `.github/ISSUE_TEMPLATE/bug.yml` | Bug Issue Form — auto-applies `type:bug` + `needs:triage` |
| `workflows/feedback-router.yml` | `.github/workflows/feedback-router.yml` | On `issues: [opened, labeled]` with `needs:triage`: mints a 1-hour App token, `actions/add-to-project` onto the central board, ack comment authored as Atlas, optional GitHub Models triage |
| `workflows/pm-commands.yml` | `.github/workflows/pm-commands.yml` | `/pm:triage` `/pm:needs-info` `/pm:ready` maintainer slash commands on the free `GITHUB_TOKEN` |

Label strings follow the canon taxonomy in [Labels.md](Labels.md). The transition semantics are defined **here**: the three workflow-state labels are mutually exclusive, and each command swaps the issue to exactly one of them.

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

Use a GitHub App — **App id `<APP_ID>`**, installation `<INSTALLATION_ID>`. Per target repo (or once as org secrets):

```bash
REPO=ORG/REPO   # the repo receiving the scaffold
gh secret set ATLAS_APP_ID          --repo "$REPO" --body "<APP_ID>"
gh secret set ATLAS_APP_PRIVATE_KEY --repo "$REPO" < <KEY_PATH>
gh secret set ATLAS_KEY_PAT         --repo "$REPO" --body "$ATLAS_KEY_PAT"
# Optional AI triage (GitHub Models free tier; gpt-4o-mini is low tier: 150 requests/day):
gh variable set PM_AI_TRIAGE --repo "$REPO" --body "true"
```

The App needs `Issues: RW` + org `Projects: RW`. The private key's canonical backup is the secret-store note `<SECRET_STORE_NOTE>`.

## Install steps

1. Copy each `FEEDBACK_AGENT_SCAFFOLD` entry from `templates/feedback-agent/` to its target path in the repo.
2. Replace the `ORG/PROJECT_NUMBER` placeholders (both Issue Forms' `projects:` key and the router's `project-url`) with the central board, e.g. `OWNER/PROJECT_NUMBER`.
3. Set the secrets above.
4. Ensure **every label the workflows can apply** pre-exists in the repo (applying a missing label fails the API call): the forms use `needs:triage`/`type:bug`, the slash commands add `needs:info`/`agent:ready`, and the AI triage allowlist can apply `type:feature`/`type:docs`/`type:chore`. Use the reconcile engine's label sync (`bun Tools/Reconcile.ts --archetype <key> --repo "$REPO"`), or:

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

Proves: *an Issue-Form submission lands on the board with an ack comment, at $0.* Needs a test repo the App can write to.

**Prerequisites:**

- The `ATLAS_KEY_PAT` grant includes the test repo. A fine-grained PAT scoped to selected repositories does not see a newly created repo until the grant is updated (or is set to "All repositories"); otherwise the ack step fails with 403.
- The ack identity needs read access to a fresh private repo.
- Cleanup needs `delete_repo` on the default gh auth: `gh auth refresh -s delete_repo`.
- Confirm `$ATLAS_KEY_PAT` is non-empty before `gh secret set` — an unset value silently stores an empty secret.

**Ordering note:** a scripted probe that creates the issue with labels atomically is stronger than a real Issue-Form submission, where the label can arrive as a follow-up `labeled` event after `opened`. The router handles that race (the ack runs on either event, deduped by the `<!-- pm-feedback-ack -->` marker). Run the manual form variant at least once for the true ack path.

```zsh
ORG=OWNER
REPO=$ORG/pm-feedback-e2e
BOARD=PROJECT_NUMBER   # the central board's number
ACK_LOGIN=<ACK_LOGIN>  # GitHub login of the ATLAS_KEY_PAT identity

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

# 4) ASSERT: the issue is on the board
gh project item-list "$BOARD" --owner "$ORG" --format json --limit 500 \
  | jq -e '.items[] | select(.content.title == "[Bug]: e2e probe")'

# 5) ASSERT: ack comment exists, authored by the PAT identity, carrying the dedupe marker
gh issue view 1 --repo "$REPO" --json comments \
  | jq -e --arg login "$ACK_LOGIN" '.comments[] | select(.author.login == $login and (.body | contains("pm-feedback-ack")))'

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

# 7) Cost check. Private-repo Actions bills each job rounded UP to a whole
#    minute against the 2,000 free min/mo; run conclusions do not prove $0.
gh api "orgs/$ORG/settings/billing/actions" \
  --jq '{total_minutes_used, included_minutes}'

# 8) Cleanup (needs delete_repo scope)
gh repo delete "$REPO" --yes
```

Manual variant of step 2 (full-fidelity form check — **the true ack-path proof**): open `https://github.com/$REPO/issues/new/choose`, pick "Bug Report", verify the required fields block empty submission, submit, then continue from step 3. This exercises the real `opened`-possibly-without-label ordering that the marker-dedupe ack was built for.
