# gap-empryo-harness

Request (2026-10-19): make Empryo a first-class gap harness, including skill sync. Review the skills and pick which to pin globally for Empryo. Detect when Empryo is the active harness. Where an Empryo-native feature beats a skill (example: `/review` against `do-adversarial-review`), the skill calls Empryo's feature inside Empryo; otherwise the skill stays as is.

## Outcome

`gap sync` publishes the GAP policy into `~/.empryo/EMPRYO.md` beside the SoulStack block, and warns when an Empryo-only skill copy shadows a managed skill. Skills have one written way to tell that Empryo is the harness. `do-adversarial-review` uses Empryo's reviewer inside Empryo only if a measured A/B says it is better. Empryo pins a small, deliberate skill set instead of 35 skills.

## Findings this plan rests on

Sources: empryo.com/docs `context/prompt-system`, `tools/skills`, `tools/agents`, `agents/review`, `reference/commands`, `reference/presets`, `tools/addons#soulstack`, read 2026-10-19. Live checks were run by the planner tab the same day.

- Empryo loads `~/.empryo/EMPRYO.md` into every tab. SoulStack keeps its own marked block there and leaves other text alone. A changed global file reaches a tab once its prompt cache goes cold.
- The GAP policy does not reach Empryo today. Project `AGENTS.md` loads only because `~/.empryo/config.json` sets `instructionFiles: ["empryo","codex"]`.
- `~/.empryo/EMPRYO.md` has no gap markers yet. `publishPolicy` refuses a foreign file without a terminal, so the first sync that adds the block runs from a terminal and the user answers yes.
- Skill lookup order is `~/.empryo/skills`, then `~/.agents/skills`, then `~/.claude/skills`, with project folders winning. gap's skills.sh install is already visible to Empryo. A copy in `~/.empryo/skills` would shadow a managed skill without warning. `skillDirs` (`cmd/gap/skills.go`) never looks there. Today that folder holds only `empryo`, which Empryo owns.
- Empryo reads `~/.claude/agents` and `~/.agents/agents`. No agent publishing is needed.
- Empryo has no user-defined slash commands; `reference/commands` lists built-ins only. `global/commands/*.md` has no Empryo destination.
- Empryo sets no environment variable for its shell children. Scripts can detect it through process ancestry (an `empryo` binary under `~/.empryo/bin` among the parents). A skill can detect it from Empryo-only tools in its toolset (`final_response`, `genome_query`, `code_script`, the `settings` tool) or the `empryo` skill being installed and loaded. `~/.empryo` existing means installed, not active.
- `/review` runs a fresh-context reviewer that returns `VERDICT: PASS|FAIL|PARTIAL` with `file:line`. It reviews this tab's edits, else this session's, else uncommitted git changes, at most 30 files, and runs no tests. Its model is the `review` router slot, falling back to `goalReview` (here `gpt-6.1-sol`). Agents reach it as `empryo --headless --review "<focus>" --json`: exit 0 is PASS or nothing reviewed (`verdict: null`), 3 FAIL, 4 PARTIAL.
- `skills.autoLoad` in `~/.empryo/config.json` pins 35 skills, each injected in full into every prompt. The `empryo` skill forbids hand-editing that file; writes go through the in-app `settings` tool. A preset is JSON `{name, version, config:{...}}` loaded from the config `presets` array, `--plugin`, or `EMPRYO_PRESETS`. Presets apply under global and project config, so the existing `skills.autoLoad` beats a preset. Whether arrays merge or replace is undocumented.

## Implementation Units

### U1. Publish the policy to Empryo
- Depends on: none
- Files: `cmd/gap/main.go`, `cmd/gap/main_test.go`, `AGENTS.md`, `cmd/gap/AGENTS.md`, `README.md`, `global/AGENTS.md`
- Change: add `filepath.Join(home, ".empryo", "EMPRYO.md")` (`cursor` false) to the `copies` list in `syncAll`. Splice behavior is the existing marker contract; no new code path. Add Empryo to the destination list in root `AGENTS.md` Publish, `cmd/gap/AGENTS.md`, the README destination table and text that names harnesses, and `global/AGENTS.md` "How this file is loaded" (one line: Empryo reads `~/.empryo/EMPRYO.md`; do not add a second copy elsewhere). The README image stays as is.
- Done when: a sync writes the gap block into `~/.empryo/EMPRYO.md`, the SoulStack block and any other text stay byte for byte, and a second sync changes nothing.
- Verify: new tests in `cmd/gap/main_test.go` with a fixture `EMPRYO.md` holding a `<!-- soulstack:start -->` block plus user text: (a) an answered-yes splice keeps everything outside the gap markers byte for byte; (b) a re-sync is a no-op and the ledger digest is stable; (c) with no terminal the foreign file is refused and the other destinations still publish. `go test ./...` passes.

