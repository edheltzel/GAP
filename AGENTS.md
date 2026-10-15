# AGENTS.md

This repo publishes GAP, the global agent policy. `global/AGENTS.md` is that contract. Depth docs live in `global/docs/agents/`.

This file only covers how to change the repo. Shared behavior rules are in `global/AGENTS.md`.

## Publish

`gap sync` copies `global/AGENTS.md` into the files these harnesses already read. Install it once with `go install ./cmd/gap`; it then runs from any directory and always publishes the checkout it was built from. Reinstall after `cmd/gap` changes or the checkout moves. `go run ./cmd/gap sync` from the repo root works without installing.

- `~/.claude/rules/global-agent-policy.md`
- `~/.pi/agent/APPEND_SYSTEM.md`
- `~/.omp/agent/APPEND_SYSTEM.md`
- `~/.codex/gap/global-agent-policy.md` (`CODEX_HOME` when set). SessionStart and SubagentStart hooks print that file. Sync leaves old policy text in `~/.codex/AGENTS.md`, so Codex loads the policy twice until `gap sync --remove-codex-legacy`. Run the flag only after both hooks are trusted in `/hooks`; it refuses unless both were in `hooks.json` before that run. Empty `CODEX_HOME` means unset. A relative `CODEX_HOME` is refused.
- `~/.cursor/rules/global-agent-policy.mdc`

The CLI writes regular files. Do not symlink those paths. Do not add Stow. Publish only from the primary branch (`origin/HEAD`, `master` here) or GitButler's `gitbutler/workspace`, not from another branch or a worktree. A destination that is not the current publish is moved to `~/.gap/backups/` after a yes. No prompt means that file is left alone and the other destinations still publish. Empty `CODEX_HOME` means unset. A relative `CODEX_HOME` is refused. Grok already loads `~/.claude/rules/global-agent-policy.md` when Claude compatibility is on. Confirm with `grok inspect` before adding a Grok destination. A second copy loads the policy twice. The human note is in `README.md`.
Do not point `~/CLAUDE.md` at the shared file. An ancestor `CLAUDE.md` suppresses that project's `AGENTS.md`.

After a shared markdown change is merged and pulled, run `gap sync`. Sync refuses while `global/` or `skills/` differs from `origin/HEAD`. A copy is stale until that command runs.

## Edit

- Change a shared rule in `global/`, then sync. Do not paste the rule into a harness home by hand.
- Add a harness destination in `cmd/gap/main.go` only when that harness actually loads the path. Sync writes that payload. If nothing reads it, the copy is dead.
- Skills live in `skills/<scope>/<group>/<skill>/`. List a skill in `global/skills.json` to have `gap sync` install it and check it for drift. After adding, moving, or renaming a skill, run `bash scripts/update-readme.sh` to regenerate the catalog.
- `docs/solutions/` holds documented solutions to past problems, by category, with YAML frontmatter (`module`, `tags`, `problem_type`). `CONCEPTS.md` is the shared domain vocabulary. Relevant when implementing or debugging in a documented area, or when a domain word is unclear.
- Go 1.24 or newer, standard library only. The one runtime exception: the skills step runs skills.sh through `npx`, so it needs Node. `go test ./...` is the self-test.
