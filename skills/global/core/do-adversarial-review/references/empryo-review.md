# Empryo review engine

Inside Empryo, the hunt runs on Empryo's fresh-context reviewer. It matched this skill on a seeded A/B at about 1/25 of the cost (GAP `docs/solutions/2026-10-19-empryo-review-ab.md`). Everything else in SKILL.md still holds: read-only, the trigger gate, self-verification, the Output format, and the Caller contract.

## When to use it

All of these must hold. Otherwise run the in-skill hunt.

- Empryo is the active harness, per GAP `global/docs/agents/harness-detection.md`.
- The change under review is the uncommitted working tree, tracked and untracked. A committed branch, a PR, or a named commit range is not: the reviewer only sees uncommitted git changes.
- At most 30 changed files: `git status --porcelain --untracked-files=all | wc -l` prints 30 or less. The reviewer stops at 30.
- In a GitButler workspace (`but status` exits 0), the reviewer sees every applied lane's uncommitted changes, not only the lane under review. List the changed files for every lane from `but status`. If the reviewed lane shares a changed file with another lane, or file ownership cannot be established, run the in-skill hunt on the reviewed lane's actual diff and attribute findings to its hunks. Otherwise name the reviewed lane's files in the brief ("Review only these files: ...") and keep only findings on them. The 30-file limit still applies to the entire working tree the engine reads.

## Run

From the repository root:

```sh
"$HOME/.empryo/bin/empryo" --headless --review "$(cat <<'EOF'
Adversarial correctness review of the uncommitted diff. Assume the change is wrong until a concrete input proves it right. Hunt correctness bugs the change introduced, not style or security. Bug classes: null/optional deref (map lookup, regex match, API value used without a guard), boundary/off-by-one (index math, slice bounds, < vs <=, empty input), swallowed error (empty catch, catch-log-continue, ignored error), happy-path gap, race/ordering, mutation/aliasing, async/await slips, type coercion, resource leak, inverted logic, contract mismatch with callers, partial change. Gate: report a finding only if you can name the concrete input that triggers it and trace it to the changed line; read surrounding guards first, since code that looks unsafe alone is often guarded. Tag each finding P0-P3 with file:line, trigger, confidence, and smallest fix. Report only; do not edit.
EOF
)" --json --cwd "$(git rev-parse --show-toplevel)"
```

## Read the result

| Exit | JSON | Meaning |
|---|---|---|
| 0 | `verdict.status` is `PASS` | The change holds. Empty report. |
| 0 | `verdict` is `null` | Nothing was reviewed. Never a pass. Run the in-skill hunt. |
| 3 | `FAIL` | Findings. Map them below. |
| 4 | `PARTIAL` | Findings. Map them below. |
| other, or no JSON | | The engine failed. Run the in-skill hunt. |

`verdict: null` also covers a GitButler lane whose changes are already committed. If `unreadPaths` is not empty, hunt those files in-skill and merge the results.

## Map the findings

The fenced `review-verdict` block in `report` lists `findings[]` with `file`, `line`, and `what`. For each:

1. Run Self-verification from SKILL.md: re-read the trigger path and drop what you cannot defend.
2. Re-grade severity with this skill's P0-P3 definitions. The reviewer's own tags run low.
3. Write it in the Output format: severity, `file:line`, why it breaks, trigger, confidence, suggested fix.

Do not edit code, and do not run the engine's fixes. Status words stay `done | blocked | needs-human`.
