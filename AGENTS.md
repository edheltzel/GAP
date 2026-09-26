# AGENTS.md

This repo publishes shared agent rules. `global/AGENTS.md` is that contract. Depth docs live in `global/docs/agents/`.

This file only covers how to change the repo. Shared behavior rules are in `global/AGENTS.md`.

## Publish

`npm run sync` copies `global/AGENTS.md` into the files Claude, Pi, and OMP already read:

- `~/.claude/rules/global-agent-instructions.md`
- `~/.pi/agent/APPEND_SYSTEM.md`
- `~/.omp/agent/APPEND_SYSTEM.md`

The script writes regular files. Do not symlink those paths. Do not add Stow.

After editing shared markdown, run `npm run sync`. A copy is stale until that command runs.

## Edit

- Change a shared rule in `global/`, then sync. Do not paste the rule into a harness home by hand.
- Add a harness destination in `src/sync.ts` only when that harness actually loads the path.
- Node 24 or newer. No extra runtime dependencies.
