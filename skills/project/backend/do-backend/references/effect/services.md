# Effect services

Treat a service as an **authority seam**: a cohesive capability whose requirements propagate through Effect context. An Effect service module owns the service contract, construction, production Layer, and any honest reusable test implementation that belong to the same capability.

Apply the `do-ts-standards` skill for ownership and Adapter decisions. For typed failures versus defects, read [`errors-and-runtime.md`](errors-and-runtime.md).

## 1. Establish the local rules

Read the nearest `AGENTS.md`, architecture docs, coding standards, project Effect guidance, pinned Effect version and source, and relevant vendored examples. Prefer current project-compatible patterns over remembered APIs. Follow the source rule in this skill's `SKILL.md`.

**Complete when:** the governing files and pinned source examples have been read and recorded for use in later decisions.

## 2. Select the branch

- **Design branch:** for a new service or a focused redesign, bound the capability, trace one caller-visible operation to every effect, then continue below.
- **Audit branch:** for a codebase, package, feature slice, or diff, follow [Audit](#audit). Apply the rules below to every candidate it finds.

**Complete when:** one branch is selected, its scope is bounded, and the caller-visible operations in scope are named.

## 3. Apply the service test

A real service owns at least one meaningful capability:

- authority over persistence, credentials, external I/O, runtime resources, configuration, time, randomness, or lifecycle;
- cohesive effect sequencing or policy reused across entrypoints;
- state or behavior with real production and test/runtime variation;
- enough implementation complexity that deleting the module would spread complexity into callers.

Prefer an existing Effect service such as `Clock`, `Crypto`, `Random`, `Config`, `HttpClient`, `FileSystem`, or `Path` before defining an application service.

Keep these as values or pure modules:

- parsed domain inputs and per-call request data;
- deterministic calculations, projections, parsers, and constructors;
- options that select policy for one call;
- framework values confined to their adapter;
- wrappers that only rename or forward another service.

A test-only desire to inject a value is not enough. The seam must represent real ownership or variability in production. Record the production evidence for the service-or-value decision and the rejected alternative.

**Complete when:** the deletion test and the existing-service/adapter audit both support either "service" or "value," and the rejected alternative is stated plainly.

## 4. Place the authority seam

- Domain modules stay pure.
- Application services own operation policy and application-owned ports.
- A port's tag and interface live beside the application operation that needs them.
- A concrete adapter owns its technology-specific `make` and `layer`; it need not share a file with the application-owned port.
- Composition roots select and provide concrete Layers. They do not become reusable policy modules.
- Runtime bindings are yielded in the composition root or owning adapter, then hidden behind application/domain types.

Yield stable dependencies while building the Layer and close over them in service methods. Yield request-, fiber-, or operation-scoped context inside the method that uses it. Let requirements propagate until the module that truthfully chooses an implementation provides them.

Authorization evidence, scoped handles, and other operation-specific capability values remain explicit inputs when they are part of the request or domain contract. Passing an external library's constructor options remains correct after the owning adapter has yielded the relevant runtime capability. React props, request values, domain inputs, and framework constructors are not Effect dependency injection.

**Complete when:** dependencies point inward, raw technology types stop at adapters, and no inner caller chooses a concrete implementation it does not own.

## 5. Shape the Effect service module

Follow the project's established equivalent of this shape:

```ts
export interface Interface {
  readonly operation: (input: Input) => Effect.Effect<Output, OperationError>
}

export class Service extends Context.Service<Service, Interface>()(
  "@app/Capability",
) {}

export const make: Effect.Effect<
  Service["Service"],
  never,
  Dependency.Service
> = Effect.gen(function* () {
  const dependency = yield* Dependency.Service

  const operation = Effect.fn("Capability.operation")(function* (input: Input) {
    return yield* dependency.operation(input)
  })

  return Service.of({ operation })
})

export const layerWithoutDependencies = Layer.effect(Service, make)

export const layer = layerWithoutDependencies.pipe(
  Layer.provide([Dependency.layer]),
)

export const layerTest = Layer.succeed(
  Service,
  Service.of({
    operation: (_input) => Effect.succeed(testOutput),
  }),
)
```

`Interface`, `Service`, `make`, `layerWithoutDependencies`, and `layer` are canonical role names within an Effect capability module. The owning module namespace and service tag identify the capability. `layerWithoutDependencies` preserves the service's requirements for composition. `layer` is the ready production assembly and provides the concrete dependency Layers chosen by this module. `layerTest` illustrates a complete static substitute; export it only when that behavior is reusable and honest. Use `layerMemory` instead when an in-memory implementation faithfully preserves the observable contract.

Choose the Layer constructor that matches acquisition: `Layer.succeed` for an already-built value, `Layer.sync` for lazy synchronous construction, and `Layer.effect` for effectful acquisition. Use `Layer.effectContext` when one acquisition intentionally supplies several tags, especially a production service and its test-control service. Use `Layer.unwrap` when configuration or runtime discovery builds the Layer. Use `Layer.fresh` or `Effect.provide(layer, { local: true })` only when an operation or test requires isolated acquisition. Reserve `Context.Reference` for ambient runtime values with a safe, truthful default.

Keep interfaces narrow and domain-shaped. Inject dependencies as yielded service objects rather than callback functions; a function capability fits only when higher-order behavior is itself the capability. Use named `Effect.fn` methods and typed expected errors. Add options, methods, services, and combinators only when each hides enough complexity to earn its place.

### Module surface

One valid module surface gives the ES module one canonical namespace while keeping file-local role names:

```ts
export interface Interface {
  readonly getUserById: (id: UserId) => Effect.Effect<User, NotFound | PersistenceError>
}

export class Service extends Context.Service<Service, Interface>()(
  "@app/UserStore",
) {}

export * as UserStore from "./user-store.js"
```

Consumers import the owning leaf directly and yield `UserStore.Service`. A folder or package entrypoint may relay the leaf's established identity with `export { UserStore } from "./user-store.js"`. Use this self-export style only where the runtime and toolchain support it; otherwise use ordinary named exports or a separate public entrypoint. Keep schemas, row codecs, helpers, and implementation details private.

### Runtime wiring

- Use `Layer.provide` when the current module truthfully chooses and hides an implementation dependency.
- Use `Layer.provideMerge` only when downstream consumers should still receive that dependency.
- Use `Layer.mergeAll` for independent exposed Layers.
- Keep runtime Layer values flat, named, and topologically ordered.
- Provide dependencies at their owning boundaries so application authority and lifecycle requirements remain visible.

A Layer that owns a stream, listener, worker, subscription, or long-lived fiber forks it into the Layer scope so acquisition can complete. Read [`streams.md`](streams.md) for the lifecycle pattern.

### Named operation boundaries

Use `Effect.fn("Capability.operation")` for public and non-trivial internal service methods. Reserve `Effect.fnUntraced` for internal helpers whose stack-frame and span metadata are intentionally unnecessary. Keep the generator focused on the operation and use one or two whole-function transforms for concerns that need the complete effect and original arguments, such as error classification, logging annotations, spans, bounded retry, timeout, cleanup, or result mapping. Each transform receives `(effect, ...originalArgs)`:

```ts
const readAttachment = Effect.fn("Attachment.read")(
  function* (ref: AttachmentRef) {
    return yield* client.read(ref)
  },
  (effect, ref) =>
    effect.pipe(
      attachmentError("Attachment.read", { attachmentId: ref.id }),
    ),
)
```

For operation-labelled boundary errors, prefer a shared curried `mapError` helper over repeated wrappers:

```ts
const persistenceError = operationError(PersistenceError.make)

const row = yield* query.pipe(
  persistenceError("UserStore.findById"),
)
```

Name the helper for the error it creates. Pair structured error fields with `Effect.fn` boundaries and spans for observability.

**Complete when:** the tag, interface, `make`, dependency-preserving Layer, production `layer`, errors, methods, and test strategy have one clear owner, and every exported symbol is required by a caller.

## 6. Choose test Layers honestly

- Use `Layer.succeed` for a complete static implementation.
- Add `layerTest` plus a test-control service when reusable state, failure injection, or observation is part of a real seam.
- Name a Layer `layerMemory` only when it faithfully implements the service's observable contract in memory.
- Prefer a real local substitute when persistence, transactions, serialization, or protocol behavior matters.
- Keep a tiny one-off fake in its test when promoting it would create production surface solely for that test.

Tests cross the same service interface as production callers. When a reusable control service exists, back its production tag and test-control tag with the same object. Partial objects with unused methods that die are focused test fixtures, not reusable in-memory adapters. `Layer.mock` fits a tiny local partial implementation whose omitted members fail loudly when called.

Read [`testing.md`](testing.md) for static implementations, shared backing objects, and focused local mocks. Name reusable implementations for their observable behavior, such as `InMemoryCache` or `RecordingEmailSender`.

**Complete when:** each test implementation is complete for its advertised name, tests observe outcomes through the public interface, and no new seam exists only to support mocking.

## Audit

For a codebase, package, feature slice, or diff, inventory every service, Layer, tag, and composition candidate in scope. Apply sections 3-6 to each row.

Each inventory row records:

- the symbol and its file/line;
- the current shape;
- a disposition: `keep`, or the change (value, reshape, or delete);
- when the disposition is not `keep`: the target module shape, the composition impact, and the test impact.

A test-only desire to inject a value is not a service. Prefer an existing Effect service before defining an application service.

**Complete when:** every inventory row has a disposition. Findings are prioritized and cite file/line or symbol evidence. Explicit `keep` decisions are recorded.

## 7. Finish the selected branch

- **Design branch:** record the service-or-value decision and its evidence. When implementation is requested, create or refactor the module, update composition roots and tests, and run the repository's required checks.
- **Audit branch:** produce prioritized findings with file/line or symbol evidence, target module shapes, composition and test impact, and explicit `keep` decisions.

**Complete when:** the designed capability has an explicit disposition and validated implementation when requested, or every audit inventory row has a disposition; validation passes or every failure is reported.

## Completion check

Complete when:

- the service-or-value decision cites production ownership or variability and the rejected alternative;
- every applicable interface, tag, construction effect, expected error, method, production Layer, and reusable test implementation has exactly one owner;
- stable runtime capabilities and implementation dependencies are captured during Layer construction, operation-specific capability values remain explicit inputs, scoped context is yielded where used, and requirements remain visible until the module that selects an implementation provides them;
- each Layer constructor matches acquisition, each provided dependency is an implementation the provider truthfully owns, and long-lived work is scoped;
- public and non-trivial service operations have named boundaries, with whole-operation concerns applied at those boundaries;
- the module surface exposes only service API intended for callers and uses canonical role names consistently; and
- the test strategy exercises the production interface at the fidelity required by the observable contract.
