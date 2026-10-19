# TypeScript

Language-wide TypeScript - standards, refactoring, and type-driven design. Install alongside frontend, backend, or tooling.

## Installation

These skills live in this repo (`edheltzel/agent-config`, being renamed GAP). `gap sync` installs every skill listed in `global/skills.json`.

```bash
gap sync
```

One skill by exact name:

```bash
npx skills add edheltzel/agent-config -s <skill-name>
```

## Skills

| Skill | Description |
|-------|-------------|
| [`do-parse-dont-validate`](./do-parse-dont-validate/) | Type-driven design principle: transform unstructured data into structured types at system boundaries, making illegal states unrepresentable. |
| [`do-ts-standards`](./do-ts-standards/) | Correct-by-construction TypeScript standards. |
| [`do-typescript-refactoring`](./do-typescript-refactoring/) | Systematically refactor TypeScript codebases for readability, type safety, and AI-friendliness. |
| [`do-write-typescript`](./do-write-typescript/) | Write clean, pragmatically functional TypeScript — simple, composable, soundly typed |

## See Also

- [Project skills](../README.md)
- [Skill catalog](../../../README.md)
- [Docs](../../../docs/project/typescript/) - human-facing pages for these skills
