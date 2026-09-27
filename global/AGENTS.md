# AGENTS.md

GAP is the global agent policy for Ed's coding agents. This file is the contract. It is not a Stow package.

The nearest project `AGENTS.md` wins on that project's local contracts. `~/AGENTS.md` is only a pointer here. Do not paste a second full copy there.

## How this file is loaded

Edit the markdown in this repo, then run `npm run sync` from the repo root. The script writes regular file copies. It does not symlink, and it does not use Stow.

- Claude reads `~/.claude/rules/global-agent-policy.md`. Do not also point `~/CLAUDE.md` here. An ancestor `CLAUDE.md` suppresses project `AGENTS.md`.
- Pi reads `~/.pi/agent/APPEND_SYSTEM.md`. Pi does not scan `rules/`. Do not add `SYSTEM.md`.
- OMP reads `~/.omp/agent/APPEND_SYSTEM.md`. Do not add `SYSTEM.md`.
- Codex reads `AGENTS.md` in Codex home (`~/.codex`, or `CODEX_HOME`). Do not add `AGENTS.override.md`.
- Cursor reads `~/.cursor/rules/global-agent-policy.mdc`.
- Grok loads `~/.claude/rules/*.md` when Claude compatibility is on. Do not add a second copy.

Pi packaging, themes, and footers stay in `~/.pi/agent/AGENTS.md`. Do not paste them here.

## Style

- No em dashes. Use a plain hyphen.
- Never auto-add an agent name as a commit co-author.
- Never manually edit `CHANGELOG.md` or any file marked auto-generated.
- Prefer quality and maintainability over a cheaper build.
- Meet the ask with the smallest change that works.

## Before editing

Read the root `AGENTS.md`, then every `AGENTS.md` on the path to the files you touch. Closer docs win on local detail. No child weakens that. Re-read that chain in the current session. Do not rely on memory. Full habit: [dox.md](docs/agents/dox.md).

## Work

- Reproduce a bug the way an end user hits it before fixing it. A claim is not proof. Details: [verification.md](docs/agents/verification.md).
- Before a non-trivial edit, use codegraph when the project has an index, and name the blast radius or why you skipped it. Details: [codegraph-blast.md](docs/agents/codegraph-blast.md).
- Investigation is not permission to change code. Details: [scout.md](docs/agents/scout.md).
- Do not invent skills, slash commands, or APIs that are not installed. Ask before a large swarm. Details: [skills-honesty.md](docs/agents/skills-honesty.md).
- Do not silently pick git versus GitButler. The nearest project contract wins. Otherwise use git and worktrees. If the project requires GitButler, use `but` and do not invent a plain-git workaround. Details: [vcs-project-wins.md](docs/agents/vcs-project-wins.md).
- Follow the project's handoff location. If it names none, use `docs/plans/handoffs/`. Never write a handoff only to `/tmp` or a harness install directory. Details: [handoffs.md](docs/agents/handoffs.md).
- Use Recall before inventing a memory store. Details: [recall.md](docs/agents/recall.md).
- If the ask would delete or replace a tree it did not name, stop and confirm the exact paths. When Ed asks to delete, use `trash`, not `rm`.

## Not here

Harness packaging, per-repo product contracts, and money or merge moves without an explicit go. Details: [out-of-scope.md](docs/agents/out-of-scope.md).
