# Recall breadcrumb and Echo adapter ledger

## Goal

Finish the two behaviors the env provisioner plan promised and left undone. When gap provisions a project, Recall gets one note of it. When gap checks a machine, it keeps a record of which Echo adapters that machine has installed.

## In scope

- `gap init` leaves one Recall breadcrumb for the project, and a second run on the same project leaves the count unchanged.
- `gap doctor` writes a record of the Echo adapters currently installed, and a later run shows the current set rather than a history of sets.
- Both keep working when the thing they talk to is absent: no `recall` on the machine, or an Echo component with nothing to report. The command still finishes and says what it skipped.

## Out of scope

- Installing, removing, or reordering Echo adapters. Echo's own installer does that.
- Writing into Recall's database directly. The note goes through Recall's own command.
- A record of rendered skill files. This record holds Echo's adapters only.
- Any change to which components are pinned, or to their pinned commits.

## Success

- After `gap init` on a project, Recall holds exactly one breadcrumb naming that project. After a second `gap init`, it still holds one.
- After `gap doctor`, the record names exactly the adapters installed at that moment. After a second `gap doctor` where the installed set changed, the record names the new set and not the old one.
- `gap init` on a machine with no `recall` command completes and prints that it skipped the breadcrumb.

## Constraints

- Gap never writes Recall's database. It calls `recall add breadcrumb`.
- Gap never changes Echo's adapters. It reads the install state through a flag Echo does not have yet. Until that flag ships, gap records nothing for echo and says so.
- The breadcrumb text is fixed, so a repeat run can recognize its own earlier note.

## Decisions

- One breadcrumb per project, matched by a quoted phrase through `recall search` with project and breadcrumb-table filters. Search covers the complete history before limiting matches, so newer notes cannot hide the setup note. Include records marked as duplicates and detect a returned record independently of its truncated preview.
- The adapter record is replaced on every run, not appended.
- A missing `recall` is a notice, not a failure.

## Open questions

None. The behavior, the boundary, and the checks are all named above.

## todo.txt

Review complete
