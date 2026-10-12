# todo.txt through tuxedo

Two kinds of list. A project's tasks live in `<repo root>/todo.txt`, with history in its `done.txt`. `~/todo.txt` holds personal and cross-project tasks plus one route per active project, so `do-regroup` can reach every project from one file. The `do-todo` skill has the full format, CLI, `--json` shapes, routing, and end-of-turn table; read it when it is installed.

## Rules

- Use `tuxedo` only. Never `todo.sh` (its numbering differs), never the TUI, never hand-edit the file.
- Pick the list, then name both files on every command, with absolute paths: `TODO_FILE=<dir>/todo.txt DONE_FILE=<dir>/done.txt tuxedo ...`, where `<dir>` is the repo root for a project task or `$HOME` for a personal or cross-project task. Never pass only `TODO_DIR`: an exported `TODO_FILE` beats it and silently writes to another file, and an exported `DONE_FILE` redirects `archive`. With neither set, tuxedo opens `./todo.txt` or a sample file.
- Pass `--json` before the command (`tuxedo --json ls`). Find a task by its `raw` text in `tuxedo --json ls`, mutate it by its `n`, and re-list before the next mutation. Numbers shift after `del`, `do`, and `archive`. Pass `-f` to `del`.
- `<project>` is the lowercased basename of the repo root. Tag every task `+<project>` in both kinds of list.
- A route is one line in `~/todo.txt`: `<Name> +<project> @route todo:<absolute path to repo todo.txt>`. Add it the first time a project gets a task, or when a repo has a `todo.txt` but no route. `do` the route when the project is finished or parked. Never copy project tasks into `~/todo.txt`.
- At the end of every turn that adds, finishes, blocks, or changes work, run the matching mutation on the right list in that turn. The human never has to. tuxedo has no daemon and each CLI write is atomic, so never leave a TUI or watcher running for this. Do not append to `inbox.txt`; only a running TUI drains it.
- Every plan, spec, and handoff has a `## Todo` section of todo.txt lines, one per open task:
  - `+<project>` on every line, using the repo name.
  - `id:<slug>` on a task others wait for, and `dep:<slug>` on the tasks that wait.
  - `@waiting` plus the reason for a hold on a person or decision. `due:YYYY-MM-DD` when a date is known.
  - `note:<absolute path>` pointing at the plan, spec, or handoff.
- Add each line with `tuxedo --json add` on the project's list, after `tuxedo --json ls` shows it is not already there.
- When a task is finished, `tuxedo --json app N pr:<url>` (or `report:<path>`), then `tuxedo --json do N`.
- Before writing a handoff, list the project's `todo.txt` and copy the open tasks into its `## Todo` section.

## Done

The document's `## Todo` lines match the open tasks in the project's `todo.txt`, the project has an open route in `~/todo.txt`, and no task from this turn is missing from its list.
