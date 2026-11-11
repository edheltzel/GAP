# gap-utilities

Requirements: `docs/plans/2026-11-11-gap-utilities-requirements.md`.

## Outcome

Any GitHub repo that ships a manifest can be added, updated, synced, listed, and removed through `gap`, the way a lazy.nvim plugin is. Hooks need approval unless the repo or its owner is trusted. Do-Skills, Echo, Recall, and Forge use the same manifest path. README, CONCEPTS, contributor docs, and the website describe it.

## Choices made here (HOW)

These settle the requirements' open questions where they are implementation detail. Each can be revised before U1 starts.

- Manifest: `gap-utility.toml` at the utility repo root, parsed by `cmd/gap/toml.go`. Fields: `schema = "1"`, `name`, `description`, `skills` (names under `skills/`), `commands` (paths of `.md` files), and an optional `[hooks]` table with `install`, `update`, `uninstall`, `doctor`.
- A component spec is still `[components.<id>]` with `repo` and `ref`. `install`, `doctor`, and `names` come from the manifest. An optional `names` in a spec still narrows the skill list, and `[components.<id>.skills.<name>]` still sets load levels.
- User utilities: `~/.config/gap/gap.toml` may declare new `[components.<id>]` tables, and an id that collides with a distro id is an error. They are pinned in `~/.config/gap/gap.lock`, which uses the same schema as `gap.lock`.
- Trust: a `[trust]` table in `~/.config/gap/gap.toml` with `repos = ["owner/repo", "owner/*"]`.
- Approvals and ownership: `~/.gap/state.json` records, per id, the approved hook digest, the last commit the install hook ran for, and every skill directory and command file GAP copied. This extends the existing ledger in `cmd/gap/doctor.go`.
- `global/skills.json` stays as it is, out of scope here.
- The author guide is a README section, so there is no new published page.

## Implementation Units

### U1. Manifest reader
- Depends on: none
- Files: `cmd/gap/manifest.go` (new), `cmd/gap/manifest_test.go` (new)
- Change: `readUtilityManifest(dir)` parses `gap-utility.toml` into `{Name, Description, Skills, Commands, Hooks}`. It rejects an unknown schema, a missing or duplicate skill, a missing `skills/<name>/SKILL.md`, a command path outside the repo or not ending `.md`, and an unknown hook key. A repo with no manifest returns a clear "not a GAP utility" error.
- Done when: valid, minimal, and malformed fixtures produce the expected struct or error.
- Verify: `go test ./cmd/gap -run TestUtilityManifest`

### U2. Specs read from the pinned manifest
- Depends on: U1
- Files: `cmd/gap/components.go`, `cmd/gap/components_test.go`, `cmd/gap/selection.go`, `cmd/gap/provision.go`
- Change: `readDistro` no longer requires `names`. While the four distro pins predate their manifests, a spec's `install`, `doctor`, and `names` are still read as a fallback when the pinned store has no `gap-utility.toml`, and U11 deletes that fallback. After the store is verified (`ensureStore`), the component's skills, commands, and hooks load from that store's manifest. `componentEntries`, `syncGroups`, and `doctorAll` read hooks from there. An optional `names` narrows the list and must be a subset.
- Done when: a fixture component whose manifest lists skills and hooks syncs and doctors without `names`, `install`, or `doctor` in `gap.toml`.
- Verify: `go test ./cmd/gap -run 'TestComponent|TestDoctor|TestSyncSelection'`

### U3. User specs and the user lock
- Depends on: U2
- Files: `cmd/gap/override.go`, `cmd/gap/components.go`, `cmd/gap/override_test.go`
- Change: `mergeOverride` on `~/.config/gap/gap.toml` adds unknown ids as user components marked `Source: user`. Project and profile files still only override existing ids. `readPins` and `effectivePins` merge `~/.config/gap/gap.lock` for user ids. `lockFor` names `gap update <id>` for a missing user pin. The distro merge gate never covers user files.
- Done when: a user component syncs from the user lock, and a distro id collision errors.
- Verify: `go test ./cmd/gap -run 'TestOverride|TestUserComponent'`

