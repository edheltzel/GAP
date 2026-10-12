# AGENTS.md

GAP is the global agent policy for coding agents. This file is the contract. It is not a Stow package.

The nearest project `AGENTS.md` wins on that project's local contracts. `~/AGENTS.md` is only a pointer here. Do not paste a second full copy there.

## How this file is loaded

Edit the markdown in this repo, then run `npm run sync` from the repo root. The script writes regular file copies. It does not symlink, and it does not use Stow.

- Claude reads `~/.claude/rules/global-agent-policy.md`. Do not also point `~/CLAUDE.md` here. An ancestor `CLAUDE.md` suppresses project `AGENTS.md`.
- Pi reads `~/.pi/agent/APPEND_SYSTEM.md`. Pi does not scan `rules/`. Do not add `SYSTEM.md`.
- OMP reads `~/.omp/agent/APPEND_SYSTEM.md`. Do not add `SYSTEM.md`.
- Codex gets the policy from SessionStart and SubagentStart hooks that print `gap/global-agent-policy.md` in Codex home (`~/.codex`, or `CODEX_HOME`). Old policy text may remain in that `AGENTS.md` until cleanup, so Codex can load the policy twice. Do not add `AGENTS.override.md`.
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
- Use GitButler (`but`) for branches, commits, and pushes. Never create a native git worktree: no `git worktree add` and no harness option that isolates work in a worktree. A project contract that names plain git wins on commit tooling, never on worktrees. Details: [vcs-project-wins.md](docs/agents/vcs-project-wins.md).
- Every new repo uses `master` as its default branch, never `main`. This is mandatory for every creation path: local init, `gh repo create`, the GitHub UI, or a template. For a local repo, run `git init -b master`. If GitHub created the repo with `main`, rename it to `master` before any other work. After the first push, confirm the GitHub default branch is `master`.
- Follow the project's handoff location. If it names none, use `docs/plans/handoffs/`. Never write a handoff only to `/tmp` or a harness install directory. Details: [handoffs.md](docs/agents/handoffs.md).
- Track follow-up tasks in `~/todo.txt` through the `tuxedo` CLI (`do-todo-txt` skill when installed). Every plan, spec, and handoff carries a todo.txt section, and its tasks go into the list. Details: [todo-txt.md](docs/agents/todo-txt.md).
- Use Recall before inventing a memory store. Details: [recall.md](docs/agents/recall.md).
- If the ask would delete or replace a tree it did not name, stop and confirm the exact paths. When asked to delete, use `trash`, not `rm`.

## Not here

Harness packaging, per-repo product contracts, and money or merge moves without an explicit go. Details: [out-of-scope.md](docs/agents/out-of-scope.md).
