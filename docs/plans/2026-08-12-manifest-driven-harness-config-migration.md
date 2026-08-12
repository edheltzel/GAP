# Manifest-driven harness config migration

## Goal

Replace Stow with one predictable installation flow for Claude Code, Pi, OMP, and Jcode. Keep harness-native config separate, install universal skills through skills.sh, and store reusable non-skill behavior once.

The governing decision is [ADR-0001](../adr/0001-manifest-driven-harness-installation.md). Domain terms are defined in [`CONTEXT.md`](../../CONTEXT.md).

## Requirements

- `harness.txt` contains one selected short harness ID per line.
- Initial IDs are `claude-code`, `pi`, `omp`, and `jcode`.
- `harness.txt` is the only harness-selection input. Remove normal-install environment overrides and per-skill target selectors.
- Every entry in `skills.txt` is a global skill intended for every selected harness.
- Skills.sh owns `~/.agents/skills` and all harness skill installation surfaces.
- Harness-native config lives under `harnesses/<id>/`.
- Shared non-skill assets live outside harness folders and are exposed through small handwritten wrappers.
- Harness runtime roots remain real local directories.
- Atlas links only declared files or asset directories.
- Existing targets are backed up before replacement.
- Stale links are pruned only when a machine-local receipt proves Atlas created them.
- Unavailable harnesses and known unsupported skills.sh targets produce end-of-run warnings.
- Unexpected skills.sh, backup, link, receipt, or prune failures make installation exit non-zero.
- `just install` is the primary apply command.

## Non-goals

- Do not build a general template engine for wrappers.
- Do not link an entire harness runtime directory.
- Do not make runtime state, credentials, sessions, caches, or generated files repository-owned.
- Do not patch skills.sh to add unsupported harnesses.
- Do not silently treat a different skills.sh agent ID as OMP or Jcode without verification.

## Proposed layout

```text
.
├── harness.txt
├── skills.txt
├── harnesses/
│   ├── claude-code/
│   │   ├── harness.json
│   │   ├── links.txt
│   │   └── files/
│   ├── pi/
│   │   ├── harness.json
│   │   ├── links.txt
│   │   └── files/
│   ├── omp/
│   │   ├── harness.json
│   │   ├── links.txt
│   │   └── files/
│   └── jcode/
│       ├── harness.json
│       ├── links.txt
│       └── files/
├── shared/
│   ├── commands/
│   ├── agents/
│   └── personas/
├── scripts/
│   └── atlas-config.ts
└── tests/
```

`files/` stores harness-native source files without mirroring credentials or runtime state. `links.txt` explicitly maps each source path to a path relative to the harness runtime root. `harness.json` stores data, not executable shell fragments.

Suggested harness metadata:

```json
{
  "id": "pi",
  "displayName": "Pi",
  "detectCommands": ["pi"],
  "detectPaths": ["~/.pi/agent"],
  "runtimeRoot": "~/.pi/agent",
  "skillsAgent": "pi"
}
```

Use `null` for an unverified or unsupported `skillsAgent`. Do not guess a mapping. The installer reports it in the final warning section.

Suggested link format:

```text
# source path under files/ | destination path under runtimeRoot
settings.json | settings.json
extensions/atlas-status-line.ts | extensions/atlas-status-line.ts
```

Reject absolute paths, parent traversal, duplicate destinations, and sources outside the harness folder.

## Install behavior

`just install` calls the repository reconciler, which runs these steps in order.

1. Parse and validate `harness.txt`.
2. Load the matching `harnesses/<id>/harness.json` files.
3. Detect which selected harnesses are available.
4. Parse `skills.txt` and reject per-skill harness selectors.
5. Resolve supported skills.sh agent IDs from available selected harness metadata.
6. Install each selected skill source globally with explicit repeated `-a <skillsAgent>` and `-s <skill>` arguments.
7. Validate the global skill and supported harness installation surfaces using `skills list -g --json` where possible.
8. Parse each eligible harness's `links.txt`.
9. For each declared destination:
   - create missing parent directories;
   - leave an already-correct Atlas link unchanged;
   - move any other existing target into a timestamped backup tree;
   - create a direct symlink to the repository source;
   - record the source, destination, and link target in the new receipt.
