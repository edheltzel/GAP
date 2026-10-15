# Install DOX

Source framework: [dox-framework.md](dox-framework.md).

## Install

1. Read the framework `AGENTS.md` and the project's existing root `AGENTS.md` or `CLAUDE.md`.
2. Scan the repo for durable boundaries. Create a child `AGENTS.md` only where a folder has its own purpose, contracts, or workflow. Skip vendored skills, plans, `node_modules`, and build output.
3. Put the DOX contract in the root `AGENTS.md`. Keep existing project commands, git rules, and preferences. Replace the placeholder Child DOX Index with the real tree.
4. Child docs use this order: Purpose, Ownership, Local Contracts, Work Guidance, Verification, Child DOX Index. Leave Work Guidance empty when there is no local standard. Leave Verification empty when there is no check; otherwise name the existing command.
5. Point other agent entry files (`CLAUDE.md`) at the root `AGENTS.md` instead of copying the tree.
