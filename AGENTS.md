# AGENTS.md

This repo publishes GAP, the global agent policy. `global/AGENTS.md` is that contract. Depth docs live in `global/docs/agents/`.

This file only covers how to change the repo. Shared behavior rules are in `global/AGENTS.md`.

## Publish

`npm run sync` copies `global/AGENTS.md` into the files these harnesses already read:

- `~/.claude/rules/global-agent-policy.md`
- `~/.pi/agent/APPEND_SYSTEM.md`
- `~/.omp/agent/APPEND_SYSTEM.md`
- `~/.codex/gap/global-agent-policy.md` (`CODEX_HOME` when set). A SessionStart hook prints that file. Sync leaves old policy text in `~/.codex/AGENTS.md`, so Codex loads the policy twice. That is expected. Do not run `npm run sync -- --remove-codex-legacy` yet: SessionStart does not reach Codex subagents, which still need that text. The flag also refuses in the run that first installs the hook. Empty `CODEX_HOME` means unset. A relative `CODEX_HOME` is refused.
- `~/.cursor/rules/global-agent-policy.mdc`

The script writes regular files. Do not symlink those paths. Do not add Stow. Publish only from the primary branch (`origin/HEAD`, `master` here), not from another branch or a worktree. A destination that is not the current publish is moved to `~/.gap/backups/` after a yes. No prompt means that file is left alone and the other destinations still publish. Empty `CODEX_HOME` means unset. A relative `CODEX_HOME` is refused. Grok already loads `~/.claude/rules/global-agent-policy.md` when Claude compatibility is on. Confirm with `grok inspect` before adding a Grok destination. A second copy loads the policy twice. The human note is in `README.md`.
Do not point `~/CLAUDE.md` at the shared file. An ancestor `CLAUDE.md` suppresses that project's `AGENTS.md`.

After editing shared markdown, run `npm run sync`. A copy is stale until that command runs.

## Edit

- Change a shared rule in `global/`, then sync. Do not paste the rule into a harness home by hand.
- Add a harness destination in `src/sync.ts` only when that harness actually loads the path. Sync writes that payload. If nothing reads it, the copy is dead.
- `docs/solutions/` holds documented solutions to past problems, by category, with YAML frontmatter (`module`, `tags`, `problem_type`). `CONCEPTS.md` is the shared domain vocabulary. Relevant when implementing or debugging in a documented area, or when a domain word is unclear.
- Node 24 or newer. No extra runtime dependencies.
