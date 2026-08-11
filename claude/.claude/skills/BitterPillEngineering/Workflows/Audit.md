# Audit Workflow

Run a full audit of the current instruction hierarchy.

## Steps

1. **Define scope.** Name the repository, harness, and target behavior being audited.
2. **Discover active instructions.** Start with active harness instructions, then walk the repository hierarchy for applicable `AGENTS.md`, `CLAUDE.md`, and equivalent files. Inspect configuration, hooks, manifests, and routing tables only when present.
3. **Record loading evidence.** For each source, state why it applies. Distinguish force-loaded, path-scoped, on-demand, generated, and merely present files.
4. **Read each applicable source completely.** Count files, lines, and independently actionable rules.
5. **Apply the six tests.** Use the tests and classifications in `../SKILL.md`.
6. **Check cross-source conflicts.** Compare scope, precedence, duplicated contracts, stale paths, retired tools, and conflicting output requirements.
7. **Estimate context value.** Identify always-loaded material that is rarely actionable and could move to on-demand guidance.
8. **Report uncertainty.** Do not invent settings semantics or model defaults. Mark anything not verified.
9. **Produce the report.** Use the output format in `../SKILL.md`.
10. **Offer edits separately.** Change files only after explicit user direction.

## Verification

Before reporting, confirm that every cited source exists, every claimed configuration key was observed, and every proposed deletion preserves higher-priority safety and tool contracts.
