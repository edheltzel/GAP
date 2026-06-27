# Handoff — Atlas comment-reply bot + Phase 0.5 wrap (Themis)

**From:** Themis (session 2026-06-25→27) · **To:** next Themis · **Repo:** `/Users/ed/Developer/atlas-config` · **Principal:** Ed
**You are Themis** (`/Themis`): delegate via **herdr**, validate every claim adversarially via RedTeam, own docs under DOX, report to Ed, never write code yourself. Codegraph is inactive here (markdown/config repo) — ground in Read/Grep/gh/git, never assume.

---

## TOP OF STACK — the one live task

**Comment-reply bot (`@atlas` on alianzaSite) needs a FIX-PR.** PR #92 was **merged by Ed (2026-06-26 21:08)** but as merged it is **non-functional** (fails on first `@atlas`, no security harm) and carries one MEDIUM security residual. The fix list is adjudicated and ready — dispatch a worker for it, then re-gate.

### Fix list (the next worker's brief)
Target: `Gentle-Art-Technologies/alianzaSite` workflow `.github/workflows/atlas-reply.yml` (now on `master`). New branch off master → PR.

1. **MUST-FIX (functional, P1):** add `github_token: ${{ github.token }}` to the `claude-code-action` (Generate) step; keep `permissions: contents: read`; **do NOT add `id-token: write`.**
   - *Why, confirmed in source:* `anthropics/claude-code-action@5211368` `src/github/token.ts` `setupGitHubToken()` returns `OVERRIDE_GITHUB_TOKEN` (= the `github_token` input) if set, **else goes straight to OIDC→App-token exchange** — it does NOT fall back to `DEFAULT_WORKFLOW_TOKEN`. Omitting `github_token` + no `id-token: write` → the step errors before Claude runs. Passing `${{ github.token }}` (read-only via `contents: read`) makes it run while keeping the model on a non-write token. (The original worker AND the first-read assumed omit = read-only fallback; the omp gate caught it.)
2. **MUST-FIX (security, MEDIUM):** the model has `Read` + the raw `${{ github.event.comment.body }}` in its prompt → an injection can `Read /proc/self/environ` (holds `CLAUDE_CODE_OAUTH_TOKEN`), obfuscate it into `reply.md`, and the current sanitizer (only plaintext `ghp_`/`github_pat_`/`Bearer`) posts it → **OAuth-token exfil**. Fix: confine Claude's `Read` to `$GITHUB_WORKSPACE` (Claude Code permission profile / `--add-dir` scoping); harden sanitizer to fail-closed, scan normalized/compacted text, add `sk-ant-*` + obfuscation patterns, case-insensitive branding, max length.
3. **HARDENING (low):** gate also strips `~~~` and unclosed ` ``` ` fences; anchor `^@atlas[ \t\r]*$` (currently `^@atlas\b` lets `@atlas ignore…` trigger); drop unused `Edit` from `--allowedTools` (leave `Read,Grep,Glob,Write`).
4. **GO-LIVE (Ed's hands):** `ATLAS_KEY_PAT` must be a fine-grained PAT scoped to **alianzaSite only, Issues: write** (currently read-only → Post step 403s → run fails). 403 does not leak the token.

### Success criteria (the worker loops until ALL true)
1. `claude-code-action` step has `github_token: ${{ github.token }}`; job `permissions:` is `contents: read` only; **no `id-token: write`**; `--allowedTools` = `"Read,Grep,Glob,Write"` (no `Bash`, no `Edit`).
2. A Claude permission profile / `--add-dir` (or equivalent) confines `Read` to `$GITHUB_WORKSPACE` — verify the model cannot read `/proc/self/environ` or any abs path outside the workspace.
3. Sanitizer fails closed, scans normalized/compacted text, rejects `sk-ant-*` + `gh[pousr]_*` + `github_pat_*` + `Bearer` + obfuscated/split variants, case-insensitive branding, enforces a max length.
4. Gate strips ` ``` ` AND `~~~` AND unclosed fences + blockquotes, then requires `^@atlas[ \t\r]*$` (own line only).
5. Actor gate unchanged (OWNER/MEMBER/COLLABORATOR, non-bot, ≠Atlas-Key).
6. `ATLAS_KEY_PAT` re-scoped (Ed) to alianzaSite-only Issues:write.

