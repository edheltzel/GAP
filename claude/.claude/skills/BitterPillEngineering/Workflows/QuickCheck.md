# QuickCheck Workflow

Audit one file or supplied rule block.

## Input

Accept a file path, an inline instruction block, or a file already present in context.

## Steps

1. Read the complete target.
2. Identify independently actionable rules.
3. Apply the six tests from `../SKILL.md`.
4. Check the nearest applicable parent instructions for contradictions and duplication.
5. Return a concise, read-only verdict.

## Output

```text
File: [path or inline]
Rules found: [count]
Verdict: [X] keep, [Y] cut, [Z] sharpen, [N] other

### Cut
- [rule]: [reason]

### Sharpen
- [rule]: [testable replacement]

### Keep
- [rule]: [load-bearing purpose]

### Uncertain
- [claim]: [evidence needed]
```
