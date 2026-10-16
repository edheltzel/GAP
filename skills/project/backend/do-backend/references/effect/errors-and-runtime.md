# Errors and runtime

Read this before changing nontrivial Effect code. These rules protect semantics that ordinary TypeScript intuition and v3 habits get wrong. Use the installed package source for exact combinator signatures.

## Effect failures are not thrown exceptions

The Effect error channel is an Effect failure yielded inside `Effect.gen`. An ordinary `try/catch` around `yield*` does not recover it. Recovery is `Effect.catch`, `Effect.catchTag`, `Effect.catchTags`, `Effect.result`, or `Effect.exit`, depending on whether the caller should recover, map, or inspect the failure.

v3 names `Effect.catchAll`, `Effect.catchAllCause`, `Effect.catchSome`, and `Effect.either` are gone. `Either` is `Result`. Wrap foreign throwing code with `Effect.try` or `Effect.tryPromise` at the boundary where it enters Effect. `Effect.sync` is for synchronous work that must not throw; a throw there is a defect.

Use `return yield*` for a failure or interruption inside a conditional generator branch so the generator's return type stays honest.

## Preserve typed failures

Model expected failures with tagged domain types rather than the global `Error` class. Use `Schema.TaggedError` when the failure crosses an encoding, persistence, API, or documentation boundary. For internal-only failures, use `Data.TaggedError`.

```ts
export class PersistenceError extends Schema.TaggedError<PersistenceError>()(
  "UserRepo.PersistenceError",
  {
    operation: Schema.String,
    message: Schema.String,
    cause: Schema.Defect(),
  },
) {}
```

Do not use `as any`, `as never`, double assertions, or a widened `Error` channel to make an Effect typecheck. Fix the service, error, or environment type that produced the mismatch. A narrow assertion at a poorly typed external boundary needs a documented reason.

Absence modeling belongs to the `do-ts-standards` skill. Normalize once at the system boundary.

## Keep defects out of expected error mapping

`Cause` is a flat `reasons` array of `Fail`, `Die`, and `Interrupt`. There is no `Sequential` or `Parallel` tree to traverse. Use `Effect.mapError` or tagged recovery for expected failures. Use `Effect.catchCause` only at a deliberate runtime, reporting, or supervision boundary where handling the whole cause is the requirement. `Cause.hasInterrupts` distinguishes interruption from other reasons.

Do not silently convert a required audit, billing, persistence, authorization, or notification effect to `Effect.void`. Propagate or translate its expected failure. A fallback is appropriate only when the product semantics make the operation optional.

## Yield only Effects

`Option`, `Result`, `Ref`, `Deferred`, and `Fiber` are plain values, not Effects. Convert explicitly: `Effect.fromOption`, `Effect.fromResult`, `Ref.get`, `Deferred.await`, `Fiber.join`, `Fiber.await`. Services and `Config` remain yieldable.

Do not wrap safe array transformations, constants, or other deterministic pure work in `Effect.try`. Do not reach for an `Unsafe` constructor, such as `Ref.makeUnsafe`, merely to avoid yielding an Effect.

## Fiber ownership

Every forked fiber needs an owner and a completion policy: join it, interrupt it, or place it in a Scope that closes. Use `Effect.forkChild` (v3 `Effect.fork`), `Effect.forkScoped`, `Effect.forkIn`, or, only for deliberately detached work, `Effect.forkDetach` (v3 `Effect.forkDaemon`). Observe results with `Fiber.join` or `Fiber.await`. Fibers are not yieldable. Do not create fire-and-forget fibers whose failures and finalizers become invisible.

Acquire resources with `Effect.acquireRelease` or `Effect.acquireUseRelease` and run them in a Scope. `Layer.effect` owns and excludes the layer Scope. v3 `Scope.extend` is gone; use `Scope.provide`.

## Runtime boundaries

There is no `Runtime.make` service capture. Capture services with `Effect.context` and run with `Effect.runForkWith` or `Effect.runPromiseWith`. For a long-lived boundary that owns a Layer, use `ManagedRuntime.make`. Process entrypoints should use the platform runner named in `Runtime.ts` (`NodeRuntime.runMain`) rather than constructing `Runtime.makeRunMain` directly, unless you are that platform.

Fiber-local settings are `Context.Reference` values, not `FiberRef`s. Read them with `yield* References.MinimumLogLevel` (and the other `References` tags). Scope a change with `Effect.provideService(effect, References.MinimumLogLevel, "Warn")` or `Layer.succeed(References.MinimumLogLevel, "Info")`.

Use Effect `Clock`, `Duration`, and `Schedule` instead of ambient time and ad hoc timer loops. `Duration.Input` accepts a number of millis or a `` `${number} ${unit}` `` string such as `"5 seconds"`. Preserve the project's established representation rather than normalizing for style alone.

For concurrency-sensitive behavior, test the coordination point explicitly rather than assuming a forked fiber has already run. Read [`testing.md`](testing.md).

Adapted from PaulRBerg/agent-skills `critical-rules.md` and `runtime.md` (MIT).
