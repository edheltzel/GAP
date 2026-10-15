---
name: do-create-cli
version: 1.1.18
description: "Generates production-ready TypeScript CLIs via a 3-tier template system (manual arg parsing, Commander.js, oclif), each shipping full implementation, docs, package.json, strict config, JSON output, and exit-code compliance. USE WHEN create CLI, build CLI, command-line tool, wrap API, add command, upgrade tier, TypeScript CLI."
effort: medium
---

# CreateCLI

`<skill-dir>` below is the folder containing this SKILL.md.

## What It Does

Generates production-ready TypeScript CLIs. Every CLI ships with full implementation, README and QUICKSTART, a Bun package.json, strict tsconfig, JSON output, and correct exit codes. A three-tier template system picks the right complexity: Tier 1 manual arg parsing with zero deps (most cases), Tier 2 Commander.js for subcommands, Tier 3 oclif as a reference for enterprise scale.

## The Problem

People build the same CLI from scratch over and over. It starts as a bash script, then needs error handling, then help text, then type safety, and ends up rewritten in TypeScript with docs bolted on at the end. Each round repeats the same boilerplate and the same mistakes. This skill takes that whole arc and produces a clean, typed, documented CLI in one pass, at the tier the job actually needs.

## How It Works

Generate production-ready TypeScript CLIs with documentation, type safety, error handling, and CLI-First Architecture principles.

---


## Workflow Routing

Route to the appropriate workflow based on the request.

**When executing a workflow, output this notification directly:**

```
Running the **WorkflowName** workflow in the **CreateCLI** skill to ACTION...
```

| Workflow | Trigger | File |
|----------|---------|------|
| CreateCli | Create a new CLI tool from scratch | `Workflows/CreateCli.md` |
| AddCommand | Add a new command to existing CLI | `Workflows/AddCommand.md` |
| UpgradeTier | Upgrade CLI to higher tier | `Workflows/UpgradeTier.md` |

---

## 🚀 WHEN TO ACTIVATE THIS SKILL

Activate when you see these patterns:

### Direct Requests
- "Create a CLI for [API/service/tool]"
- "Build a command-line interface for X"
- "Make a CLI that does Y"
- "Generate a TypeScript CLI"
- "I need a CLI tool for Z"

### Context Clues
- User describes repetitive API calls → Suggest CLI
- User mentions "I keep typing this command" → Suggest CLI wrapper
- User has bash script doing complex work → Suggest TypeScript CLI replacement
- User working with API that lacks official CLI → Suggest creating one

### Examples
- ✅ "Create a CLI for the GitHub API"
- ✅ "Build a command-line tool to process CSV files"
- ✅ "Make a CLI for my database migrations"
- ✅ "Generate a CLI that wraps this API"
- ✅ "I need a small typed CLI for the Notion API"

---

## 💡 CORE CAPABILITIES

### Three-Tier Template System

**Tier 1: Manual parsing (DEFAULT - 80% of use cases)**
- Manual argument parsing (process.argv)
- Zero framework dependencies
- Bun + TypeScript
- Type-safe interfaces
- ~300-400 lines total
- **Perfect for:** API clients, data transformers, simple automation

**When to use Tier 1:**
- ✅ 2-10 commands
- ✅ Simple arguments (flags, values)
- ✅ JSON output
- ✅ No subcommands
- ✅ Fast development

**Tier 2: Commander.js (ESCALATION - 15% of use cases)**
- Framework-based parsing
- Subcommands + nested options
- Auto-generated help
- Plugin-ready
- **Perfect for:** Complex multi-command tools

**When to use Tier 2:**
- ❌ 10+ commands needing grouping
- ❌ Complex nested options
- ❌ Plugin architecture
- ❌ Multiple output formats

**Tier 3: oclif (REFERENCE ONLY - 5% of use cases)**
- Documentation only (no templates)
- Enterprise-grade plugin systems
- **Perfect for:** Heroku CLI, Salesforce CLI scale (rare)

### What Every Generated CLI Includes

**1. Complete Implementation**
- TypeScript source with full type safety
- All commands functional and tested
- Error handling with proper exit codes
- Configuration management

**2. Comprehensive Documentation**
- README.md with philosophy, usage, examples
- QUICKSTART.md for common patterns
- Inline help text (--help)
- API response documentation

**3. Development Setup**
- package.json (Bun configuration)
- tsconfig.json (strict mode)
- .env.example (configuration template)
- File permissions configured

**4. Quality Standards**
- Type-safe throughout
- Deterministic output (JSON)
- Composable (pipes to jq, grep)
- Error messages with context
- Exit code compliance

---

## 🏗️ Conventions

### Technology Stack Alignment

Generated CLIs follow these standards:
- ✅ **Runtime:** Bun (NOT Node.js)
- ✅ **Language:** TypeScript (NOT JavaScript or Python)
- ✅ **Package Manager:** Bun (NOT npm/yarn/pnpm)
- ✅ **Testing:** Vitest (when tests added)
- ✅ **Output:** Deterministic JSON (composable)
- ✅ **Documentation:** README + QUICKSTART

