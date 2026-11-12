# Concepts

Shared domain vocabulary for this project — entities, named processes, and status concepts with project-specific meaning. Seeded with core domain vocabulary, then accretes as ce-compound and ce-compound-refresh process learnings; direct edits are fine. Glossary only, not a spec or catch-all.

## Environment

An ADE contains harnesses. A harness loads skills, commands, and a model. A provider serves models. HIL is the person, not a feature of the harness.

### ADE
Agentic Development Environment. The whole working setup: harnesses, the skills and commands they load, the models they call, and the person in the loop. It is not one harness.
*Avoid:* AI/Agent Development Environment

### Harness
The app you work in, the way a text editor is the app. It loads skills, commands, and a model. It is not the model, and it is not the provider.

### Skill
A packaged capability a harness can load, the way a plugin extends a text editor. A skill that is not installed does not exist. A skill is not a command.

### Command
A short saved invocation in a harness, the way a snippet is a saved edit. A slash command is a command. It is not a skill.

### Model
The mind that proposes the next action inside a harness. Changing the model does not change the harness. The model is not the provider.

### Provider
Who serves a model, and who holds the account, quota, and auth for that call. One provider can serve many models. The provider is not the harness.

## Policy

### GAP
Global Agent Policies: the shared agent policy for coding agents. A nearer project policy wins on that project's local contracts. GAP is edited in one place and published outward. It is not a Stow package, and it is not a second full copy pasted into the home directory.

### Depth note
A longer explanation linked from the GAP policy. It is not the policy contract.

### Publish
The step that copies the current GAP policy into the files each harness already reads. A harness that already reads another harness's copy does not get its own.
*Avoid:* sync

`gap sync` runs this step and manages skills. Publish names only the policy-copying step.

### Managed skill
A skill selected by a component, through its `gap-utility.toml` or its `names` in `gap.toml`, or by `global/skills.json`. See [Manage skills](README.md#manage-skills) for installation and drift handling.

### Utility
A repo that ships skills, commands, or its own tooling and describes them in a manifest, the way a lazy.nvim plugin describes itself. The distro's components are utilities too. User utilities have their own [lifecycle](README.md#add-a-utility).
*Avoid:* package, plugin

### Manifest
The `gap-utility.toml` at a utility's repo root that lists its skills, commands, and hooks. See [Publish a utility](README.md#publish-a-utility). Not `global/skills.json`, which selects skills by source.

### Hook
A shell command a utility's manifest runs at install, update, uninstall, or doctor time. See [Publish a utility](README.md#publish-a-utility) for timing and checkout selection. A hook is not a skill and not a command.

### User lock
`~/.config/gap/gap.lock`: the pins for utilities added with `gap add`. It is the user's own, like `lazy-lock.json`. It is not the distro's `gap.lock`.

### Trust
Permission for a user utility to run hooks without approval for each execution. See [Hooks and trust](README.md#hooks-and-trust) for configuration and approval rules.

## Decisions

### HIL
Human in the loop. A person takes part before an automated step does something consequential. Silence is not agreement. A run with nobody to answer does not proceed as if they had agreed.

GAP's [foreign-file confirmation](README.md#step-4-answer-if-gap-sync-asks-about-a-file) is one such step.
*Avoid:* human-in-the-loop

## Distro

The distro is this repo's `gap.toml` and `gap.lock`. It names components; overrides and profiles change what one machine or project takes from it.

### Component
A repo the distro selects in `gap.toml`, with its skill `names` and optional `install` and `doctor` commands. A component is installed from its pin, never from a moving branch.

### Pin
The exact commit a component installs, recorded in `gap.lock`. Only `gap pin <id>@<ref>` moves it. A ref is the branch or tag the pin was resolved from; the pin is the commit.

### Store
`~/.gap/store/<id>/<commit>`: a verified checkout of one pin. Skills are copied from the store, and a store whose files no longer match the pin is not used.

### Drift
An installed skill that differs from its source. For selected component skills, `gap status` reports drift and `gap restore` repairs it. Use `gap sync` to review and repair drift in additional skills from `global/skills.json`; sync asks before replacing drifted skills from either source.

### Load level
How one component skill installs: `listed` or `always` in the home folders, `project` only in a matching project, `command` or `off` not at all.

### Override
A later `gap.toml`, in `~/.config/gap/` or a project's `.gap/`, that can disable a component or change its ref or a skill's load level. It never writes `gap.lock`.

### Profile
A named override shipped in `profiles/`. `crew` writes project files only, for cloud agents.

## Flagged ambiguities

- "ADE" has also been expanded as AI/Agent Development Environment. This file uses Agentic Development Environment.
- A product name is not a harness, a model, and a provider at once. Say which one you mean.
- "Command" here means a harness slash command. A `gap` subcommand such as `gap sync` is a CLI command; name it in code.
