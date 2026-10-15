# Effect guide

Entry point for writing, designing, and reviewing Effect (Effect-TS) code. Load it whenever Effect code changes.

## Source rule

Check the project's pinned `effect` version before choosing an API. Every `@effect/*` package must be on that same version. Read `node_modules/effect/AGENTS.md`, and grep `node_modules/effect/ai-docs/src` and `node_modules/effect/src`, before relying on a remembered API. This guide targets Effect 4.x. For a 3.x project, say so and verify every API against that project's installed source.

## Branch chooser

Read every reference that matches the change:

- Data models, schemas, brands, variants, optional keys, or decoders: [`schema-and-data.md`](schema-and-data.md).
- Services, module surfaces, Layers, runtime wiring, `Effect.fn`, or test services: [`services.md`](services.md).
- Typed failures, defects, `Cause`, fibers, fork, or runtime boundaries: [`errors-and-runtime.md`](errors-and-runtime.md).
- Alchemy Workers, Durable Objects, Workflows, bindings, or two-phase Effectful Constructors: [`alchemy.md`](alchemy.md).
- Runtime config, environment variables, `ConfigProvider`, or `layerConfig`: [`configuration.md`](configuration.md).
- Retry, repeat, polling, backoff, jitter, rate limits, timeouts, or pass loops: [`scheduling-and-retry.md`](scheduling-and-retry.md).
- Memoization, TTL caches, concurrent lookup deduplication, or request batching: [`caching.md`](caching.md).
- Streams, event sources, async iterables, queues, pubsubs, pagination, backpressure, or stream consumers: [`streams.md`](streams.md).
- Outgoing HTTP, Effect `HttpClient`, status handling, or HTTP rate limiting: [`http-clients.md`](http-clients.md).
- SQL clients, `SqlSchema`, transactions, or `Model`: [`sql.md`](sql.md).
- Effect tests, time, sleeps, concurrency synchronization, fakes, or test Layers: [`testing.md`](testing.md).
- A v3 name, `effect/unstable/*` import, or other v3-to-v4 trap: [`v4-traps.md`](v4-traps.md).

For a v3-to-v4 migration, use the official `effect-v3-to-v4` skill if it is installed (`npx skills add Effect-TS/skills --skill effect-v3-to-v4`). Do not vendor that skill.

Topics this guide does not cover live in the pinned package docs:

- HTTP server and HttpApi: `node_modules/effect/ai-docs/src/51_http-server`
- Observability: `node_modules/effect/ai-docs/src/08_observability`
- CLI: `node_modules/effect/ai-docs/src/70_cli`
- AI: `node_modules/effect/ai-docs/src/71_ai`
- Cluster: `node_modules/effect/ai-docs/src/80_cluster`
- Child process: `node_modules/effect/ai-docs/src/60_child-process`

## Cross-cutting defaults

- Compose workflows with `Effect.gen(function* () { ... })` and the project's established `Effect.fn` patterns.
- Recover from the typed error channel at the narrowest boundary with a truthful response; preserve defects and interruption.
- Use native Effect workflows. Isolate unavoidable Promise or platform APIs in their owning Adapter.

## Completion check

Every matching reference has been read, every chosen Effect API has been verified in the pinned package source, and every cross-cutting default has been checked against each changed Effect path. Report any exception with concrete evidence.

Errors and runtime, and SQL, are adapted from PaulRBerg/agent-skills (MIT, `LICENSE-paulrberg`). v4 traps are adapted from kitlangton/skills (MIT, `LICENSE-kitlangton`).
