---
name: do-agent-docs
description: >-
  Make a repository legible to coding agents. Use when writing, creating, or
  improving AGENTS.md, writing agent instructions, onboarding an agent to a
  codebase, making a repo agent-first or agent-friendly, improving agent
  legibility, harness engineering, a Codex workflow, a docs tree for agents,
  CLAUDE.md routing, or installing DOX.
---

# Agent docs

Make the repo the thing an agent can read. Pick the mode that matches the ask, then load that reference. Do not load the others.

| Ask | Load |
|---|---|
| Write or tighten `AGENTS.md` | [references/agents-md.md](references/agents-md.md) |
| Route context: pointers, one home per fact, docs tree | [references/context-layer.md](references/context-layer.md) |
| Repo structure, enforcement, entropy | [references/agent-first-repo.md](references/agent-first-repo.md) |
| Install the DOX `AGENTS.md` hierarchy | [references/dox-install.md](references/dox-install.md) |

Architecture codemaps are a separate skill: `do-architecture-md`, if installed.

`agent-first-repo.md` loads its own detail files (progressive disclosure, mechanical enforcement, entropy). `context-layer.md` loads [references/context-layers.md](references/context-layers.md) only when the layer model is needed. DOX installs from [references/dox-framework.md](references/dox-framework.md).
