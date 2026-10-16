# Scheduling and retry

Use `Schedule` to express retry, polling, pacing, and repeated background work.

## Core rules

- `Effect.retry(...)` retries typed failures; defects and interruptions are not retried.
- `Effect.repeat(...)` repeats successful effects; failures stop repetition unless the pass handles them first.
- The source effect runs once before the schedule is stepped.
- `Schedule.recurs(3)` means three retries or repetitions after the initial run.
- `Schedule.spaced(...)` waits after work completes.
- `Schedule.fixed(...)` aligns executions to a cadence.
- Use `Schedule.exponential(...)` or `Schedule.fibonacci(...)` for backoff.
- Add `Schedule.jittered` to avoid synchronized retry storms.
- Use `Schedule.recurs(...)` for a counter schedule or `Schedule.upTo({ times })` to bound a delay schedule. `times` limits schedule outputs, so the effect can run once more than `times`.
- Use `Schedule.tap(...)` to observe schedule metadata, including the input. `Schedule.tapInput` does not exist.
- Use `Effect.retryOrElse(...)` when exhausted retries need a fallback or reporting effect.
- Retry only at the narrowest boundary with proven idempotency.
- Keep exhausted failures visible unless the boundary has a truthful fallback.

When retries can duplicate side effects, apply the `do-ts-standards` skill to choose the retry owner and safety guarantee.

## Polling workers

Handle expected pass failures through the typed error channel.

```ts
const pass = runPass().pipe(
  Effect.tapError((error) =>
    Effect.logError("Worker.pass_failed", error),
  ),
  Effect.ignore,
)

const run = pass.pipe(
  Effect.repeat(Schedule.spaced("1 second")),
)
```

This shape says expected operational pass failures are logged and the worker continues. Defects still defect and can reach supervision.

Use cause-level recovery only at supervision boundaries where the policy is truly "report non-interrupt failure and continue".

```ts
const logNonInterruptCauseAndContinue = (message: string) =>
  Effect.catchCauseIf(
    (cause) => !Cause.hasInterrupts(cause),
    (cause) => Effect.logError(message, cause),
  )
```

Recover expected typed failures with `Effect.catchIf(...)`, `Effect.catchFilter(...)`, `Effect.catchTag(...)`, or `Effect.retry(...)`.

## Per-item failure isolation

For batch workers, catch expected item-level typed failures around each item so one bad item does not stall the batch.

```ts
yield* Effect.forEach(
  items,
  (item) =>
    processItem(item).pipe(
      Effect.tapError((error) =>
        Effect.logError("Worker.item_failed", error).pipe(
          Effect.annotateLogs({ itemId: item.id }),
        ),
      ),
      Effect.ignore,
    ),
  { discard: true, concurrency: 5 },
)
```

This isolation is truthful when the product policy retries the item later or deliberately skips it.

## Reusable retry policy

```ts
const reconciliationRetrySchedule: Schedule.Schedule<unknown, ReconciliationError> =
  Schedule.exponential("100 millis").pipe(
    Schedule.jittered,
    Schedule.upTo({ times: 5 }),
  )

const reconcileWithRetry = (target: Target) =>
  reconcile(target).pipe(
    Effect.retry(
      reconciliationRetrySchedule.pipe(
        Schedule.tap((metadata) =>
          Effect.logWarning("Agent.Reconciliation.retrying").pipe(
            Effect.annotateLogs({ operation: metadata.input.operation }),
          ),
        ),
      ),
    ),
    Effect.tapError((error) =>
      Effect.logError("Agent.Reconciliation.stopped", error),
    ),
  )
```

Use this shape when retry state is useful for logs or metrics. The final `tapError` reports exhaustion while preserving the typed failure.

## Rate-limit-aware typed retry

For provider errors that carry `retryAfterMs`, let the schedule use the larger of the backoff delay and the provider delay.

```ts
type RateLimited = {
  readonly retryAfterMs?: number | undefined
}

const providerRetrySchedule: Schedule.Schedule<RateLimited, RateLimited> =
  Schedule.exponential("200 millis").pipe(
    Schedule.jittered,
    Schedule.upTo({ times: 5 }),
    Schedule.passthrough,
    Schedule.modifyDelay(({ input, duration }) =>
      Effect.succeed(
        input.retryAfterMs === undefined
          ? duration
          : Duration.max(duration, Duration.millis(input.retryAfterMs)),
      ),
    ),
  )
```

Use this for operation-level retries over typed provider errors. For Effect `HttpClient` 429 handling and proactive pacing, read [`http-clients.md`](http-clients.md).

## Timeouts and delays

- Use `Effect.timeout(...)` when the operation has a real deadline.
- Use `Effect.delay(...)` when one operation should start later.
- Use `Effect.sleep(...)` when sleeping itself is the domain behavior.
- Use `Effect.repeat(...)` with `Schedule` for recurring work.
- In tests, control time with `TestClock`. Read [`testing.md`](testing.md).

## Completion check

Every retry and repetition has an explicit owner and termination policy, including intentional unbounded workers. Every retried side effect has a proven safety guarantee. Polling and batch workers state whether each expected failure continues, retries, skips, or stops. Exhausted failures remain typed and visible unless the owning boundary provides a truthful fallback. Time-based tests use deterministic clock control.
