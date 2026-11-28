# GAP CLI UX: help, examples, errors, flags, status, man, version

Source: GAPUX tab audit of the installed `gap` (built from source, fd87eac6), 2026-11-28, plus the sync-feedback fix from the "gap sync feedback" tab.

## Outcome
A newcomer sees the key commands on the first screen of `gap --help`, can copy a working example for every command, get the same named-argument error from every command, see what each flag takes, read `gap status` at a glance, open `man gap` (and the tldr page, by path until tldr-pages accepts it), and paste a useful `gap -v` into a bug report. Piped output and scripts see no change.

## Constraints
- Pipe output stays byte for byte: `gap status | cat`, `gap sync` off a terminal, every `--json` path. `inventory_test.go:43` and scripts depend on it.
- Never run a real `gap sync` with a temp HOME to check UI (component hooks re-point the user's launchd).
- Errors from U3 must use the same shape that Track 2 A3 (structured errors, `2026-11-27-gap-tui-vim-axi-mcp-plan.md`) will encode, so A3 wraps U3 and does not replace it.
- `go test ./...` passes after every unit.

## Implementation Units

### U0. Land the sync-feedback fix
- Files: `cmd/gap/inventory.go`, `components.go` (loadLocalManifests), `main.go`, `tui_run.go`, `ui_progress.go`, `cmd/gap/AGENTS.md`.
- Change: stores checked in parallel; `startWorking` spinner from the first line until the picker; `handoff.Run` prints only the subcommand name (never its arguments, which can carry credentials) the moment the TUI closes. The store check keeps a fresh index from `HEAD`; reusing the store's own index was dropped after review (its stat cache, flags, and line-ending history can hide edits, and `gap status` must write nothing).
- Verify: pty run from `~`: first output under 0.25 s (was 13.7-16.3 s); `go test ./...`.

### U1. Grouped, short root help
- Files: `cmd/gap/cli.go` (rootCmd, AddGroup, GroupID per command).
- Change: one-line description; groups Get started (sync, status, tui, add), Manage (update, remove, check, pin, restore), Inspect (list, source, doctor, search), Setup (init, completion, self-update, uninstall). Move the distro/component/pin/lock/store/harness text to a `gap help concepts` topic that links CONCEPTS.md.
- Verify: the first 25 lines of `gap --help` show the description, usage, GET STARTED and MANAGE (fang pads headings and lists every visible command, so the whole help cannot fit); a test fails when a visible command has no GroupID; TestHelpListsCommands still passes; `concepts` is hidden, has no RunE, and `gap help concepts` prints the terms page (fang has no Additional help topics section).

### U2. Examples on every command
- Files: `cmd/gap/cli.go`.
- Change: fill `Example` (`gap sync --only policy`, `gap sync --skip recall -y`, `gap pin do-skills@master`, `gap add -g owner/repo`, and one per other command).
- Verify: a test walks rootCmd and fails on any visible command with an empty Example. Run each example against a temp HOME only for read-only commands.

### U3. One named-argument error
- Files: `cmd/gap/cli.go` (pinCmd, removeCmd; the `id@ref` shape check lives in pinCmd's Args validator), `cmd/gap/add.go` (printMissingSource). `components.go` stays unchanged.
- Change: a shared Args validator that names the missing argument and prints usage plus an example via the printMissingSource style; replaces cobra's "accepts 1 arg(s), received 0" and the lowercase `usage: gap pin` line. Return a typed error A3 can encode later.
- Verify: `gap pin`, `gap pin foo`, `gap remove`, `gap add` all print the same three-part error and exit 1; a table test covers each.

### U4. Flag value names and completion
- Files: `cmd/gap/cli.go`, `cmd/gap/completion.go`.
- Change: backtick value names in pflag usage (`--only groups`, comma-separated stated) for `--only`, `--skip`, `--profile`, `--agent`, `--skill`, `--id`; register flag and ValidArgs completion for component ids, `policy`, and profile names.
- Verify: help shows `--only groups`; `gap __complete sync --only ""` lists policy and the ids; a completion test covers it.

### U5. Readable `gap status` on a terminal
- Files: `cmd/gap/ui_table.go` (statusSummary already prints a terminal summary and Next steps on master), `cmd/gap/style.go`; change only what is still missing.
- Change: terminal only: summary line (`15 components · stores ok · 0 skills drifted`), aligned columns, and a closing next step (`gap restore <id>`) when something drifted. Off a terminal, today's key=value lines unchanged.
- Verify: golden test for the terminal view; `gap status | cat` equals today's output byte for byte (capture before the change).

### U6. Man page and tldr page
- Files: `cmd/gap/cli.go` (drop `fang.WithoutManpage()`, keep `man` hidden), `scripts/install.sh`, `scripts/install_test.sh`, `docs/tldr/gap.md` (new).
- Change: installer writes `gap man` output to `<bin dir>/../share/man/man1/gap.1` (macOS man searches `../share/man` beside each PATH entry); when `man -w gap` still fails, print a MANPATH hint like `path_hint`. tldr page in upstream format, kept in this repo; `tldr gap` by name needs the page accepted upstream (follow-up task).
- Verify: `gap man > "$TMPDIR/gap.1" && man "$TMPDIR/gap.1"` renders (BSD man has no `-l`; `mandoc -a` also works); after install, `man -w gap` in a fresh shell finds the page with no MANPATH change; install_test asserts the page path and the hint branch; render `docs/tldr/gap.md` with the installed client's render flag and check it against the tldr style guide (one `#` title, `>` summary with a `More information` link, 8 examples or fewer, `{{placeholders}}`). Update README install section and the site docs page it feeds.

### U7. Useful version from source builds
- Files: `cmd/gap/cli.go`, next to `var version`.
- Change: when no version is baked in, read `debug.ReadBuildInfo` and print `vcs.revision` short plus `dirty` when `vcs.modified=true`.
- Verify: after `go install ./cmd/gap`, `gap -v` shows the short commit; a unit test feeds fake build info.

## Order
No unit edits a U0 file, so U3 and U7 no longer wait on U0. U1 -> U2 -> U4 run in sequence because all three edit cli.go. U5 and U6 are independent; U6's tldr examples may reuse U2's. One PR per unit.

## Docs
Update `cmd/gap/AGENTS.md` for U3 (error shape) and U5 (terminal vs pipe contract); README and site for U1, U6.

## Status
U1-U7 landed in #188 (squash 680cc030). U0 ships in its own PR from branch gap-sync-feedback. Open follow-up: submit docs/tldr/gap.md to tldr-pages so `tldr gap` resolves by name; until then the page renders only by path.

## todo.txt
- (A) U0: Ship the gap sync feedback fix (spinner, handoff line, parallel store check) in its own PR +gap id:gap-sync-feedback note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-28-gap-cli-ux-plan.md
- (C) Submit docs/tldr/gap.md to tldr-pages so tldr gap resolves by name +gap id:gap-tldr-upstream note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-28-gap-cli-ux-plan.md
