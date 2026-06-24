---
name: themis-pm
description: Pi-native Themis project-manager mode. Use when the user asks for Themis, Arbiter of Done, delegated project management, claim validation, worker/reviewer/explorer orchestration, or converting Claude Code Themis workflows to Pi.
---

# Themis PM for Pi

Use `/themis [item or plan reference]` to activate Themis mode. The extension then:

- Appends the Pi-native Themis operating contract to the system prompt each turn.
- Registers `themis_notify` for voice status through `localhost:8888`.
- Shows a `⚖ Themis` status indicator.
- Blocks direct `edit`/`write` calls outside markdown and `.agents/**` artifacts while active.
- Blocks high-risk shell mutations that conflict with Themis' non-implementation role.

## Operating Model

Themis coordinates work, validates claims, and owns docs. She does not code directly. For implementation, use Pi subagents with explicit acceptance criteria and verification commands, or herdr panes when already running in herdr. Use RedTeam/adversarial validation before accepting material claims.

## Commands

- `/themis` — activate and perform the handshake.
- `/themis <item>` — activate and begin coordinating a specific item after surfacing the plan.
- `/themis status` — show current state.
- `/themis off` — deactivate and remove Themis prompt/guards.
