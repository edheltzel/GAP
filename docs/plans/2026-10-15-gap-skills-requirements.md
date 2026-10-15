# gap-skills

## Goal

GAP manages everything agent-related on a machine from one public repo. The first version adds skills alongside the policy: `gap` installs a listed set of skills (yours and other projects') and catches skills whose installed copy has drifted from its source. Later versions install harnesses too.

## In scope

- **(A) Skills manifest.** A list in the GAP repo naming every skill `gap` manages and where each comes from: this repo or another project's GitHub repo.
- **(A) Install.** `gap sync` installs every listed skill that is missing into `~/.agents/skills`, by driving skills.sh (`npx skills add/update`).
- **(A) Claude mirror.** Claude Code does not read `~/.agents/skills`, so every managed skill is also put into `~/.claude/skills`. Folders `gap` does not manage stay untouched.
- **(A) Drift.** When an installed managed skill differs from its source, `gap sync` lists every drifted skill, shows the diff, and asks per skill: keep installed, take source, or skip. With no terminal, it lists the drift, changes nothing for those skills, and exits non-zero, matching how foreign policy files are handled today.
- **(A) Unmanaged skills.** Skills in `~/.agents/skills` that the manifest does not list are ignored: not reported, not changed.
- **(B) Merge DoSkills.** Your skills (the `do-*` set) move from `edheltzel/Do-Skills` into the GAP repo and install from there through skills.sh. DoSkills is archived afterwards.
- **(B) Go public.** `edheltzel/agent-config` becomes public with its full history, after a secrets scan of every commit, and is renamed GAP.

## Out of scope

- **(C) Installing harnesses** (Claude Code, Codex, OMP, Pi, Cursor). The user wants this eventually, both installing and configuring; it gets its own requirements.
- Removing or reporting skills the manifest does not list.
- A skills installer written in `gap` itself; skills.sh does the fetching.

## Success

- On a fresh temp HOME, `gap sync` installs every manifest skill into `~/.agents/skills` and `~/.claude/skills`, and a second run changes nothing.
- After editing an installed managed skill, `gap sync` in a terminal shows that skill's diff and offers keep, take source, or skip. Without a terminal, it lists the skill, leaves it unchanged, and exits 1.
- A folder in `~/.claude/skills` or `~/.agents/skills` that the manifest does not list is byte-identical after `gap sync`.
- `npx skills add edheltzel/<GAP repo>` lists your `do-*` skills, and the GitHub repo is public under the GAP name.
- The secrets scan over all commits reports zero findings before the visibility flip.

## Constraints

- Installing goes through skills.sh (user-directed), so a machine needs Node for the skills step. `AGENTS.md` currently says "No extra runtime dependencies", and that rule must be reworded to allow the skills.sh exception.
- `global/docs/agents/out-of-scope.md` currently excludes harness packaging, so the charter has to expand for (A) and (B).
- Policy publishing keeps its existing guarantees: merged-only, a y/N prompt with a backup for files someone edited, and no prompt without a terminal.

## Decisions

- DoSkills merges into GAP, and GAP goes public (user-directed; rejected: keeping DoSkills separate, or keeping GAP private).
- Public with full history after a secrets scan (user-directed; rejected: a fresh squashed repo, or rewriting history).
- `gap` drives skills.sh (user-directed; rejected: `gap` cloning sources itself).
- Managed skills are mirrored into `~/.claude/skills` (user-directed; rejected: Claude-only skills, or leaving Claude alone).
- Drift shows the diff and you choose per skill (user-directed; rejected: source always wins, or warn and stop).
- Unmanaged skills are ignored (user-directed; rejected: reporting or removing them).
- The first version ships A and B; C comes later (user-directed).

## Open questions

- Should the Claude mirror be copies or symlinks? `~/.claude/skills` holds 79 symlinks today, while policy files must be regular files. Deferred to `do-plan`.
- Does the manifest pin a version per source (the commit hash skills.sh records), or always take latest?
- Where is "the source" of an external skill when checking for drift: skills.sh's recorded hash, or a fresh fetch?

## todo.txt

- Restore the lost DoSkills --yes fix (fix/no-mistakes-yes discarded 2026-10-15 04:31) or re-apply it in GAP after the merge
- Plan gap skills (A+B) from docs/plans/2026-10-15-gap-skills-requirements.md with do-plan
- Write requirements for (C) gap installs harnesses
