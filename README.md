# atlas-config

Shared rules for Claude, Pi, and OMP. You edit the markdown here. A Node script copies it into the files those harnesses already read.

## Prerequisites

- Node 24 or newer

## Publish the rules

1. Edit `global/AGENTS.md`. Longer notes live in `global/docs/agents/`.
2. From this directory, run:

```bash
npm run sync
```

You should see:

```text
/Users/ed/.claude/rules/global-agent-instructions.md
/Users/ed/.pi/agent/APPEND_SYSTEM.md
/Users/ed/.omp/agent/APPEND_SYSTEM.md
```

Each path is a regular file, not a symlink. A harness keeps the previous copy until you run `npm run sync` again.

## Where the rules go

| Harness | File it reads |
| --- | --- |
| Claude | `~/.claude/rules/global-agent-instructions.md` |
| Pi | `~/.pi/agent/APPEND_SYSTEM.md` |
| OMP | `~/.omp/agent/APPEND_SYSTEM.md` |

Do not point `~/CLAUDE.md` at the shared file. An ancestor `CLAUDE.md` suppresses a project's `AGENTS.md`.

Add a destination in `src/sync.ts` only when that harness actually loads the path.
