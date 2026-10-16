# Backend

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-backend
```

```bash
npx skills update do-backend
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/backend/do-backend)

## What it does

`do-backend` is the one backend skill. It has two jobs: writing and designing
Effect (TypeScript) code, and reviewing backend changes in Effect, Go, Python,
and Rust. It picks the job, detects the languages in play, and loads only the
matching guide or review. A mixed repo runs each detected language, not a
blended checklist.

Its defining constraint for Effect is that remembered APIs are not evidence:
check the project's pinned `effect` version and the installed source,
including the `AGENTS.md` and `ai-docs/` that ship inside the `effect`
package, before using a name.

An Effect *service* is a capability defined with `Context.Service`, requested
with `yield*`, and provided by a `Layer`. Service design is one part of the
Effect guide: it decides whether a capability belongs behind that seam or
should stay a plain value, then shapes the module (`make`, `layer`,
`layerWithoutDependencies`, honest test Layers).

## When to reach for it

Type `/do-backend`, or the agent reaches for it when Effect code, an Effect
service, or a Layer boundary is being written or reviewed, or when a backend
review is requested.

- **Effect:** `Effect.gen`, `Layer`, `Context.Service`, `Schema`, `Config`,
  Schedule or retry, `Cache`, `Stream`, Fiber or fork, `Cause` or defects,
  `HttpClient`, `HttpApi`, `SqlClient`, `TestClock`, and designing or auditing
  Effect services. For a v3-to-v4 migration, use the official
  `effect-v3-to-v4` skill if it is installed.
- **Review:** a pre-push or pre-PR pass over Go (including BubbleTea, Wish
  SSH, and Prometheus), Python and FastAPI, Rust (tokio, serde, sqlx), or
  Effect changes.

TypeScript that does not use Effect is not this skill's job: use
[ts-standards](../typescript/ts-standards.md) for general TypeScript and
[review-frontend](../frontend/review-frontend.md) for frontend code. For a
stack-agnostic correctness lens, use
[adversarial-review](../../global/core/adversarial-review.md).

## Prerequisites

For Effect work, the project depends on `effect`, and every `@effect/*`
package is on the same version as that pin.

## Detect, then load

| Signal | Loads |
| --- | --- |
| `.ts`/`.tsx` in a package that depends on `effect` | Effect guide and its matching branches (schema, services, errors and runtime, config, retry, caching, streams, HTTP clients, SQL, testing, v4 traps, Alchemy) |
| `.go` or `go.mod` | Go review orchestrator |
| `.py` or `pyproject.toml` | Python review orchestrator |
| `.rs` or `Cargo.toml` | Rust review orchestrator |

Each orchestrator then detects its own stack (BubbleTea, FastAPI, tokio, and
the rest) and loads only those references. Topics the Effect guide does not
cover (HTTP server, observability, CLI, AI, cluster) point to the official
docs inside the installed `effect` package.

## It's working if

- A Go-only diff never loads the Python, Rust, or Effect material.
- Every Effect API it suggests exists in the project's pinned `effect` source.
- A new Effect service passes the deletion test before it gets a `Layer`.
- Critical or Major review findings come from the enclosing function on disk,
  not diff-only reading.

## Where it fits

The backend half of the project stacks, installed with the shared
[typescript](../typescript/ts-standards.md) group for Effect projects. The
other review orchestrators are [review-ios](../swift/review-ios.md) and
[review-frontend](../frontend/review-frontend.md).

Go, Python, and Rust reviews are adapted from Beagle (Apache-2.0). Effect
errors, runtime, and SQL guidance is adapted from PaulRBerg/agent-skills, and
v4 traps from kitlangton/skills (both MIT).
