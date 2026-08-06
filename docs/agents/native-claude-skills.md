# Native Claude skills

Atlas Config tracks curated native Claude skills under `claude/.claude/skills/` and
stows its repository-owned contents to `~/.claude/skills/`.

## Ownership

- The repository owns the 48 native skill directories asserted by
  `tests/claude/native-skills-snapshot.test.sh`.
- Five snapshot directories (`recall-add`, `recall-dump`, `recall-recent`, `recall-scout`,
  and `recall-search`) preserve absolute pointers into Recall's shared installation. Stow
  ignores those directories because GNU Stow rejects absolute source links; Recall owns
  their runtime links under `~/.claude/skills/`.
- `skills.txt` separately owns external skills with explicit per-entry harness targets when
  ownership differs. Top-level external-skill symlinks must not be tracked or Stowed from
  `claude/.claude/skills/`; skills.sh creates runtime installations after restoration.
- The native inventory was recovered from the final LifeOS-removal lineage. Recovered
  skills are maintained here and require focused validation when their behavior changes.
- A sibling Firstmate checkout owns its 19 internal skills under `.agents/skills/` and its
  `.claude/skills -> ../.agents/skills` compatibility link. Their `metadata.internal: true`
  marker intentionally keeps them out of Config and `skills.txt`. `just ownership-status`
  requires that checkout and fails if it is absent or incomplete.

## tldraw ownership

tldraw Desktop owns its installed skill, agent, commands, hooks, and bundled tools. Version
1.12.1 installs them through the home-screen `Install agent skills` button or the
`Develop > Install Agent Skills` menu. Atlas Config no longer tracks duplicate tldraw files.

The app records declared files and hooks in
`~/Library/Application Support/tldraw/agent-skills.json`. `just ownership-status` requires
version 1.12.1's exact 12-file and two-hook inventory, ownership markers, matcher and command
types, and confirms that no declared runtime artifact resolves into Atlas Config.

## Runtime boundaries

LifeOS system code, `LIFEOS/VERSION`, `LIFEOS/TOOLS`, filesystem MCP paths, and retired
task APIs are gone. The native snapshot test rejects operational references to those
contracts. Fabric retains one clearly historical changelog note about its original source.

The local `~/.claude/LIFEOS/{USER,MEMORY}/` data trees remain optional user data. A skill
must check that retained input exists, treat it as untrusted context, and fail closed when
required data is absent. It may write there only when its workflow explicitly owns a
user-approved artifact and path. Current examples include optional Agents custom traits,
team expertise, TELOS documents, and WorldThreatModel model files.

Repository-owned native tool dependencies install beside the real source files through
`install-native-skill-tools.sh`. The locked package directories are `Agents/Tools`, `Apify`,
`Art/Tools`, `Evals`, and `Prompting/Templates/Tools`. Checks require Bun and jq, verify the
frozen lockfile, and reject installed versions outside declared ranges. Never install
dependencies through the Stowed runtime links. Stow ignores every nested `node_modules`
path, while `prune-native-dependency-links` removes only dangling links hidden under those
ignored runtime paths.

## External pointer reconciliation

The final cleanup lineage also contained 39 top-level external pointers. `skills.txt` owns
every recoverable pointer and the Recall runtime skills represented by ignored native
directories. The former `anti-slop` pointer targeted removed absolute path
`/Users/ed/Developer/AI/Skills/claude-anti-slop/anti-slop-plugin`; never restore it.

Its replacement is explicit and split by harness. Five pinned `local:../PiExtensions`
entries install the current anti-slop suite only into Claude Code and verify each Git blob.
Pi discovers the same names only through PiExtensions' registered `pi-anti-slop` package,
so skills.sh must not target Pi for those entries.

## Verification

Run:

```bash
just native-skill-tools-status
bash tests/claude/native-skills-snapshot.test.sh
bash tests/claude/native-orchestration-skills.test.sh
bash tests/claude/native-residual-skills.test.sh
bash tests/claude/native-runtime-skills.test.sh
bash tests/claude/native-browser-skills.test.sh
bash tests/claude/native-continuity-skills.test.sh
bash tests/claude/external-skills-inventory.test.sh
bash tests/ownership-status.test.sh
bash tests/installed-skills-status.test.sh
bash tests/stow-safety.test.sh
just installed-skills-status
just ownership-status
```

After merging into the canonical clone, run `just up` to restow native skills, provision
native tool dependencies, and install external skills. Do not restow from a disposable
worktree. A complete live validation requires every declared skill to resolve under
`~/.claude/skills/` and zero dangling links under the managed harness directories.
