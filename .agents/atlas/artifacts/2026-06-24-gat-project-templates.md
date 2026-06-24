# GAT org Projects v2 archetype templates

**Date:** 2026-06-24
**Owner org:** `Gentle-Art-Technologies`
**Source of truth:** `origin/master:skills/github-projects/templates/archetypes.ts`
**Auth:** `gh` as `edheltzel` (`GITHUB_TOKEN`, scopes include `project`, `admin:org`)

Five reusable Projects v2 boards were created on the GAT org, one per archetype, each
configured to its archetype definition (Status vocabulary + colors, extra fields + colored
options) and marked **"Use as template"** via API. None are linked to any repo.

## KEY UNKNOWN — resolved

**"Use as template" IS API-settable.** Grounded by GraphQL introspection:

- `ProjectV2` exposes a readable boolean field `template`.
- `UpdateProjectV2Input` has **no** template field — but two dedicated mutations exist:
  `markProjectV2AsTemplate(input: {projectId})` and `unmarkProjectV2AsTemplate(...)`.
- All 5 were marked via `markProjectV2AsTemplate` and independently re-queried:
  `node(id) { ... on ProjectV2 { template } }` returns `true` for each.

No human UI step is required for the template flag.

## Summary table

| Archetype | Project # | URL | Template marked? | How |
|---|---|---|---|---|
| `solo-software` | 2 | https://github.com/orgs/Gentle-Art-Technologies/projects/2 | ✅ yes | API (`markProjectV2AsTemplate`) |
| `content-pipeline` | 3 | https://github.com/orgs/Gentle-Art-Technologies/projects/3 | ✅ yes | API (`markProjectV2AsTemplate`) |
| `research-tracker` | 4 | https://github.com/orgs/Gentle-Art-Technologies/projects/4 | ✅ yes | API (`markProjectV2AsTemplate`) |
| `life-os-tasks` | 5 | https://github.com/orgs/Gentle-Art-Technologies/projects/5 | ✅ yes | API (`markProjectV2AsTemplate`) |
| `open-source-repo` | 6 | https://github.com/orgs/Gentle-Art-Technologies/projects/6 | ✅ yes | API (`markProjectV2AsTemplate`) |

All five: `public=false`, **not** linked to any repo.

## Known limitations / human follow-ups

### 1. Views are UI-only (all archetypes)
The Projects v2 GraphQL API exposes **no view-creation/configuration mutations** (introspection
of `Mutation` for `view` returns only PR-review mutations — nothing for `ProjectV2View`). Each
board therefore has only the default single view. The archetype `views` (e.g. `Board by Status`,
the `solo-software` `Ready` table filtered `status:Ready,"In Progress"`) must be created by hand:
open each project → **+ New view** → set layout (Board/Table) and the filter from `archetypes.ts`.

