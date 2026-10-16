# gap-skills

Requirements: `docs/plans/2026-10-15-gap-skills-requirements.md`

## Outcome

`gap sync` publishes the policy and also installs every skill a manifest lists, from this repo or other projects, through skills.sh into `~/.agents/skills` with the Claude mirror. It shows the diff of each drifted managed skill and lets you choose per skill. Your `do-*` skills live in this repo, DoSkills is archived, and the repo is public under the name GAP.

Facts this plan builds on (read 2026-10-15):

- skills.sh symlinks agent directories to `~/.agents/skills` by default. For example, `~/.claude/skills/do-commit -> ../../.agents/skills/do-commit`. The Claude mirror is therefore those symlinks, made by skills.sh `add -g -a <agents>`. `gap` writes nothing into `~/.claude/skills` itself, which settles the open copy-or-symlink question.
- `~/.agents/.skill-lock.json` (v3) records `source`, `skillPath` and `skillFolderHash` for each installed skill. It lists 75 skills: 55 from `edheltzel/Do-Skills` and 20 from 13 other repos.
- DoSkills keeps 131 `SKILL.md` files under `skills/<global|project>/<group>/<skill>/`, plus `skills.sh.json` (catalog groupings) and `scripts/update-readme.sh`.
- No secrets scanner is installed (`gitleaks` and `trufflehog` are both missing).

## Implementation Units

### U1. Restore the lost `--yes` fix in DoSkills

- Depends on: none
- Files: (DoSkills) `skills/global/workflow/do-commit/references/procedure.md`, `skills/global/workflow/do-git-safe-pr-workflow/SKILL.md`
- Change: re-apply `axi run --yes --intent` to all 5 call sites. Commit `xnr` was dropped by an unattributed `[DISCARD]` at 04:31, so restore it from the oplog or re-edit. This must happen before U3 copies these files.
- Done when: the DoSkills source matches the installed copies on these two files.
- Verify: `diff` of each source file against its file under `~/.agents/skills/` prints nothing, and `grep -c 'axi run --intent'` returns 0 for both.

### U2. Secrets scan of the full history

