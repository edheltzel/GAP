# Observability Dashboard

Real-time monitoring dashboard for Claude Code agent activity. Watches `~/.claude/projects/` for session transcripts and streams events via WebSocket.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     Observability System                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    WebSocket     ┌──────────────┐             │
│  │   Server    │ ◄──────────────► │    Client    │             │
│  │  :4000      │    /stream       │    :5172     │             │
│  └──────┬──────┘                  └──────────────┘             │
│         │                                                       │
│         │ watches                                               │
│         ▼                                                       │
│  ~/.claude/projects/*/*.jsonl                                   │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  MenuBarApp (macOS)     ─────►     manage.sh                    │
│  - Start/Stop/Restart              - Process management         │
│  - Status indicator                - Detached mode              │
│  - Launch at Login                 - Port cleanup               │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### Server (`apps/server/`)

**Port:** 4000

Bun-based WebSocket server that:
- Watches ALL project directories in `~/.claude/projects/`
- Parses JSONL session transcripts in real-time
- Broadcasts events to connected WebSocket clients
- Provides REST API for events, themes, tasks, and activities

**Key Files:**
- `src/index.ts` - Main server with HTTP/WebSocket endpoints
- `src/file-ingest.ts` - File watcher and JSONL parser
- `src/task-watcher.ts` - Background task monitoring
- `src/theme.ts` - Theme management API
- `src/types.ts` - TypeScript type definitions

**API Endpoints:**
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/stream` | WS | WebSocket for real-time events |
| `/events/recent` | GET | Recent events (limit param) |
| `/events/filter-options` | GET | Available filters |
| `/events/by-agent/:name` | GET | Events by agent name |
| `/api/tasks` | GET | Background task list |
| `/api/tasks/:id` | GET | Specific task details |
| `/api/tasks/:id/output` | GET | Full task output |
| `/api/themes` | GET/POST | Theme search/create |
| `/api/themes/:id` | GET/PUT/DELETE | Theme CRUD |
| `/api/activities` | GET | Kitty terminal tab titles |

### Client (`apps/client/`)

**Port:** 5172

Vue 3 + Vite dashboard featuring:
- Real-time event timeline with auto-scroll
- Agent swim lanes for multi-agent visualization
- Live pulse chart showing activity intensity
- Filter panel (agent, event type, session)
- Theme customization system
- Background task monitoring
- Toast notifications for agent requests

**Key Components:**
- `App.vue` - Main application
- `components/EventTimeline.vue` - Event stream display
- `components/AgentSwimLaneContainer.vue` - Multi-agent view
- `components/LivePulseChart.vue` - Activity visualization
- `components/FilterPanel.vue` - Event filtering
- `components/ThemeManager.vue` - Theme customization
- `composables/useWebSocket.ts` - WebSocket connection handler

**Tech Stack:**
- Vue 3 with Composition API
- Vite for bundling
- Tailwind CSS for styling
- Lucide icons
- Custom typography (Valkyrie, Triplicate)

### MenuBarApp (`MenuBarApp/`)

Native macOS menu bar application (Swift) for easy dashboard management.

**Features:**
- Status indicator (eye icon filled = running)
- Start/Stop/Restart controls
- Open Dashboard in browser
- Launch at Login toggle
- Auto-starts server on app launch

**Building:**
```bash
cd MenuBarApp
./build.sh
# Installs to /Applications/Observability.app
```

**Launch at Login:**
Creates/removes `~/Library/LaunchAgents/com.pai.observability.plist`

### Management Script (`manage.sh`)

Central control script for the observability system.

**Commands:**
```bash
./manage.sh start          # Start server + client (foreground)
./manage.sh stop           # Stop all processes
./manage.sh restart        # Stop then start
./manage.sh status         # Check if running
./manage.sh start-detached # Start in background (for MenuBarApp)
```

### Tools (`Tools/`)

- `ManageServer.ts` - TypeScript wrapper for server management

### Scripts (`scripts/`)

- `start-agent-observability-dashboard.sh` - Legacy start script
- `reset-system.sh` - Reset system state
- `test-system.sh` - System testing

## Event Types

Events parsed from JSONL transcripts:

| Event Type | Description |
|------------|-------------|
| `UserPromptSubmit` | User message sent |
| `PreToolUse` | Tool about to be called |
| `PostToolUse` | Tool result received |
| `Stop` | Assistant response complete |
| `Completed` | Todo item marked done |

## Quick Start

### Via Slash Command (Recommended)
```
/observability start   # Start dashboard
/observability open    # Start + open browser
/observability stop    # Stop dashboard
/observability status  # Check status
```

### Via MenuBarApp
1. Build and install: `cd MenuBarApp && ./build.sh`
2. Open `/Applications/Observability.app`
3. Click menu bar icon to manage

### Manual
```bash
# Start both server and client
./manage.sh start

# Or start in background
./manage.sh start-detached

# Open dashboard
open http://localhost:5172
```

## Dependencies

**Runtime:**
- Bun (server runtime)
- Node modules (installed via `bun install` in each app)

**MenuBarApp:**
- Xcode Command Line Tools (for Swift compilation)

## Troubleshooting

**"Waiting for events..." spinning:**
- Ensure server is watching correct directories
- Server scans ALL `~/.claude/projects/-*` subdirectories
- Check that JSONL files exist and are being written to

**MenuBarApp shows "Stopped" but dashboard works:**
- Server running but app can't detect it
- Restart via menu bar

**Port already in use:**
```bash
./manage.sh stop
# Force kill if needed:
lsof -ti :4000 | xargs kill -9
lsof -ti :5172 | xargs kill -9
```

**Client not starting:**
```bash
cd apps/client
bun install
bun run dev
```

## Data Flow

1. Claude Code writes session data to `~/.claude/projects/{project-dir}/{session}.jsonl`
2. Server's file watcher detects changes
3. New JSONL lines are parsed into HookEvent objects
4. Events are enriched with agent names from `~/.claude/MEMORY/STATE/agent-sessions.json`
5. Events broadcast to connected WebSocket clients
6. Dashboard displays events in real-time
