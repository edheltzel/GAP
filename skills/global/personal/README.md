# Personal

Personal extras you want everywhere - teaching, todo.txt, isometric explainer figures.

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
| [`anatomy`](./anatomy/) | Build crafted, highly detailed, interactive isometric SVG figures that explain a technical idea as a real physical object (a test rig, a dial gauge, a cabinet of drawers, a plotter, a lock), with many small precise solid parts, four-tone shading, correct painter's order, a framed card with title, hint and live readout, and calm eased motion. |
| [`do-ducky`](./do-ducky/) | Rubber-duck thinking partner for bugs and half-formed ideas. |
| [`do-teach-me`](./do-teach-me/) | Teach the user a new skill or concept, within this workspace. |
| [`do-todo`](./do-todo/) | Manage tasks through the tuxedo CLI - a todo.txt per project repo, plus a global ~/todo.txt of personal tasks and routes to each project's list - add, list, complete, prioritize, archive, dependencies, ready tasks, and end-of-turn upkeep so do-regroup can reach every project. |

## See Also

- [Global skills](../README.md)
- [Skill catalog](../../../README.md)
