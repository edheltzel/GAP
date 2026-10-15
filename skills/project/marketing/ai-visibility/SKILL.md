---
name: ai-visibility
description: >
  Report how visible a brand is inside AI answers (ChatGPT, Perplexity, Google
  AI Overviews, Claude) using Ubersuggest's AI Search Visibility data. Use when
  the user asks whether ChatGPT or other AI tools mention their brand, about AI
  search visibility, share of voice in AI answers, AEO or GEO, brand sentiment
  in LLM responses, or which competitors get recommended instead of them.
argument-hint: "[project name or domain]"
---

# AI search visibility

**Input:** `$ARGUMENTS` — the project name or domain to report on. If that
placeholder is empty or still literal, take the target from what the user asked;
if they named none, list their projects and ask which one.

## Before you start: login + a configured project

All three brand tools require an authenticated account **and** a project with
AI Search Visibility configured. Call `auth_status` first.

- Not logged in → explain that this data is tied to their Ubersuggest account,
  ask for the connection, and stop. Your impression of how often ChatGPT names
  a brand is not a measurement — never offer one in place of the report.
- Logged in but no project → run the **project-setup** skill: it creates the
  project and configures the AI visibility topics and prompts in one flow.
- Project exists but no brand configured (`has_brand` is false, or `brand_config`
  says no brand was found) → `configure_brand` sets the topics and prompts up.
  Either way the first report takes about a day to populate, so say that rather
  than looping on empty results.

## Steps

1. **Find the project.** `list_projects`, then match on the name or domain the
   user gave. If several match, ask. `get_project` for detail if needed.

2. **Read the configuration.** `brand_config` with the project id → the tracked
   topics, prompts, competitors and brand aliases. Read this *before* the
   metrics: it tells you what the numbers are measuring. If the tracked prompts
   do not reflect how customers actually ask about this business, that is
   itself a finding worth reporting.

3. **Pull the overview.** `brand_visibility_overview` → visibility %, average
   rank among mentions, share of voice, sentiment, broken down per provider.
   Pass `start_date`/`end_date` for a specific window; if the user wants a
   trend, run two windows and compare rather than guessing at direction.
   `provider` filters to one engine when the user cares about, say, ChatGPT
   only.

4. **Go prompt by prompt.** `brand_prompts`, same window → per-prompt results.
   This is where the actionable detail is. Split them:
   - **Prompts where the brand appears** and ranks well → what is working.
   - **Prompts where competitors appear and the brand does not** → the gap, and
     the priority list.
   - **Prompts with negative or mixed sentiment** → a reputation problem, not a
     visibility problem, and it needs a different fix.

5. **Compare providers.** Visibility often differs sharply between engines
   because they draw on different sources. Strong in Perplexity but absent in
   ChatGPT is a real, diagnosable pattern — say which engines are weak.

## Deliverable

- **Headline** — visibility %, share of voice, average rank, overall sentiment,
  for the stated window.
- **Per-provider table:**

| Provider | Visibility % | Avg rank | Share of voice | Sentiment |
| --- | --- | --- | --- | --- |

- **Where the brand is missing** — the prompts with no mention, with the
  competitors that appeared instead, highest-value prompts first.
- **Where competitors win** — which competitor dominates which topic.
- **Sentiment flags** — any prompt where the brand is mentioned unfavourably,
  quoted or characterised specifically.
- **Recommendations**, grounded in how answer engines actually pick sources
  (if the `seo-foundations` skill is installed alongside this one, its
  `references/methodology.md` covers this in the AEO/GEO section; if it is not
  available, do not go looking for it — the guidance below is enough). Typically: content that directly answers the missing prompts with
  extractable structure; explicit entity naming; and off-site presence in the
  reviews, roundups and directories the engines synthesise from — since a brand
  absent from third-party sources tends to be absent from AI answers regardless
  of its own site.

## When something fails

- Empty overview but a valid project → AI visibility tracking is probably not
  configured yet, or the date window predates tracking. Check `brand_config`
  before concluding the brand is invisible; "no data" and "not mentioned" are
  very different findings and must not be conflated.
- Plan/quota error → report which limit was hit.
- Tools not connected, or `auth_status` says logged out → ask for the connection (*No numbers without the connection* in `seo-foundations`) and stop. Do not fill the gap with a web search, a page fetch or prior knowledge, do not send the user to run the report in the web app, and never offer another SEO provider's connector in place of Ubersuggest.
