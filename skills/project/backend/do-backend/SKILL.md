---
name: do-backend
description: Backend engineering for Effect (TypeScript), Go, Python, and Rust - write, design, and review Effect services and code, and review Go, Python, and Rust changes. Use when working with Effect, Effect-TS, Effect v4, Effect.gen, Layer, Context.Service, Schema, Config, Schedule/retry, Stream, Fiber/fork, Cause/defects, HttpClient, HttpApi, SqlClient, or TestClock; when designing or auditing Effect services; or when reviewing Go, BubbleTea, Wish SSH, Prometheus, Python, FastAPI, Rust, tokio, serde, sqlx, or .rs changes, or for a pre-push / pre-PR backend review.
---

# Backend

One backend skill, two jobs. Pick the job, detect the language, then load only what matches. Do not load the other languages.

## Pick the job

- **Write or design Effect code** (a new service, a Layer, a schema, a retry policy, a stream, an Effect test): load the [Effect guide](references/effect/guide.md) and follow its branch chooser. No review fan-out.
- **Review backend changes** (the user asks for a review, a pre-push or pre-PR pass): detect the languages below and run each matching review.

## Detect

Files under review are the branch diff, or the paths the user named. If the user named paths, they replace the diff: pass them to the orchestrator as its file list and skip its diff-based scope check. A language is present when any file under review matches:

| Language | Signal | Review |
| --- | --- | --- |
| Effect (TypeScript) | `.ts`/`.tsx` files in a package whose `package.json` depends on `effect` | [Effect guide](references/effect/guide.md): read every matching branch, then check the changed code against it |
| Go | `.go` or `go.mod` | [Go review](references/go/review.md) |
| Python | `.py` or `pyproject.toml` | [Python review](references/python/review.md) |
| Rust | `.rs` or `Cargo.toml` | [Rust review](references/rust/review.md) |

```bash
git diff --name-only $(git merge-base HEAD main)..HEAD
```

Run each matching review and report per language. If none match, say so and stop. TypeScript without `effect` is not this skill's job: frontend code belongs to `do-review-frontend`, general TypeScript to `do-ts-standards`.

Go, Python, and Rust reviews are adapted from Beagle under Apache-2.0 (`LICENSE`). Effect references credit their sources in the guide.