### U4. Hook approval and trust
- Depends on: U2, U3
- Files: `cmd/gap/trust.go` (new), `cmd/gap/trust_test.go` (new), `cmd/gap/doctor.go`, `cmd/gap/components.go`
- Change: a hook that exits nonzero fails that component and makes the run exit `1`; a failed install does not record its commit, so the next sync retries it. Before any hook runs, `approveHook(id, repo, hook, command)` checks `[trust]` (an exact `owner/repo` or `owner/*` match), then the approved digest in `~/.gap/state.json`. Otherwise, on a terminal, it prints the command (and a diff against the last approved one) and asks `[y/N]`. With no terminal, it skips the hook and marks the run failed (exit `1`). The install hook runs only when the pinned commit differs from the last installed commit, so a second sync runs nothing.
- Done when: untrusted plus changed prompts; with no TTY it skips and exits 1; `owner/*` never prompts; an unchanged pin runs no hook.
- Verify: `go test ./cmd/gap -run TestHookTrust`

### U5. Utility commands publish
- Depends on: U2
- Files: `cmd/gap/main.go`, `cmd/gap/commands_test.go`
- Change: generalize `publishCommands` so a component's manifest `commands` publish to the same destinations as `global/commands`, recorded per id in the state ledger. A name clash between two sources errors.
- Done when: a fixture utility's command lands in `~/.claude/commands` and `~/.omp/agent/commands` and is recorded under its id.
- Verify: `go test ./cmd/gap -run TestCommandSync`

### U6. Ownership ledger for copied files
- Depends on: U4, U5
- Files: `cmd/gap/install.go`, `cmd/gap/doctor.go`, `cmd/gap/skills.go`
- Change: every skill directory and command file GAP copies for a component is recorded in `~/.gap/state.json` under that id, and existing entries are preserved (the same merge rule as the Echo adapter entry).
- Done when: after a sync, the ledger lists exactly the copied paths per id.
- Verify: `go test ./cmd/gap -run TestDoctorLedger`

### U7. `gap add`, `gap update`, `gap list`
- Depends on: U3, U4, U6
- Files: `cmd/gap/cli.go`, `cmd/gap/utility.go` (new), `cmd/gap/utility_test.go` (new), `cmd/gap/cli_test.go`
- Change: `add owner/repo[@ref]` (default id = repo basename lowercased, `--id` to override; default ref = the remote HEAD branch) fetches, reads the manifest, writes the spec to the user config and the pin to the user lock (both with `writeAtomic`), asks for hook approval, then installs that one component. `update [id]` resolves the ref again (`resolveRef`), rewrites the user lock, approves changed hooks, and runs `update` if present, otherwise `install`. A distro id points to `gap pin`. `list` prints id, source, ref, short commit, and trust state as a table (`style.go`).
- Done when: a fixture file:// utility goes add, list, update, with the user config and lock written and the distro files untouched.
- Verify: `go test ./cmd/gap -run 'TestUtilityAdd|TestUtilityUpdate|TestUtilityList|TestHelpListsCommands'`

