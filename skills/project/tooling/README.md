# Tooling

CLIs and lightweight tools - TypeScript CLI scaffolding, zero-dependency patterns, and performance profiling.

## Installation

Every skill in this group:

```bash
npx skills add https://github.com/edheltzel/Do-Skills/tree/master/skills/project/tooling
```

One skill by exact name:

```bash
npx skills add edheltzel/Do-Skills --skill=<skill-name>
```

## Skills

| Skill | Description |
|-------|-------------|
| [`do-create-cli`](./do-create-cli/) | Generates production-ready TypeScript CLIs via a 3-tier template system (manual arg parsing, Commander.js, oclif), each shipping full implementation, docs, package.json, strict config, JSON output, and exit-code compliance. |
| [`do-lean-ts-patterns`](./do-lean-ts-patterns/) | Patterns for building lightweight, zero-dependency TypeScript tools and libraries. |
| [`do-perf`](./do-perf/) | The full measure → fix → report loop for Electron and web apps via Playwright and Chromium CDP: measure the real production build, find the cause, fix it, re-measure the same way, and write an HTML report from harness JSON, with Blink style-invalidation traces and a React commit probe that name causes at file:line; use when the user asks to profile, speed up, slim down, find memory leaks, benchmark before and after, or produce a performance report; this is the complete audit with scripts — not a lightweight render-cost note such as lite-render-perf, so do not pick those in its place. |

## See Also

- [Project skills](../README.md)
- [Skill catalog](../../../README.md)
- [Docs](../../../docs/project/tooling/) - human-facing pages for these skills
