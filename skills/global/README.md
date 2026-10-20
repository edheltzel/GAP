# Global Skills

Useful in any repo, or none.

## Installation

These skills live in this repo (`edheltzel/agent-config`, being renamed GAP). `gap sync` installs every skill listed in `global/skills.json`.

```bash
gap sync
```

One skill by exact name:

```bash
npx skills add edheltzel/agent-config -s <skill-name>
```

## Groups

| Group | Skills | Coverage |
|-------|--------|----------|
| [Core](./core/) | 12 | Stack-agnostic foundations - review lenses, reasoning, testing, simplification, agent-legible repos, comments, design patterns, skill authoring, and AI-writing cleanup. |
| [Workflow](./workflow/) | 15 | Change delivery - the brainstorm, plan, work, debug, and ship loop (LFG), commits, docs, GitHub projects and stacks, PR workflow and review triage, worktrees, and technical writing. |
| [Operations](./operations/) | 4 | Operating AI agents - delegation, handoffs, prompt audits, and project status check-ins. |
| [Content](./content/) | 3 | Audience-facing media - illustrations, diagrams, images, and visual explanations. |
| [Personal](./personal/) | 4 | Personal extras you want everywhere - teaching, todo.txt, isometric explainer figures. |

## See Also

- [Skill catalog](../../README.md)
