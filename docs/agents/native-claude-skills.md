# Native Claude skills

Atlas Config tracks a curated native Claude skill snapshot under
`claude/.claude/skills/` and stows its repository-owned contents to `~/.claude/skills/`.
The retained custom agent lives at
`claude/.claude/agents/tldraw-offline.md`.

## Ownership

- The repository owns the 49 snapshot directories asserted by
  `tests/claude/native-skills-snapshot.test.sh`.
- Five snapshot directories (`recall-add`, `recall-dump`, `recall-recent`, `recall-scout`,
  and `recall-search`) preserve absolute pointers into Recall's shared installation. Stow
  ignores those directories because GNU Stow rejects absolute source links; Recall owns
  their runtime links under `~/.claude/skills/`.
- `skills.txt` separately owns external skills shared across harnesses. Top-level shared-skill
  symlinks must not be tracked or Stowed from `claude/.claude/skills/`; skills.sh creates the
  runtime links after package restoration succeeds.
- The snapshot is an exact recovery from the final LifeOS-removal lineage. Change an
  individual restored skill only as focused follow-up work with its own validation.

## Known runtime dependencies

LifeOS system code, `LIFEOS/VERSION`, and `LIFEOS/TOOLS` are gone. The local
`~/.claude/LIFEOS/{USER,MEMORY}/` data trees remain. These recovered skills still contain
references to removed LifeOS paths, commands, or environment assumptions and are not
certified standalone:

- `Agents`
- `Apify`
- `Art`
- `AudioEditor`
- `BitterPillEngineering`
- `CreateCLI`
- `Evals`
- `Fabric`
- `Hardening`
- `Ideate`
- `Loop`
- `Prompting`
- `Telos`
- `USMetrics`
- `WorldThreatModel`

Preserving those references is intentional. Adapt them one skill at a time rather than
performing a broad textual rewrite.

## External pointer reconciliation

The final cleanup lineage also contained 39 top-level external pointers. `skills.txt` now
owns the 38 recoverable skills and the five Recall runtime skills represented by ignored
snapshot directories. The former `anti-slop` pointer is intentionally retired: it targeted
removed absolute path `/Users/ed/Developer/AI/Skills/claude-anti-slop/anti-slop-plugin`,
while the current anti-slop implementation is a Pi-native package in PiExtensions. Do not
restore the dead absolute pointer as a substitute.

## Verification

Run:

```bash
bash tests/claude/native-skills-snapshot.test.sh
bash tests/claude/external-skills-inventory.test.sh
bash tests/stow-safety.test.sh
```

After merging into the canonical clone, run `just up` to restow the snapshot and provision
external skills. Do not restow from a disposable worktree. A complete live validation
requires every declared skill to resolve under `~/.claude/skills/` and zero dangling links
under the managed harness directories.
