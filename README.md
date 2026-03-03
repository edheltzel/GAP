# Atlas Config

**Atlas - Personal AI Infrastructure (Atlas)** configuration, managed with [GNU Stow](https://www.gnu.org/software/stow/).

Atlas is a general problem-solving system that magnifies human capabilities. It wraps Claude Code with a structured algorithm, 37 skills, 20 lifecycle hooks, voice integration, persistent memory, and a multi-agent coordination system.

This repo contains the full Atlas (Atlas) configuration. Files here get symlinked into `~/.claude/` via Stow — keeping everything version-controlled and portable.

---

## What is Atlas?

**Atlas** is the AI assistant identity that runs on Atlas. Think of it as a capable partner — not a tool to be commanded, but a collaborator with broad knowledge, zero ego, and genuine investment in your success.

Atlas operates across four roles simultaneously:

- **Collaborator / Sidekick** — Thinks alongside you. Offers perspective. Challenges assumptions when something could be better.
- **Mentor** — Provides guidance across domains: programming patterns, business strategy, workflow optimization, learning approaches. Frames advice as options you can interpret for yourself.
- **Force Multiplier** — Handles breadth (research, boilerplate, multi-file changes, parallel investigation) so you can focus depth on what matters.
- **Delegation Coordinator** — Orchestrates sub-agents and agent teams. Breaks complex work into parallel tracks. Manages the swarm so you see results, not process.

---

## The Algorithm

Every interaction runs through a structured 7-phase algorithm inspired by the scientific method. The core insight: you can't improve what you can't measure, and you can't measure without granular, testable criteria.

| Phase       | Purpose                                                                                    |
| ----------- | ------------------------------------------------------------------------------------------ |
| **Observe** | Reverse-engineer what you actually want. Extract constraints. Create Ideal State Criteria. |
| **Think**   | Pressure-test assumptions. Pre-mortem. Refine criteria.                                    |
| **Plan**    | Explore the codebase. Design the approach. Write execution strategy.                       |
| **Build**   | Create artifacts. Write code, content, or whatever the task requires.                      |
| **Execute** | Run the work. Deploy, apply changes, invoke capabilities.                                  |
| **Verify**  | Test every criterion. The culmination — did we actually achieve the ideal state?           |
| **Learn**   | Capture what worked, what didn't, and what to improve next time.                           |

The depth scales dynamically — a quick question gets a lightweight pass, while a complex project gets the full treatment with dozens of criteria tracked to completion.

---

## Skills

Skills are domain-specific capabilities invoked as slash commands. Type `/<skill-name>` in any Claude Code session with Atlas active.

### Thinking and Analysis

Deep reasoning tools for exploring problems from multiple angles.

| Skill                       | Command                       | What It Does                                                                                                  |
| --------------------------- | ----------------------------- | ------------------------------------------------------------------------------------------------------------- |
| **BeCreative**              | `/be-creative`                | Extended thinking and divergent ideation. Use when you need deep, creative reasoning on a problem.            |
| **FirstPrinciples**         | `/first-principles`           | Decomposes problems to their fundamental truths. Strips away assumptions to find root causes.                 |
| **IterativeDepth**          | `/iterative-depth`            | Multi-angle exploration using 2-8 different lenses on the same problem. Great for deep problem understanding. |
| **Science**                 | `/science`                    | Structured hypothesis-test-analyze cycles. The meta-skill for systematic experimentation.                     |
| **Council**                 | `/council`                    | Multi-agent structured debate. Multiple AI perspectives argue different sides of a question.                  |
| **RedTeam**                 | `/red-team`                   | Adversarial analysis with 32 agents. Stress-tests ideas, plans, and designs for weaknesses.                   |
| **WorldThreatModelHarness** | `/world-threat-model-harness` | Tests ideas against 11 time-horizon world models (6 months to 50 years). Adversarial future analysis.         |

### Research and Intelligence

Tools for gathering, parsing, and synthesizing information from multiple sources.

| Skill                   | Command                 | What It Does                                                                                                  |
| ----------------------- | ----------------------- | ------------------------------------------------------------------------------------------------------------- |
| **Research**            | `/research`             | Multi-model parallel research. Routes queries across Claude, Gemini, Grok, Perplexity, and Codex researchers. |
| **ExtractWisdom**       | `/extract-wisdom`       | Content-adaptive extraction from videos, podcasts, articles. Adapts output sections to match content type.    |
| **Parser**              | `/parser`               | Parses URLs, files, and videos into structured JSON. Handles YouTube transcripts, PDFs, articles.             |
| **OSINT**               | `/osint`                | Open source intelligence gathering. Due diligence, background research, company intel.                        |
| **PrivateInvestigator** | `/private-investigator` | Ethical people-finding and reconnection. Skip tracing with responsible practices.                             |
| **BrightData**          | `/bright-data`          | Progressive URL scraping with tiered approaches. Escalates from simple fetch to full rendering.               |
| **Apify**               | `/apify`                | Social media and business data scraping via Apify actors. Twitter, Instagram, LinkedIn, YouTube, and more.    |

### Security

Offensive and defensive security capabilities.

| Skill               | Command             | What It Does                                                                                                  |
| ------------------- | ------------------- | ------------------------------------------------------------------------------------------------------------- |
| **Recon**           | `/recon`            | Security reconnaissance and attack surface mapping for bug bounty and pentesting.                             |
| **WebAssessment**   | `/web-assessment`   | Web application security testing. Vulnerability scanning and penetration testing workflows.                   |
| **PromptInjection** | `/prompt-injection` | LLM security testing. Tests chatbots and AI applications for prompt injection vulnerabilities.                |
| **SECUpdates**      | `/sec-updates`      | Aggregates security news from tldrsec, no.security, and other sources. Stay current on breaches and research. |
| **AnnualReports**   | `/annual-reports`   | Aggregates and analyzes annual security/threat reports from major vendors.                                    |

### Building and Deployment

Tools for creating software, CLIs, skills, and deploying to the cloud.

| Skill           | Command         | What It Does                                                                                                 |
| --------------- | --------------- | ------------------------------------------------------------------------------------------------------------ |
| **CreateCLI**   | `/create-cli`   | Generates TypeScript command-line tools with argument parsing, help text, and proper structure.              |
| **CreateSkill** | `/create-skill` | Creates and validates new Atlas skills. Generates the full skill directory structure.                          |
| **Cloudflare**  | `/cloudflare`   | Deploys Cloudflare Workers and Pages projects. Handles configuration, bindings, and deployment.              |
| **Browser**     | `/browser`      | Debug-first browser automation with always-on visibility. Console logs, network requests, and error capture. |
| **Documents**   | `/documents`    | Document processing and transformation.                                                                      |
| **Remotion**    | `/remotion`     | Programmatic video creation with React. Motion graphics, intros, and rendered video output.                  |
| **Commit**      | `/commit`       | Conventional Commits workflow with CHANGELOG updates, scopes, breaking changes, and safety rules.            |
| **Evals**       | `/evals`        | Agent evaluation framework. Tests agent behavior with code-based, model-based, and human graders.            |

### Content and Creative

Tools for generating written, visual, and structured content.

| Skill          | Command        | What It Does                                                                                            |
| -------------- | -------------- | ------------------------------------------------------------------------------------------------------- |
| **Art**        | `/art`         | Complete visual content system. Illustrations, diagrams, mermaid charts, flowcharts, infographics.      |
| **WriteStory** | `/write-story` | Layered fiction writing using storytelling science. Story bibles, character arcs, worldbuilding, prose. |
| **Fabric**     | `/fabric`      | 240+ specialized prompt patterns for content analysis, extraction, and transformation.                  |
| **Prompting**  | `/prompting`   | Meta-prompting system for dynamic prompt generation using templates, standards, and patterns.           |
| **Aphorisms**  | `/aphorisms`   | Aphorism and quote management.                                                                          |

### Life and Business

Personal productivity, goal tracking, and business tools.

| Skill         | Command       | What It Does                                                                             |
| ------------- | ------------- | ---------------------------------------------------------------------------------------- |
| **Telos**     | `/telos`      | Life OS and project system. Goals, challenges, dependencies, predictions, books, movies. |
| **Sales**     | `/sales`      | Sales workflows including proposals, pricing, and pipeline management.                   |
| **USMetrics** | `/us-metrics` | US economic indicators — GDP, inflation, unemployment, gas prices.                       |

### System and Infrastructure

Tools for managing and improving Atlas itself.

| Skill               | Command             | What It Does                                                                                              |
| ------------------- | ------------------- | --------------------------------------------------------------------------------------------------------- |
| **AtlasUpgrade**      | `/pai-upgrade`      | Extracts system improvements from content. Monitors Anthropic ecosystem and YouTube for relevant updates. |
| **Agents**          | `/agents`           | Dynamic agent composition and management. Create custom agent personalities, voices, and specializations. |
| **KeybindingsHelp** | `/keybindings-help` | Customize Claude Code keyboard shortcuts, chord bindings, and keybindings.json.                           |

---

## Repo Structure

```
atlas-config/
├── .claude/                  ← Atlas configuration root (symlinked to ~/.claude/)
│   ├── settings.json         ← Hooks, identity, statusline, permissions
│   ├── CLAUDE.md             ← Agent context injection
│   ├── skills/               ← 37 skill modules
│   │   ├── Atlas/              ← Core algorithm and system docs
│   │   ├── Fabric/           ← 240+ prompt patterns
│   │   └── ...               ← Domain-specific skills
│   ├── hooks/                ← 20 lifecycle event handlers
│   ├── agents/               ← Named agent personas
│   ├── MEMORY/               ← Persistent memory
│   │   ├── WORK/             ← Active session data and PRDs
│   │   ├── STATE/            ← System state
│   │   └── LEARNING/         ← Reflections and learnings
│   ├── VoiceServer/          ← ElevenLabs TTS service
│   ├── lib/                  ← Shared TypeScript utilities
│   └── Atlas-Install/          ← Installer and setup wizard
├── CLAUDE.md                 ← Agent entry point (see below)
├── makefile                  ← Stow commands
└── README.md                 ← This file
```

---

## Setup

### Requirements

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) CLI
- [GNU Stow](https://www.gnu.org/software/stow/) (`brew install stow`)
- [Bun](https://bun.sh/) runtime (installed automatically by Atlas installer, if not already installed)

### Installation

```bash
# Clone this repo
git clone <your-repo-url> ~/Developer/atlas-config
cd ~/Developer/atlas-config

# Symlink Atlas config into ~/.claude/
make link

# Run the Atlas installer (first time only)
bash ~/.claude/install.sh

# Activate and launch
source ~/.zshrc
pai    # or: atlas (same thing)
```

### Stow Commands

```sh
make help      # Show all commands
make link      # First-time setup — create symlinks
make up        # Restow after adding/removing files
make status    # Dry run + show current symlinks
make unlink    # Remove all symlinks
make push      # Commit and push changes
```

Claude Code runtime files (`debug/`, `todos/`, `projects/`, `history.jsonl`, etc.) stay as real files in `~/.claude/` and are not tracked here.

---

## How It All Fits Together

1. **You type a request** in Claude Code (or use a slash command like `/extract-wisdom`).
2. **Claude hooks fire** — context loads, capabilities are assessed, the algorithm format is enforced.
3. **The Algorithm runs** — your request is decomposed into testable criteria, work is planned and executed, results are verified against those criteria.
4. **Skills are invoked** as needed — research agents fan out, browser automation verifies UI, security tools probe for vulnerabilities, creative tools generate content.
5. **Memory persists** — session context, learning, and work products survive across sessions via PRD files and the memory system.
6. **Voice announces** phase transitions and results (if configured with ElevenLabs), and voice mode with [Kokoro](https://huggingface.co/hexgrad/Kokoro-82M) and [Whisper by OpenAI](https://huggingface.co/openai/whisper-small), for full conversations. A startup voice greeting fires for all launch paths via an async SessionStart hook.
   The result: structured, verifiable, high-quality output that improves with every interaction.
