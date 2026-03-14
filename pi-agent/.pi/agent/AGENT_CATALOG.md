# Complete Agent Catalog

## 17 Sub-Agents Available

### Development (6)

| Agent | File | Purpose |
|-------|------|---------|
| `@frontend-dev` | `agents/dev/frontend-dev.md` | UI components, styling, React/Vue |
| `@backend-dev` | `agents/dev/backend-dev.md` | API development, databases |
| `@fullstack-task` | `agents/dev/fullstack-task.md` | End-to-end features |
| `@api-designer` | `agents/dev/api-designer.md` | API architecture, REST/GraphQL |
| `@ui-designer` | `agents/dev/ui-designer.md` | Design systems, visual design |
| `@database-designer` | `agents/dev/database-designer.md` | Schema design, migrations |

### Research & Documentation (2)

| Agent | File | Purpose |
|-------|------|---------|
| `@research-assistant` | `agents/research/research-assistant.md` | Web research, information gathering |
| `@tech-writer` | `agents/research/tech-writer.md` | Documentation, READMEs, guides |

### Quality & Testing (3)

| Agent | File | Purpose |
|-------|------|---------|
| `@test-engineer` | `agents/quality/test-engineer.md` | Unit/integration tests |
| `@qa-automation` | `agents/quality/qa-automation.md` | E2E testing, Playwright |
| `@security-auditor` | `agents/quality/security-auditor.md` | Security reviews |

### Operations (2)

| Agent | File | Purpose |
|-------|------|---------|
| `@cloud-deployer` | `agents/ops/cloud-deployer.md` | Cloudflare, edge deployment |
| `@devops-engineer` | `agents/ops/devops-engineer.md` | CI/CD, Docker, pipelines |

### Git & Release (2)

| Agent | File | Purpose |
|-------|------|---------|
| `@git-surgeon` | `agents/git/git-surgeon.md` | Complex git operations |
| `@release-manager` | `agents/git/release-manager.md` | Versioning, releases |

### Core Workflow (3)

| Agent | File | Purpose |
|-------|------|---------|
| `@router` | `agents/core/router.md` | Main routing agent |
| `@scout` | `agents/core/scout.md` | Codebase exploration |
| `@planner` | `agents/core/planner.md` | Implementation planning |
| `@worker` | `agents/core/worker.md` | Task execution |

## Usage

### Automatic Routing
```
User: "research GraphQL best practices"
→ @research-assistant handles it

User: "create a login form with validation"
→ @frontend-dev handles it
```

### Explicit Selection
```
User: "/agent:security-auditor review this auth code"
→ Direct to security-auditor

User: "/agent:database-designer design user schema"
→ Direct to database-designer
```

### Multi-Agent
```
User: "build a SaaS with auth, billing, and dashboard"
→ Router coordinates:
   1. @api-designer (endpoints)
   2. @database-designer (schema)
   3. @frontend-dev (dashboard UI)
   4. @security-auditor (review)
```

## File Structure

```
agents/
├── core/
│   ├── router.md              # Main entry point
│   ├── scout.md               # Code exploration
│   ├── planner.md             # Planning
│   └── worker.md              # Execution
├── dev/
│   ├── frontend-dev.md
│   ├── backend-dev.md
│   ├── fullstack-task.md
│   ├── api-designer.md
│   ├── ui-designer.md
│   └── database-designer.md
├── research/
│   ├── research-assistant.md
│   └── tech-writer.md
├── quality/
│   ├── test-engineer.md
│   ├── qa-automation.md
│   └── security-auditor.md
├── ops/
│   ├── cloud-deployer.md
│   └── devops-engineer.md
└── git/
    ├── git-surgeon.md
    └── release-manager.md
```

## Installation

Add to `~/.pi/agent/settings.json`:

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

Or symlink the entire agents directory and use wildcard.