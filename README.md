```
 ██████╗  █████╗ ██████╗
██╔════╝ ██╔══██╗██╔══██╗
██║  ███╗███████║██████╔╝
██║   ██║██╔══██║██╔═══╝
╚██████╔╝██║  ██║██║
 ╚═════╝ ╚═╝  ╚═╝╚═╝
```

# Global Agent Policies (GAP)

GAP is one set of rules that every coding agent on your machine follows. You, or an agent, write the rules once in `global/AGENTS.md`. The `gap sync` command copies them into the file each agent already reads.

![One AGENTS.md file, published by gap sync into Claude, Pi, OMP, Codex, and Cursor. Grok reads the Claude copy.](docs/images/publish-overview.webp)

## Before you start

- Go 1.24 or newer (`go version`)
- Go's bin folder on your `PATH`: `$(go env GOPATH)/bin`, or `GOBIN` if you set it

## Install `gap`

Run this once from the repo root:

```bash
go install ./cmd/gap
```

Check that your shell finds it:

```bash
gap --help
# gap publishes global/AGENTS.md into the file each coding agent already reads,
# and installs the skills listed in global/skills.json.
# Usage:
#   gap sync                         Publish the policy, install missing skills, check skills for drift
#   ...
```

`gap` publishes the checkout it was installed from, so it runs from any folder. Run `go install ./cmd/gap` again after `cmd/gap` changes or after you move the repo. To skip installing, run `go run ./cmd/gap sync` from the repo root.

## Commands

| Command | What it does |
| --- | --- |
| `gap sync` | Publish `global/AGENTS.md` to every agent, install missing skills, and check skills for drift |
| `gap sync --remove-codex-legacy` | Same, then strip the old policy copy from `~/.codex/AGENTS.md` |
| `go test ./...` | Run the self-test against throwaway folders |

## Manage skills

`global/skills.json` lists every skill `gap` manages and where each one comes from. `"source": "."` means a skill in this repo's `skills/` folder; any other source is a GitHub `owner/repo`, optionally followed by a subdirectory path.

```json
[
  { "source": ".", "skills": ["do-commit", "do-plan"] },
  { "source": "kunchenguid/gh-axi", "skills": ["gh-axi"] }
]
```

`gap sync` installs missing listed skills by running skills.sh (`npx -y skills add <source> -g -y -a claude-code -a codex -s <name>`, repeating `-s` for each missing skill). It goes into `~/.agents/skills`, which Codex, Cursor, Pi, and OMP read, and skills.sh links it into `~/.claude/skills` for Claude Code. You need Node for installations. If an installation is needed but `npx` is unavailable, sync still publishes the policy, skips the installation, and exits `1`.

Sync also checks every installed managed skill against its source: this repo's folder, or the hash skills.sh recorded at install. That hash is a Git tree id or a SHA-256 of the skill files; sync compares the same kind. A lock entry whose source is not that manifest source is reinstalled from the manifest after a backup, without asking, only when both destinations still match the recorded hash. An edited copy, or a hash sync cannot check, is reviewed like other drift, against the manifest source. For each skill whose content drifted, it shows the diff and asks:

- `k` keeps the installed copies for this run. Later runs check them again. An incomplete installation still exits `1`.
- `t` takes the source. Sync backs up both destinations under `~/.gap/backups/`, then reinstalls. A failed install restores the previous destinations.
- `s` skips it for now. Sync exits `1`.

With no terminal to ask in, sync lists the drifted skills, including a foreign lock that was not migrated, changes none of them, and exits `1`. Skills the manifest doesn't list are never touched.

