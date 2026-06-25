# Claude Code GitHub Actions — ToS / Subscription-OAuth Research

> Research date: 2026-06-24. Author: Atlas (research agent). Purpose: decide whether an event-triggered GitHub Actions bot may be powered by a Claude Max/Pro subscription OAuth token, and design our `Atlas-Key`-attributed variant. Every claim below is cited inline.

---

## 1. ToS Verdict — LOAD-BEARING

### Verdict: **GRAY-AREA, leaning ALLOWED** for the official `anthropics/claude-code-action` with `claude_code_oauth_token`; **NOT ALLOWED** for any non-Anthropic harness on a subscription token.

The question turns on one sentence in Anthropic's Consumer Terms and one exception inside it. The official GitHub Action lands inside that exception; a custom/third-party runner does not.

#### The governing rule (Consumer Terms, Section 3 — "Use of our Services")

> "Except when you are accessing our Services via an Anthropic API Key **or where we otherwise explicitly permit it**, to access the Services through automated or non-human means, whether through a bot, script, or otherwise."
> — Anthropic Consumer Terms §3, quoted via <https://www.anthropic.com/legal/consumer-terms> and corroborated at <https://www.daimonlegal.com/blog/anthropic-banned-my-account-for-using-openclaw-heres-what-to-do-if-it-happens-to-you>

So automated/bot/script access on a **subscription** (non-API-key) credential is prohibited **unless Anthropic explicitly permits it**. Two facts establish that the official Action is explicitly permitted:

