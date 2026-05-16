<!-- BEGIN COMPOUND PI TOOL MAP -->
## Compound Engineering (Pi compatibility)

This block is managed by compound-plugin.

Pi extensions used by this plugin:
- Required: `pi-subagents` (by nicobailon) provides the `subagent` tool used by skills that dispatch parallel agents
- Recommended: `pi-ask-user` (by edlsh) provides the `ask_user` tool; skills fall back to numbered options in chat when it is missing

Install with:
  pi install npm:pi-subagents
  pi install npm:pi-ask-user
<!-- END COMPOUND PI TOOL MAP -->

## Atlas Flow safety note

Before changing Pi's global package set or implementing `/atlas-flow`, read `/Users/ed/Developer/atlas-config/.atlas/plans/2026-05-16-atlas-flow-research-digest.md` for the minimal-global baseline, flow manifest schema direction, safety rules, and GitHub issue roadmap.
