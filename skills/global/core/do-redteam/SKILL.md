---
name: do-redteam
description: >-
  Attack an idea, judge one named option against the evidence, or develop
  competing alternatives and synthesize a choice. Use when the user wants a
  red team, a devil's advocate, a stress test, the strongest objection, a
  verdict on a named candidate, or a choice that needs alternatives built
  out first. NOT FOR code diffs (`do-adversarial-review`, if installed) or
  instruction-file bloat such as CLAUDE.md (`do-bitter-pill`, if installed).
---

# Red team

Attack ideas, strategies, and plans. It can also judge one named option, or develop competing alternatives and then choose. It does not implement the winner.

## Route

| Request | Workflow |
|---------|----------|
| Stress-test an existing argument | [ParallelAnalysis](Workflows/ParallelAnalysis.md) |
| A defined goal whose alternatives still need to be built before a fair choice | [AdversarialValidation](Workflows/AdversarialValidation.md) |
| A verdict on one named option | Judge, below |

A code diff goes to `do-adversarial-review`, if installed. Instruction-file bloat goes to `do-bitter-pill`, if installed.

## Judge

Verdict on one named candidate against evidence. Do not implement, and do not invent rival candidates.

1. Name the option and the decision it would settle. Do not interview the user to discover the question. If a missing fact would change the verdict and you cannot find it, return blocked: what is missing, why it matters, and what would resolve it.
2. Ground the verdict in the project's own evidence: the code, tests, docs, and recorded decisions. A claim in the conversation is a pointer to check, not proof. Check any external claim the verdict depends on.
3. Lead with the verdict. Then the evidence, the tradeoff that could flip it, and what is still uncertain. Identifiers must make sense without reopening the subject.

Done when the verdict is delivered, or when you returned blocked. The caller decides what to do next. This mode does not implement.

## Parallel analysis

The deliverable is a steelman a proponent would accept, and the strongest counter-argument that beats that steelman, not a strawman. Each is 8 points, 12-16 words a point, ranked by severity, with a remediation for each real weakness. Volume is not a finding. One core flaw beats a pile of nits.

Before the attack, use `do-first-principles`, if installed, to split the argument and class constraints as hard, soft, or assumption. See [Integration](Integration.md) and [Philosophy](Philosophy.md).

## Adversarial validation

Develop independent alternatives to one brief, compare them, and synthesize one coherent approach. The caller adopts it and does the later work. Details: [AdversarialValidation](Workflows/AdversarialValidation.md).

One named option that only needs a verdict is Judge. Several options that are already concrete skip generation and go straight to compare-and-synthesize. A settled decision stays settled.
