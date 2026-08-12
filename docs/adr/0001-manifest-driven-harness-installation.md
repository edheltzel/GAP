# ADR-0001: Use manifests and direct links instead of Stow

- Status: Accepted
- Date: 2026-08-12

## Context

Atlas Config currently mirrors harness runtime paths in top-level Stow packages. Shared skills are partly selected through `skills.txt`, but targeting rules also live in environment variables, per-entry exceptions, native skill trees, and harness packages. This makes ownership and installation behavior difficult to understand.

The configuration must support several coding harnesses without duplicating reusable behavior or redirecting runtime-generated state into the repository.

## Decision

Atlas Config will use these rules:

1. `harness.txt` is the only source of selected harness IDs.
2. Harness-specific source files live under `harnesses/<id>/`.
3. Every skill in `skills.txt` is universal. The installer asks skills.sh to install every listed skill globally for every selected harness that skills.sh supports.
4. Skills.sh exclusively owns its global skill store and harness skill links. Atlas does not repair or create skill links itself.
5. Reusable non-skill behavior is stored once. Small handwritten wrappers expose it in harness-native formats.
6. Harness runtime roots remain normal local directories. Atlas creates direct symlinks only for declared files or asset directories.
7. Existing targets are backed up before Atlas replaces them with managed links.
8. Atlas records the links it owns and removes stale recorded links during later installs. It never prunes unknown files or links.
9. `just install` is the primary reconciliation command.
10. A selected harness that is unavailable is skipped and reported. A known missing skills.sh target is also reported without failing the run. Unexpected install or linking errors produce a non-zero exit after the final summary.

## Consequences

### Benefits

- Harness selection has one clear source.
- Shared skills have one installation path.
- Harness runtime state, credentials, caches, and generated files stay outside Git.
- Adding a harness does not require a new Stow package.
- Managed links can be updated and removed safely without claiming unrelated files.

### Costs

- Atlas must maintain its own small reconciliation tool and install receipts.
- Each harness needs detection, runtime-path, link, and skills.sh capability metadata.
- Backups and receipts become machine-local state that must be tested carefully.
- Unsupported skills.sh targets cannot receive an explicit `-a` installation until skills.sh supports them.

## Rejected alternatives

- **Complete duplicated harness configs**: Simple to install, but reusable behavior drifts.
- **Generated complete harness bundles**: Reduces source duplication but adds templates and generation rules before they are needed.
- **Whole-runtime-directory symlinks**: Too likely to place credentials and runtime-generated state in Git.
- **Copying managed files**: Avoids repository coupling but makes source and installed files diverge.
- **Per-skill harness selectors**: Conflicts with the rule that `skills.txt` contains universal skills.
