# AGENTS.md

This repo publishes GAP, the global agent policy. `global/AGENTS.md` is that contract. Depth docs live in `global/docs/agents/`. `global/AGENTS.md` is the published policy, not a DOX child. Do not add repo workflow to it.

This file covers how to change the repo. Shared behavior rules are in `global/AGENTS.md`. DOX rules are in `global/docs/agents/dox.md`. Before editing, read this file, then every AGENTS.md on the path to the files you touch. Closer docs win on local detail. No child may weaken DOX.

## Publish

`gap sync` writes `global/AGENTS.md` between `<!-- gap:begin -->` and `<!-- gap:end -->` in the files these harnesses already read, publishes each `global/commands/*.md` unchanged into the command files those harnesses already load, then installs and checks the skills in `global/skills.json`. Sync steps, the ledger, backups, source-identity migration, and drift choices are in `cmd/gap/AGENTS.md`. Install it once with `go install ./cmd/gap`; it then runs from any directory and always publishes the checkout it was built from. Reinstall after `cmd/gap` changes or the checkout moves. `go run ./cmd/gap sync` from the repo root works without installing.

- `~/.claude/rules/global-agent-policy.md`
- `~/.pi/agent/APPEND_SYSTEM.md`
- `~/.omp/agent/APPEND_SYSTEM.md`
- `~/.codex/gap/global-agent-policy.md` (`CODEX_HOME` when set). SessionStart and SubagentStart hooks print that file. Sync leaves old policy text in `~/.codex/AGENTS.md`, so Codex loads the policy twice until `gap sync --remove-codex-legacy`. Run the flag only after both hooks are trusted in `/hooks`; it refuses unless both were in `hooks.json` before that run. Empty `CODEX_HOME` means unset. A relative `CODEX_HOME` is refused.
- `~/.cursor/rules/global-agent-policy.mdc`
- `~/.empryo/EMPRYO.md`. Empryo loads it into every tab. SoulStack keeps its own marked block there, so the first sync is a foreign-file splice and needs a terminal yes.
- `~/.omp/agent/commands/<name>.md` for each `global/commands/<name>.md`
- `~/.claude/commands/<name>.md` for the same file. Claude Code loads that directory as user slash commands. `$ARGUMENTS` and `argument-hint` are Claude's own syntax, so the file is copied unchanged.

The CLI writes regular files. Do not symlink those paths. Do not add Stow. Publish only from the primary branch (`origin/HEAD`, `master` here) or GitButler's `gitbutler/workspace`, not from another branch or a worktree. Sync writes the policy between `<!-- gap:begin -->` and `<!-- gap:end -->`. A file that already has those markers is updated in place. Only the block changes, and the rest stays byte for byte. A foreign policy file with no markers stays unless a terminal answers yes. Yes backs that file up under `~/.gap/backups/` and appends the block, so the harness keeps loading the rest. No prompt leaves that file alone and the other destinations still publish. A symlink is still moved aside after a yes. That harness stops loading the old target. The backup keeps those bytes only. Command files are whole-file copies. A foreign command file is replaced after a yes. That harness stops loading the old file. The backup keeps the bytes only. Empty `CODEX_HOME` means unset. A relative `CODEX_HOME` is refused. Grok already loads `~/.claude/rules/global-agent-policy.md` when Claude compatibility is on. Confirm with `grok inspect` before adding a Grok destination. A second copy loads the policy twice. The human note is in `README.md`.
Do not point `~/CLAUDE.md` at the shared file. An ancestor `CLAUDE.md` suppresses that project's `AGENTS.md`.

After a shared markdown change is merged and pulled, run `gap sync`. Sync refuses while `global/` or `skills/` differs from `origin/HEAD`. An ignored file blocks only when skills.sh would install it or gap reads it; the rule is in `cmd/gap/AGENTS.md`. A copy is stale until that command runs.

## Edit

- Change a shared rule in `global/`, then sync. Do not paste the rule into a harness home by hand.
- Add a harness destination in `cmd/gap/main.go` only when that harness actually loads the path. Sync writes that payload. If nothing reads it, the copy is dead.
- Skills live in `skills/<scope>/<group>/<skill>/`. List a skill in `global/skills.json` to have `gap sync` install it and check it for drift. After adding, moving, or renaming a skill, run `bash scripts/update-readme.sh` to regenerate the catalog. Layout and catalog ownership are in `skills/AGENTS.md`.
- `docs/solutions/` holds documented solutions to past problems, by category, with YAML frontmatter (`module`, `tags`, `problem_type`). `CONCEPTS.md` is the shared domain vocabulary. Relevant when implementing or debugging in a documented area, or when a domain word is unclear.
- Go 1.24 or newer, standard library only. The one runtime exception: the skills step runs skills.sh through `npx`, so it needs Node. `go test ./...` is the self-test.

## Child DOX Index

- `cmd/gap/AGENTS.md`: how `gap sync` publishes, installs, migrates, and checks drift. Parent keeps the destination list and the publish limits in Publish.
- `skills/AGENTS.md`: skill source layout, manifest, and generated catalog.
- `docs/plans/AGENTS.md`: plans, requirements, and handoffs.
- `regroup/AGENTS.md`: dated status snapshots. Do not overwrite older files.

Not indexed as DOX children: `global/AGENTS.md` (published policy), `docs/solutions/` (frontmatter contract stays in Edit), `scripts/` (the catalog command is named above).
