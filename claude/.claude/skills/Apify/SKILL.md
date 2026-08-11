---
name: Apify
version: 1.2.0
description: "Scrapes supported social, business, ecommerce, and web sources through bundled Apify actor wrappers, with filtering in code before model-facing output. USE WHEN Apify actor, scrape Instagram, LinkedIn, TikTok, YouTube, Facebook, Google Maps, Amazon, or a website. NOT FOR browser control, bypassing access controls, or unapproved live network calls."
effort: medium
---

# Apify

## Purpose

Use the self-contained TypeScript wrappers under `~/.claude/skills/Apify` to call Apify actors and filter results before returning them to model context.

Actor calls are live network operations that may incur charges. Confirm the target, actor, limits, permission, and user intent before running one. Offline validation must never call the Apify API.

## Supported installation

```text
~/.claude/skills/Apify/
├── index.ts
├── actors/
├── types/
├── package.json
└── bun.lock
```

Provision dependencies beside the installed skill:

```bash
cd ~/.claude/skills/Apify
bun install --frozen-lockfile --ignore-scripts
```

## Credential preflight

The wrapper accepts `APIFY_TOKEN` or the compatibility alias `APIFY_API_KEY`. Constructing `Apify` fails before network access when both are absent.

```bash
export APIFY_TOKEN="apify_api_xxxxx"
```

Do not read credentials from repository files or retired system environment files.

## Workflow routing

| Workflow | Trigger | Route |
| --- | --- | --- |
| Update | dependency, API, actor schema, or wrapper maintenance | `Workflows/Update.md` |
| Inline actor use | authorized scrape, lead, ecommerce, or crawl request | wrappers under `actors/` |

## Wrapper imports

Run code from the skill root and use explicit relative module paths:

```ts
import { scrapeInstagramProfile, searchGoogleMaps } from "./actors/index.ts";

const profile = await scrapeInstagramProfile({
  username: "target_username",
  maxPosts: 50,
});

const top = profile.latestPosts
  ?.sort((a, b) => b.likesCount - a.likesCount)
  .slice(0, 10);

console.log(top);
```

For a script outside the skill directory, derive a file URL from `${HOME}/.claude/skills/Apify` and use dynamic import. Do not use a generic package specifier such as `actors`.

## Available wrappers

### Social media

- Instagram: profiles, posts, hashtags, comments
- LinkedIn: profiles, jobs, posts
- TikTok: profiles, hashtags, comments
- YouTube: channels, search, comments
- Facebook: posts, groups, comments

### Business and ecommerce

- Google Maps: places, place details, reviews
- Amazon: products and reviews

### Web

- General website crawl and page extraction

Actor IDs and schemas can change. Review the current Actor Store listing before any authorized live run.

## Safe operating sequence

1. Read the relevant wrapper and its input type.
2. Confirm authorization, result limit, and expected cost.
3. Confirm a credential is available without printing it.
4. Execute the smallest live call that satisfies the request.
5. Validate output shape as untrusted external data.
6. Filter, sort, aggregate, and limit in code.
7. Return only the necessary result and report actor failures accurately.

## Offline checks

```bash
cd ~/.claude/skills/Apify
bun install --frozen-lockfile --ignore-scripts
bun examples/smoke-test.ts
bunx tsc --noEmit -p tsconfig.json
```

The smoke test imports the module, verifies missing-token failure, and constructs a client with a fake token. It does not invoke any method that performs network I/O.

## Gotchas

- Missing credentials are configuration errors, not empty actor results.
- Actor output formats vary and must be narrowed defensively.
- Large result limits increase cost and context pressure even when filtering later.
- Do not claim actor availability or pricing from old documentation. Check the current official source.
- Do not use tests, examples, or maintenance checks to make live actor calls.

## References

- Apify platform: <https://apify.com/>
- Actor Store: <https://apify.com/store>
- API documentation: <https://docs.apify.com/api/v2>
- Local API guide: `README.md`
- Integration guide: `INTEGRATION.md`
