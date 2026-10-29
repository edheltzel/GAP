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

![Blip keeps the AGENTS.md rulebook on a GAP lectern while a crew of very different robots each works from the same copy.](docs/images/publish-overview.webp)

## Before you start

- Go 1.25 or newer (`go version`)
- Go's bin folder on your `PATH`: `$(go env GOPATH)/bin`, or `GOBIN` if you set it

## Install `gap`

```bash
curl -fsSL https://raw.githubusercontent.com/edheltzel/GAP/master/scripts/install.sh | sh
```

That one line does four things:

1. `curl` downloads `scripts/install.sh` from the `master` branch of `github.com/edheltzel/GAP`.
2. The piped script clones that repo into a new checkout under `~/.gap/checkouts/` and keeps it there. Each curl install gets its own checkout, so earlier installs and published documentation links keep their source.
3. It builds `gap` with `go install ./cmd/gap` and puts the binary in Go's bin folder.
4. It checks whether your shell can find that folder. If not, it prints the exact `export PATH=...` line to add to your shell config. Open a new terminal after adding it.

It needs Go 1.25 or newer and git. Installation writes the source checkout and Go binary. `gap sync` publishes the policy and installs the selected components and skills.

Already have the repo checked out? From its root, `sh scripts/install.sh` builds that copy instead of cloning, including when stdin is redirected. Keep that checkout too.

Then publish:

```bash
gap sync
```

`gap` publishes the checkout it was built from, so it runs from any folder. Run the curl command again after `cmd/gap` changes. To skip installing, run `go run ./cmd/gap sync` from a checkout.

## Commands

| Command | What it does |
| --- | --- |
| `gap sync` | Publish `global/AGENTS.md` and each `global/commands/*.md`, install missing skills, and check skills for drift |
| `gap sync --profile crew` | Prepare project files for cloud agents without fetching components or writing to the home directory |
| `gap sync --remove-codex-legacy` | Same, then strip the old policy copy from `~/.codex/AGENTS.md` |
| `gap status` | Show each component's pin, its store copy, and skill drift. Writes nothing; exits `1` on skill drift or an unpinned, missing, changed, or unreadable source checkout |
| `gap restore [id]` | Back up and replace missing or drifted component skills at the locked commit, reusing a verified store copy |
| `gap pin <id>@<ref>` | Resolve a branch, tag, or commit and write it to `gap.lock`. The only command that moves a pin |
| `gap doctor` | Run configured component doctors and report which harness commands are on `PATH` |
| `gap init [--adopt] [dir]` | Create project configuration and a policy pointer; `--adopt` reports and preserves existing project files |
| `go test ./...` | Run the self-test against throwaway folders |

Use `gap --help` or `gap <command> --help` for usage and flags. Status and doctor reports use color and tables in terminals, and plain text when redirected.

## Manage skills

