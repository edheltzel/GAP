# gap-tui

## Goal

A terminal UI inside the `gap` binary, built like AUP (`~/Developer/Tools/AUP`), where you can see and manage everything GAP installed: components, utilities, and the local projects that use GAP.

## In scope

- **Items view.** Lists every component and utility with its source (distro or user), scope (global or project), ref, pinned commit, store state, skill drift, and the `new=N` count from the last `gap check`.
- **Update one.** Updates the selected item. The result matches `gap update <id>`: a distro pin moves in `gap.lock`, and a user utility installs right away. The TUI shows the commits it brought in.
- **Check.** Runs `gap check` for the selected item or for all items, with a spinner, and refreshes `new=N`.
- **Delete one.** Removes the selected user utility, with the same result as `gap remove <id>`, after one confirmation.
- **Mass delete.** Mark several user utilities, review the list of marked ids, confirm once, and remove them in order. One failure does not stop the rest, and the summary reports each id as removed or failed.
- **Status view.** Shows the same counts and states as `gap status` (pins, store missing/changed/unpinned, drift), plus the repair command for each problem.
- **Projects view.** Lists every registered local project with its root, profile, the project-scope items it uses, and their drift. A root that no longer exists is flagged, and you can forget it.
- **Project registry.** `gap init`, `gap add --project`, and `gap sync` run inside a repo each record that repo's root in `~/.gap`. The Projects view reads only this registry. Existing `projectSkills` roots in `~/.gap/state.json` are imported the first time.
- **Look and keys like AUP.** `j`/`k` to move, a running pane with live output, a progress bar for batch actions, a summary at the end, `esc` asks before cancelling, `q` quits.
- **Entry.** `gap tui` opens the TUI. Plain `gap` opens it too when stdin and stdout are a terminal.

## Out of scope

- Adding a utility from the TUI (`gap add` stays a CLI command).
- Publishing the policy (`gap sync` stays a CLI command).
- Editing `gap.toml` by hand in the TUI, except the disable action below.
- Deleting distro components (do-skills, echo, recall, forge, and the other `gap.toml` sources).
- Scanning folders to find projects.
- Windows.

## Success

- On a terminal, `gap` and `gap tui` open the TUI. When piped, with `TERM=dumb`, or with `NO_COLOR`, `gap` prints today's help and the TUI does not start (plain output, like AUP's `--plain`).
- Updating one item changes only that item's entry in the right lock. A `git diff` of `gap.lock` and `~/.config/gap/gap.lock` shows nothing else.
- Deleting one utility, or a marked set, leaves each removed copy under `~/.gap/backups/<time>/`. After the run, `gap list` no longer shows those ids.
- A mass delete where one id fails still removes the others, and the summary names the failed id.
- The Status view's counts match `gap status` run at the same moment.
- After `gap init` in a new repo, the Projects view lists that repo. After that folder is deleted, the view flags it, and forgetting it removes it from the registry.
- A distro component shows as locked, with no delete action available.
- `go test ./...` passes, with end-to-end tests that drive the TUI against throwaway homes.

## Constraints

- Go, inside the existing `gap` binary, using the Charm v2 libraries GAP already depends on.
- Behavior matches the CLI commands. The TUI must not add its own update or remove rules.
- Everything destructive keeps GAP's existing backup and ledger rules.
- macOS and Linux, the same platforms as the `tty_*` files.

## Decisions

- Projects come from a registry, not a folder scan (user choice, 2026-11-23).
- Assumed: delete applies only to user utilities. Distro components get a "disable" action, which writes `enabled = false` under `[components.<id>]` in `~/.config/gap/gap.toml`.
- Assumed: plain `gap` on a terminal opens the TUI, and `gap tui` is the explicit way in.
- Assumed: every delete asks for one confirmation, and a mass delete lists every marked id before you confirm.

## Open questions

- Where does the registry live: a new key in `~/.gap/state.json`, or its own `~/.gap/projects.json`? (This is a HOW question for do-plan.)
- Should the TUI offer "update all" (`gap update` with no id), or only per-item updates?
- Should the Projects view allow project actions such as `gap restore` inside that repo, or only display status?

## todo.txt

- Plan the gap TUI from docs/plans/2026-11-23-gap-tui-requirements.md with do-plan +gap