### 2. `open-source-repo` "Type" field could not be created (reserved name)
`createProjectV2Field` with `name: "Type"` is rejected by the API:
`Name cannot have a reserved value, Name has already been taken`. "Type" is reserved in
Projects v2 (collides with GitHub's native issue **Type**). The board has Status, Priority, and
Difficulty; the archetype's `Type` single-select (Bug/Feature/Docs/Chore) is **missing**.
Not renamed silently (would violate "exactly per archetype"). Human options:
- Add a custom single-select in the UI under a non-reserved name (e.g. `Work Type`), or
- Use GitHub's native org-level issue **Types** feature instead of a board field, or
- Decide `type:bug` / `type:feature` canon labels already cover this and drop the field.

### 3. Labels not applied
Archetype `labels` are a **repo-level** concern (`gh label create`), not a project-board
property, and these templates are intentionally repo-unlinked. Labels are applied per-repo at
bootstrap (see the skill's `Labels.md`), so nothing was created here.

## Captured field + option IDs

> For later GraphQL mutations (`updateProjectV2ItemFieldValue`, etc.). Built-in fields
> (Title/Assignees/Labels/Milestone/Repository/…) omitted. IDs are stable until a field/option is
> renamed or recreated in the UI.

### Template: Solo Software (#2) — archetype `solo-software`

- Project ID: `PVT_kwDOEaJtNs4BblYQ`
- URL: https://github.com/orgs/Gentle-Art-Technologies/projects/2
- Template: `true`

| Field | Field ID | Options (name → id) |
|---|---|---|
| Status | `PVTSSF_lADOEaJtNs4BblYQzhWU59E` | Backlog → `baf3f38a`<br>Ready → `0fb2845f`<br>In Progress → `ac2d3529`<br>In Review → `c9eab387`<br>Done → `4b91d52f` |
| Priority | `PVTSSF_lADOEaJtNs4BblYQzhWU598` | P0 → `139000d9`<br>P1 → `2305fa52`<br>P2 → `052e8573`<br>P3 → `94ba6ab2` |
| Phase | `PVTSSF_lADOEaJtNs4BblYQzhWU5-E` | Phase 1 → `d9866aff`<br>Phase 2 → `4a2b6013`<br>Phase 3 → `c69c97e8` |
| Area | `PVTF_lADOEaJtNs4BblYQzhWU5-I` | — (TEXT) |

### Template: Content Pipeline (#3) — archetype `content-pipeline`

- Project ID: `PVT_kwDOEaJtNs4BblYR`
- URL: https://github.com/orgs/Gentle-Art-Technologies/projects/3
- Template: `true`

| Field | Field ID | Options (name → id) |
|---|---|---|
| Status | `PVTSSF_lADOEaJtNs4BblYRzhWU5-U` | Idea → `68e9f52a`<br>Drafting → `e1963681`<br>Editing → `837cd378`<br>Review → `55e2dab4`<br>Scheduled → `09d5149d`<br>Published → `050cb97b` |
| Priority | `PVTSSF_lADOEaJtNs4BblYRzhWU5_E` | P0 → `9e304b46`<br>P1 → `1f9880c0`<br>P2 → `6e241027`<br>P3 → `e2c6464a` |
| Channel | `PVTSSF_lADOEaJtNs4BblYRzhWU5_I` | blog → `b748834f`<br>newsletter → `f555a33a`<br>social → `7e7b4977` |
| Publish Date | `PVTF_lADOEaJtNs4BblYRzhWU5_M` | — (DATE) |

### Template: Research Tracker (#4) — archetype `research-tracker`

- Project ID: `PVT_kwDOEaJtNs4BblYT`
- URL: https://github.com/orgs/Gentle-Art-Technologies/projects/4
- Template: `true`

| Field | Field ID | Options (name → id) |
|---|---|---|
| Status | `PVTSSF_lADOEaJtNs4BblYTzhWU6AM` | Open Question → `d10a3395`<br>Investigating → `41ce4303`<br>Evidence Gathered → `e728c661`<br>Concluded → `63710a18`<br>Parked → `d577b460` |
| Priority | `PVTSSF_lADOEaJtNs4BblYTzhWU6BA` | P0 → `c8a47a15`<br>P1 → `6a29c611`<br>P2 → `43e40f6f`<br>P3 → `563f2835` |
| Confidence | `PVTSSF_lADOEaJtNs4BblYTzhWU6BE` | Low → `28b03d3c`<br>Medium → `862af3e9`<br>High → `5330ac71` |
| Source | `PVTF_lADOEaJtNs4BblYTzhWU6BI` | — (TEXT) |

### Template: Life-OS Tasks (#5) — archetype `life-os-tasks`

- Project ID: `PVT_kwDOEaJtNs4BblYU`
- URL: https://github.com/orgs/Gentle-Art-Technologies/projects/5
- Template: `true`

| Field | Field ID | Options (name → id) |
|---|---|---|
| Status | `PVTSSF_lADOEaJtNs4BblYUzhWU6BU` | Inbox → `f3cc0360`<br>Next → `a3ac8987`<br>In Progress → `2f7e300d`<br>Waiting → `95f7f3cd`<br>Done → `1e457317` |
| Priority | `PVTSSF_lADOEaJtNs4BblYUzhWU6DM` | P0 → `efddfc71`<br>P1 → `13ee1c81`<br>P2 → `81c03ba0`<br>P3 → `5f97a6ed` |
| Area | `PVTF_lADOEaJtNs4BblYUzhWU6DQ` | — (TEXT) |
| Due | `PVTF_lADOEaJtNs4BblYUzhWU6DU` | — (DATE) |
| Energy | `PVTSSF_lADOEaJtNs4BblYUzhWU6DY` | Low → `cd72ae82`<br>Medium → `93f5e937`<br>High → `015aeb84` |

### Template: Open-Source Repo (#6) — archetype `open-source-repo`

- Project ID: `PVT_kwDOEaJtNs4BblYY`
- URL: https://github.com/orgs/Gentle-Art-Technologies/projects/6
- Template: `true`
- **Missing field:** `Type` (reserved name — see limitation #2)

| Field | Field ID | Options (name → id) |
|---|---|---|
| Status | `PVTSSF_lADOEaJtNs4BblYYzhWU6FM` | Triage → `554abae3`<br>Accepted → `3a2f2589`<br>In Progress → `cf21aca3`<br>In Review → `d2c47e72`<br>Done → `0273e801` |
| Priority | `PVTSSF_lADOEaJtNs4BblYYzhWU6F8` | P0 → `e346203e`<br>P1 → `48b0bb38`<br>P2 → `b105f14f`<br>P3 → `85a91f58` |
| Difficulty | `PVTSSF_lADOEaJtNs4BblYYzhWU6SU` | Easy → `bb8e5313`<br>Medium → `e4f1943a`<br>Hard → `7ed2ccb6` |

## How it was built (reproducible)

1. GraphQL introspection to confirm `markProjectV2AsTemplate`, `updateProjectV2Field`,
   `createProjectV2Field`, and the absence of view mutations.
2. `gh project create --owner Gentle-Art-Technologies --title "Template: <Name>"` per archetype.
3. `updateProjectV2Field` on the built-in **Status** field to replace `Todo/In Progress/Done`
   with the archetype's options + colors.
4. `createProjectV2Field` per extra field (SINGLE_SELECT options carry `color` + `description`).
5. `markProjectV2AsTemplate(projectId)` per board.
6. Independent re-query of `template`, fields, and option colors for verification.

Orchestration was an idempotent Bun/TypeScript script (reuses same-titled projects, skips
existing fields) — run-only tooling, not committed to the repo.
