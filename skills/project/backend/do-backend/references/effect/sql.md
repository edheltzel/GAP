# Effect SQL

Core SQL modules ship inside `effect` under `effect/sql`: `SqlClient`, `SqlSchema`, `SqlResolver`, `SqlError`, `Statement`, `Migrator`, and `SqlModel`. They are `@stability unstable`. Drivers are separate `@effect/sql-*` packages and must be the same version as `effect`. Check the installed driver before copying an import. Variant models live in `effect/schema` as `Model`.

```ts
import { SqlClient, SqlSchema } from "effect/sql"
import { Model } from "effect/schema"
```

Keep SQL at repository boundaries and return domain values rather than unchecked row shapes.

## Decode rows

Build queries with `SqlSchema.findAll`, `findNonEmpty`, `findOne`, `findOneOption`, or `void`. Each takes `{ Request, Result, execute }` except `void`, which has no `Result`. Pick the constructor whose cardinality matches:

- `findAll` returns an array.
- `findNonEmpty` fails with `Cause.NoSuchElementError` on zero rows.
- `findOne` returns the first row or fails with `Cause.NoSuchElementError` (v3 `single`).
- `findOneOption` returns `Option<A>` (v3 `findOne`).
- `SqlSchema.void` encodes the request and discards the result.

A raw SQL type parameter describes a row but does not validate database output. Use precise schemas for identifiers, literals, decimals, and encoded values. When no row is normal, use `findOneOption`. When the service contract requires existence, translate absence to a tagged domain error.

## SqlError

`SqlError` carries a tagged `reason` and an `isRetryable` getter derived from that reason. Reason tags include `ConnectionError`, `AuthenticationError`, `AuthorizationError`, `SqlSyntaxError`, `UniqueViolation`, `ConstraintError`, `DeadlockError`, `SerializationError`, `LockTimeoutError`, `StatementTimeoutError`, and `UnknownError`. Branch with `Effect.catchReason("SqlError", "UniqueViolation", handler)` (parent tag, then reason tag). Do not parse driver messages.

Repository services may expose domain errors while retaining driver and decode causes for diagnostics. Map expected SQL or decode failures with `Effect.mapError`. Do not map defects through `Effect.catchCause` in ordinary repository code. Read [`errors-and-runtime.md`](errors-and-runtime.md).

## Transactions and Model

Use the client method `withTransaction` for writes that must commit atomically. Include audit, outbox, or ledger writes in the same transaction only when the product invariant requires one commit boundary. Close the transaction before network calls or long-running work. Retry and idempotency ownership belong to the `do-ts-standards` skill.

`Model.Class` defines a domain model with database and JSON variants: the class itself for select, plus `insert`, `update`, `json`, `jsonCreate`, and `jsonUpdate`. Field helpers such as `Model.GeneratedByDb` and `Model.DateTimeInsert` live on the same module. `SqlModel.makeRepository` builds a repository over a `Model` and a table name when that helper matches the store.

Adapted from PaulRBerg/agent-skills `sql.md` (MIT).
