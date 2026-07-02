# Triage Labels

The skills speak in terms of triage roles. This file maps those roles to the actual `:`-namespaced label strings used in this repo's issue tracker. The canonical full taxonomy lives in `skills/pm-tools/references/Labels.md`; this file covers the workflow-state (who-acts-next) subset.

| Role (mattpocock/skills vocab) | Label in our tracker | Meaning                                  |
| ------------------------------ | -------------------- | ---------------------------------------- |
| `needs-triage`                 | `needs:triage`       | Maintainer needs to evaluate this issue  |
| `needs-info`                   | `needs:info`         | Waiting on reporter for more information |
| `ready-for-agent`              | `agent:ready`        | Fully specified, ready for an AFK agent  |
| `ready-for-human`              | `needs:human`        | Requires human implementation            |
| `wontfix`                      | `wontfix`            | Will not be actioned                     |

Related agent-lifecycle labels (not strictly triage, but part of the `agent:` namespace): `agent:blocked`, `agent:complete`.

When a skill mentions a role (e.g. "apply the AFK-ready triage label"), use the corresponding label string from this table.

> Migration note: older repos may still carry `needs-triage` / `needs-info` (hyphen). The canon is the colon form; the label-sync target migrates them.
