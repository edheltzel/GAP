# Schema and data modeling

Use this when touching data models, DTOs, row schemas, wire contracts, brands, variants, optional fields, or decoders.

## Records

Default to `Schema.Struct(...)` plus a same-name `interface`.

```ts
export const User = Schema.Struct({
  id: UserId,
  name: Schema.NonEmptyString,
  email: Schema.optionalKey(Schema.String),
})

export interface User extends Schema.Schema.Type<typeof User> {}
```

Guidance:

- Add `.annotate({ identifier: "User" })` only when tooling consumes it: HTTP API, RPC, OpenAPI/JSON Schema, docs, diagnostics, or codegen.
- Use `schema.make(...)` when construction is trusted.
- Use `schema.makeEffect(...)` when construction failure should stay in the Effect error channel. That failure is `SchemaIssue.Issue`, not `Schema.SchemaError`. Wrap with `new Schema.SchemaError(issue)` only when the boundary wants a `SchemaError`.
- Apply the `do-ts-standards` skill to boundary ownership and trust decisions. Decode unknown input with `Schema.decodeUnknownEffect(...)` by default. That failure is `Schema.SchemaError`.
- Use `Schema.decodeUnknownSync(...)` only in scripts, tests, or startup paths where throwing is acceptable.
- Use `Schema.decodeUnknownOption(...)` only when mismatch details are intentionally discarded.
- Use `Schema.decodeUnknownResult(...)` for pure code that wants explicit success/failure without Effect.

## Field and contract reuse

Reuse fields directly when contracts are semantically related.

```ts
export const CreateUserInput = Schema.Struct({
  name: User.fields.name,
  email: User.fields.email,
})

export const StoredUser = User.pipe(
  Schema.fieldsAssign({
    createdAt: Schema.DateTimeUtcFromString,
  }),
)
```

Guidance:

- Use `.fields`, `Schema.fieldsAssign(...)`, and `.mapFields(...)` to build small contracts with a genuine semantic relationship. `.mapFields` is an instance method, not a `Schema.mapFields` function.
- Use `Schema.encodeKeys(...)` when decoded TypeScript names differ from encoded wire/storage keys and naming is the only difference.
- Keep explicit mapping when behavior, joins, validation, or domain translation is involved.
- Use `Schema.extendTo(...)` sparingly for decoded-only derived fields.

## Optionality and defaults

Apply the `do-ts-standards` skill to decide domain optionality. Represent the encoded contract precisely:

- Use `Schema.optionalKey(...)` for absent JSON/storage keys.
- Use `Schema.optional(...)` only when explicit `undefined` is part of the contract.
- Use `Schema.NullOr`, `Schema.UndefinedOr`, or `Schema.NullishOr` only when nullish values are part of the encoded contract.
- Keep normalized defaulted values as required fields and apply defaults in constructors or decoding.

## Nominal values

Apply the `do-ts-standards` skill to decide which values require brands or refinements.

- Implement scalar IDs and value objects as constrained branded schemas.
- Apply normal schema constraints before `Schema.brand(...)` for most code.
- Use `Schema.fromBrand(...)` when the project already models brands with `Brand` constructors or needs the check packaged with the brand constructor.

## Variants

```ts
type Step = Data.TaggedEnum<{
  Continue: { readonly cursor: number }
  Finished: { readonly count: number }
}>

export const Step = Data.taggedEnum<Step>()

const next = Step.Continue({ cursor: 10 })
const label = Step.$match(next, {
  Continue: ({ cursor }) => `continue at ${cursor}`,
  Finished: ({ count }) => `finished ${count}`,
})
```

```ts
export const Event = Schema.TaggedUnion({
  Started: { runId: RunId },
  Finished: { runId: RunId, result: Schema.Json },
})

export type Event = typeof Event.Type

const event = Event.cases.Started.make({ runId })
const label = Event.match(event, {
  Started: ({ runId }) => `started ${runId}`,
  Finished: ({ runId }) => `finished ${runId}`,
})
```

Guidance:

- Use `Data.TaggedEnum` for internal control-flow algebras. `Data.taggedEnum` provides constructors, `$is`, and exhaustive `$match`.
- Use `Schema.TaggedStruct` for the ordinary Effect-owned `_tag` variant.
- Use `Schema.TaggedUnion` when the union needs decoding, encoding, persistence, wire validation, JSON Schema derivation, or schema composition.
- Use `Schema.tag(...)` when an external contract has a custom discriminator field such as `type` or `kind`; combine those structs with `Schema.toTaggedUnion("type")` when union helpers are needed.
- If the encoded contract omits the discriminant, use `Schema.tagDefaultOmit(...)` deliberately.
- Use structural schemas, `Schema.Struct`, `Schema.TaggedStruct`, or `Schema.TaggedUnion`, for new data models.

## Errors

Apply the `do-ts-standards` skill to the error's meaning, granularity, context, message, and recovery guidance. `Schema.TaggedError` is the schema-backed class for typed Effect errors. `Schema.TaggedErrorClass` does not exist.

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

Guidance:

- Use schema unions for public API or transport error surfaces.
- Use `Schema.Defect()` for defect-like payloads.
- Read [`errors-and-runtime.md`](errors-and-runtime.md) for the error channel, `Cause`, and defects.

## Completion check

Every changed Effect data model uses the selected record or variant representation; every reused field preserves the same meaning; every encoded optional or nullish state is intentional; every default produces a required normalized value; every decoder matches its boundary's trust and failure policy; and every serialized error surface follows the `do-ts-standards` skill.
