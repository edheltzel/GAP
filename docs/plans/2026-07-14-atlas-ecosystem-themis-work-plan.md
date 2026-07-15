# Atlas Ecosystem — Prioritized Work Plan for Themis

**Date:** 2026-07-14 (rev 2 — board consolidation applied)
**Owner:** Ed → dispatched via Themis (delegating PM; workers/reviewers/explorers over herdr)
**Scope:** GitHub Project boards under `edheltzel`, prioritized against Strategy **S2** — *make every Atlas component agnostic and plug-n-play so it ports across harnesses.*
**Track emphasis:** **Recall leads** (structural consolidation done first); Echo runs in parallel.

---

## Board structure (after this session's consolidation)

Seven boards were closed. Recall's six phase/track boards collapsed into **one board + milestones**; Better Solarized (100% done) was closed.

**Active boards:**

| Board | Open | Repo |
|-------|------|------|
| **#28 Recall** (consolidated) | 28 active issues, grouped by Milestone | Recall |
| #20 Atlas Echo | 20 Todo | Echo |
| #26 github-projects skill overhaul | 2 Backlog | atlas-config |

**Recall milestones (replace the old phase boards; dated now → end of summer):**

| Milestone | Progress | Due | State |
|-----------|----------|-----|-------|
| Phase 0 — Hardening & Dedup | 13/13 | 2026-06-15 | closed |
| Phase 1 — Memory Intelligence | 35/35 | 2026-07-01 | closed |
| Code Knowledge Graph | 10/10 | 2026-07-10 | closed |
| **Phase 2 — Distribution & Packaging** | 9/16 | **2026-07-31** | open |
| **Phase 3 — Backend Portability** | 11/19 | **2026-08-22** | open |
| **Phase 4 — Beyond Coding Agents** | 4/9 | **2026-09-19** | open |

**Closed boards:** #14 Better Solarized, #15/#16/#17/#18/#19 Recall phases, #27 Recall Code-KG. All recoverable via `gh project close --undo`.

---

## Priority ranking (most important → blocked)

**P1 — Recall / Phase 2: Distribution & Packaging (due Jul 31).** Dead-center on S2, fully actionable, nearest due date, and the gate for Recall's public story. Contains the installer/uninstall-safety debt that must land before any publish.

**P2 — Recall / Phase 3: Backend Portability (due Aug 22).** Hardening items (`#163 #192 #226 #227`) actionable now; the Turso/libSQL epic (`#102`) is **blocked on go/no-go**.

**P3 — Atlas Echo: agent-agnostic + distribution (#20).** Largest actionable surface (20 items); the `#30` "make voice agent-agnostic" epic is the purest S2 work. Independent of Recall — fans out in full parallel.

**P4 — Recall / Phase 4: Beyond Coding Agents (due Sep 19).** Mostly gated by epic `#103`; its redaction/scrub items (`#182 #194 #178`) pull *forward* into the security gate below.

**P5 — github-projects skill overhaul (#26).** Internal PM tooling, 2 backlog items. Lowest — does not advance S2.

**⛔ Blocked — needs Ed's decision (the real "blockers"):**

- `Recall #102` — Turso/libSQL multi-device (Phase 3 epic) *(pending go/no-go)*
- `Recall #103` — Beyond Coding Agents build-out (Phase 4 epic) *(pending go/no-go; 4/9 already done, gate is on the remaining push)*

---

## Cross-cutting constraint (hard sequencing rule)

**Security hardening gates all public distribution.** Do **not** publish an npm package (`Recall #1`, `Echo #8`) or reduce install surface until redaction and uninstall-safety land + pass RedTeam:

- `Recall #182` — scrub `learnings.tags` (un-redacted secret persist)
- `Recall #194` — email/phone/path redaction patterns
- `Recall #236` — uninstall wipes user-authored files
- `Recall #237` — `doctor --fix` symlink repairs unguarded

---

## Themis execution plan

Recall leads; Echo runs parallel. Each issue → one worker in a **worktree against the GitHub repo** (`edheltzel/Recall`, `edheltzel/Echo`, `edheltzel/atlas-config`), one reviewer, RedTeam on anything security-touching. Every brief carries the codegraph + "never assume structure" rule. Track Recall progress by **milestone due date**.

### Wave 0 — Themis only (no code)
1. **Triage the unslotted open Recall issues** (`#195 #211 #212 #221 #231 #232 #233 #234` — on the board, no milestone) into Phase 2/3/4 before dispatch.
2. **Surface the 2 go/no-go epics** (`#102`, `#103`) to Ed. Nothing in the blocked tier moves until decided.

### Wave 1 — Security hardening (parallel · gates distribution)
`Recall #182 · #194 · #236 · #237` — each worker + reviewer + **RedTeam validate**. Unlocks all publish work.

### Wave 2 — Recall Phase 2 → shippable (parallel · after Wave 1 · due Jul 31)
`Recall #124 · #174 · #165 · #187 · #238` (installer/doc/CI debt) → then `Recall #1` (npm package).

### Wave 3 — Echo track (parallel · independent · start immediately)
- Agnostic epic `Echo #30` → adapters `#16 Codex`, `#17 OpenCode`.
- Distribution: `#8 npm` (holds behind Echo install hardening `#12 #28`).
- Backlog hardening/parsers (fan out wide): `#44 #45 #50 #51 #55 #63 #94 #95 #96 #98 #102 #106 #29 #13`.

### Wave 4 — Recall Phase 3 reliability (parallel · due Aug 22)
`Recall #163 · #192 · #226 · #227 · #155 · #193` + spike `#106` (oMLX embedding backend). Non-blocking quality/perf debt. Epic `#102` waits on go/no-go.

### Wave 5 — Post go/no-go (only if Ed green-lights)
`Recall #102` (Phase 3 Turso/libSQL) and `Recall #103` (Phase 4 build-out) decompose into their own waves under their milestones.

### Wave 6 — Tooling (lowest)
`atlas-config #25` (interview + scaffold), `#26` (golden projects).

---

## Decisions needed from Ed

1. **Go/no-go on `Recall #102`** (Turso/libSQL multi-device) — proceed, defer, or drop?
2. **Go/no-go on `Recall #103`** (Beyond Coding Agents remaining push) — proceed, defer, or drop?
