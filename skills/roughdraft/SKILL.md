---
name: Roughdraft
description: Install and drive the published `roughdraft` CLI — a local-first markdown editor/viewer for reviewing markdown with an AI agent (comments + CriticMarkup suggestions). USE WHEN open markdown in roughdraft, review a draft, roughdraft open/start/status/stop, install roughdraft CLI, hand a markdown file to a human for review, roughdraft mcp. NOT FOR generic markdown linting (use a linter), building roughdraft from source (this is the published CLI, not the repo), or scaffolding new skills (use CreateSkill).
alwaysAllow: ["Bash"]
---

# Roughdraft

`roughdraft` is a published, local-first CLI tool (`npm i -g roughdraft`) that opens a single markdown file in a local browser app for review — inline comments and suggested edits via CriticMarkup, written back to the `.md` file on disk. No cloud, no account, no telemetry. It's built for an agent ↔ human review handoff: the agent writes markdown, opens it in roughdraft, and waits for the human's feedback.

This skill is about **using that CLI**, not building the project from source.

## Voice Notification

**When executing a workflow, do BOTH:**

1. **Send voice notification**:
   ```bash
   curl -s -X POST http://localhost:31337/notify \
     -H "Content-Type: application/json" \
     -d '{"message": "Running WORKFLOWNAME in Roughdraft"}' \
     > /dev/null 2>&1 &
   ```

2. **Output text notification**:
   ```
   Running **WorkflowName** in **Roughdraft**...
   ```

**Full documentation:** `~/.claude/PAI/DOCUMENTATION/Notifications/NotificationSystem.md`

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **Install** | "install roughdraft", "set up the roughdraft CLI" | `Workflows/Install.md` |
| **Review** | "open in roughdraft", "review this markdown", "hand off for review" | `Workflows/Review.md` |

## Quick Reference

```bash
roughdraft open <file.md>             # open AND wait for "Done Reviewing" (BLOCKS by default)
roughdraft open <file.md> --no-watch  # open without waiting (returns immediately)
roughdraft open <file.md> --print-url # print only the URL, don't open a browser (still waits unless --no-watch)
roughdraft open <file.md> --json      # add machine-readable output (does NOT change blocking)
roughdraft watch <file.md>            # wait for a "Done Reviewing" event (decoupled from open)
roughdraft start                      # start/reuse the background server, print URL, exit
roughdraft status [--json]            # is the server running? on what URL?
roughdraft stop                       # stop the background server
roughdraft doctor [path]              # diagnose setup / validate markdown (non-blocking)
roughdraft help agent                 # print the copyable agent setup prompt
roughdraft mcp                        # experimental stdio MCP server
```

- Default URL is `http://localhost:7373`. Server state lives in `~/.roughdraft/server.json`.
- `roughdraft open` reuses or auto-starts the server — you do **not** need to run `start` first.
- CriticMarkup in the `.md` file is the durable source of truth; everything stays as plain markdown on disk.
- Remote mode: `ROUGHDRAFT_HOST` (+ `ROUGHDRAFT_TOKEN` for non-loopback) routes `open` through a hosted instance. Default is local-only.

## Gotchas

The highest-value knowledge in this skill. Add to it after every failure.

- **`roughdraft open` WAITS for "Done Reviewing" BY DEFAULT** — bare `open <file>` blocks until the human finishes, with **no timeout by default**. Blocking is a property of `open`, **not** of `--json`. To NOT block, you must pass **`--no-watch`** (or use `--timeout <seconds>` to bound the wait). An agent that runs `open` expecting a quick return will hang.
- **`--json`, `--print-url`, and `--no-watch` are independent.** `--json` = machine-readable output (does not affect blocking). `--print-url` = print the URL and don't open a browser (but it STILL waits unless you also pass `--no-watch`). `--no-watch` = the actual "don't block" flag. For "just give me the URL and don't block," you need **`--print-url --no-watch` together**. For an agent handoff that should wait for feedback, use plain `open` (add `--json` if you want to parse the result).
- **`open` auto-starts the server.** Don't script `start` then `open` — `open` reuses a running server or launches one itself. Use `start` only when you want a server up without opening a file.
- **Decouple open from waiting with `watch`.** `roughdraft open <file> --no-watch` to open, then `roughdraft watch <file>` (optionally `--json`) to block on the next "Done Reviewing" — useful when you want to do other work between opening and waiting.
- **Pass an absolute path (or a clearly path-like arg).** `roughdraft open /abs/path/file.md` is safest; `roughdraft <path>` is a shortcut only when the input obviously looks like a path.
- **It does not touch `~/CLAUDE.md` / `~/AGENTS.md`.** The setup prompt (`https://roughdraft.md/setup.md`) asks the agent to update its *own* guidance — roughdraft never edits user-level agent files itself.
- **Global install with a bun-first toolchain:** `bun add -g roughdraft` instead of `npm i -g roughdraft`, or skip the global install entirely and run `bunx roughdraft ...` / `npx roughdraft ...`. It's a published bin, so any runner works.
- **Don't confuse the published CLI with the repo.** The source repo is a pnpm monorepo with its own dev wrappers (e.g. `roughdraft-dev-<worktree>`); this skill targets the published global `roughdraft` command only.

## Examples

**Example 1: Install the CLI**
```
User: "install roughdraft"
→ Invokes Install workflow
→ Runs `npm i -g roughdraft` (or `bun add -g roughdraft`)
→ Verifies with `roughdraft status`
```

**Example 2: Open a draft for review**
```
User: "open my-essay/draft.md in roughdraft"
→ Invokes Review workflow
→ Runs `roughdraft open ./my-essay/draft.md`
→ Reports the localhost URL; review server is up
```

**Example 3: Agent handoff — write, hand off, wait for feedback**
```
User: "draft the post, then let me review it in roughdraft"
→ Agent writes the markdown file
→ Runs `roughdraft open ./post.md --json` (blocks)
→ Human comments + clicks "Done Reviewing"
→ CLI prints event JSON (path, version, feedback counts, overallComment)
→ Agent revises based on the feedback
```
