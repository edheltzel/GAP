# omp-themis — DOX Instructions

## Purpose

OMP-native Themis implementation packaged as an installable OMP extension.

## Ownership

- `src/main.ts` owns `/themis`, prompt injection, guardrails, persisted state, and `themis_notify`.
- `skills/themis-pm/SKILL.md` owns OMP skill guidance for Themis mode.
- `package.json` owns OMP package metadata; keep `omp.extensions` accurate.
- `README.md` owns human install and usage instructions.

## Local Contracts

- Themis must remain a delegating project manager, not an implementation agent.
- Use OMP-native orchestration: `task`, `job`, `irc`, `todo`, `lsp`, and OMP extension hooks/tools.
- Host harness (OMP) is the default dispatch surface. External harnesses (claude, pi, codex, …) require Ed's confirmation through a mandatory once-per-session cross-vendor bridge check-in; reviewers add a bridge review pass and explorer pools mix host-native and bridge-routed model families.
- Herdr tabs hosting dispatched agents are labeled `GH#<issue>-<role>` (e.g. `GH#21-worker`) when the work is tied to an issue number; otherwise a short descriptive label.
- Voice status targets the Atlas voice server on `localhost:8888`; never use Pulse port `31337`.
- Guardrails should block direct code mutation while active, but allow markdown and `.agents/**` project-management artifacts.
- When Themis starts work from a handoff, she immediately moves it to the handoff archive selected by the nearest repository contract; the default is `docs/handoffs/archive/`.
- Any meaningful behavior change must update both README and `skills/themis-pm/SKILL.md`.

## Work Guidance

- Do not copy Claude Code hook semantics into this package; adapt behavior to OMP APIs.
- Keep the public command surface small: `/themis`, `/themis status`, and deactivation aliases.
- Prefer explicit acceptance criteria and verification commands in delegated implementation briefs.

## Verification

After changes, run the narrow checks that apply:

```bash
bun --check packages/omp-themis/src/main.ts
omp --extension ./packages/omp-themis -p '/themis status'
```

Also parse `packages/omp-themis/package.json` after manifest edits.

## Child DOX Index

No child AGENTS.md files yet.
