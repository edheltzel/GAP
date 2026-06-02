# Red Team Philosophy

## Origin

Military red teaming: dedicated teams attack plans, strategies, and assumptions to find vulnerabilities before an opponent or reality does.

## Core insight

**The goal is not destruction.** The goal is finding the fundamental flaw that, if challenged, causes the structure to collapse or reveals where it must be reinforced.

The most powerful critique is usually one core issue:

- A hidden assumption that is false.
- A logical step that does not follow.
- A category error: treating X like Y.
- An ignored precedent that directly contradicts the argument.
- A second-order effect the plan cannot survive.

## Success criteria

- The steelman is strong enough that a proponent would say: "yes, that's my argument."
- The counter-argument defeats the steelman, not a weaker strawman.
- Multiple perspectives converge on the same core insights.
- The reader says: "I hadn't thought of that."
- Weaknesses include remediation paths when possible.

## Agent/persona types

| Type | Count | Focus |
|---|---:|---|
| Engineers | 8 | Technical and logical rigor |
| Architects | 8 | Structural and systemic issues |
| Pentesters | 8 | Adversarial thinking against the argument, not systems |
| Interns | 8 | Fresh eyes and naive questions |

Use actual Pi subagents when available. If subagent dispatch is unavailable or overkill, simulate these personas inline and disclose the fallback.
