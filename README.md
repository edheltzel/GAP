```
 ██████╗  █████╗ ██████╗
██╔════╝ ██╔══██╗██╔══██╗
██║  ███╗███████║██████╔╝
██║   ██║██╔══██║██╔═══╝
╚██████╔╝██║  ██║██║
 ╚═════╝ ╚═╝  ╚═╝╚═╝
```

# GAP

GAP is the global agent policy. Edit `global/AGENTS.md`, then publish that file into the paths Claude, Pi, OMP, Codex, and Cursor already read. Grok reads the Claude copy.

Stay in this directory.

## Prerequisites

- Node 24 or newer

```bash
node -v
```

You should see `v24` or newer. If the shell says command not found, install Node 24 and return here.

## Step 1: Edit the policy

Open `global/AGENTS.md`. Change one line you can spot later. Longer notes live in `global/docs/agents/`. Do not add a second policy file for one harness.

Save the file.

## Step 2: Publish

```bash
npm run sync
```

Sync prints an absolute path for each of these:

```text
~/.claude/rules/global-agent-policy.md
~/.pi/agent/APPEND_SYSTEM.md
~/.omp/agent/APPEND_SYSTEM.md
~/.codex/gap/global-agent-policy.md
~/.cursor/rules/global-agent-policy.mdc
```

Each path is a regular file, not a symlink.

If sync asks about an existing file, type `y`. Sync moves that file under `~/.gap/backups/` and prints the path. Read the backup. Put anything you still want into `global/AGENTS.md`, then run `npm run sync` again. Codex policy is a file GAP owns alone. A SessionStart hook prints it. Sync does not rewrite `~/.codex/AGENTS.md` on a policy edit.

If you see `Refusing to replace foreign file`, that file was left in place. The other destinations still publish. Run the same command in a terminal and type `y` to replace the refused file.

The next sync overwrites a file without asking once it matches this publish, or the last publish recorded in `~/.gap/published.json`.

Run sync from the primary branch (`origin/HEAD`, `master` here), not from another branch or a worktree. Empty `CODEX_HOME` means `~/.codex`. A relative value is refused.

## Step 3: Check Grok

Grok reads `~/.claude/rules/global-agent-policy.md` when Claude compatibility is on. That is the default. Sync does not write a Grok file. A second file would load the policy twice.

```bash
grok inspect
```

You should see `~/.claude/rules/global-agent-policy.md` in the global rules list.

If that line is missing, Claude compatibility is off. Turn it back on. Do not add a Grok destination unless Grok loads some other path.

## Where each file goes

| Harness | File it reads |
| --- | --- |
| Claude | `~/.claude/rules/global-agent-policy.md` |
| Pi | `~/.pi/agent/APPEND_SYSTEM.md` |
| OMP | `~/.omp/agent/APPEND_SYSTEM.md` |
| Codex | `~/.codex/gap/global-agent-policy.md`, printed by a SessionStart hook |
| Cursor | `~/.cursor/rules/global-agent-policy.mdc` |
| Grok | no extra file; reads the Claude rules file |

Do not point `~/CLAUDE.md` at the shared file. Claude looks through parent folders for `CLAUDE.md`. A file at `~/CLAUDE.md` sits above every project and stops Claude from reading that project's `AGENTS.md`.
