```
 ██████╗  █████╗ ██████╗
██╔════╝ ██╔══██╗██╔══██╗
██║  ███╗███████║██████╔╝
██║   ██║██╔══██║██╔═══╝
╚██████╔╝██║  ██║██║
 ╚═════╝ ╚═╝  ╚═╝╚═╝
```

# Global Agent Policy

This repo is just what I call, GAP (global agent policy) for Claude, Pi, and OMP. You can manually edit this or have an agent edit it, but these are the global agent policies you want your harness to follow.

**For Example:** You never want emojis or emdash in content writing.

## Prerequisites

- Node 24 or newer
- claude, pi, or omp installed

## Publish the rules

1. Edit `global/AGENTS.md`. Longer notes live in `global/docs/agents/`.
2. From this directory, run:

```bash
npm run sync
```

You should see:

```text
/Users/ed/.claude/rules/global-agent-policy.md
/Users/ed/.pi/agent/APPEND_SYSTEM.md
/Users/ed/.omp/agent/APPEND_SYSTEM.md
/Users/ed/.codex/AGENTS.md
/Users/ed/.cursor/rules/global-agent-policy.mdc
```

Each path is a regular file, not a symlink. A file that is not the current publish is moved to `~/.gap/backups/` after a yes, then replaced. A later sync overwrites quietly. Grok already reads the Claude file, so it has no extra copy.

## Where the rules go

| Harness | File it reads                            |
| ------- | ---------------------------------------- |
| Claude  | `~/.claude/rules/global-agent-policy.md` |
| Pi      | `~/.pi/agent/APPEND_SYSTEM.md`           |
| OMP     | `~/.omp/agent/APPEND_SYSTEM.md`          |
| Codex   | `~/.codex/AGENTS.md`                     |
| Cursor  | `~/.cursor/rules/global-agent-policy.mdc` |

