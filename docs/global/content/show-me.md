# Show Me

Quickstart:

This skill lives in this repo (`edheltzel/agent-config`, being renamed GAP) at `skills/global/content/do-show-me`. `global/skills.json` lists it with `"source": "."`. Install and refresh it with:

```bash
gap sync
```

One skill by exact name:

```bash
npx skills add edheltzel/agent-config -s do-show-me
```

[Source](https://github.com/edheltzel/agent-config/tree/master/skills/global/content/do-show-me)

Upstream: [humanlayer/skills](https://github.com/humanlayer/skills/tree/main/plugins/show-me) ([post](https://www.humanlayer.com/blog/show-me-skill))

## What it does

`do-show-me` makes the agent explain the current topic with compact visuals instead of walls of prose. It picks the smallest view that makes the point: pseudocode, a call tree, a component tree, a shallow file tree, a Mermaid diagram, a `diff` of any of those, or one focused HTML file when the idea is too dense for text.

An HTML file is reviewed in Plannotator when that tool or the `plannotator` command is installed. Approved ends the review. Annotated feedback is applied and the file is reviewed again. If Plannotator is not installed, the file is opened locally instead. The skill never fires on its own. You call it when a reply got too dense to read.

## When to reach for it

Type `/do-show-me` after a reply full of jargon, or before the agent writes code, to talk through the shape of the change: types, signatures, call stacks. It also helps when reviewing a large diff to decide what to dig into.

## Where it fits

A standalone you reach for anytime. [art](./art.md) makes polished static visuals for an audience. `do-show-me` makes quick working sketches for you, inside the conversation.