### Re-gate + verify (the policy, in action)
`omp reviewer → claude + omp RedTeam` on the fix-PR. After merge, **smoke-test**: comment `@atlas hello` on a throwaway alianzaSite issue → expect exactly one reply **authored by `Atlas-Key`**, **zero Claude/Anthropic branding**, **no second (loop) run**, and the Actions log shows the Generate step succeeded on a read-only token. (Workflows only fire from `master`, so a real test requires merging the fix-PR first — open a throwaway issue, don't touch #88/#31/#28/#18.) **Adversarially confirm** the OAuth-exfil path is closed: post `@atlas` with a payload asking Atlas to read `/proc/self/environ` and echo it — the reply must NOT contain a token.

### Dispatch incantations (copy-paste; verify pane IDs live first)
```bash
# WORKER (claude, herdr) — write the brief to THIS session's scratchpad first, then:
WP=$(herdr tab create --workspace w8 --label "atlas-reply-fix" | jq -r '.result.root_pane.pane_id')
herdr pane run "$WP" "claude"
herdr wait output "$WP" --match "auto mode" --timeout 30000
herdr pane run "$WP" "Read <brief-path> and execute it exactly. Ground in real claude-code-action docs+gh+git; never assume. New branch off master, PR into alianzaSite, push early."
# IMPORTANT: pane-run Enter often does NOT submit — verify + resend:
herdr wait agent-status "$WP" --status working --timeout 8000 || herdr pane send-keys "$WP" Enter

# omp REVIEWER / omp REDTEAM (background; --auto-approve is BLOCKED, use -p --yolo):
omp -p --yolo --model gpt-5.5 @<brief> 'directive' > <out>.txt 2>&1   # run via Bash run_in_background

# claude REDTEAM → Agent tool, run_in_background:true, subagent_type general-purpose.
```

---

## Current architecture of the bot (no-write-token — the P0 fix that DID land)
On `@atlas`: (1) **Gate** (Python, deterministic) validates exact mention + actor (OWNER/MEMBER/COLLABORATOR, non-bot, ≠Atlas-Key); (2) **Generate** runs `claude-code-action` with a read-only token + **no Bash**, Claude writes `reply.md` only; (3) **Sanitize** (deterministic) strips branding/tokens; (4) **Post** uses `GH_TOKEN=ATLAS_KEY_PAT` to post exactly one comment. The PAT NEVER enters Claude's process. This closed the original **P0** (v1 gave the LLM the PAT + `Bash(gh:*)` = arbitrary-`gh` prompt-injection endpoint).

**Secrets on alianzaSite:** `CLAUDE_CODE_OAUTH_TOKEN` (set; Ed will cycle the exposed one), `ATLAS_KEY_PAT` (set, but read-only — see go-live). Compute = Ed's Claude subscription OAuth via the official action (ToS: gray-leaning-allowed for owner-controlled low-volume; research at `.agents/atlas/artifacts/2026-06-24-claude-actions-tos-research.md`).

---

## DISPATCH POLICY (locked this session — memory `[[themis-dispatch-harness]]`)
Roles via **herdr**, by harness — NOT model-selection:
- **Worker** → `claude`  ·  **Reviewer** → `omp` (pi alt)  ·  **Explorer** → `claude`  ·  **RedTeam** → **both `claude` and `omp`** (cross-harness; each has the skills).

### herdr/omp gotchas (learned the hard way)
- `herdr pane run <pane> "claude"` then sending a long prompt **does not reliably submit** — the Enter doesn't register. Pattern: send prompt, then `herdr wait agent-status <pane> --status working --timeout 8000`; if not working, `herdr pane send-keys <pane> Enter`. (Cost a silent stall this session.)
- **herdr churn kills live workers** — Ed actively reshuffles tabs (piThemis/ompThemis run in the same `w8`/ConfigAtlas workspace). Lost 2 workers this way. Mitigation: resilience-critical/validation/swarm work → **background tasks** (Agent tool for claude RedTeam; `Bash run_in_background` for omp); keep only the *primary steerable* worker in herdr and don't reshuffle its tab.
- omp validators: `omp -p --yolo --model gpt-5.5 @<brief-file> 'directive' > out.txt 2>&1` run via `Bash run_in_background`. **`--auto-approve` is blocked** by the auto-mode classifier; Ed authorized `-p --yolo` instead. omp binary: `/Users/ed/.bun/bin/omp` (pi: `/Users/ed/.vite-plus/bin/pi`, codex: `/Users/ed/.bun/bin/codex`).
- Background Agent launches trigger a watchdog hook → start once: `Monitor({persistent:true, command:"bun $HOME/.claude/PAI/TOOLS/AgentWatchdog.ts"})`.

---

## DONE THIS SESSION (don't redo)
- **§8.7 capstone PROVEN + RedTeam PASS (both halves):** `gat-atlas-bot[bot]` App token wrote/deleted a GAT Projects v2 board (`organization_projects: write`); `Atlas-Key` PAT authored an issue+comment on a GAT repo as a User. App key **rotated** (fp `34ae8bad…`; PEM at `~/.config/atlas-bot/atlas-bot.private-key.pem`, chmod 600; env via `~/.env`: `ATLAS_APP_ID=4127929`, `ATLAS_APP_PRIVATE_KEY_PATH`). Memory `[[atlas-bot-app-token-env]]`.
- **5 GAT org project templates** (#2–#6, archetypes solo-software/content-pipeline/research-tracker/life-os-tasks/open-source-repo), marked `template:true` via `markProjectV2AsTemplate`, RedTeam PASS. **PR #29** (atlas-config) has the artifact doc — **merge-ready, not yet merged.** Known gaps (real platform limits): views are UI-only (no GraphQL mutation); `open-source-repo` `Type` field collides with GAT native issue Types (Ed to decide: `Work Type` field / native Types / drop for `type:*` labels).
- **#7 "ALI-Content Pipeline"** (GAT project, alianzaSite's 4 open issues, content-pipeline) — **deferred to a later phase** by Ed. Not a template. Leave it.
- **Three-identity model persisted:** memory `[[atlas-key-vs-atlas-bot]]` + `docs/agents/github-identities.md` + AGENTS.md pointer. (edheltzel = owner/admin/default gh; Atlas-Key = teammate User via `ATLAS_KEY_PAT`, use `GH_TOKEN=$ATLAS_KEY_PAT gh …`; gat-atlas-bot = App `4127929`, ephemeral installation tokens.)
- **§8.6 default-label template:** Ed dropped the manual UI route → per-repo `gh label` canon sync only (already applied to alianzaSite: 16 canon labels verified).
- **`~/.env` propagation fact:** editing `~/.env` does NOT reach a running agent; either relaunch, or `set -a; . ~/.env; set +a` inside the command.

## OPEN THREADS (besides the fix-PR)
- **Avatar** — parked on Ed's call: an agent (tab `atlas-avatar`) went off-brief and made South Park / SpongeBob "Atlas mascot" variants (in `~/Downloads`) instead of my specced minimal armillary-sphere emblem (#3B82F6 on slate). Ed never chose: keep mascots / reset to emblem / both.
- **PR #29** templates artifact — merge when ready.
- **OAuth token** Ed merged into the transcript earlier — Ed said he'll cycle it; don't re-flag (his standing instruction).
- **alianzaSite real issues** (#88/#31/#28/#18) are genuine project work — never close them.

## KEY IDS / PATHS
- Org `Gentle-Art-Technologies` (GAT), id `O_kgDOEaJtNg`. App `gat-atlas-bot` id `4127929`, installation `142206211`. Atlas-Key user id `296298943`; edheltzel id `402910`.
- ⚠️ This session's `scratchpad/` briefs + omp `*-out.txt` are **EPHEMERAL — gone next session.** Don't reference them. The durable equivalents are: the **fix list + success criteria above** (your worker brief — copy it into your own scratchpad), `.agents/atlas/artifacts/2026-06-24-claude-actions-tos-research.md` (ToS), `.agents/atlas/artifacts/2026-06-25-comment-reply-setup.md` (setup), and `docs/agents/github-identities.md`.
- Plan (canonical, older): `.agents/atlas/plans/2026-06-22-github-projects-skill-overhaul.md`.

## LIVE HERDR STATE (workspace `w8` / ConfigAtlas) — verify before trusting; this workspace churns
Leftover agent tabs from this session (reuse or close):
- **`atlas-reply-v2`** (`w8:tH` / pane `w8:p1C`, **done/idle**) — the **claude worker that built the rebuilt PR #92** (its scratchpad brief is gone). Closeable, or reuse the pane for the fix-PR.
- **`w8:p13`** (**done**) — the original v1 comment-reply worker. Closeable.
- `atlas-avatar` tab is already closed; the off-brief mascot PNGs are in `~/Downloads` (avatar decision still open).
- Other `w8` tabs (`ConfigAtlas` = the Themis session, `ompAtlas`, `NeoVim`, `atlas-reply-aliSiteTest`, `pi-themis`) are Ed's own — **leave them.**
- All re-gate validators (omp reviewer/RedTeam, claude RedTeam) ran as **background tasks** and have completed — nothing of theirs is still live. Pane IDs above will have recompacted; re-read with `herdr pane list` before driving.

## FIRST ACTIONS FOR YOU
1. `/Themis`; read this handoff + `docs/agents/github-identities.md` + memory `[[themis-dispatch-harness]]`.
2. Re-verify don't-trust-blindly: `gh pr view 92 --repo Gentle-Art-Technologies/alianzaSite --json state,mergedAt`; confirm the workflow on `master`; `gh secret list --repo Gentle-Art-Technologies/alianzaSite`.
3. Dispatch the **fix-PR worker** (claude, herdr, with the submit-verify pattern) on the 4-item fix list → omp review → claude+omp RedTeam → merge → smoke-test. Surface the avatar + PR #29 decisions to Ed.
4. Archive this handoff to `.agents/atlas/handoffs/archive/` once consumed.
