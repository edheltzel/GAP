# pi-themis — Agent Instructions

This package is a Pi-native Themis implementation. It intentionally adapts the Claude Code persona to Pi APIs instead of copying Claude-specific hooks, agent metadata, or permissions.

## Scope

- `extensions/themis.ts` owns the `/themis` command, prompt injection, guardrails, persisted state, and `themis_notify` tool.
- `skills/themis-pm/SKILL.md` documents when to use Themis mode from Pi's skill system.
- `package.json` is the Pi package manifest; keep `pi.extensions` and `pi.skills` accurate.
- `README.md` is the install/usage surface for humans.

## Conventions

- Themis must remain a delegating project manager, not an implementation agent.
- Prefer Pi-native orchestration: `subagent` acceptance contracts by default, herdr only when the session is already inside herdr.
- Voice status targets the Atlas voice server on `localhost:8888` with `voice_id: "themis"`; never use Pulse port `31337`.
- Guardrails should block direct code mutation while active, but allow markdown and `.agents/**` project-management artifacts.
- Any meaningful behavior change should update both this package's README and the `themis-pm` skill so command behavior and agent guidance stay aligned.

## Verification

After changes, run the narrow checks that apply:

```bash
node -e "JSON.parse(require('fs').readFileSync('packages/pi-themis/package.json','utf8'))"
pi list | rg -n "pi-themis|atlas-config/packages/pi-themis"
pi -p "/themis status"
```
