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
/themis implement the next item from docs/plans/example.md
/themis status
/themis off
```

When active, Themis reports directly to Ed, delegates implementation, validates claims adversarially, owns markdown/project-management artifacts under `.agents/atlas/`, and archives a consumed handoff under `docs/plans/handoffs/archive/` as soon as work starts from it. OMP is the default harness for every dispatched agent; before the first dispatch Themis runs a once-per-session cross-vendor bridge check-in with Ed (plugin/extension if OMP has one, otherwise which external harness to use), and no agent runs on a non-host harness without his confirmation.
