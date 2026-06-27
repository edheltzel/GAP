# `@atlas` comment-reply workflow — setup & prerequisites

> Date: 2026-06-25. Pilot repo: `Gentle-Art-Technologies/alianzaSite` (private, default branch `master`).
> Workflow file: `.github/workflows/atlas-reply.yml` (added via PR, not merged).
> Companion research: `2026-06-24-claude-actions-tos-research.md` (ToS verdict + mechanism).

## What it does

When a **human** comments `@atlas` on an issue or PR in alianzaSite, GitHub Actions runs
`anthropics/claude-code-action@v1`, which runs Claude on Ed's subscription OAuth token and
posts **one reply authored by the `Atlas-Key` user** — with no Claude/Anthropic branding.

## How each requirement is met (all verified against the action's real source/docs)

| Requirement | Mechanism | Ground truth |
|---|---|---|
| Reply authored by `Atlas-Key`, not `claude[bot]` | `github_token: ${{ secrets.ATLAS_KEY_PAT }}` | action FAQ: *"if you provide a `github_token` … comments appear under a different username."* `src/entrypoints/run.ts:181-182` sets `GITHUB_TOKEN`/`GH_TOKEN` to that token for the Claude run, so `gh` posts as Atlas-Key. |
| No "Claude finished…" header / no spinner / no branding | Provide `prompt` → **agent mode** | `src/modes/detector.ts:41` (prompt on comment events → agent); `src/modes/agent/index.ts:104` *"No tracking comment in agent mode."* The `**Claude finished @user's task**` header (`src/github/operations/comment-logic.ts:127`) only exists in tag mode. The `"Generated with Claude Code"` signature (`src/create-prompt/index.ts:781`) is emitted only in the PR-creation branch flow, which this reply bot never enters. Prompt also explicitly forbids any footer/branding. |
| Subscription OAuth (not API key) | `claude_code_oauth_token: ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}` | `action.yml:70` input; `docs/setup.md:10` (`claude setup-token`). |
| No infinite-loop risk | `if:` actor filter `login != 'Atlas-Key'` + `type != 'Bot'` + must contain `@atlas`; `concurrency` per issue/PR; `--max-turns 15`; reply forbidden from containing `@atlas` | A PAT is **not** recursion-suppressed like `GITHUB_TOKEN`, so the actor filter is load-bearing. Agent mode also runs `checkHumanActor` (`src/modes/agent/index.ts:29`) as a second line. |
| Trigger phrase `@atlas` | `if: contains(..., '@atlas')` is the real gate (agent mode does not re-check `trigger_phrase`) | `src/modes/agent/index.ts` bypasses mention checking. |
| Least privilege | `permissions: contents: read, issues: write, pull-requests: write`; no `id-token: write`, no `actions: read`, no `contents: write` | `id-token` is only for OIDC/Bedrock/Vertex/federation, not OAuth-token auth. |

## Does the official Claude GitHub App need to be installed? — NO

Installing `https://github.com/apps/claude` is the action's *default* GitHub-auth path. It is **not
required here** because we supply our own `github_token` (the Atlas-Key PAT): `action.yml:88`
documents `github_token` as *"optional if using GitHub App"*, and the prepare step uses
`inputs.github_token || github.token` — a provided token replaces the App entirely for GitHub access.
The `CLAUDE_CODE_OAUTH_TOKEN` authenticates Claude to Anthropic and is **not** a GitHub credential, so
it imposes no App requirement either. The existing `gat-atlas-bot` App is also not used by this workflow.

## Human prerequisites

1. **`ATLAS_KEY_PAT` secret — DONE.** Set on alianzaSite by this task (`gh secret set`, value never printed).
   The PAT authenticates as `Atlas-Key` and expires **2027-06-24**.
   - ⚠️ **VERIFY PERMISSIONS (outstanding):** the PAT currently shows only collaborator `pull`
     (read) on alianzaSite and `issues=read` was the only confirmable grant. To post comments on a
     **private** repo it needs **Issues: Read and write** and **Pull requests: Read and write** for
     `Gentle-Art-Technologies/alianzaSite`. If the first real `@atlas` reply 403s, elevate the
     fine-grained PAT's repository permissions (Issues R/W, Pull requests R/W) and re-set the secret.
2. **`CLAUDE_CODE_OAUTH_TOKEN` secret — already present** on alianzaSite (set 2026-06-25 20:45 UTC).
   Confirm it is Ed's current token from `claude setup-token` on his Pro/Max account (regenerate +
   re-set if it has been rotated/expired). This draws Ed's subscription quota.
3. **Merge the PR.** `issue_comment` / `pull_request_review_comment` workflows only run from the copy
   on the **default branch (`master`)**. The workflow therefore **cannot be triggered until merged** —
   no pre-merge live test is possible (a GitHub platform behavior, not a config gap). After merge,
   smoke-test by commenting `@atlas hello` on an issue and confirming an `Atlas-Key`-authored reply.

## ToS posture (from the companion research)

Subscription-OAuth on the **official** action is gray-leaning-allowed for low-volume, owner-controlled
repos; the April 2026 crackdown targeted *third-party* harnesses, which this is not. If volume or
account-risk becomes a concern, switch to `anthropic_api_key` (pay-per-use) — a one-line swap.
