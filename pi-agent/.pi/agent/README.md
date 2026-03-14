# Pi Agent System

Simplified sub-agent routing for Pi. Just run `pi` and type naturally, or use `/agent:name` for explicit selection.

## How It Works

### Option 1: Natural Language (Automatic)

```
User: lets do some research on ACME company
→ Router detects "research" → delegates to @research-assistant

User: create a responsive navbar with glassmorphism
→ Router detects "navbar", "glassmorphism" → delegates to @frontend-dev

User: design database schema for users and posts
→ Router detects "database schema" → delegates to @database-designer

User: setup CI/CD pipeline
→ Router detects "CI/CD" → delegates to @devops-engineer
```

### Option 2: Explicit Selection

```
User: /agent:frontend-dev create a login form
→ Directly uses frontend-dev

User: /agent:security-auditor review this code
→ Directly uses security-auditor

User: /agent:git-surgeon help me rebase
→ Directly uses git-surgeon
```

### Option 3: Multi-Agent (Complex Tasks)

```
User: build a full auth system with JWT and login UI
→ Router coordinates:
   1. @api-designer → designs endpoints
   2. @frontend-dev → builds UI
   3. @security-auditor → reviews
```

## 19 Sub-Agents Available

### Development (6)
- `@frontend-dev` - UI components, styling
- `@backend-dev` - API development, databases
- `@fullstack-task` - End-to-end features
- `@api-designer` - API architecture
- `@ui-designer` - Design systems
- `@database-designer` - Schema design

### Research & Docs (2)
- `@research-assistant` - Web research
- `@tech-writer` - Documentation

### Quality (3)
- `@test-engineer` - Unit/integration tests
- `@qa-automation` - E2E testing
- `@security-auditor` - Security reviews

### Operations (2)
- `@cloud-deployer` - Cloudflare deployment
- `@devops-engineer` - CI/CD, Docker

### Git & Release (2)
- `@git-surgeon` - Complex git operations
- `@release-manager` - Version management

### Core (4)
- `@router` - Main routing agent
- `@scout` - Code exploration
- `@planner` - Implementation planning
- `@worker` - Task execution

## Installation

```bash
# Run setup to symlink skills/extensions
bash ~/.pi/agent/setup-evolutionary-agents.sh
```

Then add to `~/.pi/agent/settings.json`:

```json
{
  "agents": [
    "agents/core/router.md",
    "agents/core/scout.md",
    "agents/core/planner.md",
    "agents/core/worker.md",
    "agents/dev/frontend-dev.md",
    "agents/dev/backend-dev.md",
    "agents/dev/fullstack-task.md",
    "agents/dev/api-designer.md",
    "agents/dev/ui-designer.md",
    "agents/dev/database-designer.md",
    "agents/research/research-assistant.md",
    "agents/research/tech-writer.md",
    "agents/quality/test-engineer.md",
    "agents/quality/qa-automation.md",
    "agents/quality/security-auditor.md",
    "agents/ops/cloud-deployer.md",
    "agents/ops/devops-engineer.md",
    "agents/git/git-surgeon.md",
    "agents/git/release-manager.md"
  ]
}
```

## No CLI Tools Needed

- No shell functions to remember
- No profile switching
- Just run `pi` and type naturally
- Or use `/agent:name` for specific agents

## Files

- `AGENT_CATALOG.md` - Complete agent reference
- `agents/` - All agent definitions
- `setup-evolutionary-agents.sh` - One-time setup

## Quick Start

```bash
# After setup, just run:
pi

# Then type naturally:
> research best practices for X
> create a login form with validation
> /agent:git-surgeon help me squash commits
```