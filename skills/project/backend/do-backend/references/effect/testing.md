# Effect testing

Apply the test levels, observable-outcome rules, and completion check in the `do-ts-standards` skill to every Effect test. This reference adds Effect runtime, time, synchronization, and test-Layer rules.

`it.effect` and `it.live` come from `@effect/vitest`. `TestClock` comes from `effect/testing`, not the root `effect` barrel.

## Defaults

- Use `it.effect` by default.
- Use `it.live` when real time or live runtime services are the behavior under test.
- Drive sleeps, schedules, retries, leases, and timeouts with `TestClock.setTime` or `TestClock.adjust`.
- Fork a sleeping effect before advancing `TestClock`. The pinned package docs fork with `Effect.forkChild`.
- Assert interruption and finalization when they are observable parts of the behavior under test.

For retry or schedule tests, also read [`scheduling-and-retry.md`](scheduling-and-retry.md).

```ts
import { it } from "@effect/vitest"
import { Effect, Option } from "effect"

it.effect("finds a user", () =>
  Effect.gen(function* () {
    const users = yield* UserRepo.Service
    const result = yield* users.find(UserId.make("u1"))
    expect(Option.isSome(result)).toBe(true)
  }).pipe(Effect.provide(UserRepo.layerTest)),
)
```

## Explicit synchronization

- Use `Deferred` for one-shot readiness or completion signals. Wait with `Deferred.await`.
- Use `Queue` to hand test-controlled work or observed events across fibers.
- Use `Latch` for reusable open/close coordination gates.
- Use `Ref` for shared test observation state.
- Coordinate through an existing lifecycle, status, or result interface. Put test-only controls on a test-control service; public interfaces expose observations required by production callers.

```ts
it.effect("publishes exactly once", () =>
  Effect.gen(function* () {
    const ready = yield* Deferred.make<void>()

    const fiber = yield* Effect.gen(function* () {
      yield* Deferred.succeed(ready, undefined)
      return yield* publisher.publishNext()
    }).pipe(Effect.forkScoped)

    yield* Deferred.await(ready)
    const message = yield* Fiber.join(fiber)

    expect(message).toEqual(expectedMessage)
  }),
```

## Reusable test implementations

Read [`services.md`](services.md#6-choose-test-layers-honestly) before designing a reusable test Layer. When reusable state, failure injection, or observation belongs to a real service seam, expose a `TestService` for test control and inspection while production code continues through the real `Service` tag.

```ts
export interface Interface {
  readonly send: (message: Message) => Effect.Effect<void, SendError>
}

export class Service extends Context.Service<Service, Interface>()(
  "@app/Notifier",
) {}

export interface TestInterface extends Interface {
  readonly sentMessages: () => Effect.Effect<ReadonlyArray<Message>>
  readonly failNextSend: (error: SendError) => Effect.Effect<void>
}

export class TestService extends Context.Service<TestService, TestInterface>()(
  "@app/Notifier/Test",
) {}

export const layerTest = Layer.effectContext(
  Effect.gen(function* () {
    const sent = yield* Ref.make<ReadonlyArray<Message>>([])
    const nextFailure = yield* Ref.make<Option.Option<SendError>>(Option.none())

    const service = TestService.of({
      send: Effect.fn("Notifier.Test.send")(function* (message) {
        const failure = yield* Ref.getAndSet(nextFailure, Option.none())
        if (Option.isSome(failure)) return yield* Effect.fail(failure.value)
        yield* Ref.update(sent, (messages) => [...messages, message])
      }),
      sentMessages: Effect.fn("Notifier.Test.sentMessages")(function* () {
        return yield* Ref.get(sent)
      }),
      failNextSend: Effect.fn("Notifier.Test.failNextSend")(function* (error) {
        yield* Ref.set(nextFailure, Option.some(error))
      }),
    })

    return Context.empty().pipe(
      Context.add(Service, service),
      Context.add(TestService, service),
    )
  }),
)
```

Keep service members function-valued, including zero-argument operations, so `Effect.fn` applies uniformly. `Layer.mock` fits a tiny local partial implementation whose omitted members fail loudly when called.

## Configuration

For tests that provide runtime configuration, replace the provider with `ConfigProvider.layer` or provide an already-decoded test Layer. Do not rely on the process environment unless the test is specifically about env parsing. Read [`configuration.md`](configuration.md#providers) for the recipe.

## Completion check

The `do-ts-standards` completion check passes; the test runtime matches the behavior under test; every temporal test drives time with `TestClock`, with sleeping effects started before the clock advances; concurrent readiness and ordering use explicit synchronization; reusable test implementations cross the production service tag while test controls remain on the test-control tag; observable interruption and finalization contracts are asserted; and every applicable service, retry, and configuration pointer above has been followed.