To add a skill, add it to `global/skills.json`, then follow [the merge and publish steps](#step-2-merge-the-change).

## Change the policy

Do these steps every time you change the rules.

### Step 1: Edit `global/AGENTS.md`

Make your change in `global/AGENTS.md`. Longer notes it links to live in `global/docs/agents/`. Keep one policy for every agent; don't add a policy file for a single agent.

### Step 2: Merge the change

Commit the edit, merge it to `master`, and pull. `gap sync` publishes only merged rules. It refuses when:

- you're on a branch other than `master` or GitButler's `gitbutler/workspace`
- you're in a git worktree
- `global/` or `skills/` differs from `origin/HEAD`, including untracked and ignored files under either directory

### Step 3: Run `gap sync`

```bash
gap sync
```

It prints each file it wrote:

```text
~/.claude/rules/global-agent-policy.md
~/.pi/agent/APPEND_SYSTEM.md
~/.omp/agent/APPEND_SYSTEM.md
~/.codex/gap/global-agent-policy.md
~/.cursor/rules/global-agent-policy.mdc
```

Each is a regular file, not a symlink. Start a new session in any agent to load the new rules.

### Step 4: Answer if `gap sync` asks about a file

`gap sync` asks before it replaces a file that you or another tool edited.

1. Type `y`. Sync backs up the old file under `~/.gap/backups/` and prints where it went.
2. Open the backup and move anything you want to keep into `global/AGENTS.md`.
3. Merge that change and run `gap sync` again.

Any other answer leaves that file alone. With no terminal to ask in, such as an agent's shell, sync prints `Refusing to replace foreign file`, skips that file, publishes the rest, and exits `1`. Run `gap sync` yourself in a terminal to answer.

## Set up Codex (once per machine)

Codex has no rules folder. The first `gap sync` adds two hooks to `~/.codex/hooks.json`, SessionStart and SubagentStart. They print `~/.codex/gap/global-agent-policy.md` into every Codex session and subagent.

![Two hooks, SessionStart and SubagentStart, cat ~/.codex/gap/global-agent-policy.md into each Codex session and subagent.](docs/images/codex-hooks.webp)

1. Open Codex and run `/hooks`. Trust both GAP hooks.
2. Start a session and a subagent. Confirm both see the policy.
3. If `~/.codex/AGENTS.md` still holds an old copy of the policy, remove it:

   ```bash
   gap sync --remove-codex-legacy
   ```

   The flag refuses unless both hooks were already in `hooks.json` before that run. On a clean machine it reports that there's nothing to strip.

GAP uses `~/.codex` unless you set `CODEX_HOME`. An empty `CODEX_HOME` counts as unset. A relative path is refused.

## Check Grok (once per machine)

Grok has no file of its own. It reads the Claude copy when Claude compatibility is on, which is the default.

```bash
grok inspect
```

Look for `~/.claude/rules/global-agent-policy.md` in the global rules list. If it's missing, turn Claude compatibility back on. Don't add a Grok file. A second copy would load the policy twice.

## Skill catalog

Skills live in `skills/<scope>/<group>/<skill>/`. Regenerate this table with `bash scripts/update-readme.sh`.

<!-- skills-start -->

### [Global](./skills/global/)

Useful in any repo, or none. Install once per machine with `-g`.

| Group | Skills | Coverage |
|-------|--------|----------|
| [Core](./skills/global/core/) | 11 | Stack-agnostic foundations - review lenses, reasoning, testing, simplification, agent-legible repos, comments, design patterns, skill authoring, and AI-writing cleanup. |
| [Workflow](./skills/global/workflow/) | 15 | Change delivery - the brainstorm, plan, work, debug, and ship loop (LFG), commits, docs, GitHub projects and stacks, PR workflow and review triage, worktrees, and technical writing. |
| [Operations](./skills/global/operations/) | 4 | Operating AI agents - delegation, handoffs, prompt audits, and project status check-ins. |
| [Content](./skills/global/content/) | 3 | Audience-facing media - illustrations, diagrams, images, and visual explanations. |
| [Personal](./skills/global/personal/) | 2 | Personal extras you want everywhere - teaching, todo.txt. |

### [Project](./skills/project/)

Pay off only when a project uses that stack or product. Install into the project.

| Group | Skills | Coverage |
|-------|--------|----------|
| [TypeScript](./skills/project/typescript/) | 4 | Language-wide TypeScript - standards, refactoring, and type-driven design. Install alongside frontend, backend, or tooling. |
| [Frontend](./skills/project/frontend/) | 7 | Web frontend - Astro, CSS, React effects, design systems, UX flows, cleanup, and review. |
| [Backend](./skills/project/backend/) | 1 | Backend engineering - Effect (TypeScript) services and code, and review for Effect, Go, Python, and Rust. |
| [Tooling](./skills/project/tooling/) | 3 | CLIs and lightweight tools - TypeScript CLI scaffolding, zero-dependency patterns, and performance profiling. |
| [Swift](./skills/project/swift/) | 3 | Swift and Apple platforms - macOS desktop apps, iOS review, and cleanup. |
| [Product](./skills/project/product/) | 1 | Product marketing - positioning, SEO and GEO discovery, launches, conversion, and retention. |
| [Marketing](./skills/project/marketing/) | 9 | SEO marketing on the Ubersuggest MCP - keyword research, site audits, competitors, content briefs, AI visibility, and action plans. |
| [Personal](./skills/project/personal/) | 1 | Personal extras for specific projects - recipe diagrams. |

<!-- skills-end -->

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

### How sync tells its own files apart

`~/.gap/published.json` holds a fingerprint of each file sync last wrote. Sync overwrites a file without asking when it is missing, matches the new publish, or matches that fingerprint. Anything else counts as edited, and sync asks first.

![If the file is missing, unchanged, or matches the last publish, sync overwrites it. If someone edited it, sync asks y/n. Yes backs it up and writes. No answer skips that file.](docs/images/existing-file.webp)

### Don't create `~/CLAUDE.md`

Claude searches parent folders for `CLAUDE.md`. A file at `~/CLAUDE.md` sits above every project, and it stops Claude from reading that project's `AGENTS.md`. The Claude rules file above already covers every project.

### Self-test

```bash
go test ./...
```

The test builds `gap` inside a throwaway clone of this repo and runs it against a temporary home folder. It never touches your real agent files. It exits `0` when every check passes.
