# Regroup: GAP

Date: 2026-11-16
Sources: GAP `todo.txt` and `done.txt` via tuxedo; current checkout and installed jcode files. Compared with `regroup/2026-11-15-regroup-gap.md`.
Resume point: `docs/plans/handoffs/archive/2026-10-15-no-mistakes-autonomous-policy.md` (old archive; no current handoff is present).

## Last completed

- **Ship jcode integration: done (2026-11-16).** Verified with a live `jcode run -p grok-build` session: it quoted the GAP policy and reported one `gap:begin` block. The overlay exists at `~/.jcode/prompt-overlay.md`; `~/AGENTS.md` is absent.

## Up next

- **Merge fix/release-version, then tag v0.5.1: stalled.** PR #125 is merged, but `v0.5.1` is not tagged. The task line is unchanged since the previous check-in.

## Remaining

- **Pin Echo at the first tag with `install.sh --list-installed`: blocked.** Waiting on the `echo-list-installed` dependency; no task resolving that dependency appears in the routed list.
- **Move GAP's Echo pin to the first release tag with `gap-utility.toml`: blocked.** Waiting on an Echo release after `v0.13.3`.

## Raw open lines

```
2026-11-05 Pin Echo at the first tag with install.sh --list-installed, then confirm gap doctor records adapters +gap dep:echo-list-installed note:/Users/ed/Developer/Atlas/GAP/docs/plans/2026-10-22-gap-breadcrumb-ledger-plan.md
2026-11-12 Move GAP's echo pin to the first Echo release tag that ships gap-utility.toml and drop its spec install/doctor/names +gap dep:gap-utilities-migrate @waiting Echo release after v0.13.3
2026-11-14 Merge fix/release-version, then tag v0.5.1 so release binaries report their version (v0.5.0 says 'version unknown') +gap
```

## Open decisions

No open decision was recorded in the previous check-in.
