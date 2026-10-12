# todo.txt through tuxedo

The task list is `~/todo.txt`, with done history in `~/done.txt`. The `do-todo-txt` skill has the full format, CLI, and `--json` shapes; read it when it is installed.

## Rules

- Use `tuxedo` only. Never `todo.sh` (its numbering differs), never the TUI, never hand-edit the file.
- Unless `TODO_FILE` or `TODO_DIR` is set, prefix every command with `TODO_DIR=~`. Without it, tuxedo opens `./todo.txt` or a sample file.
- Pass `--json` before the command (`tuxedo --json ls`). Find a task by its `raw` text in `tuxedo --json ls`, mutate it by its `n`, and re-list before the next mutation. Numbers shift after `del`, `do`, and `archive`. Pass `-f` to `del`.
- Every plan, spec, and handoff has a `## Todo` section of todo.txt lines, one per open task:
  - `+<project>` on every line, using the repo name.
  - `id:<slug>` on a task others wait for, and `dep:<slug>` on the tasks that wait.
  - `@waiting` plus the reason for a hold on a person or decision. `due:YYYY-MM-DD` when a date is known.
  - `note:<absolute path>` pointing at the plan, spec, or handoff.
- Add each line with `tuxedo --json add`, after `tuxedo --json ls +<project>` shows it is not already there.
- When a task is finished, `tuxedo --json app N pr:<url>` (or `report:<path>`), then `tuxedo --json do N`.
- Before writing a handoff, list `+<project>` and copy the open tasks into its `## Todo` section.

## Done

The document's `## Todo` lines match the open `+<project>` tasks in `~/todo.txt`.
