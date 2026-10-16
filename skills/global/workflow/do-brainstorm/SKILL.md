---
name: do-brainstorm
description: "Requirements dialogue for what to build, not how. Use when the user wants to brainstorm or scope an idea, or has no idea yet and needs a short options pass first. NOT FOR attacking an idea (do-redteam), implementation planning (do-plan), executing (do-work), PRD or spec prose (do-tech-writing), or UX flow trees (do-ux-flow-plan)."
---

# Brainstorm

Answer WHAT to build. `do-plan` answers HOW. This skill does not implement and does not write the plan.

**Done:** the requirements file is written, the Ready check passed or the user stopped with Open questions listing what is unsettled, and the Next block was shown or a caller return was sent.

## Start

The invocation's idea is the subject. If it carries `mode:return-to-caller`, strip that token and remember caller mode. If no idea was given, ask what to explore and wait.

No idea yet, meaning they asked for options or what to build: ask one question for the area or constraint. Then a short ideas pass. Read the repo's top-level docs and the area that question names, and stop once 3 to 5 options are grounded in what is already there. Each option is one sentence. The user picks one. That pick is the idea. "Surprise me" is a real pick. Take the option most grounded in the repo and say so. Do not write an ideas file. Do not plan. A second question means they need to name an idea. Ask for that instead of interviewing.

If that read finds nothing, say so and ground the options in what the user said.

A flow tree is `do-ux-flow-plan`, if installed. A throwaway visual is `do-show-me`, if installed. Attacking the idea is `do-redteam`, if installed. Name the neighbor and stay on requirements unless the user switches.

## Dialogue

One question per turn. Use the harness's blocking question tool when it has one. Ask only a decision the request and the repo cannot settle. Do not re-ask a decision already made.

Stay on behavior, who it is for, what is in and out, and how someone will know it worked. File layout, libraries, and sequencing are HOW. Record them as constraints only when the user stated them. Park the rest for `do-plan`.

Stop when the Ready check passes. If the user says stop earlier, write what is settled and list the rest under Open questions.

## Write

Location: if the project `AGENTS.md` overrides where requirements or plans go, use that directory. Otherwise `docs/plans/`. Create it if missing.

File: `YYYY-MM-DD-<topic>-requirements.md` in that directory. Date is the local calendar date. Topic is a short lowercase hyphenated slug. If the path exists, append `-2`, then `-3`.

```
# <topic>

## Goal
## In scope
## Out of scope
## Success
## Constraints
## Decisions
## Open questions
## todo.txt
- <one follow-up per line, or leave the list empty>
```

Omit a section only when it would be empty, except `## todo.txt`, which always ends the file. Success is something a later run can observe fail. Do not add units, implementer tasks, or a technology pick the user did not make.

Add each `## todo.txt` line to the repo-root `todo.txt` with `do-todo`, if installed, tagged `+<project>` (lowercased repo-root basename) and `note:<absolute artifact path>`.

**Ready check:** a later reader can name the behavior, the boundary, and the success test without guessing. If not, ask another question. Do not show Next while the check fails, unless the user asked to stop.

Caller mode: skip Next. Return `status: done` when the check passed, `needs-human` when Open questions still block planning, `blocked` when the file could not be written. Include the absolute artifact path when one exists.

## Next

End with this block. Run the next skill only when the user picks it. When a caller started this run (`mode:return-to-caller`, used by `do-it` and `do-work`), skip the block and return `status: done | blocked | needs-human` plus the artifact path.

Next: `do-plan` - turn these requirements into an implementation plan, passing `<absolute artifact path>`.

Other moves:
1. `do-redteam` - pressure-test the idea, if installed
2. Review in Plannotator - annotate or approve `<artifact>` first. (Only when the artifact is a file and Plannotator is installed.)
3. Stop - the artifact is already saved.

Reply with a number, the label, or what you want instead.

Ask with the harness's blocking question tool when it has one, and drop the last line. Otherwise print the block and wait. A pick loads that skill now, in this session.

Review in Plannotator: if a `plannotator` tool exists, call it with `{action: "annotate", target: <path>, gate: true}`, end the turn, and wait for the feedback message. Otherwise run `plannotator annotate <path> --gate --json` with a long or no timeout (it blocks until the user acts). `approved`: offer the Next rung. `annotated`: revise from the feedback, then offer review again. `dismissed`: show the block again.
