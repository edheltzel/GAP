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
A skill listed in `global/skills.json`. `gap sync` installs it when it's missing and checks it for drift from its source. A skill the manifest doesn't list is unmanaged, and `gap` never touches it.

## Decisions

### HIL
Human in the loop. A person takes part before an automated step does something consequential. Silence is not agreement. A run with nobody to answer does not proceed as if they had agreed.

Publish replacing a harness file that differs from the current policy, and is not the last published copy, is one such step.
*Avoid:* human-in-the-loop

## Flagged ambiguities

- "ADE" has also been expanded as AI/Agent Development Environment. This file uses Agentic Development Environment.
- A product name is not a harness, a model, and a provider at once. Say which one you mean.
