---
name: project-setup
description: >
  Set a website up in Ubersuggest for the first time — create the project,
  describe the business, pick competitors, choose the topics and prompts tracked
  in AI answers, and select keywords to rank-track. Use when the user has no
  project yet, says they just signed up, asks how to get started, wants to track
  a site, add a domain, set up rank tracking, or start monitoring their brand in
  ChatGPT and other AI assistants.
argument-hint: "<your website> [where your customers are]"
---

# First-time project setup

**Input:** `$ARGUMENTS` — the website to set up, and where its customers are. If
that placeholder is empty or still literal, take the website from what the user
asked; if they named none, ask for the domain — nothing else is required.

You are setting up the thing every other Ubersuggest workflow reads from. A
project is what makes rankings, competitors and AI visibility trackable over
time; without one, every other skill can only look at public data.

## Before you start

1. **`auth_status`.** Setting a project up writes to the user's account, so it
   needs login. Not logged in → ask for the connection and stop; there is no
   version of this flow that works without it, and telling the user to create
   the project in the web app instead just gives them the manual work.
2. **`user_limits`.** Tells you how many projects, keywords, competitors,
   locations and prompts the plan allows. Read it before promising anything —
   on a free plan this is 1 project, 25 keywords, 2 competitors, 1 location and
   10 prompts.
3. **`list_projects`.** If the domain is already set up, say so and stop; this
   skill is for a domain that has none. To change an existing one, use
   `add_project_keywords`, `add_project_competitors` or `configure_brand`.
4. **Resolve the market.** Ask where their customers are and call
   `location_suggest` to get the real `loc_id`. Never guess one — a wrong id
   silently tracks the wrong country, and the numbers look plausible. Default is
   English / United States (2840) if the user does not care.

## Running the setup

`onboard_project` does the whole flow. It takes minutes of server work, so it
returns early with `done: false` and a `stage`. **Call it again, passing back
every field it returned** (`business_summary`, `competitors`, `topics`,
`keywords`) — that is what stops it redoing work.

```
onboard_project({ domain, locations })
  → stage: "business_summary" | "generation" | "keywords"   // keep calling
  → stage: "review"                                          // STOP HERE
  → stage: "done"                                            // project exists
```

While it works, say what it is doing — reading the site, finding competitors,
writing prompts, generating keywords. Do not go silent for minutes.

If it reports the analysis could not read the site, do not invent a business
summary. Ask the user what the business does, who it sells to, and what it
offers, then pass `business_summary` yourself.

## The review stage — never skip it

When `stage` comes back `"review"`, **nothing has been created yet**, and you
must not call again with `confirm: true` on your own judgement. This is the one
point where the user decides what gets tracked.

Show them, in plain language, not as raw JSON:

- **The business summary** — is this actually what they do? It is what the AI
  prompts and article generation are grounded on, so a wrong summary poisons
  everything downstream.
- **The competitors** — each domain, and say which ones you would question.
  Suggested competitors are guesses from the site's content; the user knows
  who they actually lose deals to.
- **Each topic with its prompts** — this is the most important list on the
  screen. The prompts *are* the measurement: visibility is defined as how often
  the brand shows up in the answers to exactly these questions. Ask whether
  these are how their customers would really ask.
- **The keywords** — how many are selected and what the plan allows.

Then read out anything in `notes`: lists get trimmed to the plan's limits, and
the user should hear what was dropped rather than discover it later.

Ask what they want to change. To apply edits, call again with `confirm: true`
**and** the corrected `competitors`, `topics` or `keywords` — what you pass wins
over what was generated. Removing a prompt, renaming a topic or swapping a
competitor is just sending the edited list.

Mention once, without labouring it: saving the brand spends one of the account's
monthly AI-visibility operations, and prompts can only be changed by replacing
the whole list — so fixing it now is free and fixing it later is not.

## After it is done

Confirm what exists now, and set expectations honestly:

- Rank tracking and the first AI visibility report take **about a day** to
  populate. Reading them immediately returns empty, and that is not a failure.
- `site_audit` works right away and is the useful thing to do next.
- `brand_config` shows what ended up tracked; `configure_brand` changes it.

Then offer one next step, not a menu — usually the site audit, or
`seo-action-plan` if they do not know what to do with any of it.

## When things go wrong

- **"domain already exists"** — the project is already there. `list_projects`,
  then work with it instead of creating another.
- **Project or brand limit reached** — the plan is full. Say which limit, and
  that it means archiving something or upgrading. Do not retry.
- **The project was created but the brand was not** — `notes` will say why
  (usually no AI-visibility slot on the plan, or the monthly operations pool is
  spent). The project is still fully usable; retry the brand later with
  `configure_brand`.
- **No topics were suggested** — the site gave the analyser nothing to work
  with. Ask the user which two or three topics their customers ask about, then
  use `industry_prompts` to write prompts for them and `configure_brand` to save.