### U2. Warn on Empryo-only skill shadows
- Depends on: none
- Files: `cmd/gap/skills.go`, `cmd/gap/skills_test.go`
- Change: during the drift check, for each managed skill name, if `~/.empryo/skills/<name>` exists and its content hash differs from the installed `~/.agents/skills/<name>`, print one warning naming both paths and saying Empryo loads the `~/.empryo` copy first. Warn only; never write or delete there. Skip the name `empryo`.
- Done when: a differing shadow copy produces exactly one warning, a matching copy or no copy produces none, and the exit code is unchanged.
- Verify: table test in `cmd/gap/skills_test.go` with a fake home: no shadow, identical shadow, differing shadow, and a shadow named `empryo` (ignored). `go test ./...`.

### U3. Harness-detection contract
- Depends on: none
- Files: `global/docs/agents/harness-detection.md` (new), `global/AGENTS.md` (one link line under Work)
- Change: document how a skill decides it runs inside Empryo: (1) Empryo-only tools are in the agent's toolset (`final_response`, `genome_query`, `code_script`), or (2) for a script, an ancestor process is `~/.empryo/bin/empryo`. Say that `~/.empryo` existing and `~/.empryo/bin` on `PATH` only mean installed. Give the fallback rule: when unsure, use the portable path.
- Done when: the page exists, `global/AGENTS.md` links it in one line, and U4 and U5 cite it instead of restating it.
- Verify: `rg -n "harness-detection" global skills` shows the policy link and each skill that uses it; `go test ./...` (link rewriting) passes.

### U4. Review A/B, then the swap if Empryo wins
- Depends on: U3
- Files: `skills/global/core/do-adversarial-review/SKILL.md`, `skills/global/core/do-adversarial-review/references/empryo-review.md` (new, only if Empryo wins), `docs/solutions/2026-10-19-empryo-review-ab.md` (new)
- Change: spike first. Seed 3 throwaway diffs in a scratch repo (not GAP), each with one known bug from the skill's table (null deref, off-by-one, swallowed error) plus one decoy that looks wrong but is guarded. For each diff run (a) the skill in-session and (b) `empryo --headless --review "<the skill's hunt brief: bug classes and the trigger gate>" --json`. Record true bugs found, false positives, wall time, and cost. Empryo wins only if it finds at least as many true bugs with no more false positives. If it wins: inside Empryo (per U3) the skill runs that command as its engine, maps FAIL/PARTIAL findings to its P0-P3 report, and keeps its contract (read-only, `mode:return-to-caller`, status words). A `verdict: null` means not reviewed, never PASS, and falls back to the in-skill path; this covers a GitButler lane whose changes are already committed. More than 30 changed files also falls back. In a GitButler workspace the reviewer sees every applied lane's uncommitted changes: when more than one lane holds uncommitted changes, the skill lists the reviewed lane's files from `but status`, applies the 30-file limit to that list, names those files in the brief, and keeps only findings on them; if the lane's files cannot be listed, it falls back. Outside Empryo nothing changes. If it loses, the skill is unchanged. Either way the solutions note records the numbers and the decision.
- Done when: the solutions note holds the A/B table and the decision, and the skill matches that decision.
- Verify: the A/B table is in the note with the commands used. If swapped: one run of `do-adversarial-review` inside a real Empryo tab on a seeded diff reports the seeded bug, and one run with no uncommitted changes falls back instead of passing. A nested `empryo --headless` inside a running Empryo tab completes. `go test ./...` still passes (skills tree gate).

