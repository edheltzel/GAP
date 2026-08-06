# PropertyTest Workflow

## Purpose

Strengthen existing tests by expressing a deterministic universal claim with fast-check, then capture reproducible evidence when the claim fails.

## Candidate properties

| Category | Form |
| --- | --- |
| Round trip | `decode(encode(x)) === x` |
| Idempotence | `f(f(x)) === f(x)` |
| Commutativity | `f(a, b) === f(b, a)` when order is irrelevant |
| Associativity | `f(f(a, b), c) === f(a, f(b, c))` when grouping is irrelevant |
| Identity | `f(x, identity) === x` |
| Conservation | a documented count, sum, membership, or ordering invariant is preserved |
| Model based | implementation matches a simpler trusted reference |
| Metamorphic | a controlled input change produces a specified output relation |
| State machine | every generated legal transition sequence preserves invariants |
| Oracle | result matches an independent trusted implementation |

If no deterministic invariant applies, keep or add example-based tests instead.

## Procedure

1. Read the repository's applicable instructions, dependency policy, and test conventions.
2. Identify a pure candidate and state its valid input domain.
3. Write the property in prose: "for all valid X, P(X) holds."
4. Choose the least complex generator that covers the full valid domain.
5. Set a bounded run budget. Use 1,000 by default, 100 for expensive generators, or 10,000 only for critical and inexpensive invariants.
6. Write the property beside the repository's related tests using its naming and import conventions.
7. Run the smallest focused test command first.
8. On failure, record the seed, path when provided, and shrunk counterexample. Decide whether the code or the property is wrong before editing.
9. Convert important counterexamples into explicit regression examples while retaining the seed in a comment.
10. Run relevant type, lint, and broader test checks required by the repository.

## Example

```ts
import { expect, test } from "bun:test";
import * as fc from "fast-check";
import { normalizeSlug } from "../src/slug";

test("slug normalization is idempotent", () => {
  fc.assert(
    fc.property(fc.string(), (value) => {
      const once = normalizeSlug(value);
      expect(normalizeSlug(once)).toBe(once);
    }),
    { numRuns: 1000 },
  );
});
```

## Completion evidence

Report:

- the universal claim and valid input domain
- generator constraints and run budget
- focused command and result
- seed and shrunk counterexample for any failure
- regression example added, if any
- relevant repository checks and results

## Portable references

- fast-check documentation: <https://fast-check.dev/>
- fast-check reproducibility and shrinking: <https://fast-check.dev/docs/core-blocks/runners/>
- QuickCheck paper by Koen Claessen and John Hughes: <https://www.cs.tufts.edu/~nr/cs257/archive/john-hughes/quick.pdf>
