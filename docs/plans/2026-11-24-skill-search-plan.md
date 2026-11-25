# skill search in gap tui

## Outcome

In `gap tui`, a user types to search the skills.sh catalog, sees each hit's install count, trusted-source mark, and whether it is already installed, and installs one skill everywhere or only in this project with one confirmation. Like Empryo's skills browser: `GET https://skills.sh/api/search?q=<text>&limit=N` returns `{skills:[{id, source, skillId, name, installs}]}` (checked 2026-11-24).

That needs `gap add` to accept a repo that has no `gap-utility.toml` when the caller names the skills to take. Distro components already work that way through `names` (`applyManifest`, cmd/gap/components.go:167); user components are refused there today.

## Implementation Units

### U1. gap add --skill for repos without a manifest
- Status: already on master. `gap add <source> --skill <name> [--global|--project]` installs chosen skills from any repo, so this unit was dropped when the work was rebased onto master.

### U2. skills.sh search client
- Depends on: none
- Files: cmd/gap/skillsearch.go, cmd/gap/skillsearch_test.go
- Change: `searchSkills(ctx, query, limit)` calls the search endpoint (base URL a package var for tests), 5 s timeout, parses hits, ranks by installs, marks trusted sources (vercel-labs, anthropics, microsoft, google-labs-code). Hits whose `source` is not `owner/repo` are dropped, because `gap add` needs a GitHub repo.
- Done when: a fake server's response parses into ranked hits with trust marks; HTTP errors and bad JSON return an error, not a panic.
- Verify: `go test ./cmd/gap -run TestSearchSkills` against httptest.

### U3. Find view in gap tui
- Depends on: U1, U2
- Files: cmd/gap/tui_search.go, cmd/gap/tui_model.go, cmd/gap/tui_view.go, cmd/gap/inventory.go, cmd/gap/tui_search_test.go
- Change: `/` opens a search pane: a text input, results that refresh 300 ms after typing stops, each row `name  source  installs  trusted  installed` in the two-pane layout with the hit's id and repo on the right. Installed-state comes from the loaded items: `itemRow` gains the component's skill names so a hit matches on repo plus skill. `enter` asks Everywhere or This project (the add form's scope select), then hands the terminal to `gap add <source> --skill <skillId> [--project]`. `esc` closes. A network error shows in the note line.
- Done when: typing queries the fake client once per pause, results render with marks, enter on a hit hands over `gap add <repo> --skill <name> --global|--project`, installed hits are marked.
- Verify: `go test ./cmd/gap -run TestTUISearch` with a stub search func; a tmux run with the real endpoint shows hits for "react".

### U4. Docs
- Depends on: U1, U3
- Files: README.md, cmd/gap/AGENTS.md
- Change: README Terminal UI gets `/`; `gap add` row and Publish a utility say a repo without `gap-utility.toml` installs with `--skill`. AGENTS.md names tui_search.go and skillsearch.go.
- Done when: every new key and flag is documented, and no heading moved.
- Verify: `grep -n -- '--skill' README.md` and `grep -n 'tui_search' cmd/gap/AGENTS.md` both hit.

## Out of scope
- Installing skills.sh entries from non-GitHub sources (for example `open.feishu.cn`); they are listed by the API but dropped.
- Searching any catalog other than skills.sh.
- Showing a skill's SKILL.md body before install.

## todo.txt
- Build skill search in gap tui (plan: docs/plans/2026-11-24-skill-search-plan.md) +gap

Review complete
