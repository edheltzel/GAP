# Recall breadcrumb and Echo adapter ledger

## Outcome

Close the two items the env provisioner plan named and M3, M5, and M6 left unfinished. `gap init` writes one Recall breadcrumb for the project it provisions, and `gap doctor` records which Echo adapters are installed into `~/.gap/state.json`. The breadcrumb goes through `recall`. The adapter list needs one new flag in Echo, because nothing Echo exposes today returns it.

## Redteam, 2026-10-22

Four angles attacked the argument, then two candidates were built for the adapter mechanism and a critic judged them.

Held:

- Idempotency needs a project-scoped search of the full breadcrumb history. A recent-record limit of 1,000 still loses the setup note after 1,000 newer entries. Recall's `search` filters by project and table before limiting matches, so a quoted phrase query can find the setup note regardless of its age. Search previews truncate at 100 characters; use the returned breadcrumb record header to detect a match.
- A missing `recall` stays a notice. Init has already written its files by then, and failing the command over an optional memory tool punishes a machine that never installed one.
- Replace Echo's prior report after a successful read, rather than appending history. The [components reference](../../README.md#components) owns ledger updates and skipped-report behavior.

Rejected:

- "Shell the `*_installed` functions." They are not exported, and `scripts/install.sh` has no flag that runs only them. Sourcing the file is not a read: after the function definitions the top level calls `warn_if_linked_worktree` and then the install path.
- "`install.sh --check` already reports this." It calls the `*_installed` functions and then, for each that returns true, runs `bun run adapters/<name>/reconcile.ts --check`. It also checks the LaunchAgent plist and the workspace links. Its output is colored prose, and exit 3 means stale paths, not an adapter list. Parsing it would couple gap to Echo's wording.
- Reimplementing the ten checks in Go. Each one is a path plus a grep that Echo changes when it changes an adapter. A copy in gap drifts, and a drifted copy writes a record that looks authoritative and is wrong.

Judged:

- The Pragmatist was told to ship from gap's tree and proposed a new `cli/echo` command anyway. That broke the brief, so it lost.
- The Skeptic's `--list-installed` flag is the smallest change that works, and it won. The critic overstated its risk: the flag is one more branch in the argument `case` at `scripts/install.sh:67`, next to `--check`, and the top-level install only runs when `CHECK_ONLY` is 0. A new branch that prints and exits never reaches it. No guard refactor is needed.
- Taken from the Pragmatist: the record names the store commit alongside the adapter names, so a later run can see which Echo produced the list.

Open, and accepted: the flag does not exist in the pinned Echo (`f8001264`, v0.13.0). Until Echo ships it and gap pins that commit, `gap doctor` records nothing for echo and says so. That is a visible gap, not a guessed list.

## Implementation Units

### U1. Recall breadcrumb from gap init

- Depends on: none
- Files: `cmd/gap/provision.go`, `cmd/gap/init.go`, `cmd/gap/init_test.go`, `cmd/gap/env_review_test.go`
- Change: validate `.gap/gap.toml` and its non-empty string slug before writing project files, preserving existing configuration. Only after successful provisioning, run `recall search '"gap provisioned <slug>"' -t breadcrumbs -p <slug> --include-duplicates -l 1`. Double any quotes in the note inside the FTS phrase. This searches all matching project breadcrumbs before limiting the result, including records Recall marked as duplicates. A returned breadcrumb record skips the add regardless of preview truncation. Otherwise run `recall add breadcrumb "gap provisioned <slug>" -p <slug> -c setup`. Adopted files stay byte-identical; keeping an existing adopted or marked file still counts as successful provisioning. Failed provisioning writes no breadcrumb. An explicit or persisted crew profile never calls Recall. `recall` missing from `PATH` prints one line and does not fail init. Gap does not write Recall's database itself.
- Done when: `gap init` on a repo with no prior breadcrumb runs the add command once, and a second `gap init` on the same repo runs no add command.
- Verify: `go test -count=1 -run 'TestInit|TestEnvReviewInit|TestEnvReviewCrew|TestParseTOML|TestOverride|TestProjectOverride|TestCrewProfile|TestNoOverride|TestEnvReviewEffectiveSelection' ./cmd/gap`. The tests use a Recall CLI fixture with isolated history, check one setup note after 1,000 newer notes, and exercise persisted slugs, truncated previews, adoption, crew, and provisioning failures without touching the real Recall database. Parser and override checks cover boolean types and component selection.

### U2. Echo adapter ledger

- Depends on: U1
- Files: `cmd/gap/provision.go`, `cmd/gap/doctor.go`, `cmd/gap/doctor_test.go`
- Change: implement the [Echo adapter reporting contract](../../README.md#components) in `gap doctor`. Read adapter names, one per line, from `scripts/install.sh --list-installed` in the verified pinned store. Do not parse `--check` output, source `install.sh`, or reimplement Echo's checks.
- Blocked on: Echo. `scripts/install.sh` needs a `--list-installed` branch beside `--check` that runs the ten `*_installed` functions, prints each installed name, and exits 0 before `warn_if_linked_worktree`. Gap cannot pin that until Echo tags it. This unit's test uses a fixture script, so it does not wait on the tag. The real pin does.
- Done when: after a successful Echo adapter report, `~/.gap/state.json` lists exactly the adapters whose checks returned installed, and a later successful report with a different set replaces the list.
- Verify: `go test -count=1 -run 'TestDoctorRecordsAdapters|TestDoctorLedgerReplaces|TestDoctorLedgerNoFlag' ./cmd/gap`. The first two point the echo store at a fixture script that prints two adapter names and exits 0, then read `state.json` back and compare the names and the commit. The third points it at a script that exits 2, and asserts there is no echo entry and the run still finishes.

### U3. Contract

- Depends on: U2
- Files: `README.md`, `cmd/gap/AGENTS.md`
- Change: keep user-facing behavior in README's [project setup](../../README.md#project-setup) and [components](../../README.md#components) sections. Reduce the stale Local Contracts copy to pointers to those owners.
- Done when: README describes setup breadcrumbs, Echo adapter reporting, and their limits, with pointers from `cmd/gap/AGENTS.md`.
- Verify: re-read those sections against `provision.go`, `init.go`, and `doctor.go`; `go test -count=1 ./cmd/gap` remains the behavioral check.

## Out of scope

- Installing or removing Echo adapters. Echo's installer owns that, and the plan records what it did.
- Writing into Recall's SQLite database. The breadcrumb goes through `recall add`.
- The render ledger the env provisioner plan describes for skill files. This ledger holds Echo's adapters only.
- Component and pin changes, as defined in the [requirements](2026-10-22-gap-breadcrumb-ledger-requirements.md#out-of-scope).

## todo.txt

- 2026-10-22 U1 gap init writes one Recall breadcrumb per project and skips when one exists +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md
- 2026-10-22 U2 gap doctor records Echo's installed adapters in ~/.gap/state.json +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md
- 2026-10-22 U3 document both behaviors in README and point to them from cmd/gap/AGENTS.md +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md

Review complete
