# Recall breadcrumb and Echo adapter ledger

## Outcome

Close the two items the env provisioner plan named and M3, M5, and M6 left unfinished. `gap init` writes one Recall breadcrumb for the project it provisions, and `gap doctor` records which Echo adapters are installed into `~/.gap/state.json`. The breadcrumb goes through `recall`. The adapter list needs one new flag in Echo, because nothing Echo exposes today returns it.

## Redteam, 2026-10-22

Four angles attacked the argument, then two candidates were built for the adapter mechanism and a critic judged them.

Held:

- The idempotency check is sound. `recall recent breadcrumbs -p <slug>` filters on the project column (`recentBreadcrumbs` queries `WHERE project = ?`), and `showRecentBreadcrumbs` prints `b.content` in full. The 60-character truncation the attack cited is in `showRecentAll`, which only runs when no table is named. Passing `-l 1000` removes the default limit of 10, so a busy project cannot scroll its own provisioning note out of view.
- A missing `recall` stays a notice. Init has already written its files by then, and failing the command over an optional memory tool punishes a machine that never installed one.
- Replacing `state.json` whole is right. The file is a picture of one moment. A history of pictures is a log, and nobody asked for one.

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
- Files: `cmd/gap/init.go`, `cmd/gap/init_test.go`, `cmd/gap/AGENTS.md`
- Change: after `initProject` writes `.gap/gap.toml`, it runs `recall add breadcrumb "gap provisioned <slug>" -p <slug> -c setup`. The slug is the one written into `.gap/gap.toml`. A second `gap init` on the same repo does not add another. Gap asks `recall recent breadcrumbs -p <slug> -l 1000` first and skips the add when any returned line contains `gap provisioned <slug>`. `-l 1000` matters: the default limit is 10, so a project with a long breadcrumb history would otherwise hide its own note and get a second one. The project filter is real (`src/index.ts` registers `-p`, and `recentBreadcrumbs` queries `WHERE project = ?`). The text match stays, because the project filter alone would also match a breadcrumb someone else wrote for that project. `recall` missing from `PATH` prints one line and does not fail the init. The breadcrumb text and both commands are fixed. Gap does not write Recall's database itself.
- Done when: `gap init` on a repo with no prior breadcrumb runs the add command once, and a second `gap init` on the same repo runs no add command.
- Verify: `go test -count=1 -run 'TestInitBreadcrumb|TestInitBreadcrumbIdempotent|TestInitBreadcrumbNoRecall' ./cmd/gap`. The tests stub `recall` on `PATH` the way `doctor_test.go` stubs it, and assert the argv it received.

### U2. Echo adapter ledger

- Depends on: U1
- Files: `cmd/gap/doctor.go`, `cmd/gap/doctor_test.go`, `cmd/gap/AGENTS.md`
- Change: `gap doctor` grows a `~/.gap/state.json`. For the `echo` component it runs `scripts/install.sh --list-installed` inside the pinned store and reads adapter names, one per line. It records them keyed by component id, with the store commit from `gap.lock`, and rewrites the file whole on each run so a second run replaces the list. The flag does not exist in the pinned Echo yet. Until it does, the command exits 2 with `Unknown argument`, and gap prints that echo reported nothing and writes no echo entry. Gap does not parse `--check` output, does not source `install.sh`, and does not reimplement the checks. It does not install, remove, or reorder adapters.
- Blocked on: Echo. `scripts/install.sh` needs a `--list-installed` branch beside `--check` that runs the ten `*_installed` functions, prints each installed name, and exits 0 before `warn_if_linked_worktree`. Gap cannot pin that until Echo tags it. This unit's test uses a fixture script, so it does not wait on the tag. The real pin does.
- Done when: after `gap doctor`, `~/.gap/state.json` lists exactly the adapters whose checks returned installed, and a second run with a different set replaces the list.
- Verify: `go test -count=1 -run 'TestDoctorRecordsAdapters|TestDoctorLedgerReplaces|TestDoctorLedgerNoFlag' ./cmd/gap`. The first two point the echo store at a fixture script that prints two adapter names and exits 0, then read `state.json` back and compare the names and the commit. The third points it at a script that exits 2, and asserts there is no echo entry and the run still finishes.

### U3. Contract

- Depends on: U2
- Files: `cmd/gap/AGENTS.md`
- Change: the Local Contracts paragraph names both behaviors. `gap init` writes one Recall breadcrumb per project and skips when one exists. `gap doctor` records Echo's installed adapters in `~/.gap/state.json` and replaces that record on each run. Both say what they do not do: gap does not write Recall's database, and it does not change Echo's adapters.
- Done when: `cmd/gap/AGENTS.md` states both behaviors and both limits.
- Verify: `grep -n "breadcrumb\|state.json" cmd/gap/AGENTS.md` shows both, and `go test -count=1 ./cmd/gap` passes.

## Out of scope

- Installing or removing Echo adapters. Echo's installer owns that, and the plan records what it did.
- Writing into Recall's SQLite database. The breadcrumb goes through `recall add`.
- The render ledger the env provisioner plan describes for skill files. This ledger holds Echo's adapters only.
- Any change to `gap.toml` or to a pinned commit other than echo's. Pinning the Echo commit that ships `--list-installed` is the one pin this plan allows, and it waits on Echo's tag.

## todo.txt

- 2026-10-22 U1 gap init writes one Recall breadcrumb per project and skips when one exists +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md
- 2026-10-22 U2 gap doctor records Echo's installed adapters in ~/.gap/state.json +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md
- 2026-10-22 U3 record both behaviors in cmd/gap/AGENTS.md +gap note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md

Review complete