10. Compare the previous receipt with the new desired links. Remove a stale destination only when it is still the exact Atlas-owned symlink recorded previously.
11. Write receipts atomically after successful reconciliation for that harness.
12. Print one summary containing installed skills, linked paths, backups, pruned links, skipped harnesses, unsupported capabilities, and real errors.
13. Exit non-zero when real errors occurred. Known unavailable harnesses and unsupported skills.sh targets remain warnings.

## Machine-local state

Keep backups and receipts outside the repository and harness runtime roots:

```text
${XDG_STATE_HOME:-~/.local/state}/atlas-config/
├── receipts/<harness-id>.json
└── backups/<timestamp>/<harness-id>/...
```

Receipts must be written through a temporary file followed by an atomic rename. A receipt may authorize removal only of the exact destination and link target it records.

Provide commands to inspect state and restore backups before removing Stow support:

```bash
just status
just backups
just restore <timestamp> <harness-id>
```

`just status` is read-only. It reports selected, available, eligible, linked, stale, unsupported, and conflicting state.

## Skills migration

1. Simplify `skills.txt` to source and skill selection only.
2. Remove `|agents=...` entries and the `SKILLS_AGENTS` normal-install override.
3. Preserve immutable source and integrity verification where it remains useful.
4. Inventory every currently active skill surface:
   - manifest-installed skills;
   - native Claude skills;
   - Pi package skills;
   - OMP package or compatibility-loaded skills;
   - Recall-owned and other external skill stores.
5. For each skill, choose one source repository and one `skills.txt` entry.
6. Remove duplicate harness-owned copies only after skills.sh installation and runtime discovery are verified.
7. Resolve the current anti-slop split before broad installation. Skills in the universal manifest cannot remain Claude-only, and Pi must not receive two owners for the same skill.
8. Reconcile `skills-lock.json` with the installed skills.sh version. Either generate and verify it as authoritative or rename/document it as non-authoritative state.
9. Treat OMP and Jcode skills.sh mappings as unsupported until exact agent IDs are proven. Their native ability to discover `~/.agents/skills` may be documented, but it does not justify inventing a `-a` value.

## Harness migration

### Claude Code

- Move curated source from `claude/.claude/` into `harnesses/claude-code/files/`.
- Exclude credentials, sessions, runtime state, caches, backups, and installed skills.
- Replace reusable command bodies with small wrappers where a shared asset or global skill owns the behavior.
- Preserve native hooks as Claude-owned files.

### Pi

- Move curated source from `pi/.pi/agent/` into `harnesses/pi/files/`.
- Preserve Pi-only extensions, theme, settings, prompts, and status-line behavior.
- Do not modify or absorb the pre-existing local `pi/.pi/agent/settings.json` change or its untracked backup files without a separate ownership review.
- Remove package-owned skill duplication only after skills.sh parity is verified.

### OMP

- Move curated source from `omp/.omp/agent/` into `harnesses/omp/files/`.
- Keep OMP plugin installation as an OMP harness capability until it can be represented safely in the main reconciler.
- Preserve machine-local databases, generated integrations, secrets, sessions, caches, and plugin registry data.
- Verify whether skills.sh has an exact OMP target. Warn if it does not.

### Jcode

- Move curated source from `jcode/.jcode/` into `harnesses/jcode/files/`.
- Preserve the clean-filter behavior needed for app-rewritten config or replace it with an equivalent normalization check.
- Keep credentials, sessions, memory, caches, generated assets, and machine-specific output local.
- Warn that skills.sh has no verified Jcode target.

## Shared assets and wrappers

1. Inventory commands, agents, and personas across all harnesses.
2. Move behavior used by more than one harness into `shared/`.
3. Keep wrappers handwritten and minimal. A wrapper should name or load the shared asset without copying its body.
4. If two harnesses accept the same file format and semantics, they may link the same shared file directly.
5. If a shared behavior is naturally an Agent Skill, prefer making it a global skill and keep only friendly command wrappers.
6. Add a parity test that fails when a wrapper points to a missing shared asset or skill.

