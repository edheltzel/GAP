# The Context Layers and the Docs Tree

Supporting detail for [context-layer.md](context-layer.md). The three operating rules (pointers over
payload, one home per fact, name the section) and the honest limits of doc-based context live in
that file; this file does not restate them. Here is where docs sit and how the tree is
organized.

## The layer model (optional lens)

ICM describes five context layers the agent reads down, stopping when it has enough. A code repo
has no execution stages, so the mapping is not one-to-one. This is the honest correspondence:

| ICM layer | Question it answers | Loaded | Code-repo artifact |
|---|---|---|---|
| 0 Entry catalogue | Where am I? | Always | `AGENTS.md` (root) |
| 1 Routing | Where do I go? | On entry | `ARCHITECTURE.md` + `docs/**/index.md` catalogues |
| 2 Local contract | What do I do here? | Per domain | Nested `AGENTS.md` / `docs/<domain>/index.md` |
| 3 Reference (factory) | What rules apply? | Selectively | `docs/design-docs/`, `docs/references/`, `docs/generated/`, `DESIGN.md`, `SECURITY.md` |
| 4 Working artifacts (product) | What am I working with? | Selectively | `docs/exec-plans/active/`, the code and tests under change |

**Layer 2 is often empty.** In an ICM pipeline this is a stage control point. A code repo has an
analogue only when a sub-directory is a real domain boundary with its own rules; then a nested
`AGENTS.md` (or `docs/<domain>/index.md`) scopes an agent working there. If there are no such
sub-domains, Layer 2 collapses into Layer 1. Do not invent nested contracts for domains that do
not exist.

**Factory vs product (Layers 3 vs 4).** Reference material (Layer 3) is stable across every task:
design docs, conventions, schemas, security model. Working artifacts (Layer 4) are specific to the
task: the active exec-plan, the diff. Keep them structurally apart so an agent can load the stable
rules without dragging in unrelated in-flight state.

The layer numbers are a placement aid, not a mechanism. You can apply the whole skill without ever
referring to them.

## Directory structure

Start lean. Add layers only as complexity demands; do not build this whole tree on day one.

```
docs/
├── design-docs/
│   ├── index.md              # Catalogue: title, status, date, owner
│   ├── core-beliefs.md       # Foundational principles
│   ├── auth-architecture.md  # How auth works end-to-end
│   └── data-pipeline.md      # ETL design and constraints
├── exec-plans/
│   ├── active/
│   │   └── migrate-to-v2.md  # Current work with progress log
│   ├── completed/
│   │   └── initial-launch.md # Done, kept for context
│   └── tech-debt-tracker.md  # Prioritized list of known debt
├── product-specs/
│   ├── index.md              # Catalogue of product specs
│   ├── onboarding-flow.md    # User journey, acceptance criteria
│   └── billing.md            # Billing domain spec
├── references/
│   ├── stripe-llms.txt       # LLM-friendly Stripe API reference
│   └── postgres-patterns.md  # DB patterns specific to this project
└── generated/
    └── db-schema.md          # Auto-generated, never hand-edited
```

## Index files

Every directory with 3+ documents gets an `index.md` (a Layer 1 catalogue) with metadata:

```markdown
# Design Documents

| Document | Status | Last Updated | Summary |
|---|---|---|---|
| [auth-architecture.md](auth-architecture.md) | Approved | 2025-12-01 | OAuth2 + session tokens |
| [data-pipeline.md](data-pipeline.md) | Draft | 2025-11-15 | ETL from Postgres to analytics |
```

Status values: `Draft`, `In Review`, `Approved`, `Superseded`, `Deprecated`. This gives the agent
a fast scan of what exists without reading every document.

## Cross-linking

Documents reference each other by relative path. When a product spec has a technical constraint,
link to the design doc that owns it. Do not deep-link to line numbers; they go stale. Link to the
document and to a named section, and let the agent search within it.

## Freshness validation

Structural freshness is cheap to enforce and worth enforcing:

**Mechanical checks:**
- CI job that validates every `index.md` lists every document in its directory
- Lint that checks cross-links resolve to existing files
- Warning on docs not updated in 90+ days (configurable)

**Metadata in frontmatter** makes structural freshness queryable:

```markdown
---
status: approved
last_reviewed: 2025-12-01
owner: @team-platform
---
```

These checks catch broken links and missing indexes. They do **not** catch a doc that reads
correctly but no longer describes the code. That gap, and why progressive disclosure makes it more
dangerous, is covered under "Honest limits" in SKILL.md. The practical consequence for this tree:
keep the canonical set small, and back load-bearing claims with a test rather than trusting prose.

## The filesystem is the state machine

Status is derivable, not narrated. What is in flight is whatever sits in `docs/exec-plans/active/`;
what is done moves to `completed/`. Generated indexes (file maps, schema dumps) are rebuilt by
script and never hand-edited. An exec-plan tracks its own progress:

```markdown
# Migrate to API v2

## Status: In Progress (Step 3 of 5)

## Steps
1. [x] Define v2 schema types
2. [x] Implement v2 endpoints alongside v1
3. [ ] Migrate frontend to v2 endpoints  <-- CURRENT
4. [ ] Add deprecation headers to v1
5. [ ] Remove v1 after 30-day deprecation window

## Decision Log
- 2025-11-20: Chose parallel deployment over feature flags (simpler rollback)

## Known Risks
- Frontend has 3 undocumented v1 endpoint usages (tracked in tech-debt-tracker.md)
```
