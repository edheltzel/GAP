# AGENTS.md

## Purpose

`gap` publishes this checkout's `global/AGENTS.md` and each `global/commands/*.md`, and installs the skills listed in `global/skills.json`.

## Ownership

- `cmd/gap/` owns publish, install, source-identity migration, and drift.
- Policy text stays in `global/`. Command prompts stay in `global/commands/`. Skill files stay in `skills/`. The manifest stays in `global/skills.json`.

## Local Contracts

`gap` publishes the checkout it was built from. `sourceRoot` is that compile-time path, so `gap sync` uses it from any directory. Reinstall with `go install ./cmd/gap` after `cmd/gap` changes, after the checkout moves, or after a `-trimpath` build. `go run ./cmd/gap sync` from the repo root uses that checkout without installing.

`gap sync` and `gap sync --remove-codex-legacy` publish. `gap -h`, `gap --help`, and `gap help` print usage and exit 0. No command, an unknown command, or extra arguments exit 2. A skipped destination or a failed skills step exits 1. Drift kept with both destinations complete exits 0. Skip, an incomplete keep, or drift with no terminal exits 1.

### Sync steps

1. Resolve `$HOME` and `CODEX_HOME`. Empty `CODEX_HOME` means unset. A relative `CODEX_HOME` is refused.
2. Merge gate. Refuse unless HEAD is the primary branch (`origin/HEAD`, `master` here) or `gitbutler/workspace`. Refuse a worktree (`git-dir` differs from `git-common-dir`). Refuse when `global/` or `skills/` differs from `origin/HEAD`, including untracked files under either directory. An ignored file blocks only if skills.sh would install it (a path under `skills/`) or gap reads it (`global/AGENTS.md`, `global/skills.json`). `.DS_Store` and `node_modules` do not block. Unset `origin/HEAD` is a refuse.
3. Publish `global/AGENTS.md`. Rewrite `](docs/agents/` links to the absolute `global/docs/agents/` path in this checkout. The Cursor file gets an `alwaysApply: true` frontmatter wrap. Write regular files only. A symlink is foreign. Then publish each top-level `global/commands/*.md` through the same `publish` function to `~/.omp/agent/commands/<name>.md` and `~/.claude/commands/<name>.md`. A missing `global/commands/` publishes nothing.
4. Ledger each written destination in `~/.gap/published.json` (path to the sha256 of the payload). Overwrite without asking only when the file is missing, already the payload, or its digest matches the ledger. Otherwise ask `[y/N]`. Yes backs the old file up under `~/.gap/backups/` and replaces it. A regular file is copied; a symlink is renamed. That harness stops loading the old file. The backup keeps the bytes only. No, or no terminal, leaves that file alone. Other destinations still publish.
5. If this run wrote the Codex policy file, add SessionStart and SubagentStart hooks that `cat` it. `--remove-codex-legacy` runs before that hook install. It strips an old policy prefix from `CODEX_HOME/AGENTS.md` only when this run published the policy file and both hooks were already in `hooks.json`. The cleaned ledger value is `cleaned`, not a digest.
6. Read `global/skills.json`. Source `"."` is this checkout. Any other source is `owner/repo`, optionally with a subpath.
7. Install missing listed skills through skills.sh: `npx -y skills add <source> -g -y -a claude-code -a codex -s <name>` (`-s` repeats). `"."` is passed as the checkout path. Required destinations are `~/.agents/skills/<name>` and the Claude mirror `~/.claude/skills/<name>`. Both must exist and match the source hash. That hash skips `node_modules`, `.git`, `__pycache__`, `__pypackages__`, and `metadata.json` on both the source and the installed copy. Missing `npx` skips install; policy publish already done; exit 1. Skills the manifest does not list are never touched.
8. Source-identity migration. The lock file is `$XDG_STATE_HOME/skills/.skill-lock.json` when `XDG_STATE_HOME` is set, otherwise `~/.agents/.skill-lock.json`. An entry whose source is not the manifest source is foreign. `"."` matches the absolute checkout path. `owner/repo`, a github.com URL, and a subpath are one identity. A missing or unreadable lock file is not a mismatch. Automatic migration runs only when both copies match the recorded hash (40-hex git tree or 64-hex skills.sh SHA-256). It backs up both destinations under `~/.gap/backups/` and reinstalls from the manifest. A failed migration restores those destinations and the previous lock entry. A missing or unverifiable hash, or an edited copy, goes to drift review and is not replaced without a choice.
9. Drift check. Compare each managed skill with its source: this checkout's skill directory, or the lock hash for an external source. On drift, a terminal shows the diff and asks `[k]eep installed`, `[t]ake source`, or `[s]kip`. An edited external skill clones the manifest source for that diff, not the lock's old source. Keep with both destinations complete leaves the copies and exits 0. An incomplete keep still exits 1. Take source backs up both destinations, reinstalls, and on failure restores those destinations and the previous lock entry. Skip exits 1. No terminal: the skill is left in place, the drift is listed, and the run exits 1.

### Publish destinations

- `~/.claude/rules/global-agent-policy.md`
- `~/.pi/agent/APPEND_SYSTEM.md`
- `~/.omp/agent/APPEND_SYSTEM.md`
- `$CODEX_HOME/gap/global-agent-policy.md` (`~/.codex` when unset)
- `~/.cursor/rules/global-agent-policy.mdc`
- `~/.omp/agent/commands/<name>.md` for each `global/commands/<name>.md`
- `~/.claude/commands/<name>.md` for the same file. Claude Code loads that path as a user slash command. The prompt uses Claude's `$ARGUMENTS` and `argument-hint`, so it is copied unchanged.

Do not add a destination nothing reads. Grok loads the Claude file when Claude compatibility is on. Confirm with `grok inspect` before adding a Grok destination. A second copy loads the policy twice.

## Work Guidance

- Go 1.24 or newer, standard library only. The skills step is the Node exception (`npx` runs skills.sh).
- Add a destination in `main.go` only when that harness loads the path.
- Tests run against throwaway homes. They must not touch the real `$HOME`.

## Verification

- `go test ./...`

## Child DOX Index

None.