- Depends on: none
- Files: none (report only, saved to `docs/plans/2026-10-15-secrets-scan.md`)
- Change: install `gitleaks` as a dev tool, not a `gap` runtime dependency. Run it over every commit of this repo and of DoSkills, since DoSkills' history comes along in U3.
- Done when: both scans report zero findings, or every finding has been removed or rotated with your sign-off.
- Verify: `gitleaks git --log-opts="--all" .` exits 0 in each repo, and the saved report shows the commit counts scanned (458 and DoSkills' count).

### U3. Bring DoSkills into this repo

- Depends on: U1, U2
- Files: `skills/**`, `skills.sh.json`, `scripts/update-readme.sh`, `README.md` (catalog section)
- Change: copy `skills/`, `skills.sh.json` and `scripts/update-readme.sh` from DoSkills into the repo root, plus `docs/global` and `docs/project`, because the generated group READMEs link to those pages. Add `<!-- skills-start -->`/`<!-- skills-end -->` markers to README and rerun the script. (User-directed during the build: copy the files without history, since `git subtree add` would need a raw merge commit inside GitButler. DoSkills' history stays in the archived repo. Also user-directed: keep DoSkills' script as is. `REPO_URL` stays `edheltzel/Do-Skills` until U8 renames the repo, because `edheltzel/GAP` is still the old scaffold.)
- Done when: all 131 `SKILL.md` files exist at the same relative paths, and skills.sh lists the same skills from GAP as from DoSkills.
- Verify: `find skills -name SKILL.md | wc -l` prints 131. `npx -y skills add <this checkout> --list` lists every `do-*` skill. `bash scripts/update-readme.sh` leaves no diff after a second run.

### U4. Skills manifest

- Depends on: U3
- Files: `global/skills.json`
- Change: write a JSON list of `{ "source": "<owner/repo or this repo>", "skills": ["<name>", ...] }` entries, seeded from today's `~/.agents/.skill-lock.json` so that it describes what is installed now. The 55 Do-Skills entries point at this repo. JSON keeps `gap` on the standard library.
- Done when: every skill in today's lock appears in the manifest exactly once.
- Verify: a throwaway check that the set of names in the manifest equals the set of keys in the lock.

### U5. `gap sync` installs missing managed skills

- Depends on: U4
- Files: `cmd/gap/main.go` or a new `cmd/gap/skills.go`, plus `cmd/gap/main_test.go`
- Change: after the policy publish, read `global/skills.json`. For each source with listed skills missing from `~/.agents/skills`, run `npx -y skills add <source> -g -y -s <names> -a <agents>`. `<agents>` are the skills.sh agent ids for the harnesses that have a skills directory on this machine, and must include Claude Code. Confirm the exact ids from skills.sh first; that is the first step of this unit. If `npx` is missing, print one line naming the skills step, skip it, still publish the policy, and exit 1. Never touch skills the manifest does not list.
- Done when: a fresh temp HOME gets every manifest skill in `~/.agents/skills`, and each one also resolves in `~/.claude/skills` (a symlink created by skills.sh). A second run makes no skills.sh call.
- Verify: an e2e test with a stub `npx` on `PATH` records its argv. The first run calls it once per source that has missing skills, and the second run doesn't call it at all. Without `npx` on `PATH` the policy files are still written and the exit code is 1. An unlisted folder in `~/.agents/skills` is byte-identical before and after.

### U6. Drift: show the diff, choose per skill

- Depends on: U5
- Files: `cmd/gap/skills.go`, `cmd/gap/main_test.go`
- Change: for each installed managed skill, compute the git tree hash of the installed folder with `git hash-object` and `mktree` over its files. Compare it with the source:
  - Skills from this repo compare against `git rev-parse HEAD:<skillPath dir>`.
  - External skills compare against the lock's `skillFolderHash`.
  - On a mismatch, print the skill name and a unified diff (`git diff --no-index <source> <installed>`). For a skill from this repo, the source is `<repo>/<skillPath dir>`. For an external skill, the source is a `git clone --depth 1 <sourceUrl from the lock>` into a temp directory.
  - In a terminal, ask `[k]eep installed / [t]ake source / [s]kip`. Take source reinstalls through skills.sh after copying the installed folder to `~/.gap/backups/`.
  - Without a terminal, list every drifted skill, change none of them, and exit 1.
- Done when: an edited installed skill is always caught, and never silently overwritten.
- Verify: tests cover each path. An unchanged skill shows no drift. An edited skill without a terminal is listed, left byte-identical, and the exit code is 1. In a pty, `t` creates a backup and makes the stub `npx` call, `k` makes no call, and `s` makes no call.
- Spike first: confirm that the lock's `skillFolderHash` equals the tree hash of a freshly installed folder. If it doesn't, external drift compares against a fresh fetch instead, and this unit records that.

### U7. Docs and contract

- Depends on: U5, U6
- Files: `AGENTS.md`, `README.md`, `CONCEPTS.md`, `global/docs/agents/out-of-scope.md`, `cmd/gap/main.go` (help text)
- Change:
  - `AGENTS.md`: replace "No extra runtime dependencies" with "Go standard library; the skills step needs Node (`npx`) for skills.sh".
  - `out-of-scope.md`: GAP now owns skills distribution; harness installs stay out until (C).
  - README: add a "Manage skills" section covering the manifest, drift choices and the no-terminal behavior.
  - `CONCEPTS.md`: add an entry for "Managed skill".
  - `gap --help`: say that sync also installs skills.
- Done when: every claim in these docs matches U5 and U6.
- Verify: `do-readme-update` check, `gap --help` output reviewed, `go test ./...` passes.

### U8. Archive DoSkills, go public, rename to GAP

- Depends on: U2, U3, U7, and your explicit go at that moment (irreversible)
- Files: `scripts/update-readme.sh` (`REPO_URL`), plus the README files that script regenerates
- Change:
  1. Rename the old `edheltzel/GAP` scaffold repo to free the name (todo task 8).
  2. Make `edheltzel/agent-config` public.
  3. Rename it to `GAP`.
  4. Run `git remote set-url origin`.
  5. Set `REPO_URL` in `scripts/update-readme.sh` to `https://github.com/edheltzel/GAP/tree/master` and rerun it.
  6. Replace DoSkills' README with a pointer, then archive it.
  7. Reinstall the 54 Do-Skills skills from GAP so the lock shows `edheltzel/GAP`.
- Done when: `gh repo view edheltzel/GAP --json visibility` shows `PUBLIC`, DoSkills is archived, and the lock has no `edheltzel/Do-Skills` sources.
- Verify: those two `gh` calls, `npx skills add edheltzel/GAP --list` from outside the repo, and a lock count by source.

## Out of scope

- (C) Installing harnesses (Claude Code, Codex, OMP, Pi, Cursor) gets its own requirements.
- Reporting or removing skills the manifest does not list.
- A Go installer replacing skills.sh.
- Pinning external skills to versions: the manifest names sources only, and skills.sh's lock keeps the hashes. Revisit if updates bite.

## Blockers

- U8 needs your explicit go when it runs: it makes the repo public and renames repos.
- U8's "public with full history" decision is reopened (user, 2026-10-15). The U2 scan found pi session transcripts, the old harness-config era, and 2 real-looking values (`docs/plans/2026-10-15-secrets-scan.md`). Choose a fresh public repo, filter-repo, or full history before U8.

## todo.txt

- U1 restore DoSkills --yes fix before the merge
- U2 gitleaks scan of agent-config and DoSkills history
- U3-U7 build gap skills from docs/plans/2026-10-15-gap-skills-plan.md
- U8 publish: rename old GAP repo, flip agent-config public, rename to GAP, archive DoSkills (needs explicit go)

Review complete