### U5. One-line Empryo mappings in two skills
- Depends on: U3
- Files: `skills/global/operations/do-delegation/SKILL.md`, `skills/global/core/do-redteam/Workflows/AdversarialValidation.md`
- Change: one line each, citing U3: in Empryo, `do-delegation`'s subagent tool is `dispatch` / `background_dispatch`, with `find_agent` for custom agents; `AdversarialValidation` may use Empryo's `council` to develop competing alternatives. No other rewrites.
- Done when: each file gains exactly that line and still reads correctly for other harnesses.
- Verify: `git diff --stat` shows one or two added lines per file; `bash scripts/update-readme.sh` leaves the catalog unchanged.

### U6. Empryo pin set
- Depends on: U1
- Files: `cmd/gap/main.go`, `cmd/gap/main_test.go`, `cmd/gap/AGENTS.md`, `README.md`
- Change: the presets spike found that the `settings` tool cannot write `presets`, headless ignores `--plugin` and `--verbose-presets`, and a global `skills.autoLoad` beats any preset. So there is no preset. `gap sync` reads `~/.empryo/config.json` and, when `skills.autoLoad` differs from `empryoPins` (`do-todo`, `do-commit`, `but-axi`) as a set, prints the exact `settings` call that sets it. Order does not matter. gap never writes `~/.empryo/config.json`. The user asks an agent inside Empryo to run the printed call once.
  Pin reasoning: `do-todo` is required by the policy every turn; `do-commit` and `but-axi` cover the mandated GitButler flow. `do-debug`, `do-plan`, `do-work` stay unpinned because the Next blocks load them on demand. Everything else loads on demand through the skills tool. Specifically unpinned: `do-humanize`, `do-show-me`, `do-teach-me` (marked `disable-model-invocation`, which pinning defeats); `do-art`, `do-illo`, `do-gh-pm`, `do-design-patterns-gof` (long); `do-delegation` (Empryo has native dispatch).
- Done when: `gap sync` prints the call while the pins differ, prints nothing once they match in any order, and never changes Empryo's config.
- Verify: table test `TestEmpryoPinAdvice` in `cmd/gap/main_test.go` covers missing config, invalid JSON, the matching set, the same set in another order, and a different set, and checks the config file is unchanged. `go test ./...`.

### U7. Prove it in the real app
- Depends on: U1, U2, U4, U6
- Files: `todo.txt`, `TODO.md`
- Change: after the units merge, the user runs `go install ./cmd/gap && gap sync` from a terminal and answers yes for `~/.empryo/EMPRYO.md`. Open a new Empryo tab and confirm: the GAP policy is in its global instructions exactly once, the SoulStack block is intact, and after the printed `settings` call the pin set matches U6. Run `/soulstack update` (TUI) and confirm the gap block survives; if that command does not exist, the SoulStack addon's updater in `/addons` is the round trip. Record each check in `todo.txt` and sync the `TODO.md` mirror.
- Done when: each check above is recorded with its result.
- Verify: `rg -c 'gap:begin' ~/.codex/AGENTS.md` prints 0 (the `codex` instruction source adds no second copy); `rg -c "gap:begin" ~/.empryo/EMPRYO.md` is 1 before and after the SoulStack update; `gap sync` a second time prints no change for Empryo.

## Out of scope

- gap installing Empryo itself (rejected 2026-10-16).
- Editing Empryo's own `empryo` skill or anything under `~/.empryo/skills/empryo`.
- Empryo memory against Recall.
- An Empryo destination for `global/commands/*.md` (Empryo has no user commands).
- `do-git-worktree`, which contradicts the policy's no-native-worktree rule. It goes to `todo.txt` for the user to decide.
- Updating `docs/images/publish-overview.webp`.

## todo.txt

- U1-U3: Empryo policy destination, shadow warning, harness-detection doc
- U4: A/B do-adversarial-review vs empryo --headless --review, swap only if Empryo wins
- U5: one-line Empryo mappings in do-delegation and do-redteam
- U6: Empryo pin set (do-todo, do-commit, but-axi); no preset, gap sync prints the settings call
- U7: gap sync from a terminal, prove policy, SoulStack round trip, and pin set in a new Empryo tab
- Decide do-git-worktree: remove, or rewrite for GitButler lanes (contradicts the no-native-worktree rule)

Review complete
