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
- `skills.txt` separately owns other external skills shared across harnesses. Do not replace
  the native snapshot with manifest links or copy manifest-managed skills into it.
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

## Verification

Run:

```bash
bash tests/claude/native-skills-snapshot.test.sh
```

After merging into the canonical clone, run `just claude` to restow the snapshot. Do not
restow from a disposable worktree.
