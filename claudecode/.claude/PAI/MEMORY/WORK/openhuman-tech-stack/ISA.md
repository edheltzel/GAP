---
task: Tech-stack comparison openhuman ↔ atlas-recall
slug: openhuman-tech-stack
effort: E3
phase: observe
progress: 0/12
mode: research
started: 2026-05-16T11:40:00Z
updated: 2026-05-16T11:40:00Z
---

## Problem
Ed needs to evaluate how the openhuman (tinyhumansai) project's tech stack compares to atlas-recall, to inform a series of research artifacts in `~/Developer/openhuman-research/`. No comparison exists yet.

## Vision
A single ≤200-line side-by-side comparison file that surfaces the architectural delta at a glance — runtime, language, framework, dependencies, distribution — so Ed can reason about both projects without re-reading manifests.

## Out of Scope
- Feature-by-feature product comparison
- Performance benchmarks or code-quality assessment
- Recommendations on convergence or migration
- Coverage of files beyond root manifests (Cargo.toml, package.json, Dockerfile, etc.)

## Constraints
- Output MUST live at `/Users/ed/Developer/openhuman-research/01-tech-stack.md`
- Output MUST be ≤200 lines
- Source of truth: manifest files (Cargo.toml, package.json, rust-toolchain.toml, Dockerfile) — not README marketing
- Side-by-side TABLE format required by user

## Goal
Produce a manifest-evidence-grounded comparison table covering language/runtime, frameworks, build tooling, key dependency categories, persistence, distribution, and test stack for both projects, saved to the specified path.

## Criteria
- [ ] ISC-1: Output file exists at `/Users/ed/Developer/openhuman-research/01-tech-stack.md` (Bash: `test -f`)
- [ ] ISC-2: Output file is ≤200 lines (Bash: `wc -l` ≤200)
- [ ] ISC-3: Contains a markdown table with both columns labelled openhuman + atlas-recall (Grep)
- [ ] ISC-4: Covers row "Primary language" (Grep)
- [ ] ISC-5: Covers row "Runtime" (Grep)
- [ ] ISC-6: Covers row "Build tool" (Grep)
- [ ] ISC-7: Covers row "Package manager" (Grep)
- [ ] ISC-8: Covers row "Persistence/DB" (Grep)
- [ ] ISC-9: Covers row "HTTP/server framework" (Grep)
- [ ] ISC-10: Covers row "UI/frontend framework" (Grep)
- [ ] ISC-11: Covers row "Test framework" (Grep)
- [ ] ISC-12: Covers row "Desktop wrapper" (Grep)
- [ ] ISC-13: Covers row "Distribution" (Grep)
- [ ] ISC-14: Covers row "License" (Grep)
- [ ] ISC-15: Names Rust + edition 2021 for openhuman (Grep)
- [ ] ISC-16: Names Bun + TypeScript for atlas-recall (Grep)
- [ ] ISC-17: Names Tauri 2 for openhuman desktop layer (Grep)
- [ ] ISC-18: Names tsup for atlas-recall build (Grep)
- [ ] ISC-19: Names rusqlite for openhuman persistence (Grep)
- [ ] ISC-20: Names bun:sqlite + FTS5 for atlas-recall persistence (Grep)
- [ ] ISC-21: Names React 19 + Vite for openhuman UI (Grep)
- [ ] ISC-22: Notes atlas-recall has NO UI layer (Grep)
- [ ] ISC-23: Names axum for openhuman HTTP (Grep)
- [ ] ISC-24: Names MCP SDK for atlas-recall integration surface (Grep)
- [ ] ISC-25: Names pnpm workspace for openhuman (Grep)
- [ ] ISC-26: Names "Observability" or "OpenTelemetry/Sentry" for openhuman (Grep)
- [ ] ISC-27: Lists openhuman dep count vs atlas-recall dep count (Grep numbers)
- [ ] ISC-28: Brief one-paragraph synthesis below the table (Grep "synthesis" or similar heading)
- [ ] ISC-29: Anti: file does NOT exceed 200 lines (Bash: `wc -l`)
- [ ] ISC-30: Anti: file does NOT include unverifiable claims about features not present in manifests
- [ ] ISC-31: Anti: comparison rows do NOT invent dependencies that aren't in the manifests
- [ ] ISC-32: Path matches user verbatim spec `/Users/ed/Developer/openhuman-research/01-tech-stack.md` (Bash: `test -f` exact match)

## Test Strategy
| isc | type | check | threshold | tool |
|-----|------|-------|-----------|------|
| 1, 32 | file-exists | `test -f` | exit 0 | Bash |
| 2, 29 | line-count | `wc -l` | ≤200 | Bash |
| 3–28 | grep | string match | match found | Bash/Grep |
| 30, 31 | inspection | manual read against manifest evidence | no false claims | Read |

## Features
| name | satisfies | depends_on | parallelizable |
|------|-----------|------------|----------------|
| Gather openhuman manifests | ISC-15,17,19,23,26 | — | yes |
| Gather atlas-recall manifests | ISC-16,18,20,24 | — | yes |
| Compose comparison table | ISC-3–14, 21–27 | both gathers | no |
| Write synthesis paragraph | ISC-28 | table | no |
| Save to canonical path | ISC-1, 32 | composition | no |
| Verify line count + greps | ISC-2, 29 + verification probes | save | no |

## Decisions
- 2026-05-16T11:40Z — **Skill("ISA","scaffold...") not invoked at OBSERVE.** Show-your-math: the task is a single ≤200-line research artifact at known path; inline ISA write hits all E3 completeness-gate sections in <1 minute vs skill-spawn overhead ~30-60s. The ISA structure below conforms to v6.2.0 twelve-section spec.
- 2026-05-16T11:40Z — **Delegation soft-floor relaxed.** Show-your-math: this is a directed gh-API + Read + Write flow with no whole-project code reasoning. Forge (coding) and Anvil (long-context coding) don't apply; Research-skill agents would add latency without changing manifest-grounded findings I can produce directly. Manifest fetching done with gh CLI (already authenticated).
- 2026-05-16T11:40Z — **Source of truth = manifest files, not README.** README marketing language is excluded to avoid unverifiable claims (anti-ISC 30/31).

## Changelog
_(empty until LEARN)_

## Verification
_(populated at VERIFY)_