### U8. `gap remove`
- Depends on: U6, U7
- Files: `cmd/gap/utility.go`, `cmd/gap/utility_test.go`, `cmd/gap/cli.go`
- Change: `remove id` (user ids only, since distro ids use `enabled = false`) runs an approved `uninstall` hook. It backs up each ledger-owned path whose content differs from the pinned store under `~/.gap/backups/`, moves every other owned skill dir and command file into the same backup run (a recoverable trash, per the requirements' no-`rm` constraint), drops the spec, lock entry, and ledger entry, and warns when `install` exists without `uninstall` unless the utility is trusted. A failing `uninstall` stops remove before anything moves. Removal moves files to backup or deletes only ledger-listed paths, never a path outside the ledger.
- Done when: after remove, nothing owned remains, an edited skill has a backup, the warning prints, and the second sync is a no-op.
- Verify: `go test ./cmd/gap -run TestUtilityRemove`

### U9. End-to-end lifecycle test
- Depends on: U8
- Files: `cmd/gap/utility_e2e_test.go` (new)
- Change: drive the built CLI against a temp HOME and a file:// fixture repo with a manifest, a skill, a command, and hooks that write marker files. Covers add, sync, sync again (no writes, no hook), update to a commit whose hook changed (no TTY: exit 1, hook skipped), trust `owner/*`, update again (no prompt), remove.
- Done when: every behavior Success line in the requirements has an assertion, including the backup of an edited skill and the install-without-uninstall warning (and its absence when trusted).
- Verify: `go test ./cmd/gap -run TestUtilityLifecycle` and `go test ./...`

### U10. Manifests in the four component repos
- Depends on: U1
- Files: `gap-utility.toml` in edheltzel/Do-Skills, edheltzel/Echo, edheltzel/Recall, and edheltzel/Forge (other repos, one PR each)
- Change: Do-Skills lists its skills. Echo gets `[hooks] install = "scripts/install.sh"` and `doctor = "cli/echo doctor"`. Recall gets `install = "./packaging/install.sh"` and `doctor = "recall doctor"`. Forge lists `forge-interview`. Each repo's tag or branch is merged first.
- Done when: each repo has the manifest on the ref GAP pins.
- Verify: after U7, for each clone: `HOME=$(mktemp -d) go run ./cmd/gap add file://<clone>` succeeds and `gap list` shows its skills.

### U11. Distro migration
- Depends on: U2, U10
- Files: `gap.toml`, `gap.lock`, `profiles/*.toml` if they name fields that moved
- Change: drop `install`, `doctor`, and `names` from the four components, keeping `do-astro`'s skills table, and delete the U2 spec fallback with its tests. Move each pin with `gap pin <id>@<ref>` to a commit that has the manifest.
- Done when: `gap.toml` has only repo, ref, and selections for the four, and `gap sync --yes` plus `gap doctor` on this machine report the same installed skills and doctor results as before (save the output before U11 and diff it after). The installers now run only on a pin change (U4), not on every sync.
- Verify: `go run ./cmd/gap status`, `go run ./cmd/gap doctor`, `go test ./...`

### U12. User and author docs
- Depends on: U9, U11
- Files: `README.md`, `CONCEPTS.md`
- Change: in README, add "Add a utility" (add, update, sync, list, remove, user config and lock, approval, `[trust]`) and "Publish a utility" (the full `gap-utility.toml` reference, hook semantics and exit codes, what `uninstall` must undo, and one copyable minimal manifest). Rewrite Components and Manage skills for the manifest model, and update the Commands list. In CONCEPTS, add Utility, Manifest, Hook, User lock, and Trust, and update Managed skill. Plain hyphens only.
- Done when: a person following only "Publish a utility" writes a manifest that `gap add` accepts (check with the U9 fixture copied from the README example).
- Verify: a test that extracts the README example manifest and runs `readUtilityManifest` on it (`go test ./cmd/gap -run TestReadmeManifestExample`)

### U13. Contributor docs
- Depends on: U12
- Files: `cmd/gap/AGENTS.md`, `AGENTS.md`
- Change: in `cmd/gap/AGENTS.md`, update Sync steps for manifest loading, the user lock, approvals, the ownership ledger, and remove. In the root `AGENTS.md`, update Publish and Edit wherever they describe components as hand-written in `gap.toml`. Leave `global/AGENTS.md` alone, since no shared rule changes.
- Done when: no doc still says `install`, `doctor`, or `names` live in the distro `gap.toml`.
- Verify: `rg -n 'names\s*=|install\s*=|doctor\s*=' AGENTS.md cmd/gap/AGENTS.md README.md` shows only manifest or example context

### U14. Website
- Depends on: U12
- Files: `site/src/site.js`, `site/src/pages/index.astro`, `site/public/_redirects`
- Change: the landing page gets a short "add any utility" block with a `gap add owner/repo` example and links to the new README anchors. `siteDescription` mentions utilities. Every README heading renamed or removed in U12 gets a `_redirects` line.
- Done when: the built landing page shows the example, and `/docs/` shows both new sections.
- Verify: from `site/`: `npm ci && npm run typecheck && npm run build && npm run links`

## Out of scope

- A registry, index, catalog, or search.
- A project-level add (`.gap/gap.toml` declaring new components).
- Installing harnesses.
- Changes to `global/skills.json` or the skills.sh path.
- A separate published page for the author guide.
- Moving the distro's own pins without `gap pin`.

## todo.txt
- Build GAP utilities from the plan with do-work +gap
- Ship gap-utility.toml in Do-Skills, Echo, Recall, and Forge (U10) +gap

Review complete
