# Skill Review

Review Agent Skill PRs for structural validity, design quality, and marketplace consistency. This mode runs only on explicit request: `/do-skill-craft review [output-path] [--base <branch>]`.

Apply [writing-principles.md](writing-principles.md) when judging description quality, information hierarchy, leading words, and failure modes. The checks below are the procedure; those principles are the design bar.

## Arguments

- `output-path` (also `$ARGUMENTS`): file for review results. If omitted, default to `.review-output.md`.
- `--base <branch>`: base branch for diff comparison (default: `main`)

## Workflow

### Step 1: Parse Arguments

Extract the output path from `$ARGUMENTS`. If no path is provided, default to `.review-output.md`. Extract `--base` if provided, otherwise default to `main`.

### Step 2: Identify Changed Skills

```bash
git diff --name-only $(git merge-base HEAD <base>)..<HEAD> | grep -E '(SKILL\.md|skills/[^/]+/)'
```

Group changed files by skill directory. Each unique skill directory is a review target. If no skill files changed, write "No skill files changed" to the output path and stop.

### Step 3: Read Each Changed Skill End-to-End

For each changed skill:
1. Read the full `SKILL.md` (not just diff lines)
2. Read all files in the skill directory (references, scripts)
3. Note the parent plugin directory and locate its `plugin.json`

### Step 4: Run Structural Checks (HIGH Confidence)

Apply every check in [structural-checks.md](structural-checks.md). These are binary pass/fail — violations are clear-cut.

Structural violations are HIGH confidence because they can be verified mechanically.

### Step 5: Run Design Checks (MEDIUM Confidence)

Apply every check in [design-checks.md](design-checks.md), and apply [writing-principles.md](writing-principles.md) as the design bar. These require judgment — flag only when the issue is clear, not when the approach is merely different from what you'd choose.

Design issues are MEDIUM confidence because they involve subjective assessment.

### Step 6: Run Marketplace Checks (HIGH Confidence)

Apply every check in [marketplace-checks.md](marketplace-checks.md). These verify the skill integrates correctly with the marketplace it belongs to.

Marketplace violations are HIGH confidence because they can be verified by reading manifest files.

### Step 7: Load and Apply Verification Protocol

Load [review-verification-protocol.md](review-verification-protocol.md). Before reporting any finding, verify:

1. You read the actual skill content, not just the diff context
2. The issue is real, not a style preference
3. The issue applies to skill files specifically (not general code review concerns)
4. You can point to the specific line that proves the issue

Remove any finding you cannot verify.

### Step 8: Write Output

Write all findings to the output path specified in Step 1, using the exact format below.

## Output Format

```markdown
## Review Summary

[1-2 sentence overview of findings across all reviewed skills]

## Issues

### Critical (Blocking)

1. [FILE:LINE] ISSUE_TITLE
   - Issue: Description of what's wrong
   - Why: Why this matters for skill quality or marketplace health
   - Fix: Specific recommended fix
   - Confidence: HIGH

### Major (Should Fix)

N. [FILE:LINE] ISSUE_TITLE
   - Issue: ...
   - Why: ...
   - Fix: ...
   - Confidence: HIGH|MEDIUM

### Minor (Nice to Have)

N. [FILE:LINE] ISSUE_TITLE
   - Issue: ...
   - Why: ...
   - Fix: ...
   - Confidence: HIGH|MEDIUM

### Informational (For Awareness)

N. [FILE:LINE] SUGGESTION_TITLE
   - Suggestion: ...
   - Rationale: ...
   - Confidence: MEDIUM

## Verdict

Ready: Yes | No | With fixes 1-N
Rationale: [1-2 sentences — only Critical and Major items block approval]
```

Every issue gets a sequential number. Every issue includes `Confidence: HIGH|MEDIUM`. The Verdict ignores Minor and Informational items.

## Severity Calibration

### Critical (Block Merge)

- Invalid or missing YAML frontmatter (skill won't load)
- Missing required `name` or `description` field
- SKILL.md exceeds 500 lines (performance degradation)
- Name collision with existing skill in the marketplace

### Major (Should Fix)

- `name` violates format rules (not kebab-case, too long, reserved word)
- `description` missing "what" or "when" component, wrong person, or exceeds 1024 chars
- Nested reference chains (references that reference other references)
- Description too vague to trigger accurately
- Structured output with no format template or example
- Windows-style paths in file references

### Minor (Nice to Have)

- Time-sensitive content (hardcoded dates, "recently", "new")
- Inconsistent terminology across skill files
- Progressive disclosure not used when SKILL.md is dense
- Workflows missing validation steps
- Missing cross-reference (SKILL.md links to file that doesn't exist)

### Informational (For Awareness)

- Trigger keyword overlap with existing marketplace skills
- Suggestions for additional reference files
- Opportunities to improve description specificity
- Script lacks `--help` or uses interactive prompts

## Re-Review Rules

On subsequent review passes after fixes are applied:
1. ONLY verify that previously flagged issues were addressed correctly
2. Do NOT introduce new findings unrelated to previous issues
3. Accept Minor items that weren't fixed — do not re-flag
4. The goal of re-review is verification, not discovery

## References

- **Writing principles**: [writing-principles.md](writing-principles.md). Apply when judging design.
- **Structural checks**: [structural-checks.md](structural-checks.md) — frontmatter, naming, line limits, path format
- **Design checks**: [design-checks.md](design-checks.md) — description quality, progressive disclosure, degrees of freedom
- **Marketplace checks**: [marketplace-checks.md](marketplace-checks.md) — name collision, plugin.json consistency, trigger overlap
