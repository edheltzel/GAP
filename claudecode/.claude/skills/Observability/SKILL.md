---
name: Observability
description: Manage the PAI Observability dashboard — real-time Claude Code session monitoring. USE WHEN observability, dashboard, start dashboard, stop dashboard, open dashboard, monitor agents, watch sessions, /observability.
---

# Observability

Real-time monitoring dashboard for Claude Code agent activity. Watches `~/.claude/projects/` and streams events to a Vue dashboard at `http://localhost:5172`.

## Commands

The user will invoke with an argument: `start`, `stop`, `restart`, `status`, or `open`.

Parse the argument from their message and execute the appropriate action below.

---

### `start`

```bash
~/.claude/Observability/manage.sh start-detached
```

Wait 2 seconds, then check status:
```bash
~/.claude/Observability/manage.sh status
```

Report: server running on port 4000, client on port 5172. Remind Ed to open the app if not already open, or suggest `open http://localhost:5172`.

---

### `stop`

```bash
~/.claude/Observability/manage.sh stop
```

Confirm stopped. Report ports 4000 and 5172 are free.

---

### `restart`

```bash
~/.claude/Observability/manage.sh stop
~/.claude/Observability/manage.sh start-detached
```

Wait 2 seconds, verify status, report running.

---

### `status`

```bash
~/.claude/Observability/manage.sh status
```

Report whether running or stopped. If running, include the dashboard URL: `http://localhost:5172`.

---

### `open`

Start if not already running:
```bash
~/.claude/Observability/manage.sh status
# If not running:
~/.claude/Observability/manage.sh start-detached
```

Then open the dashboard:
```bash
open http://localhost:5172
```

---

## MenuBar App

`/Applications/Observability.app` provides a native macOS menu bar controller.

- Eye icon (filled) = running
- Eye icon (outline) = stopped
- Click → Start/Stop/Restart/Open Dashboard/Launch at Login

To open it: `open /Applications/Observability.app`

---

## What the Dashboard Shows

| View | Description |
|------|-------------|
| Event Timeline | Every tool call, prompt submit, and stop event in real-time |
| Agent Swim Lanes | Multi-agent sessions shown as parallel lanes |
| Live Pulse Chart | Activity intensity over time |
| Filter Panel | Filter by agent, event type, or session |
| Background Tasks | Monitored background agent progress |

Data source: `~/.claude/projects/*/*.jsonl` (Claude Code's native session transcripts).

---

## Troubleshooting

**Ports in use:**
```bash
lsof -ti :4000 | xargs kill -9
lsof -ti :5172 | xargs kill -9
~/.claude/Observability/manage.sh start-detached
```

**Client not loading:**
```bash
cd ~/.claude/Observability/apps/client && bun install && bun run dev
```

**Server not responding:**
```bash
cd ~/.claude/Observability/apps/server && bun install
~/.claude/Observability/manage.sh start-detached
```
