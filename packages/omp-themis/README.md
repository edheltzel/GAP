# omp-themis

OMP-native Themis: a delegating project-manager persona packaged as an OMP extension.

## Why an OMP extension

The Claude Code Themis relied on Claude-specific agent frontmatter, slash-command prompts, and hook semantics. This package uses OMP-native surfaces instead:

- `omp.extensions` manifest entry pointing at `src/main.ts`.
- `/themis` extension command to activate/deactivate the persona.
- `before_agent_start` prompt injection while active.
- `tool_call` guardrails that prevent Themis from directly editing code.
- `themis_notify` custom tool for voice status through the local Atlas voice server (`localhost:8888`).
- Bundled `themis-pm` skill for on-demand operating guidance.

## Install

Local development:

```bash
omp install ./packages/omp-themis
```

One-shot test run:

```bash
omp --extension ./packages/omp-themis
```

If this repo is added as a marketplace source:

```bash
omp marketplace add <owner>/<repo>
omp install themis@atlas-config
```

Confirm loading:

```bash
omp -p '/extensions'
```

## Usage

```text
/themis
/themis implement the next item from .agents/atlas/plans/example.md
/themis status
/themis off
```

When active, Themis reports directly to Ed, delegates implementation, validates claims adversarially, and owns markdown/project-management artifacts under `.agents/atlas/`.
