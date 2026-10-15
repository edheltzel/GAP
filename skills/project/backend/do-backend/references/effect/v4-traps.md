# v4 API traps

Checked against Effect 4.0.1. Always inspect the consuming project's pinned version before applying these changes. For a full migration, use the official `effect-v3-to-v4` skill if it is installed. Do not vendor it.

## Imports

v4 groups modules by domain. There is no `effect/unstable/*` segment.

```ts
import { Config, Context, Effect, Layer, Schema } from "effect"
import { FetchHttpClient, HttpClient } from "effect/http"
import { HttpApi, HttpApiBuilder } from "effect/http-api"
import { SqlClient } from "effect/sql"
import { Rpc, RpcGroup } from "effect/rpc"
import { Atom } from "effect/reactivity"
import { TestClock } from "effect/testing"
```

Direct imports such as `effect/http/HttpClient` also work through the package `exports` wildcard. Check `effect/package.json` exports instead of moving every `@effect/*` package into `effect`. Platform adapters, framework integrations, SQL drivers, and `@effect/vitest` are still separate packages and must match the `effect` version. The root barrel does not re-export every domain module.

## Common traps

| Older API or assumption | Current v4 |
| --- | --- |
| `Config.string`, `Config.boolean`, `Config.redacted` | `Config.String`, `Config.Boolean`, `Config.Redacted` |
| `Config.mapOrFail` | `Config.mapEffect`; the failure must be `ConfigError` |
| `effect/unstable/http` | `effect/http` |
| Root `TestClock` import | `effect/testing` |
| `Context.Tag` or `Effect.Service` service definitions | `Context.Service` plus an explicit `Layer` |
| `Effect.catchAll` | `Effect.catch`; prefer `Effect.catchTag` for one tagged error |
| `Effect.fork` / `Effect.forkDaemon` | `Effect.forkChild` / `Effect.forkDetach`; prefer `Effect.forkScoped` for layer-owned work |
| `Either` | `Result`; inspect constructors and payload fields rather than only renaming the import |
| `Schema.TaggedErrorClass` | `Schema.TaggedError<Self>()("Tag", fields)` |
| `schema.makeEffect` failures are `SchemaError` | Instance `schema.makeEffect` fails with `SchemaIssue.Issue` |
| `Schedule.tapInput` | `Schedule.tap`; the callback receives metadata, including `input` |
| `FiberRef` | `Context.Reference` (`References` for built-in settings) |
| `Scope.extend` | `Scope.provide` |
| `Runtime.make` | `Effect.context` plus `Effect.runForkWith` / `Effect.runPromiseWith`, or `ManagedRuntime.make` |

A name appearing in a source comment is not evidence that it is exported.

## Validation boundaries

```ts
const Port = Schema.Int.check(Schema.isBetween({ minimum: 1, maximum: 65535 }))

const config = Config.schema(Port, "PORT")
const constructed = Port.makeEffect(8080) // failure: SchemaIssue.Issue
const decoded = Schema.decodeUnknownEffect(Port)(8080) // failure: Schema.SchemaError
const wrapped = constructed.pipe(
  Effect.mapError((issue) => new Schema.SchemaError(issue)),
)
```

Schema and config recipes are in [`schema-and-data.md`](schema-and-data.md) and [`configuration.md`](configuration.md). Virtual time is in [`testing.md`](testing.md).

Adapted from kitlangton/skills `effect/references/V4_APIS.md` (MIT).