## Implementation phases

### Phase 1: Build the read-only model

- Add `harness.txt` with all four IDs.
- Add harness metadata and link manifests without moving files.
- Implement `just status` and manifest validation.
- Confirm exact runtime roots, detection rules, and skills.sh mappings from installed versions.
- Add tests for duplicate IDs, unknown IDs, unsafe paths, duplicate destinations, malformed skill entries, and unsupported capabilities.

Exit criterion: status can describe the desired migration without changing the machine.

### Phase 2: Implement safe link reconciliation

- Implement backup, link, receipt, stale-link, and atomic-write behavior in isolated temporary homes.
- Add interruption and rerun tests.
- Verify that unknown files and links are never removed.
- Add restore and backup-list commands.

Exit criterion: repeated installs are idempotent, conflicts are recoverable, and stale cleanup is ownership-safe.

### Phase 3: Integrate skills.sh

- Refactor `install-skills.sh` or replace it with the reconciler's skills module.
- Derive every `-a` argument only from selected harness metadata.
- Remove alternate target inputs and per-entry exceptions.
- Preserve source pinning and integrity checks.
- Test command construction without paid-model calls or destructive global writes.

Exit criterion: all supported selected harnesses receive explicit skills.sh targets, while unsupported targets appear only as warnings.

### Phase 4: Move harness source trees

- Move one harness at a time into `harnesses/<id>/files/`.
- Populate explicit link manifests.
- Run focused runtime and config validation after each move.
- Preserve unrelated and machine-local changes.
- Keep compatibility commands temporarily while the old Stow paths still exist.

Exit criterion: all four harnesses install through managed direct links and retain their current behavior.

### Phase 5: Consolidate shared assets

- Move reusable behavior into `shared/` or global skills.
- Replace duplicated native bodies with thin wrappers.
- Add cross-harness parity checks.
- Resolve duplicate skill ownership before deleting old copies.

Exit criterion: shared behavior has one source, and each harness has only native settings, hooks, extensions, and wrappers.

### Phase 6: Remove Stow

- Run the new installer and verify receipts and backups.
- Remove Stow recipes, safety scripts, package ignore files, and Stow-specific documentation.
- Rename command groups in `justfile` around install, status, restore, and verification.
- Update root `AGENTS.md`, child DOX files, README, and harness-specific docs to the implemented contract.
- Keep a tested migration note for restoring the pre-migration backup.

Exit criterion: no install, validation, or documentation path depends on Stow.

## Verification

Automated checks must cover:

- manifest parsing and path safety;
- harness detection and selection;
- exact skills.sh argument construction;
- warning versus error exit behavior;
- conflict backup and restore;
- idempotent links;
- atomic receipts;
- stale recorded-link pruning;
- refusal to prune unknown or changed targets;
- universal skill inventory and missing bundled references;
- shared-asset and wrapper parity;
- harness config parsing and existing focused status-line tests;
- `git diff --check` and Pi Lens diagnostics for changed source files.

Run tests against temporary home and state directories. Do not mutate live harness config during unit or integration tests.

## Completion criteria

- `just install` reconciles all four selected harnesses without Stow.
- `harness.txt` is the only harness selector.
- Every `skills.txt` entry is installed globally with every verified selected skills.sh target.
- Unsupported OMP or Jcode targeting is summarized as a warning, not hidden.
- Real failures return a non-zero status.
- Runtime roots remain local directories.
- Conflicts are backed up and restorable.
- Only receipt-owned stale links are pruned.
- Skills.sh exclusively owns global and harness skill installs.
- Shared non-skill behavior has one source and small native wrappers.
- No credentials, sessions, caches, generated state, or unrelated user files enter Git.
- Existing harness behavior and focused tests remain green.
- All Stow code and documentation are removed only after the new path is verified.
