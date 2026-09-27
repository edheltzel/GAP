---
title: Policy links resolve against the wrong relative base
date: 2026-09-27
category: logic-errors
module: gap-sync
problem_type: logic_error
component: documentation
symptoms:
  - "Source links prefixed with global/ resolve to global/global/docs/agents"
  - "A repo-root existence check accepts that bad prefix and rejects the correct relative target"
root_cause: logic_error
resolution_type: code_fix
severity: high
tags:
  - relative-links
  - gap-sync
  - docs-agents
---

# Policy links resolve against the wrong relative base

## Problem

Depth links in `global/AGENTS.md` are relative to that file. Prefixing `global/` makes them resolve to `global/global/docs/agents/`. A check that joins the URL to the repo root reports the opposite of the truth, so the bad prefix looks fixed.

## Symptoms

- From `global/AGENTS.md`, `docs/agents/dox.md` opens `global/docs/agents/dox.md`. `global/docs/agents/dox.md` opens `global/global/docs/agents/dox.md`, which is not in the tree.
- Joining the repo root to `docs/agents/dox.md` misses. Joining it to `global/docs/agents/dox.md` hits. The check trains the next edit to ship the doubled path.
- A published copy that still contains `](docs/agents/` resolves against the harness directory, for example `~/.claude/rules/docs/agents/dox.md`, not the depth doc.

## What Didn't Work

- Prefixing `global/` in the source. Markdown resolves from the file's directory, which is already `global/`. The prefix shifts the path a second time, and the published copy still does not point at the repo.
- Treating "the file exists at the repo root" as proof the link opens. No reader uses the repo root as the link base. The source reader uses `global/`. The published reader uses a harness home.
- Leaving the relative URL in the copy. Sync writes regular files. It does not symlink. Relative `docs/agents/` next to `~/.claude/rules/global-agent-policy.md` cannot see `global/docs/agents/`.

## Solution

Keep the source links relative to `global/`. Rewrite only the bytes that get copied.

`adapt` substitutes the absolute docs directory into every depth link:

```ts
const docsDir = join(repoRoot, "global/docs/agents")

function adapt(source: string, wrap: Wrap): string {
  const body = source.replaceAll("](docs/agents/", `](${docsDir}/`)
  // ...
}
```

That is `src/sync.ts`. The match string is the literal `](docs/agents/`. Every depth link in `global/AGENTS.md` uses that prefix, with no `./` and no leading `global/`. A link written `](./docs/agents/` or `](global/docs/agents/` is not rewritten.

Do not put absolute machine paths, or a `global/` prefix, into `global/AGENTS.md` to make the copies work. Those relative links are already correct when the file is read from `global/`.

## Why This Works

A markdown link resolves from the file that contains it, not from the repo root and not from the directory where `npm run sync` was launched.

No relative URL is relative to both `global/` and `~/.claude/rules/`. Publish has to rewrite. Absolute paths in the source would open on this machine and would still be the wrong edit: a machine path does not belong in git.

Grok reads `~/.claude/rules/*.md` when Claude compatibility is on. It does not get its own copy. A broken Claude link is a broken Grok link.

## Prevention

- Resolve each source link from `global/`, the directory of `global/AGENTS.md`. The resolved path must be `global/docs/agents/<note>.md` and that file must exist. It must not be `global/global/docs/agents/<note>.md`.
- Run the same source through `adapt`. Every `](docs/agents/` must be gone. Each replacement must be `](` plus `join(repoRoot, "global/docs/agents")` plus `/`, and that absolute file must exist. Resolving that absolute target from a stand-in harness directory must still land on the repo file.
- `src/sync.ts --check` proves identity: the payload is not the source, a matching payload is ours, other text is foreign, and the Cursor wrap did not drift. It does not stat either base. A rewrite to a missing path still passes that fixture. The two-base check above is the missing guard.

## Related Issues

- None linked. `docs/solutions/` had no earlier learning. Closed GitHub issues were not listed in this run.
