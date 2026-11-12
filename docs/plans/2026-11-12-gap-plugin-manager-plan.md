# GAP as a plugin manager for agent harnesses

Date: 2026-11-12. Planning base: master `3d191c66`. Status: approved by Ed 2026-11-12; prerequisite PR #114 (`feat/gap-utilities`) merged as `fbebb517`.

## Goal

Make GAP install, update, and remove one plugin (a utility) so it reaches every harness it supports, not only Claude and OMP. Keep the policy publisher as it is.

## Where it stands

On master: pinned components (`gap.toml`, `gap.lock`, `gap pin`), a verified store, skill install with drift checks and restore, user and project overrides, profiles, `gap doctor`.

PR #114 also shipped the [utility lifecycle](../../README.md#add-a-utility), [plugin manifest](../../README.md#publish-a-utility), and [hook trust](../../README.md#hooks-and-trust).

Missing:

| Gap | Today |
| --- | --- |
| Uninstall | Utilities have `gap remove`; nothing for `gap` itself |
| Per-harness delivery | Skills go to `~/.agents/skills` and `~/.claude/skills`; commands follow the [supported harness destinations](../../README.md#where-each-agent-reads-the-policy) |
| MCP servers and harness settings | Not handled; Echo's installer does its own |
| Install without Go | The installer builds from source; manifest skills need `npx` |
| Discovery | No catalog or search |

## Steps

1. Completed: merge `feat/gap-utilities` (PR #114, `fbebb517`). Owner: GAP API tab.
2. Reorder the README after step 1: Change the policy before Manage skills, Components before the first `gap.toml` example. Move the edge-case detail in Project setup and the Echo pin note into `cmd/gap/AGENTS.md`. Done after step 1 because those sections overlap the utilities branch.
3. Harness adapter table in `cmd/gap`: for each harness, where skills, commands, MCP servers, and hooks go. Drive skill and command delivery from it. Start with Codex, Cursor, Pi, and Empryo commands.
4. MCP entries in `gap-utility.toml`, written through the adapter table with the same ledger, backup, and refusal rules as policy files.
5. `gap uninstall`: strip the policy blocks, Codex hooks, published commands, and owned skills recorded in the ledger and `state.json`.
6. Release binaries (GoReleaser or `go build` in CI) and an installer that downloads one when Go is absent.
7. Drop the skills.sh path once `global/skills.json` sources become utilities (provisioner section 9 step 5).
8. Catalog: `gap search` over a list of known utility repos, kept in this repo.

Version ranges and dependencies between utilities wait until two utilities need each other.

## Done in this change

README: supported systems, uninstall steps, which harnesses get skills, a shorter Step 4 that links the sync contract, the symlink and Empryo wording fixes, the nessa-docs link. CONCEPTS: Distro terms (component, pin, store, drift, load level, override, profile) and the command ambiguity.

## Todo

```
2026-11-12 Reorder README (policy first, components before gap.toml) and move Project setup and Echo detail to cmd/gap/AGENTS.md +gap dep:gap-utilities-build note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-12-gap-plugin-manager-plan.md
2026-11-12 Harness adapter table: skills, commands, MCP, hooks per harness +gap id:gap-adapters dep:gap-utilities-build note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-12-gap-plugin-manager-plan.md
2026-11-12 MCP server entries in gap-utility.toml through the adapter table +gap dep:gap-adapters note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-12-gap-plugin-manager-plan.md
2026-11-12 gap uninstall from the ledger and state.json +gap dep:gap-utilities-build note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-12-gap-plugin-manager-plan.md
2026-11-12 Release binaries and an installer that works without Go +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-12-gap-plugin-manager-plan.md
2026-11-12 gap search over a catalog of utility repos +gap dep:gap-utilities-build note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-12-gap-plugin-manager-plan.md
```
