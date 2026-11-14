# GAP as a plugin manager for agent harnesses

Date: 2026-11-12. Planning base: master `3d191c66`. Status: completed in v0.5.0. Approved by Ed 2026-11-12; prerequisite PR #114 (`feat/gap-utilities`) merged as `fbebb517`.

## Goal

Make GAP install, update, and remove one plugin (a utility) so it reaches every harness it supports, not only Claude and OMP. Keep the policy publisher as it is.

## Where it stands

On master: pinned components (`gap.toml`, `gap.lock`, `gap pin`), a verified store, skill install with drift checks and restore, user and project overrides, profiles, `gap doctor`.

PR #114 also shipped the [utility lifecycle](../../README.md#add-a-utility), [plugin manifest](../../README.md#publish-a-utility), and [hook trust](../../README.md#hooks-and-trust).

Delivered in v0.5.0:

| Capability | Outcome |
| --- | --- |
| Uninstall | Done: `gap uninstall` (step 5) |
| Per-harness delivery | Done: skills go to `~/.agents/skills` and `~/.claude/skills`; commands follow the [supported harness destinations](../../README.md#where-each-agent-reads-the-policy) |
| MCP servers and harness settings | Done: MCP delivery follows the [manifest contract](../../README.md#publish-a-utility); other harness settings remain installer-owned |
| Install without Go | Done: release binaries, and skills no longer need `npx` (steps 6 and 7) |
| Discovery | Done: `catalog.toml` and `gap search` (step 8) |

## Steps

1. Completed: merge `feat/gap-utilities` (PR #114, `fbebb517`). Owner: GAP API tab.
2. Completed: reordered the README with Change the policy before Manage skills and Components before the first `gap.toml` example. Project setup edge cases and the Echo pin note live in `cmd/gap/AGENTS.md`.
3. Completed for supported destinations: `cmd/gap/harness.go` drives policy, command, and MCP delivery. Skills use `~/.agents/skills` and `~/.claude/skills`; commands go to Claude, OMP, and Pi. Codex no longer reads custom prompts; Cursor and Empryo command folders remain unproven. Hooks and other harness settings stay installer-owned.
4. Completed: MCP entries in `gap-utility.toml` follow the [manifest and lifecycle contract](../../README.md#publish-a-utility) through the harness adapter table and the Claude, Codex, and Pi MCP CLIs.
5. Completed (PR #120): `gap uninstall`: strip the policy blocks, Codex hooks, published commands, and owned skills recorded in the ledger and `state.json`.
6. Completed (PR #120): release binaries (GoReleaser or `go build` in CI) and an installer that downloads one when Go is absent.
7. Completed: replaced the skills.sh path with [pinned components](../../README.md#components), including [adoption of matching installs](../../README.md#manage-skills). This also completes provisioner section 9 step 5. Branch `feat/drop-skillssh`.
8. Completed (PR #120): catalog, `gap search` over a list of known utility repos, kept in this repo.

Version ranges and dependencies between utilities wait until two utilities need each other.

## Documentation delivered

README: supported systems, uninstall steps, which harnesses get skills, a shorter Step 4 that links the sync contract, the symlink and Empryo wording fixes, the nessa-docs link. CONCEPTS: Distro terms (component, pin, store, drift, load level, override, profile) and the command ambiguity.

Website: pinned skills, MCP registration, utility lifecycle, release binary installation, and uninstall behavior.

## Todo

None. All eight steps are complete for the supported harness destinations above.
