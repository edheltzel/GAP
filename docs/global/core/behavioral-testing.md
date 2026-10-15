# Behavioral Testing

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-behavioral-testing
```

```bash
npx skills update do-behavioral-testing
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/core/do-behavioral-testing)

## What it does

`behavioral-testing` is a testing methodology that asserts on what a user
observes — rendered errors, state transitions, recovery from failure — and never
on how the code is wired underneath. The defining constraint is a single rule:
if a refactor breaks a test while behavior is unchanged, the test was wrong. That
one line reframes every decision the skill makes, from what to assert to when a
mock is allowed to exist.

## When to reach for it

Type `/do-behavioral-testing`, or the agent reaches for it automatically when
writing tests, planning a feature's test strategy, or reviewing a suite that has
gone brittle.

Reach for it when tests keep breaking on green refactors, when mock setup dwarfs
the actual assertion, or when "coverage" has drifted into testing internal call
order. It governs *what* a test should prove; for the red-green loop of driving
new code test-first, that's a separate discipline this one still applies to.

## Test behavior, not structure

- **Assert on user-observable outcomes.** Empty and null input, the error
  message a user sees, loading -> success -> error transitions, API-failure
  recovery. Skip internal method-call order and which helper fired.
- **Arrange / Act / Do the thing the user would do / Assert what they'd see.**
  No red-green-red ritual, no ceremony. If setup is longer than the assertion,
  something is wrong.
- **Name tests after the behavior** ("shows error when email is empty"), never
  after the function under test ("test validateEmail").
- **Mock external boundaries only** — network, third-party services, timers.
  Never mock modules you own. If a thing must be mocked to be tested, the design
  has a coupling problem; fix the design, not the test.

The `references/` files carry the depth: `anti-patterns.md`,
`test-templates.md`, and `branch-coverage.md` for systematic coverage without
testing every permutation.

## It's working if

- A test names a user-visible behavior and stays green through an internal
  refactor.
- Mock setup is a small fraction of the test, and no assertion checks a mock's
  call count unless the call *is* the behavior.
- Tests written after the implementation are verified to fail when behavior
  breaks, not merely when code moves.

## Where it fits

A reach-for-it-anytime standard for day-to-day test authoring. It favors terse
over thorough and deletes what doesn't earn its keep, and complements the code-craft skills in
[`project/`](../../project/): they shape the code, this proves it behaves.
