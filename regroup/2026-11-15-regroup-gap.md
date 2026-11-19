# Regroup: GAP

Date: 2026-11-15
Sources: `todo.txt` and `done.txt` (tuxedo), GitHub PRs on edheltzel/GAP, `~/.jcode` and `~/AGENTS.md` on disk. Previous check-in: `regroup/2026-11-08-regroup-gap-2.md`.
Resume point: `docs/plans/handoffs/2026-11-05-gap-plans-sync.md` (predates the utilities and jcode work).

## Last completed

- 2026-11-14 Drop skills.sh: global/skills.json sources moved into gap.toml as pinned components (#123 merged).
- 2026-11-14 Tag v0.5.0 with the release workflow and Go-free install (docs #124 merged).
- 2026-11-14 Output kit: framed tables, spinners, NO_COLOR, themed help (landed with #123).
- 2026-11-13 jcode policy overlay built and merged (#121), plus gap uninstall, gap search, MCP entries, and release binaries (#120).

## Up next

Up next not specified: no open task has a priority or due date. Today's state points at:

- Merge #125 (fix/release-version), then tag v0.5.1 so release binaries report their version. Not started; PR open.
- Run gap sync, now unblocked since #123 merged: `~/.jcode/prompt-overlay.md` does not exist yet.
- Trash ~/AGENTS.md after that sync (jcode plan U5). Blocked on the overlay existing.

## Remaining

- Build jcode integration (id:jcode-build): #121 merged, so the code is done; the todo line was not closed.
- Ship jcode integration, U5-U6 after merge: U6 blocked on `jcode login` (Claude OAuth expired).
- Move the echo pin to the first Echo release tag that ships gap-utility.toml: blocked, waiting on an Echo release after v0.13.3.
- Pin Echo at the first tag with install.sh --list-installed: blocked on dep echo-list-installed; unchanged since 2026-11-08 (stalled).

## Raw open lines

```
2026-10-27 Ship jcode integration: see docs/plans/2026-11-13-jcode-integration-plan.md (U5-U6 after merge) +gap
2026-11-05 Pin Echo at the first tag with install.sh --list-installed, then confirm gap doctor records adapters +gap dep:echo-list-installed note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md
2026-11-11 Trash ~/AGENTS.md after the jcode overlay ships (plan U5; confirmed: identical copy of ~/.pi/agent/AGENTS.md, nothing unique) +gap dep:jcode-build note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-13-jcode-integration-plan.md U1-U4 committed on feat/jcode-overlay, publishing via No Mistakes; jcode 0.99.1 installed via brew; U6 needs jcode login (Claude OAuth expired)
2026-11-12 Move GAP's echo pin to the first Echo release tag that ships gap-utility.toml and drop its spec install/doctor/names +gap dep:gap-utilities-migrate @waiting Echo release after v0.13.3
2026-11-13 Build jcode integration from the plan with do-work +gap id:jcode-build note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-11-13-jcode-integration-plan.md
2026-11-14 Merge fix/release-version, then tag v0.5.1 so release binaries report their version (v0.5.0 says 'version unknown') +gap
```