### Output location

Create the CLI where the user asks. If they do not name a directory, create `./<cli-name>/` in the current working directory.

Do not assume a home tools directory or a personal projects folder. Confirm the directory before any git operation.

### CLI-First Architecture Principles

Every generated CLI follows:
1. **Deterministic** - Same input → Same output
2. **Clean** - Single responsibility
3. **Composable** - JSON output pipes to other tools
4. **Documented** - Comprehensive help and examples
5. **Testable** - Predictable behavior

---

## 📚 EXTENDED CONTEXT

**For detailed information, read these files:**

### Workflow Documentation
- `Workflows/CreateCli.md` - Main CLI generation workflow (decision tree, 10-step process)
- `Workflows/AddCommand.md` - Add commands to existing CLIs
- `Workflows/UpgradeTier.md` - Migrate simple → complex

### Reference Documentation
- `FrameworkComparison.md` - Manual vs Commander vs oclif (with research)
- `Patterns.md` - Common CLI patterns
- `TypescriptPatterns.md` - Type safety patterns (from tsx, vite, bun research)

---

## 📖 EXAMPLES

### Example 1: API Client CLI (Tier 1)

**User Request:**
"Create a CLI for the GitHub API that can list repos, create issues, and search code"

**Generated structure** (new folder in the working directory, unless the user names another path):
```
./ghcli/
├── ghcli.ts
├── package.json          # Bun + TypeScript
├── tsconfig.json         # Strict mode
├── .env.example          # GITHUB_TOKEN=your_token
├── README.md             # Full documentation
└── QUICKSTART.md         # Common use cases
```

**Usage:**
```bash
ghcli repos --user exampleuser
ghcli issues create --repo owner/repo --title "Bug fix"
ghcli search "typescript CLI"
ghcli --help
```

---

### Example 2: File Processor (Tier 1)

**User Request:**
"Build a CLI to convert markdown files to HTML with frontmatter extraction"

**Generated Structure:**
```
./md2html/
├── md2html.ts
├── package.json
├── README.md
└── QUICKSTART.md
```

**Usage:**
```bash
md2html convert input.md output.html
md2html batch *.md output/
md2html extract-frontmatter post.md
```

---

### Example 3: Data Pipeline (Tier 2)

**User Request:**
"Create a CLI for data transformation with multiple formats, validation, and analysis commands"

**Generated Structure:**
```
./data-cli/
├── data-cli.ts           # Commander.js with subcommands
├── package.json
├── README.md
└── QUICKSTART.md
```

**Usage:**
```bash
data-cli convert json csv input.json
data-cli validate schema data.json
data-cli analyze stats data.csv
data-cli transform filter --column=status --value=active
```

---

## ✅ QUALITY STANDARDS

Every generated CLI must pass these gates:

### 1. Compilation
- ✅ TypeScript compiles with zero errors
- ✅ Strict mode enabled
- ✅ No `any` types except justified

### 2. Functionality
- ✅ All commands work as specified
- ✅ Error handling comprehensive
- ✅ Exit codes correct (0 success, 1 error)

### 3. Documentation
- ✅ README explains philosophy and usage
- ✅ QUICKSTART has common examples
- ✅ --help text comprehensive
- ✅ All flags/options documented

### 4. Code Quality
- ✅ Type-safe throughout
- ✅ Clean function separation
- ✅ Error messages actionable
- ✅ Configuration externalized

### 5. Integration
- ✅ Follows the Bun / TypeScript tech stack
- ✅ CLI-First Architecture principles
- ✅ Deterministic output (JSON)
- ✅ Composable with other tools

---

## 🎯 PHILOSOPHY

### Why This Skill Exists

Developers repeatedly create CLIs for APIs and tools. Each time:
1. Starts with bash script
2. Realizes it needs error handling
3. Realizes it needs help text
4. Realizes it needs type safety
5. Rewrites in TypeScript
6. Adds documentation
7. Now has production CLI

**This skill automates steps 1-7.**

### The Tier 1 pattern

Manual parsing is the default because a small CLI does not need a framework:
- Zero dependencies
- Typed interfaces and error handling
- Help text and a README written for the commands that exist
- JSON on stdout, errors on stderr

Escalate only when the decision tree says so.

### Design Principles

1. **Start Simple** - Default to Tier 1 (manual parsing)
2. **Escalate When Needed** - Tier 2 only when justified
3. **Complete, Not Scaffold** - Every CLI is production-ready
4. **Documentation First** - README explains "why" not just "how"
5. **Type Safety** - TypeScript strict mode always

---

**This skill turns "I need a CLI for X" into a typed, documented CLI, using manual parsing and CLI-First Architecture.**

## Gotchas

- **Default to bun.** Use npm or npx only if the user asks.
- **TypeScript only.** Never generate Python CLIs unless the user explicitly approves.
- **3-tier system:** Start with the simplest tier that fits. Don't over-engineer a Tier 3 CLI when Tier 1 suffices.
