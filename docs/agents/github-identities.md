# GitHub identities: edheltzel / Atlas-Key / gat-atlas-bot

Three distinct GitHub principals operate on the `Gentle-Art-Technologies` (GAT) org. They are **separate accounts** — none authenticates "as" another. An agent picks one **per action type**.

| Principal | What it is | Auth | Acts as | Use for |
|---|---|---|---|---|
| **edheltzel** | Ed, the human owner | broad classic `ghp_` token in `~/.env` as `GITHUB_TOKEN` (the default `gh` auth) | `edheltzel` | admin / org config / setup / read-verification |
| **Atlas-Key** | a real GitHub **User** account (id `296298943`), active GAT member | fine-grained PAT in `~/.env` as `ATLAS_KEY_PAT` | `Atlas-Key` (a User, not a bot) | teammate-voice actions: issue/PR comments, opening issues, assignment, @-mentions, reviews |
| **gat-atlas-bot** | a GitHub **App** (id `4127929`, GAT installation `142206211`, installed all-repos) | short-lived installation tokens minted from a private key | `gat-atlas-bot[bot]` | privileged automation: **Projects v2 board writes**, CI, cross-repo work at scale |

## Decision rule

- Conversational / attributed / human-shaped → **Atlas-Key PAT**.
- Privileged automation / board mutation / unattended CI → **gat-atlas-bot installation token**.

## How to use each (the mechanism is shell-level, not agent-specific)

Any agent that can run a shell can use both — `gh` resolves auth as `GH_TOKEN` > `GITHUB_TOKEN` > keyring.

**As the teammate (Atlas-Key):** explicitly override per call. The override is the switch; without it `gh` uses the default `GITHUB_TOKEN` (edheltzel).

```bash
GH_TOKEN=$ATLAS_KEY_PAT gh issue comment <num> --repo Gentle-Art-Technologies/<repo> --body "…"
```

Never store the PAT as `GITHUB_TOKEN`/`GH_TOKEN` — it would silently re-auth every `gh` call as Atlas-Key.

**As the bot (gat-atlas-bot):** mint an installation token from the App key.

```
JWT (RS256, ≤10 min, iss = App ID 4127929, signed with $ATLAS_APP_PRIVATE_KEY_PATH)
  → GET /app/installations            (find the GAT installation id)
  → POST .../access_tokens            (mint a ~1h installation token)
  → use as: Authorization: Bearer <token>
```

In GitHub Actions, mint the same token with `actions/create-github-app-token` (App ID + private-key **contents** as secrets — the CI exception to the path convention).

## Env caveat

Editing `~/.env` does **not** reach an already-running agent process (env is inherited at launch). Either relaunch the agent, or source inside the command:

```bash
set -a; . ~/.env; set +a; GH_TOKEN=$ATLAS_KEY_PAT gh …
```

Relevant `~/.env` vars: `ATLAS_KEY_PAT`, `ATLAS_APP_ID` (=`4127929`), `ATLAS_APP_PRIVATE_KEY_PATH` (→ `~/.config/atlas-bot/atlas-bot.private-key.pem`, chmod 600). The App private key's canonical backup is the Bitwarden secure note "GAT atlas-bot (GitHub App)".

## Status

The §8.7 capstone proved both halves (2026-06-24, each adversarially gated): `gat-atlas-bot[bot]` created/wrote/deleted a GAT Projects v2 board (`organization_projects: write` — the scope `GITHUB_TOKEN` lacks), and `Atlas-Key` authored an issue + comment on a GAT repo as a User.

**Not yet built:** event-driven comment replies (an `@claude`-style integration). That is a separate layer — a GitHub Actions workflow (or webhook) that triggers on comment events, runs an agent, and posts a reply via one of these identities — built *on top of* these credentials, which are its foundation.
