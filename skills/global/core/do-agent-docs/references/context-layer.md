# Agent Context Layer

Structure the documentation and context of a code repository so an agent can build an accurate
mental model of the project fast, then load only what the current task needs. This is the
**Interpretable Context Methodology (ICM)** by Jake Van Clief (also published as the Model
Workspace Protocol) applied to the context layer of a code repo.

Think of the repo's docs as a library the agent walks. The entry file is the catalogue: small,
stable, it points at everything and stores almost nothing. Content lives on the shelves. The
agent starts at the catalogue, follows one or two pointers, and stops as soon as it has enough.

## Scope

This skill owns the **context/documentation layer** only. It does not turn a code repo into a
staged pipeline.

- For the **code side** of agent-first work (repo as single source of truth, mechanical
  enforcement of architecture via lint/CI, entropy management, quality scoring), use
  [agent-first-repo.md](agent-first-repo.md).

## The three rules that do the work

These are load-bearing. Applied on their own, they deliver most of the value.

- **Pointers over payload.** `AGENTS.md` and index files are a catalogue: they point at
  everything and hold almost nothing. Target a thin entry file (~100 lines). The moment a
  routing file starts carrying the actual rules, definitions, or examples, that content belongs
  in a reference doc with a pointer left behind.

- **One home per fact.** Every fact has a single authoritative home; other files link to it.
  Two authoritative copies always drift, and an agent following the stale one propagates the
  error. Smell test: search for a distinctive phrase; if it appears in two files and both mean
  to be authoritative, one must become a pointer.

- **Name the section, not just the file.** When a pointer sends the agent to a large doc, say
  which section to read. A 400-line design doc may hold 40 relevant lines; point at those.

As a rough heuristic (borrowed from ICM's content-pipeline work, not a measured figure for code
repos), a well-scoped task read lands the agent in a few thousand tokens rather than tens of
thousands. Treat it as a smell, not a limit: a coding agent legitimately loads a whole file, its
test, and two callers. If assembling routine context needs *far* more than expected, the layering
is too flat or a routing file is carrying payload it should have delegated.

## The layer model (an optional lens)

ICM describes context as five layers the agent reads down. You do **not** need the layer numbers
to apply the three rules above; the numbers are a mental model for where a doc sits, not a
mechanism. A code repo has no execution stages, so the mapping is not one-to-one (Layer 2, the
stage control point, collapses into Layer 1 unless the repo has real sub-domains).

For the layer-to-artifact mapping, factory-vs-product separation, directory structure, index
files, and freshness validation, see [context-layers.md](context-layers.md).

## Honest limits: what docs cannot enforce

This is the discipline that keeps the context layer from becoming a liability.

**CI enforces structure, not truth.** A freshness job can check that links resolve and every
index lists its directory. It cannot check whether a doc still *describes the code*. A design doc
that says "OAuth2 + session tokens" after the code moved to JWT is still valid markdown with
working links and a complete index: every mechanical check passes while the doc lies.

**Progressive disclosure raises the stakes of that lie.** Routing an agent to a single canonical
doc and telling it to stop means it reads fewer corroborating sources. Scattered copies at least
give the agent a chance to notice a contradiction; a lone canonical source removes that chance,
and the agent trusts it *more* because it is labeled authoritative. A stale canonical doc
therefore misleads worse than no doc would. The repo's own maxim holds: stale docs are worse
than no docs, because agents follow them faithfully.

**So:**
- Keep the canonical set small. Every authoritative doc is a liability proportional to how
  authoritative it looks.
- Put load-bearing invariants under mechanical enforcement where you can, so truth is checked by
  a test rather than trusted from prose (see [agent-first-repo.md](agent-first-repo.md)).
- Treat a green freshness build as "structurally intact," never as "accurate."

## Companion references

| Topic | Reference | What it covers |
|---|---|---|
| Code-side agent-first work | [agent-first-repo.md](agent-first-repo.md) | Single source of truth, mechanical enforcement, entropy management, quality scoring |
| Writing the entry file | [agents-md.md](agents-md.md) | Structure, sections, anti-patterns for AGENTS.md |
| Architecture docs | `do-architecture-md` skill, if installed | Codemap with boundaries, invariants, cross-cutting concerns |

## Workflow

### New repo
1. Write `AGENTS.md` as a thin routing catalogue: dev commands, repo map, and a "you want to X,
   go to Y" pointer table. Pointers, not payload.
2. Add docs only as complexity demands. Do not build the full `docs/` tree on day one.
3. Give any directory with 3+ docs an `index.md` catalogue.
4. For every canonical doc, ask what mechanical check could guard its central claim, and add it.

### Existing repo
1. Inventory the docs. Find facts stated in two authoritative places; pick one home, link the rest.
2. Thin the entry file: move payload to reference docs, leave pointers.
3. Add section-scoped pointers where large docs are referenced whole.
4. Identify the canonical docs whose staleness would mislead an agent most, and either shrink the
   canonical set or back the claim with a test.
