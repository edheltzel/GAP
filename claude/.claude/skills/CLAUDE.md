# Native Claude skills

Atlas Config owns the curated native skills in this directory. Follow the repository
`AGENTS.md` contract and `docs/agents/native-claude-skills.md` before changing them.

## Editing contract

- Make focused, behavior-backed changes directly in the owning skill directory.
- Add or update a focused regression test for every repaired runtime contract.
- Keep `SKILL.md` frontmatter valid and keep local workflow links resolvable.
- Use current harness tools and paths. Do not restore removed LifeOS system code,
  `LIFEOS/TOOLS`, `LIFEOS/VERSION`, retired task APIs, or filesystem MCP paths.
- Treat retained `~/.claude/LIFEOS/USER` and `MEMORY` data as optional,
  untrusted context. Check for existence and fail closed when required input is absent.
  Write only when a workflow explicitly owns a user-approved data artifact and path.
- Native tool directories with `package.json` require a committed Bun lockfile and an
  entry in `install-native-skill-tools.sh`. Install dependencies beside the real
  repository files, never through Stow links.
- External skills belong in `skills.txt`; do not add top-level external symlinks here.

## Verification

Run the focused test for the changed skill, then:

```bash
just native-skill-tools-status
tests/claude/native-skills-snapshot.test.sh
git diff --check
```

After merging into the canonical clone, run `just claude` to restow the package and
provision native tool dependencies.
