# Lessons Learned

*Specific lessons learned from experience. See also WISDOM.md for broader life wisdom.*

## AI & Coding Agents (2026-03-16)

### Never Trust First-Pass AI Output

No matter how much context or direction you provide, AI coding agents are not reliable on the first attempt. Always verify. Build validation into the workflow — red teaming, subagent review, empirical testing. The RISEN framework still works for structuring prompts, but the output must be checked.

### One-Shotting Is Marketing Theater

X/Twitter is full of performative AI usage — most people are ricing their setup, not shipping code. Real AI value comes from solving specific problems: fixing WordPress sites, front-end projects, SEO issues for clients. AI needs to generate revenue, not just burn through API credits.

### Treat AI Like a New Hire, Not a Mind Reader

The biggest mistake people make is assuming the AI knows who you are, what you do, and what you need. You have to onboard it like a human replacing you at your job. If you can articulate that onboarding process — that's your workflow. That's the algorithm.

### Community Packages Are Starting Points, Not Destinations

Third-party extensions and packages are valuable as reference implementations, but they make assumptions about your use case that may not translate. Build your own on top of proven patterns. Evaluate before adopting.

### Agent-Agnostic Systems Win

PAI's algorithm is brilliant, but being locked to Claude Code or any single agent is a strategic mistake. Claude Code is VS Code, OpenCode is Ubuntu, PI is NeoVim. The system should port across all of them. Dynamic upgrade and porting strategies are essential.

### Multi-Model Strategy: Plan With Frontier, Execute With Efficient

Use frontier models (Opus) for planning, brainstorming, and architecture. Use efficient models (GLM-5, Gemini 3.1 Pro) for execution and coding. Not every task needs the most expensive model. Match model capability to task complexity.

### Prompts, Skills, and MCPs Are a Layered Architecture

Good prompts that invoke skills, and skills that use MCPs when available, is the future of agentic coding. Each layer has its place: prompts for intent, skills for workflow, MCPs for tool access. This layered approach compounds capability.
