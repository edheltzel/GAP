# Red Team Philosophy

<!-- markdownlint-disable MD013 -->

## Origin

Military red teaming: dedicated teams attack plans, strategies, and assumptions to find vulnerabilities before an opponent or reality does.

## Core insight

**The goal is not destruction.** Find the fundamental flaw that, if challenged, causes the structure to collapse or reveals where it must be reinforced.

The most powerful critique is usually one core issue:

- A hidden assumption that is false.
- A logical step that does not follow.
- A category error: treating X like Y.
- An ignored precedent that directly contradicts the argument.
- A second-order effect the plan cannot survive.

## Success criteria

- The steelman is strong enough that a proponent would say, “Yes, that is my argument.”
- The counter-argument defeats the steelman, not a weaker strawman.
- Multiple independent perspectives converge on the same core insights.
- A lone non-convergent finding still survives when its evidence and consequence are strong.
- The reader says, “I had not thought of that.”
- Weaknesses include remediation paths when possible.

## The 32 persona types

| Family | Count | Focus |
| --- | ---: | --- |
| **Engineers** | 8 | Technical and logical rigor |
| **Architects** | 8 | Structural and systemic issues |
| **Pentesters** | 8 | Adversarial thinking against the argument, not computer systems |
| **Interns** | 8 | Fresh perspectives and naive questions |

The complete roster and attack angles are in [`Workflows/ParallelAnalysis.md`](Workflows/ParallelAnalysis.md).

Use actual Pi subagents when available. If subagent dispatch is unavailable or disproportionate, simulate a representative set inline, preserve the same synthesis contract, and disclose the fallback.