1. **Anthropic itself ships and documents subscription-OAuth auth for the Action.** The official `anthropics/claude-code-action` setup doc lists, verbatim:
   > "Or `CLAUDE_CODE_OAUTH_TOKEN` for OAuth token authentication (Pro and Max users can generate this by running `claude setup-token` locally)"
   > — <https://github.com/anthropics/claude-code-action/blob/main/docs/setup.md>

   The `action.yml` defines the input verbatim as:
   > `claude_code_oauth_token`: "Claude Code OAuth token (alternative to anthropic_api_key)"
   > — <https://raw.githubusercontent.com/anthropics/claude-code-action/main/action.yml>

   Anthropic publicly announced Max-subscription support for GitHub Actions: "Pro and Max subscribers can authenticate with an OAuth token instead, and the workflow will use your existing subscription quota." (<https://wain.blog/en/claude-code-github-actions-max-support-8NB583zS/>)

2. **The April 2026 crackdown scoped the ban to *third-party* harnesses and explicitly exempted Anthropic's own products — including Claude Code.** Effective **April 4, 2026**, Anthropic "restricted OAuth token authentication to its own products — Claude.ai and Claude Code — only." Coverage states: "Claude Code, Claude.ai, and the Claude Desktop app continue working with your Pro or Max subscription exactly as before." (<https://venturebeat.com/technology/anthropic-cuts-off-the-ability-to-use-claude-subscriptions-with-openclaw-and>, <https://news.ycombinator.com/item?id=47633568>, <https://www.mindstudio.ai/blog/anthropic-openclaw-ban-third-party-harnesses-claude-subscriptions>). The enforced reading: "The ToS for Claude.ai have always prohibited automated, programmatic, and commercial use through consumer subscription accounts. What changed is enforcement: Anthropic actively detects and terminates accounts using third-party harnesses." (search synthesis of the OpenClaw coverage above).

   `claude-code-action` is part of **Claude Code** — it is "built on top of the Claude Agent SDK" and is the Anthropic-published, Anthropic-maintained Action (<https://code.claude.com/docs/en/github-actions>). It is therefore on the *permitted* side of the line the crackdown drew.

#### Why it is still GRAY, not clean ALLOWED

- **The main Anthropic docs do NOT mention subscription/OAuth auth at all.** `code.claude.com/docs/en/github-actions` documents only `ANTHROPIC_API_KEY`, Bedrock, and Vertex. The OAuth path lives only in the action repo's `docs/setup.md`/`docs/usage.md` and the `action.yml`. The first-party narrative still treats API key as the default ("Simple setup: Get started in minutes with our installer and **API key**"). (<https://code.claude.com/docs/en/github-actions>)
- **Anthropic frames Max as single-user and recommends API keys for teams.** "Anthropic emphasizes that Max subscriptions are intended for single users. For repositories with multiple users, traditional API key usage is still recommended." (<https://wain.blog/en/claude-code-github-actions-max-support-8NB583zS/>) A bot that auto-replies to *other people's* issues/PRs blurs the "single user" framing — usage is driven by third parties even though the credential and the Action are first-party.
- **No published rate-limit or account-risk carve-out for CI.** Neither `docs/setup.md` nor `docs/usage.md` contains any rate-limit, team-use, or ToS caveat (confirmed: "None found"). Subscription quota is shared with your interactive Claude Code usage, and heavy unattended runs draw it down. There is no documented guarantee that high-volume automated bursts won't be flagged.
- **The contradiction is real and acknowledged.** Legal commentary notes Anthropic "actively promotes the exact conduct the ToS appears to ban" (CLI piping, GitHub Actions, autonomous operation), and argues banning the official shipped use case would be Anthropic "going after their own shipped use case" — but concludes "the ToS language creates ambiguity that leaves automated/unattended operation in CI genuinely at risk." (<https://www.daimonlegal.com/blog/anthropic-banned-my-account-for-using-openclaw-heres-what-to-do-if-it-happens-to-you>)

#### Practical risk posture

- **Lowest risk / clearly compliant:** use `ANTHROPIC_API_KEY` (console.anthropic.com, pay-per-use) — the unambiguous, explicitly-permitted path for automation. This sidesteps §3 entirely.
- **Acceptable but gray:** `claude_code_oauth_token` from `claude setup-token` on a **personal Max/Pro** account, on **your own low-volume repos**, single-user-driven. This is Anthropic's own documented feature and survives the April crackdown. Risk = quota draw + the "single user" framing if many external people trigger it.
- **Not allowed:** feeding that same subscription OAuth token into a **non-Anthropic** harness/custom runner (e.g. a bespoke script that hits the Anthropic OAuth endpoint outside Claude Code). That is exactly what the April 4 2026 ban terminates accounts for.

---

## 2. Official Integration Architecture

**Components** (<https://code.claude.com/docs/en/github-actions>, <https://github.com/anthropics/claude-code-action>):
- **Claude GitHub App** (`https://github.com/apps/claude`) — requests repo permissions Contents R/W, Issues R/W, Pull requests R/W. Installed via `/install-github-app` in the Claude Code CLI, or manually.
- **Workflow YAML** in `.github/workflows/` referencing `anthropics/claude-code-action@v1`.
- **Secrets:** one of `ANTHROPIC_API_KEY` (direct API, default) or `CLAUDE_CODE_OAUTH_TOKEN` (Pro/Max subscription) — or Bedrock/Vertex/Foundry via OIDC (no static key). API key is `Yes**` required for direct API, "not needed for Bedrock/Vertex/Foundry" (<https://raw.githubusercontent.com/anthropics/claude-code-action/main/action.yml>).

**Trigger model** (verbatim basic workflow, <https://code.claude.com/docs/en/github-actions>):
```yaml
on:
  issue_comment:
    types: [created]
  pull_request_review_comment:
    types: [created]
```
Common practice adds `issues: [opened, assigned]`. Default trigger phrase is `@claude` (configurable via `trigger_phrase`). v1 auto-detects "interactive mode" (responds to `@claude` mentions) vs "automation mode" (runs immediately with a `prompt`). Recommended `if:` guard filters on `contains(github.event.comment.body, '@claude')` so the job only runs on real mentions.

**Key `action.yml` inputs** (verbatim, <https://raw.githubusercontent.com/anthropics/claude-code-action/main/action.yml>):
- `github_token`: "GitHub token with repo and pull request permissions (optional if using GitHub App)"
- `anthropic_api_key`: "Anthropic API key (required for direct API, not needed for Bedrock/Vertex/Foundry)"
- `claude_code_oauth_token`: "Claude Code OAuth token (alternative to anthropic_api_key)"
- `use_bedrock` / `use_vertex`: switch provider (default false)
- `prompt`, `trigger_phrase` (default `@claude`), `claude_args` (raw Claude CLI passthrough — `--model`, `--max-turns`, `--mcp-config`, `--allowedTools`, `--append-system-prompt`), `settings` (Claude Code settings JSON string or path)

**Default attribution:** comments are posted by the **Claude GitHub App** identity, i.e. as `claude[bot]`, when you rely on the App / default token (<https://github.com/anthropics/claude-code-action>).

---

## 3. The `Atlas-Key`-Attribution Approach (post as a user, not a `[bot]`)

**Goal:** comments authored by our `Atlas-Key` *user* account, not `claude[bot]`.

**Mechanism:** the `github_token` input overrides the posting identity. The action repo's `docs/usage.md` marks it verbatim:
> `github_token`: "GitHub token for Claude to operate with. **Only include this if you're connecting a custom GitHub app of your own!**"
> — <https://github.com/anthropics/claude-code-action/blob/main/docs/usage.md>

To attribute comments to a **specific human/service user** (`Atlas-Key`):
- Generate a **classic/fine-grained PAT for the `Atlas-Key` account** (scopes: repo / issues / pull-requests R/W) and store as a secret.
- Pass it as `github_token: ${{ secrets.ATLAS_KEY_PAT }}`. All issue/PR comments the action posts will then be authored by `Atlas-Key` (the token owner), not `claude[bot]`.
- (Anthropic's own "branded usernames / custom authentication flows" path uses a custom GitHub App + `actions/create-github-app-token`; that attributes to `<app-name>[bot]`, still a bot. To get a real *user* author, a PAT for that user is the route. — <https://code.claude.com/docs/en/github-actions> "Custom GitHub App" note.)

**Loop / recursion hazard — important.** GitHub deliberately suppresses recursion for the default `GITHUB_TOKEN`:
> "when a workflow run pushes code using the repository's GITHUB_TOKEN, a new workflow will not run … Activities authorized with the GITHUB_TOKEN never trigger additional workflows to avoid unintentional recursive runs."
> — <https://docs.github.com/en/actions/concepts/security/github_token>

A **PAT does NOT get this protection** — comments/pushes made with a PAT *do* fire new `issue_comment`/`push` events, which can re-trigger your own workflow → infinite loop. (<https://deku.posstree.com/en/github_actions/github-actions-workflow-retrigger/>) Guards to add when posting with `Atlas-Key`'s PAT:
1. **Actor filter:** `if: github.event.comment.user.login != 'Atlas-Key'` (skip when the comment author is our own bot user).
2. **Trigger-phrase filter:** only run when the comment contains `@claude` (or your custom phrase) — our own replies should not include the trigger phrase.
3. **Concurrency + `--max-turns`** caps to bound any runaway.
4. Consider `if: !contains(github.event.comment.body, '<!-- atlas-bot -->')` sentinel marker in our own replies as a belt-and-suspenders.

---

## 4. Fallback Assessment — OpenRouter + DeepSeek (if subscription OAuth is ruled out)

**Does `anthropics/claude-code-action` support non-Anthropic models / a custom base URL?**
Officially, **no** — the action documents only Anthropic direct API, Bedrock, Vertex, and Microsoft Foundry as providers (<https://github.com/anthropics/claude-code-action>). There is no first-party `ANTHROPIC_BASE_URL`/OpenRouter input.

**Unofficially / workaround:** the underlying **Claude Code CLI honors `ANTHROPIC_BASE_URL` + `ANTHROPIC_AUTH_TOKEN`**, and the action exposes `settings` (Claude Code settings JSON) and `claude_args`. So you *can* point Claude Code at an OpenRouter (Anthropic-compatible) or LiteLLM endpoint via env/settings:
> "When you set ANTHROPIC_BASE_URL to https://openrouter.ai/api, Claude Code speaks its native protocol directly to OpenRouter. ANTHROPIC_AUTH_TOKEN is the bearer token Claude Code sends to whatever endpoint ANTHROPIC_BASE_URL points at."
> — <https://openrouter.ai/docs/cookbook/coding-agents/claude-code-integration>, <https://www.morphllm.com/use-different-llm-claude-code>

Model remapping is required because Claude Code hardcodes `sonnet`/`opus`/`haiku` internally — remap via `ANTHROPIC_MODEL` and `ANTHROPIC_SMALL_FAST_MODEL` (<https://www.morphllm.com/use-different-llm-claude-code>). For OpenAI-format/self-hosted models you'd front them with **LiteLLM** translating Anthropic Messages API ↔ upstream (<https://www.morphllm.com/claude-code-litellm>).

**Caveat:** this is unsupported by Anthropic, brittle across Claude Code releases, and non-Anthropic models lose tool-use fidelity the action assumes. A cleaner fallback is a **different action / custom runner** that calls OpenRouter directly and posts via the GitHub API — decoupled from `claude-code-action` entirely. Decision point for design: bolt OpenRouter onto Claude Code (fast, fragile) vs. write a thin custom Action (more work, robust).

**Current DeepSeek model IDs + pricing on OpenRouter** (<https://openrouter.ai/deepseek>, model pages linked below; the user's "DeepSeek 4 flash or pro" maps to the V4 line):

| Model | OpenRouter ID | Input $/M | Output $/M | Context | Source |
|---|---|---|---|---|---|
| DeepSeek V4 Pro | `deepseek/deepseek-v4-pro` | $0.435 | $0.87 | 1,048,576 | <https://openrouter.ai/deepseek/deepseek-v4-pro> |
| DeepSeek V4 Flash | `deepseek/deepseek-v4-flash` (non-thinking) / thinking variant | (sub-Pro; cheapest tier) | — | — | per deprecation note below |
| DeepSeek V3.2 | `deepseek/deepseek-v3.2` | $0.2288 | $0.3432 | 131,072 | <https://openrouter.ai/deepseek/deepseek-v3.2> |
| DeepSeek V3.1 | `deepseek/deepseek-chat-v3.1` | $0.21 | $0.79 | 163,840 | <https://openrouter.ai/deepseek/deepseek-chat-v3.1> |
| DeepSeek V3 (chat) | `deepseek/deepseek-chat` | $0.2002 | $0.8001 | 131,072 | <https://openrouter.ai/deepseek/deepseek-chat> |

Note: "The model names `deepseek-chat` and `deepseek-reasoner` will be deprecated on 2026/07/24 … they correspond to the non-thinking mode and thinking mode of **deepseek-v4-flash**." So "DeepSeek 4 flash" = `deepseek/deepseek-v4-flash` (cheapest, fast) and "pro" = `deepseek/deepseek-v4-pro` (1M context, ~$0.44/$0.87 per M). All are ~10-50x cheaper than Claude. (<https://api-docs.deepseek.com/quick_start/pricing>)

**Fallback recommendation:** if you abandon subscription OAuth, prefer a **purpose-built custom Action/runner calling OpenRouter** with `deepseek/deepseek-v4-flash` for cheap triage and `deepseek/deepseek-v4-pro` for substantive replies, posting via the GitHub API as `Atlas-Key`. Reserve the `claude-code-action`+`ANTHROPIC_BASE_URL` hack only if you specifically want Claude Code's agentic tool loop and accept the brittleness.

---

## Sources

- Anthropic Consumer Terms — <https://www.anthropic.com/legal/consumer-terms>
- Claude Code GitHub Actions docs — <https://code.claude.com/docs/en/github-actions>
- claude-code-action setup — <https://github.com/anthropics/claude-code-action/blob/main/docs/setup.md>
- claude-code-action usage — <https://github.com/anthropics/claude-code-action/blob/main/docs/usage.md>
- action.yml — <https://raw.githubusercontent.com/anthropics/claude-code-action/main/action.yml>
- Max-subscription GitHub Actions support — <https://wain.blog/en/claude-code-github-actions-max-support-8NB583zS/>
- April 2026 third-party harness ban — <https://venturebeat.com/technology/anthropic-cuts-off-the-ability-to-use-claude-subscriptions-with-openclaw-and>, <https://news.ycombinator.com/item?id=47633568>, <https://www.mindstudio.ai/blog/anthropic-openclaw-ban-third-party-harnesses-claude-subscriptions>
- ToS legal analysis — <https://www.daimonlegal.com/blog/anthropic-banned-my-account-for-using-openclaw-heres-what-to-do-if-it-happens-to-you>
- GITHUB_TOKEN recursion behavior — <https://docs.github.com/en/actions/concepts/security/github_token>, <https://deku.posstree.com/en/github_actions/github-actions-workflow-retrigger/>
- OpenRouter + Claude Code / custom base URL — <https://openrouter.ai/docs/cookbook/coding-agents/claude-code-integration>, <https://www.morphllm.com/use-different-llm-claude-code>, <https://www.morphllm.com/claude-code-litellm>
- DeepSeek on OpenRouter — <https://openrouter.ai/deepseek>, model pages linked in table
