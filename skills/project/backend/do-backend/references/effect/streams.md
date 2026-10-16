# Streams

Use this when working with `Stream`, event sources, async iterables, queue or pubsub-backed streams, pagination, backpressure, throttling, debouncing, or long-lived stream consumers.

## Mental model

`Stream<A, E, R>` is an effectful source that can emit many `A` values over time, fail with `E`, and require services `R`. Streams are pull-based and backpressured; consumption controls demand.

Use streams for sources that are naturally many-valued and time-ordered:

- gateway events
- provider callbacks adapted through queues
- subscription or event logs
- paginated APIs
- file, stdin, or platform streams
- scheduled ticks when values matter
- pipelines with filtering, mapping, buffering, throttling, or bounded concurrent processing

Use `Effect.repeat(...)` with `Schedule` for one repeated effect with no emitted values. Read [`scheduling-and-retry.md`](scheduling-and-retry.md). Reserve streams for work that emits values.

## Source chooser

- In-memory values: `Stream.make(...)` or `Stream.fromIterable(...)`.
- Queue-backed callback boundary: `Queue` plus `Stream.fromQueue(...)`.
- Broadcast events: `PubSub` plus `Stream.fromPubSub(...)`.
- Latest-value state plus updates: `SubscriptionRef`.
- Schedule-generated ticks or values: `Stream.fromSchedule(...)`.
- Paginated pull APIs: `Stream.paginate(initial, step)`. The step returns `Effect<readonly [ReadonlyArray<A>, Option<S>]>`.
- Async iterable or platform source: prefer a native Effect source; otherwise use `Stream.fromAsyncIterable(...)`.
- Effect that produces a stream after reading services or config: `Stream.unwrap(...)`.

## Transformation chooser

- Pure transformation: `Stream.map(...)`.
- Effectful transformation: `Stream.mapEffect(...)`.
- Bounded concurrent effectful transformation: `Stream.mapEffect(fn, { concurrency })`.
- Drop ordering when order is irrelevant and latency matters: `Stream.mapEffect(fn, { concurrency, unordered: true })`.
- One input to zero or many outputs: `Stream.flatMap(...)`.
- Multiple inner streams concurrently: `Stream.flatMap(fn, { concurrency })`.
- Keep only matching values: `Stream.filter(...)` or `Stream.filterEffect(...)`.
- Stateful transformation: `Stream.mapAccum(...)` or `Stream.mapAccumEffect(...)`.

## Consumption chooser

- Side-effecting consumer: `Stream.runForEach(...)`.
- Ignore elements but run the stream: `Stream.runDrain`.
- Materialize a finite stream: `Stream.runCollect`.
- Fold into a value: `Stream.runFold(...)`.
- Long-lived consumer: use the scoped layer pattern below.

Use `Stream.runCollect` only when the stream is known to terminate.

## Long-lived consumers

Own long-lived stream consumers in layers and fork them into the layer scope.

```ts
export const layer = Layer.effectDiscard(
  Effect.gen(function* () {
    const gateway = yield* Gateway.Service

    yield* gateway.events.pipe(
      Stream.filter(isMessageEvent),
      Stream.runForEach(handleEvent),
      Effect.forkScoped,
    )
  }),
)
```

If service methods must fork work into the layer lifetime, capture `Scope.Scope` during layer acquisition, use `Effect.forkIn(scope)` internally, and keep the scope private. Let stream failures reach the owning boundary unless it has a truthful recovery policy.

## Queues, PubSub, and SubscriptionRef

- Use `Queue` when each event or item should be consumed by one consumer or worker.
- Use `PubSub` when every subscriber should see every event.
- Use `SubscriptionRef` when consumers need the current value and a stream of changes. Construct it with `SubscriptionRef.make`.
- Expose a `Stream` from service interfaces for caller-consumed events.
- Keep producer queues and mutable refs inside the implementation or test service.

Good service shape:

```ts
export interface Interface {
  readonly events: Stream.Stream<ProviderEvent, ProviderError>
  readonly status: Stream.Stream<ProviderStatus>
}
```

Implementation can use a private `Queue` or `SubscriptionRef`; consumers see streams.

## Backpressure and buffers

Prefer natural stream backpressure first.

Use `Stream.buffer({ capacity, strategy })` only when producer and consumer should decouple. Finite buffers take `strategy`:

- `"suspend"`: apply backpressure when full.
- `"dropping"`: drop new values when full.
- `"sliding"`: keep the latest values by dropping old ones.
- `capacity: "unbounded"`: rare; use only when growth is bounded elsewhere.

`Stream.buffer` destroys chunking. Use `Stream.rechunk` afterward if fixed chunk sizes matter.

Use `Stream.debounce(...)` for quiet-period behavior and `Stream.throttle(...)` or `Stream.throttleEffect(...)` for rate-shaped streams.

## Error handling

- Translate typed errors at boundaries with `Stream.mapError(...)`.
- Recover typed errors with `Stream.catchIf(...)`, `Stream.catchTag(...)`, or `Stream.catchFilter(...)`.
- Reserve `Stream.catchCause(...)` for explicit supervision boundaries.

## Keyed concurrency

For keyed work, preserve ordering within each key while allowing different keys to run concurrently. Prefer an existing named keyed-run helper; otherwise keep the required fiber bookkeeping in one named helper rather than scattering it through consumers. Choose queueing, replacement, or coalescing semantics from the owning operation's policy.

## Tests

- Use `Stream.fromIterable(...)` for finite fixtures. Compose it with `Stream.concat(Stream.never)` when the fixture represents an open subscription.
- Use `Stream.empty` for no events.
- Use `Stream.fromQueue(...)` with a test-owned `Queue` when the test needs to drive events interactively.
- Bound open streams with `Stream.take(n)` before `Stream.runCollect`.

For stream tests involving time or concurrency, read [`testing.md`](testing.md).

## Completion check

The source matches the producer's delivery semantics; ordering and concurrency are explicit; every collected stream is finite; buffers have a bounded-growth policy; long-lived consumers have a scoped owner; queue, `PubSub`, and `SubscriptionRef` internals stay behind stream-facing service interfaces; and typed failures, defects, and interruption reach a boundary with an explicit policy.
