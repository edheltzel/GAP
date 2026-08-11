# The 8 Lenses of Iterative Depth

Each lens is a structured perspective that forces exploration of a problem from a fundamentally different angle. They are ordered from most concrete to most abstract, and from most commonly useful to most specialized.

---

## Lens 1: LITERAL (Surface Requirements)

**Question:** "What did they explicitly say? What are the concrete, stated requirements?"
**Grounded in:** Requirements elicitation fundamentals
**Focus:** Parse the exact words. Identify every stated requirement, constraint, preference. Do not interpret beyond what was said.
**Criteria output:** Criteria for every explicitly stated requirement.
**Example prompt variation:** "List every concrete, testable requirement explicitly stated in this request. Do not infer. Only extract."

---

## Lens 2: STAKEHOLDER (Who Else Cares?)

**Question:** "Who are all the people, systems, and entities affected by this? What does each need?"
**Grounded in:** Viewpoint-Oriented RE (Finkelstein & Nuseibeh), Triangulation (Denzin)
**Focus:** Identify every stakeholder beyond the requester. End users, maintainers, administrators, downstream systems, and future developers may add needs that were not stated.
**Criteria output:** Criteria for stakeholder needs not in the original request.
**Example prompt variation:** "Identify every stakeholder affected by this work. For each, what requirement would they add that the requester did not mention?"

---

## Lens 3: FAILURE (What Goes Wrong?)

**Question:** "What could fail? What would an adversary exploit? What are the edge cases?"
**Grounded in:** Misuse Cases (Sindre & Opdahl), Pre-Mortem (Klein), STRIDE Threat Modeling
**Focus:** Assume the solution exists, then break it. Examine error states, race conditions, security holes, data corruption, user confusion, and performance under load.
**Criteria output:** Anti-criteria and defensive criteria.
**Example prompt variation:** "This solution ships tomorrow. List every way it fails in the first week. Be adversarial."

---

## Lens 4: TEMPORAL (Past, Present, Future)

**Question:** "How does this change over time? What is the history? What happens in six months?"
**Grounded in:** Causal Layered Analysis (Inayatullah), Progressive Elaboration (PMBOK)
**Focus:** Ask why the problem exists now, what was tried before, and what future changes could break the solution. Include migration, backwards compatibility, and scale changes.
**Criteria output:** Criteria for durability, migration, and future-proofing.
**Example prompt variation:** "What context created this request? What will change in 3 to 12 months that could invalidate this solution?"

---

## Lens 5: EXPERIENTIAL (How Should It Feel?)

**Question:** "When this works perfectly, how does the user feel? What is the experience?"
**Grounded in:** Appreciative Inquiry (Cooperrider), de Bono Red Hat (emotions)
**Focus:** Go beyond functional correctness to speed, elegance, confidence, trust, and the difference between working and working beautifully.
**Criteria output:** Quality-of-experience criteria.
**Example prompt variation:** "Describe the perfect user experience. What distinguishes exactly right from technically functional?"

---

## Lens 6: CONSTRAINT INVERSION (What If?)

**Question:** "What if we removed all constraints? What if we added extreme ones?"
**Grounded in:** TRIZ (Altshuller), Lateral Thinking (de Bono), Reframing (Dorst)
**Focus:** Remove assumed constraints, then impose extreme constraints such as offline operation, a 100ms limit, or zero dependencies. Both directions reveal hidden assumptions.
**Criteria output:** Criteria that challenge assumptions and reveal what is essential.
**Example prompt variation:** "Which constraints are assumed rather than stated? Remove them, then impose extreme constraints. What remains essential?"

---

## Lens 7: ANALOGICAL (What Patterns Apply?)

**Question:** "What similar problems have been solved before? What patterns from other domains apply?"
**Grounded in:** Cognitive Flexibility Theory (Spiro), Cross-Domain Transfer
**Focus:** Find analogous problems in other codebases, industries, or fields. Extract patterns that worked and mistakes that recurred.
**Criteria output:** Criteria derived from proven patterns and analogous lessons.
**Example prompt variation:** "What are three to five analogous problems? What criteria do their successful solutions imply here?"

---

## Lens 8: META (Is This the Right Question?)

**Question:** "Are we solving the right problem? Is the framing itself correct?"
**Grounded in:** Hermeneutic Circle (Gadamer), Double-Loop Learning (Argyris), Soft Systems Methodology (Checkland)
**Focus:** Ask whether the request is a symptom of a deeper issue and whether reframing could produce a better outcome.
**Criteria output:** Criteria that reframe or expand the problem definition.
**Example prompt variation:** "What is the underlying need? Would a different question produce a better outcome than the stated request?"

---

Draw from this catalog. Pick the lenses the problem calls for in any order. A short run often benefits from the earlier concrete lenses, but no fixed sequence is required.
