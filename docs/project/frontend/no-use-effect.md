# No useEffect

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-no-use-effect
```

```bash
npx skills update do-no-use-effect
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/frontend/do-no-use-effect)

## What it does

`no-use-effect` steers React and Next.js code away from unnecessary `useEffect`,
toward derived state, event handlers, memoisation, `key`-based resets,
`useSyncExternalStore`, and framework or query-library data APIs. The defining
test is a single question you must answer before writing an Effect: *what external
system am I synchronising with, and why can't this happen during render, in an
event handler, or with a narrower primitive?* If you can't name the external
system, you don't write the Effect. It draws on the React docs plus TkDodo, Dan
Abramov, and Kent C. Dodds.

## When to reach for it

Type `/do-no-use-effect`, or the agent reaches for it automatically when writing or
reviewing React components, refactoring effect-heavy code, or when a change
introduces `useEffect`, `useLayoutEffect`, or dependency-array churn.

Reach for it any time an Effect is about to appear in a component. For the broader
question of how the surrounding TypeScript should read, use
[write-typescript](../typescript/write-typescript.md); for systematically excising Effects across a
component tree as a cleanup pass, pair it with
[typescript-refactoring](../typescript/typescript-refactoring.md).

## What replaces the Effect

Derived values (a full name, a filtered list) belong in render, not mirrored into
more state. Expensive pure work goes in `useMemo`, not an Effect that calls
`setState`. Anything that runs because the user clicked, submitted, or typed
belongs in the event handler. To get a fresh component instance when an identity
changes, pass a different `key` rather than resetting state in an Effect. When the
source of truth lives outside React, subscribe with `useSyncExternalStore`. And
data loading should route through a framework loader or a query library
(`TanStack Query`, `SWR`) before it ever reaches raw fetch-in-Effect.

An Effect is only correct when synchronising with a genuine external system —
sockets, observers, widgets, media, layout measurement. Then keep it tiny, name
the system, and give it symmetrical cleanup behind a custom hook.

## It's working if

- No Effect exists purely to `setState` from props/state, to run after a click, or
  to notify a parent that local state changed.
- Every surviving Effect names the external system it syncs with and has matching
  setup and cleanup that is idempotent under Strict Mode remounting.
- `react-hooks/exhaustive-deps` is never silenced with `eslint-disable`, and
  dependency lists are honest.

## Where it fits

A focused React-specific standalone you reach for at authoring or review time,
narrower than the general [write-typescript](../typescript/write-typescript.md) style skill it sits
beside. It also supplies a ready-made team policy for codebases that want to ban
or heavily restrict raw `useEffect`.
