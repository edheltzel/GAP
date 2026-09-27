# AGENTS.md

This repo publishes GAP, the global agent policy. `global/AGENTS.md` is that contract. Depth docs live in `global/docs/agents/`.

This file only covers how to change the repo. Shared behavior rules are in `global/AGENTS.md`.

## Publish

`npm run sync` copies `global/AGENTS.md` into the files these harnesses already read:

- `~/.claude/rules/global-agent-policy.md`
- `~/.pi/agent/APPEND_SYSTEM.md`
- `~/.omp/agent/APPEND_SYSTEM.md`
- `~/.codex/AGENTS.md` (`CODEX_HOME` when set)
- `~/.cursor/rules/global-agent-policy.mdc`

The script writes regular files. Do not symlink those paths. Do not add Stow. A destination that is not the current publish is moved to `~/.gap/backups/` after a yes. No prompt means that file is left alone. Grok already loads `~/.claude/rules/global-agent-policy.md` when Claude compatibility is on. Confirm with `grok inspect` before adding a Grok destination. A second copy loads the policy twice. The human note is in `README.md`.
Do not point `~/CLAUDE.md` at the shared file. An ancestor `CLAUDE.md` suppresses that project's `AGENTS.md`.

After editing shared markdown, run `npm run sync`. A copy is stale until that command runs.

## Edit

- Change a shared rule in `global/`, then sync. Do not paste the rule into a harness home by hand.
- Add a harness destination in `src/sync.ts` only when that harness actually loads the path. Sync writes that payload. If nothing reads it, the copy is dead.
- Node 24 or newer. No extra runtime dependencies.