`gap.toml` selects the pinned [component skills](#components), including Do-Skills. `global/skills.json` selects additional skills by source. In that manifest, `"source": "."` means a skill in this repo's `skills/` folder; any other source is a GitHub `owner/repo`, optionally followed by a subdirectory path.

```json
[
  { "source": "kunchenguid/gh-axi", "skills": ["gh-axi"] }
]
```

`gap sync` copies selected component skills directly from the pinned store into `~/.agents/skills` and `~/.claude/skills`. Component skill installation, drift repair, and restore do not use Node or `npx`. Project skills use the same two paths under the invoking project instead of the home directory; see [load levels](#load-levels).

Additional skills in `global/skills.json` still install through skills.sh (`npx -y skills add <source> -g -y -a claude-code -a codex -s <name>`, repeating `-s` for each missing skill). They use the home-directory destinations above, with a Claude link managed by skills.sh, and require Node. If such an installation is needed but `npx` is unavailable, sync still publishes the policy, skips that installation, and exits `1`.

Sync checks installed managed skills against their selected source. Component skills use the pinned store as their baseline and leave the skills.sh lock unchanged; matching existing copies stay in place. Manifest skills retain skills.sh source-identity migration and hash checks, described in the [sync contract](cmd/gap/AGENTS.md#sync-steps). For each drifted skill, sync shows the diff and asks:

- `k` keeps the installed copies for this run. Later runs check them again. An incomplete installation still exits `1`.
- `t` takes the source. Sync backs up both destinations under `~/.gap/backups/`, then reinstalls. A failed install restores the previous destinations.
- `s` skips it for now. Sync exits `1`.

With no terminal to ask in, sync lists the drifted skills, including a foreign lock that was not migrated, changes none of them, and exits `1`. Skills selected by neither `gap.toml` nor `global/skills.json` are never touched.

To add a skill from an existing component, add its name to that component's `names` array in `gap.toml`. For another source, add an entry to `global/skills.json`. Then follow [the merge and publish steps](#step-2-merge-the-change).

For a direct skills.sh install that does not use `gap.lock`, run `npx skills add edheltzel/Do-Skills -s <skill-name>`.

### Components

A component is a repo pinned to one commit that supplies skills or its own installer and doctor. [`gap.toml`](gap.toml) selects the components and their skills; [`gap.lock`](gap.lock) records the exact commits and trees:

```toml
# gap.toml
schema = "2"
[components.do-skills]
repo  = "github.com/edheltzel/Do-Skills"
ref   = "master"
names = ["do-commit", "do-todo"]
```

`gap pin do-skills@master` writes the requested ref, resolved commit, and tree to `gap.lock`. If a branch and tag share a name, use `refs/heads/<name>` or `refs/tags/<name>` to choose one. Machine sync fetches the locked commit into `~/.gap/store/<id>/<commit>/`, checks its tree against the lock, and installs the selected skills as described above. Sync never moves a pin. A component missing from `gap.lock` stops the sync and names the `gap pin` command to run. `gap.toml` and `gap.lock` pass the same merge gate as `global/`.

The distro includes Do-Skills, Forge, Echo, and Recall. Forge supplies `forge-interview`. Echo and Recall have empty skill lists: machine sync runs their configured `install` commands from their verified stores, and those installers own their skills and runtime prerequisites. Gap copies no skills from them and never installs a harness.

`gap doctor` runs each configured `doctor` command from its verified pinned store, then reports harness presence using `PATH`. Missing pins, missing or changed stores, and failed doctors produce exit `1`; doctor does not fetch missing components.

Doctor also asks Echo's verified pinned store for `scripts/install.sh --list-installed`. A successful report replaces Echo's adapter list and store commit in `~/.gap/state.json`, preserving other component entries. The current Echo pin lacks this optional flag: doctor prints `echo reported nothing; no adapters recorded` and leaves the ledger unchanged. This skipped report does not itself fail doctor. Wait for a supporting Echo tag before changing the pin. GAP records only reported names and does not install or remove adapters.

`gap status` checks component source checkouts and selected component skills, using the locked ref when pinned. It does not check Recall's memory database or additional manifest skills. `store=missing` means the checkout is absent, `store=changed` means its source files differ from the pin, and `store=unpinned` means `gap.lock` has no entry. For these states, it prints the matching repair command. Ignored, untracked build output does not change the result; tracked edits, deletions, and mode changes still do. Skill drift is checked only when the source checkout matches the lock; an unreadable checkout reports an error.

Status and `gap restore [id]` use the same overrides and skill selection as sync. `gap restore do-skills` repairs that component's selected missing or drifted skills without asking; omit the id to repair all components. Restore backs up replaced copies and rolls them back if installation fails. A verified store avoids network access.

### Load levels

Each component's `names` array is its skill list. An optional `[components.<id>.skills.<name>]` table selects how an existing name installs:

| `load` | Installation |
| --- | --- |
| `listed` (default), `always` | Install in the home-directory skill destinations |
| `project` | Install under the invoking project only when any `when.files` glob matches |
| `command`, `off` | Not selected for installation by sync or restore |

For example, the distro's `do-astro` selection uses `load = "project"` with `when.files = ["astro.config.*"]`. Globs are relative to the invoking project's Git root, or the current directory outside Git.

### Overrides

Sync, status, restore, and doctor read the installed distro's `gap.toml`, then `~/.config/gap/gap.toml`, then the invoking project's `.gap/gap.toml`. Later values win. `gap sync --profile <name>` applies the distro's `profiles/<name>.toml` last. A project with `profile = "crew"` also loads the crew profile on later runs.

Overrides can replace an existing component's `enabled` and `ref`, or a listed skill's `load` and `when.files`. A later `enabled = true` can re-enable a component. `when.files` replaces the earlier list. For example:

```toml
[components.recall]
enabled = false

[components.do-skills.skills.do-astro]
load = "project"
[components.do-skills.skills.do-astro.when]
files = ["astro.config.*", "apps/site/astro.config.*"]
```

Overrides never write `gap.lock`. If an overridden `ref` differs from the locked ref, the command stops and names the `gap pin <id>@<ref>` command needed to change the pin.

### Project setup

Run `gap init --adopt [dir]` for an existing project, or `gap init [dir]` for a new one; `dir` defaults to the current directory. Both create `.gap/gap.toml` only when absent, using `profile = "project"` and the directory name as the slug. Existing configuration must parse and contain a non-empty string `slug`; init reports an error before project writes otherwise. Init appends a short policy pointer to an unmarked `AGENTS.md`, preserving its existing text. With `--adopt`, an existing `AGENTS.md` stays byte-identical, including an empty file. A missing `AGENTS.md` is created in either mode, and a file containing either GAP marker stays unchanged. Non-regular `AGENTS.md` files are refused. `--adopt` also lists existing agent, Forge, task, and stack files. Successful project setup checks Recall through its CLI and adds `gap provisioned <slug>` using the persisted slug, unless a matching breadcrumb already exists anywhere in that project's history. Project-file provisioning failures add no breadcrumb; keeping an adopted or marked file still reaches the Recall check. Missing Recall prints a notice and succeeds. A failed Recall search or add reports an error and makes init exit `1`; project files already written stay in place for a retry. GAP never writes Recall's database directly. Crew setup, including a later init on a project with persisted `profile = "crew"`, never calls Recall. Init does not run the Forge interview.

For cloud agents, run `gap sync --profile crew`. It uses the same protected project-file writes, creating missing configuration with `profile = "crew"`. The [crew profile](profiles/crew.toml) excludes Recall and Echo; this sync skips home publication, store fetching, component installers, and skill installation. Commit the project files for the cloud agent. With an effective crew profile, status reports `crew: project files only` and restore writes nothing.

## Change the policy

Do these steps every time you change the rules.

### Step 1: Edit `global/AGENTS.md`

Make your change in `global/AGENTS.md`. Longer notes it links to live in `global/docs/agents/`. Keep one policy for every agent; don't add a policy file for a single agent.

### Step 2: Merge the change

Commit the edit, merge it to `master`, and pull. `gap sync` publishes only merged rules. It refuses when:

- you're on a branch other than `master` or GitButler's `gitbutler/workspace`
- you're in a git worktree
- `global/`, `skills/`, `gap.toml`, or `gap.lock` differs from `origin/HEAD`, including untracked files. An ignored file blocks only when skills.sh would install it, or when gap reads it (`global/AGENTS.md`, `global/skills.json`, top-level `global/commands/*.md`, `gap.toml`, `gap.lock`). Ignored `node_modules` and `.DS_Store` do not block.

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
~/.empryo/EMPRYO.md
~/.omp/agent/commands/state-docs.md
~/.claude/commands/state-docs.md
```

Policy files are the shared rules. Each top-level `global/commands/*.md` is a slash-command prompt, copied unchanged as a whole file to OMP and Claude Code. See [how sync handles existing files](#step-4-answer-if-gap-sync-asks-about-a-file).

The bundled [`/state-docs`](global/commands/state-docs.md) prompt writes project maps under `docs/state`. See the prompt for arguments and local Nessa docs requirements.

Each is a regular file, not a symlink. Start a new session in any agent to load the new rules.

### Step 4: Answer if `gap sync` asks about a file

`gap sync` asks before it appends the policy to a foreign file without a complete GAP marker pair, or replaces a foreign command file or a symlink. An empty file without a matching ledger entry also needs confirmation.

1. Type `y`. For a policy file, sync copies it under `~/.gap/backups/` and appends the policy between `<!-- gap:begin -->` and `<!-- gap:end -->`. The rest of the file stays, so the harness keeps loading it. For a command file, sync backs up the old contents and replaces them. For a symlink, sync retains the link and writes a regular file. If that replace stops before the move, the empty recovery folder is removed. In those replacement cases, the harness stops loading the old file. The backup keeps the bytes only.
2. Later policy syncs replace only that block and do not ask again. When the live file already equals the publish, sync leaves it in place. Otherwise it moves the previous file or symlink under `~/.gap/recovery/<dest-id>/<timestamp>/` (`<dest-id>` is the destination path relative to `$HOME`) and prints that path. If the destination is on another filesystem, the previous file is copied with its mode, fsync'd, and verified, then the live destination is re-checked and only then removed. A destination that changes or disappears during that copy is refused, and the rest of the sync continues. Recovery stays out of harness rule directories. Leftover `.<filename>.gap-recovery-*` directories next to a managed destination are moved into `~/.gap/recovery/<dest-id>/` and the new path is printed. A failed move prints a warning and leaves the folder in place. GAP keeps the 3 newest regular copies per destination. A displaced user symlink is kept and is never pruned. A moved leftover `.<filename>.gap-recovery-*` folder is never pruned. A prune failure prints a warning and does not fail the sync. Concurrent edits, including writes through an already open file, are kept in that recovery copy; if those bytes are no longer in the live file, sync prints a warning with the recovery path. Installation refuses to overwrite a file created during publication and names the recovery path. On failure, sync restores the previous file if the destination is still absent; otherwise both versions remain available, and a failed restore is printed with that path. Existing regular files keep their access permissions. Command files stay whole-file copies of `global/commands/`.
3. Rules you want every harness to follow still belong in `global/AGENTS.md`. Text outside the policy markers stays in that file. Command edits belong in the matching source under `global/commands/`.

Any other answer leaves that file alone. With no terminal to ask in, such as an agent's shell, sync prints `Refusing to replace foreign file`, skips that file, publishes the rest, and exits `1`. Run `gap sync` yourself in a terminal to answer.

## Set up Codex (once per machine)

Codex has no rules folder. The first `gap sync` adds two hooks to `~/.codex/hooks.json`, SessionStart and SubagentStart. They print `~/.codex/gap/global-agent-policy.md` into every Codex session and subagent.

![Blip slides the policy out of ~/.codex/gap. Two hooks, SessionStart and SubagentStart, cat ~/.codex/gap/global-agent-policy.md into each Codex session and subagent.](docs/images/codex-hooks.webp)

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

## Set up Empryo (once per machine)

Empryo reads `~/.empryo/EMPRYO.md`, which already holds SoulStack's block. The first `gap sync` asks once to append the GAP block there, so run it from a terminal and type `y`. SoulStack's block and anything else in the file stay as they are.

Empryo injects every skill in `skills.autoLoad` in full into every prompt. GAP recommends three: `do-todo` (the policy tracks tasks every turn), `do-commit` and `but-axi` (the GitButler flow). Everything else loads by name when a task needs it. Until your pins match, `gap sync` prints the exact `settings` call. Ask an agent inside Empryo to run it. gap never writes Empryo's config.

`gap sync` also warns when `~/.empryo/skills/<name>` differs from a managed skill. Empryo loads that copy first, so the managed one never runs there.

## Skill catalog

GAP keeps no skill source. [Do-Skills](https://github.com/edheltzel/Do-Skills#readme) supplies the `do-*` and marketing skills, pinned in `gap.toml` and `gap.lock`; its README holds the catalog and the human docs for each skill. External skills are listed in `global/skills.json`.

## Reference

### Where each agent reads the policy

| Agent | File it reads |
| --- | --- |
| Claude | `~/.claude/rules/global-agent-policy.md` |
| Pi | `~/.pi/agent/APPEND_SYSTEM.md` |
| OMP | `~/.omp/agent/APPEND_SYSTEM.md` |
| Codex | `~/.codex/gap/global-agent-policy.md`, printed by the SessionStart and SubagentStart hooks |
| Cursor | `~/.cursor/rules/global-agent-policy.mdc` |
| Empryo | `~/.empryo/EMPRYO.md`, beside the SoulStack block. The first sync asks once, so run it from a terminal. |
| Grok | No file of its own. It reads the Claude copy. |

### How sync tells its own files apart

The [sync contract](cmd/gap/AGENTS.md#sync-steps) defines ledger fingerprints, legacy-file recognition, and Cursor frontmatter handling.

### Don't create `~/CLAUDE.md`

Claude searches parent folders for `CLAUDE.md`. A file at `~/CLAUDE.md` sits above every project, and it stops Claude from reading that project's `AGENTS.md`. The Claude rules file above already covers every project.

### Self-test

```bash
go test ./...
```

The test builds `gap` inside a throwaway clone of this repo and runs it against a temporary home folder. It never touches your real agent files. It exits `0` when every check passes.
