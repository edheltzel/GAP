# Apify Code-First API

This skill provides TypeScript wrappers around `apify-client`. The supported installation root is:

```text
~/.claude/skills/Apify
```

## Provisioning

The skill tracks `package.json` and `bun.lock`. Install exactly the locked dependencies beside the real skill files:

```bash
cd ~/.claude/skills/Apify
bun install --frozen-lockfile --ignore-scripts
```

Do not install dependencies through an unrelated working directory.

## Authentication preflight

Set one token in the process environment:

```bash
export APIFY_TOKEN="apify_api_xxxxx"
# APIFY_API_KEY is accepted as a compatibility alias.
```

Module import is offline and does not require a token. Constructing `Apify` performs a fail-fast preflight and throws before any network request when neither variable is set.

```bash
cd ~/.claude/skills/Apify
bun -e 'import { resolveApifyToken } from "./index.ts"; console.log(resolveApifyToken() ? "token available" : "unreachable")'
```

## Importing wrappers

Use relative imports from the skill root, not a package-like `actors` specifier:

```ts
import { scrapeInstagramProfile, searchGoogleMaps } from "./actors/index.ts";

const profile = await scrapeInstagramProfile({
  username: "target",
  maxPosts: 50,
});

const topPosts = profile.latestPosts
  ?.sort((a, b) => b.likesCount - a.likesCount)
  .slice(0, 10);

console.log(topPosts);
```

For a script outside the skill directory, use a file URL derived from the supported root:

```ts
const root = `${process.env.HOME}/.claude/skills/Apify`;
const { searchGoogleMaps } = await import(`${root}/actors/index.ts`);
```

## Direct client API

```ts
import { Apify } from "./index.ts";

const apify = new Apify();
const actors = await apify.search("web scraper", { limit: 5 });
const run = await apify.callActor(actors[0].id, { startUrls: ["https://example.com"] });
const items = await apify.getDataset(run.defaultDatasetId).listItems({ limit: 100 });
console.log(items.slice(0, 10));
```

Actor calls require network access and may incur Apify charges. Confirm the actor, input, limits, and user authorization before running them. Tests and preflight checks must not call the live API.

## Local checks

```bash
cd ~/.claude/skills/Apify
bun install --frozen-lockfile --ignore-scripts
bunx tsc --noEmit -p tsconfig.json
bun examples/smoke-test.ts
```

The smoke test is offline. It verifies module loading and credential failure behavior without calling Apify.
