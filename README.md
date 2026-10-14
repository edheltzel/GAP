```
 ██████╗  █████╗ ██████╗
██╔════╝ ██╔══██╗██╔══██╗
██║  ███╗███████║██████╔╝
██║   ██║██╔══██║██╔═══╝
╚██████╔╝██║  ██║██║
 ╚═════╝ ╚═╝  ╚═╝╚═╝
```

# GAP

GAP is a Global Agent Policy. It's set of rules that every coding agent on your machine follows. You or Agents write the rules once in `global/AGENTS.md` and then a sinlge command copies them into your harness.

![One AGENTS.md file to rule them all. run sync into Claude, Pi, OMP, Codex, and Cursor. Grok reads the Claude copy.](docs/images/publish-overview.webp)

## Prereq:

You need Go 1.24+ to run the `gap` CLI.

```bash
go version
```

## TL;DR:

| What you want | Command |
| --- | --- |
| Publish the policy | `go run ./cmd/gap sync` |
| Run the self-test | `go test ./...` |
| Remove old Codex text | `go run ./cmd/gap sync --remove-codex-legacy` |

*`go install ./cmd/gap` puts `gap` on your PATH. Run `gap sync` from inside this repo.*

## TL;DR;

### Step 1: Edit the policy

Open `global/AGENTS.md` and change one line you'll recognize later. Save the file.

That file is the whole contract. Longer explanations it links to live in `global/docs/agents/`. Don't make a separate policy file for one agent. Every agent should read the same rules.

### Step 2: Merge before you publish

Sync only publishes rules that are already merged. It refuses in three cases:

- You're on a branch other than `master` or GitButler's `gitbutler/workspace`.
- You're in a git worktree.
- `global/` differs from `origin/HEAD`.

So commit your edit, merge it, and pull `master` before the next step. This keeps every agent on reviewed rules.

### Step 3: Publish

```bash
go run ./cmd/gap sync
```


Sync prints one line for each file it wrote:
```shell 
~/.claude/rules/global-agent-policy.md
~/.pi/agent/APPEND_SYSTEM.md
~/.omp/agent/APPEND_SYSTEM.md
~/.codex/gap/global-agent-policy.md
~/.cursor/rules/global-agent-policy.mdc
```

Each one is a real file, not a symlink. Start a new session in any agent and your changed line is there.

### Step 4: Answer if sync asks about a file

Sync remembers what it wrote last time. It keeps a fingerprint of each file in `~/.gap/published.json`. That lets it tell its own copy apart from one that you or another tool edited.

![If the file is missing, unchanged, or matches the last publish, sync overwrites it. If someone edited it, sync asks y/n. Yes backs it up and writes. No answer skips that file.](docs/images/existing-file.webp)

If sync asks before replacing a file:

1. Type `y`. Sync copies the old file under `~/.gap/backups/` and prints where it went.
2. Open that backup. Move anything you still want into `global/AGENTS.md`.
3. Merge that change and run `go run ./cmd/gap sync` again.

If sync can't ask, it prints `Refusing to replace foreign file` and leaves that one file alone. This happens when an agent runs it, not you. The other files still publish, and the command exits with an error. Run it yourself in a terminal to answer the question.

### Step 5: Set up Codex once

Codex doesn't have a rules folder, so GAP gives it the policy through two hooks. Sync adds both to `~/.codex/hooks.json`. Each one prints the policy file when a session or subagent starts.

![Two hooks, SessionStart and SubagentStart, cat ~/.codex/gap/global-agent-policy.md into each Codex session and subagent.](docs/images/codex-hooks.webp)

Codex asks you to trust new hooks before it runs them:

1. Open Codex and run `/hooks`. Trust both GAP hooks.
2. Start a session and a subagent. Confirm both see the policy.

Older setups also kept the policy text inside `~/.codex/AGENTS.md`, so Codex loads it twice. Once both hooks work, remove that old copy:

```bash
go run ./cmd/gap sync --remove-codex-legacy
```

The flag refuses unless both hooks were already in `hooks.json` before that run. Running it again on a clean machine is safe. It reports that there's nothing to strip.

GAP uses `~/.codex` unless you set `CODEX_HOME`. An empty `CODEX_HOME` counts as unset. A relative path is refused.

### Step 6: Check Grok

Grok has no file of its own. It reads the Claude copy when Claude compatibility is on, which is the default.

```bash
grok inspect
```

Look for `~/.claude/rules/global-agent-policy.md` in the global rules list. If it's missing, turn Claude compatibility back on. Don't add a Grok file. A second copy would load the policy twice.

## Reference

### Where each agent reads the policy

| Agent | File it reads |
| --- | --- |
| Claude | `~/.claude/rules/global-agent-policy.md` |
| Pi | `~/.pi/agent/APPEND_SYSTEM.md` |
| OMP | `~/.omp/agent/APPEND_SYSTEM.md` |
| Codex | `~/.codex/gap/global-agent-policy.md`, printed by the SessionStart and SubagentStart hooks |
| Cursor | `~/.cursor/rules/global-agent-policy.mdc` |
| Grok | No file of its own. It reads the Claude copy. |

### Don't create `~/CLAUDE.md`

Claude searches parent folders for `CLAUDE.md`. A file at `~/CLAUDE.md` sits above every project, and it stops Claude from reading that project's `AGENTS.md`. The Claude rules file above already covers every project.

### Self-test

```bash
go test ./...
```

This builds the CLI and runs it against throwaway folders in your temp directory. It never touches your real agent files. It exits `0` when everything passes.
