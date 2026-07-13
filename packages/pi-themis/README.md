# pi-themis

Pi-native Themis: a delegating project-manager persona for Pi, packaged as a Pi extension.

## Why an extension

The Claude Code version relied on Claude-specific agent metadata and hook mechanics. This package uses Pi-native surfaces instead:

- `/themis` extension command to activate/deactivate the persona.
- `before_agent_start` prompt injection while active.
- `tool_call` guardrails that prevent Themis from editing code directly.
- `themis_notify` custom tool for voice status through the local Atlas voice server (`localhost:8888`).
- Pi package manifest so it can be installed locally, from git, or published to npm.

## Install

Local development:

```bash
pi install /Users/ed/Developer/atlas-config/packages/pi-themis
```

Temporary test run:

```bash
pi -e /Users/ed/Developer/atlas-config/packages/pi-themis
```

If published later:

```bash
pi install npm:pi-themis
```

## Usage

```text
/themis
/themis implement the next item from .agents/atlas/plans/example.md
/themis status
/themis off
```

When active, Themis reports directly to Ed, delegates implementation, validates claims adversarially, and owns markdown/project-management artifacts under `.agents/atlas/`. Pi is the default harness for every dispatched agent; before the first dispatch Themis runs a once-per-session cross-vendor bridge check-in with Ed (plugin/extension if Pi has one, otherwise which external harness to use), and no agent runs on a non-host harness without his confirmation.
