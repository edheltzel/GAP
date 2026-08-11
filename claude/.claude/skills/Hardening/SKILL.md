---
name: Hardening
version: 1.1.0
description: "Strengthens existing tests with property-based testing. USE WHEN harden tests, property test, property based testing, PBT, fast-check, universal quantified claim, shrink counterexample, or find inputs example tests miss. NOT FOR mutation testing, CRAP analysis, duplication scans, acceptance-criteria mutation, security review, or new feature implementation."
---

# Hardening

## Current capability

PropertyTest is the only supported workflow. It identifies deterministic code with useful invariants, writes focused fast-check properties, executes them with the repository's test runner, and captures seeds and shrunk counterexamples for reproduction.

| Workflow | Status | Route |
| --- | --- | --- |
| PropertyTest | Supported | `Workflows/PropertyTest.md` |

Mutation testing, CRAP scoring, duplication analysis, and acceptance-criteria mutation are possible future techniques, but this skill has no runnable routes, scripts, or integration contracts for them. If requested, explain that they are unsupported here rather than implying execution.

## Scope

Use PropertyTest for pure functions, parsers and serializers, normalizers, deterministic transforms, collection invariants, and state transitions with a bounded legal input model.

Do not use it directly against disk, network, clock, random, or process-global side effects. Mock those dependencies or extract a pure core first.

## Guardrails

- Constrain generators to the function's documented valid domain, no narrower and no wider.
- Start with a bounded run budget appropriate to repository cost. The workflow default is 1,000 runs.
- Capture the fast-check seed and shrunk counterexample for every failure.
- Preserve useful example tests. Properties complement examples rather than replacing them.
- Run the repository's existing formatting, type, and test checks after adding a property.
- Do not add fast-check unless the repository's dependency policy permits it.

## Examples

- "Property-test the slug parser" routes to PropertyTest and checks round-trip, idempotence, or normalization invariants.
- "What inputs are the example tests missing?" routes to PropertyTest candidate discovery.
- "Mutation-test this suite" receives a clear unsupported-capability response and no fabricated result.
