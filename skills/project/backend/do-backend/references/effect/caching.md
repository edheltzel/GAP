# Caching, memoization, and request dedupe

Use `Cache` from `effect` when its keyed memoization, TTL, capacity, lifecycle, and eviction semantics fit. When it fits, do not hand-roll `Map`/TTL/prune caches, in-flight deduplication maps, or LRU logic.

## Core rules

- `Cache.make({ capacity, lookup, timeToLive })` caches per-key lookups. `capacity` is required. `timeToLive` is optional and, when set, is one fixed TTL for all entries.
- Concurrent `Cache.get` calls for the same missing key share one pending lookup. The cache stores the lookup `Exit`, so a failed lookup stays failed until it expires, is invalidated, or is refreshed.
- `Cache.invalidate(cache, key)` and `Cache.refresh(cache, key)` handle explicit staleness. `Cache.has` checks without triggering a lookup.
- Cache construction is effectful. Build each cache once in its owning Layer or Scope and share the handle.
- For a single value without a key, use `Effect.cached(effect)` or `Effect.cachedWithTTL(effect, ttl)`.
- For cached resources that need cleanup, such as connections or clients, use `ScopedCache`.

## Exit-aware TTL

`Cache.makeWith(lookup, { capacity, timeToLive })` computes each entry's TTL from the lookup's `Exit`. Give transient failures and degraded fallbacks a zero TTL (`Duration.zero`) so the caller receives the result while the next lookup can try again. A short negative-cache TTL can protect an upstream from repeated stable failures such as not-found results.

```ts
import { Cache, Duration, Effect, Exit } from "effect"

const makeResolver = Effect.gen(function* () {
  const cache = yield* Cache.makeWith(
    (channelRef: string) => resolveUncached(channelRef),
    {
      capacity: 300,
      timeToLive: (exit) =>
        Exit.isSuccess(exit) && exit.value.cacheable ? "10 minutes" : Duration.zero,
    },
  )
  return (channelRef: string) =>
    Cache.get(cache, channelRef).pipe(Effect.map((resolved) => resolved.where))
})
```

## Acquire expensive clients once

Construct or authenticate clients while building the owning Layer, then close over the yielded service or client in the cache lookup. Each cache miss then pays only for the provider operation. When this changes service or Layer construction, also read [`services.md`](services.md).

## Request batching

A `RequestResolver` batches pending requests when one backend call can answer multiple keys, such as SQL `IN (...)`, DataLoader-style endpoints, or batch GET. Per-item endpoints use `Effect.forEach(items, f, { concurrency: n })`, optionally through a `Cache` for deduplication and memoization.

`RequestResolver.batchN(resolver, n)` bounds batch size. `RequestResolver.makeGrouped` groups requests that resolve through different targets. `Effect.request` is the request constructor used with a resolver.

Selection guide:

- Same key requested repeatedly over time: `Cache`.
- Same key requested concurrently in one burst: `Cache` (shared pending lookup).
- Many distinct keys, backend has a batch endpoint: `Effect.request` plus `RequestResolver`.
- Many distinct keys, per-item endpoint only: `Effect.forEach(..., { concurrency: n })`, optionally through a `Cache`.

## Completion check

For every cache or batching path, the chosen primitive matches the key pattern and backend; each keyed cache has an intentional capacity, TTL and invalidation policy, and lifetime owner; stable dependencies are acquired once; same-key concurrency uses built-in deduplication; and each `RequestResolver` maps multiple keys to one backend call.
