# jcode-integration

Scope: U1-U4 ship in this change; U5-U6 are machine steps after merge.

## Outcome

`gap sync` writes the GAP policy between the GAP markers in jcode's prompt overlay (`$JCODE_HOME/prompt-overlay.md`, or `~/.jcode/prompt-overlay.md` when unset). A set but empty or relative `JCODE_HOME` is refused. The sync warns once per managed skill whose `~/.jcode/skills/<name>` copy differs. `~/AGENTS.md` is gone, and no GAP doc calls it a pointer.

## Findings that shape the plan

- `~/AGENTS.md` is not a pointer. It is a byte-identical copy of `~/.pi/agent/AGENTS.md` (Pi packaging, theme, footer, plus a SoulStack block), dated 2026-10-04.
- Empryo loads `~/AGENTS.md` as a global instruction file, so its text appears in every Empryo prompt today. The same SoulStack block is already in `~/.empryo/EMPRYO.md`, so Empryo gets it twice now.
- Pi and OMP walk parent directories for `AGENTS.md`. For a project under `$HOME`, `~/AGENTS.md` is a second copy of Pi's own file.
- Removing `~/AGENTS.md` therefore drops nothing unique: Pi keeps `~/.pi/agent/AGENTS.md`, Empryo keeps `EMPRYO.md`, and jcode gets the policy from the overlay.
- jcode is not installed on this machine right now: there is no `jcode` on PATH or in `/Applications`, and the newest `~/.jcode` log is from 2026-11-06. `~/.jcode/skills` holds 53 entries and `~/.jcode/config.toml` exists.

## Implementation Units

### U1. jcode policy destination
- Depends on: none
- Files: `cmd/gap/harness.go`, `cmd/gap/harness_test.go`, `cmd/gap/main.go`, `cmd/gap/main_test.go`
- Change: add `{name: "jcode", policy: "prompt-overlay.md", homeEnv: "JCODE_HOME", homeDir: ".jcode"}` to `harnesses`. `policyDests` uses `~/.jcode` when `JCODE_HOME` is unset and accepts an absolute override. Refuse a set but empty `JCODE_HOME` with an "unset it" message, because jcode reads it as its working folder. Empty `CODEX_HOME` still means unset. Both overrides refuse relative values through the shared home resolver. The overlay uses the plain markdown splice, so the marker, ledger, foreign-file, and backup rules apply unchanged.
- Done when: sync writes the block to `~/.jcode/prompt-overlay.md` when `JCODE_HOME` is unset, or to `$JCODE_HOME/prompt-overlay.md` when it is nonempty and absolute. Empty or relative values fail before publishing. Text outside the markers stays byte for byte, and a second sync writes nothing.
- Verify: `go test ./cmd/gap -run 'TestJcodeHomeEmptyRefused|TestJcodeOverlay|TestResolveCodexHome|TestSplice'`. `TestJcodeHomeEmptyRefused` covers unset versus set but empty `JCODE_HOME`, the "unset it" message, destination-resolution failure, and unchanged empty `CODEX_HOME` behavior. `TestJcodeOverlay` covers the default path, an absolute `JCODE_HOME`, a relative `JCODE_HOME` (refused), foreign text preserved, and the second-sync no-op.

### U2. jcode shadow warning
- Depends on: none
- Files: `cmd/gap/skills.go`, `cmd/gap/skills_test.go`
- Change: generalize `empryoShadow(home, name)` into `shadowWarnings(home, name) []string`. Empryo uses `~/.empryo/skills` and keeps its `empryo` skill exception; jcode uses `$JCODE_HOME/skills`, or `~/.jcode/skills` when unset. `checkDrift`'s deferred loop prints each warning. The wording for jcode states only that both copies exist and differ, because jcode's copy precedence is unverified.
- Done when: a differing `~/.jcode/skills/<name>` prints exactly one line naming both paths, a matching or missing copy prints nothing, and the exit code is unchanged.
- Verify: `go test ./cmd/gap -run 'TestShadow'`

### U3. Doctor lists jcode
- Depends on: U1
- Files: `cmd/gap/provision.go`
- Change: add `jcode` to the harness presence list at `provision.go:161`, so `gap doctor` shows `jcode: present` or `absent`.
- Done when: `gap doctor` prints a `jcode:` row.
- Verify: `go run ./cmd/gap doctor | grep '^jcode:'`

### U4. Docs
- Depends on: U1, U2
- Files: `AGENTS.md`, `cmd/gap/AGENTS.md`, `README.md`, `global/AGENTS.md`, `global/docs/agents/out-of-scope.md`
- Change:
  - Root `AGENTS.md` Publish list: add `~/.jcode/prompt-overlay.md` (or absolute `JCODE_HOME`). State that set but empty `JCODE_HOME` is refused with an "unset it" message, while empty `CODEX_HOME` means unset; both refuse relative values.
  - `cmd/gap/AGENTS.md`: add the destination line, and add the jcode shadow to the drift-check step next to Empryo's.
  - `README.md`: add a jcode row to the destination table.
  - `global/AGENTS.md`, "How this file is loaded": add "jcode reads `~/.jcode/prompt-overlay.md`." Replace "`~/AGENTS.md` is only a pointer here. Do not paste a second full copy there." with "Do not create `~/AGENTS.md`; harnesses that walk parent folders would load it into every project under home."
  - `out-of-scope.md:9`: drop "(home stays a thin pointer)" to match.
  - Plain hyphens only.
- Done when: no GAP doc outside `docs/plans/` and `regroup/` calls `~/AGENTS.md` a pointer, and jcode is named in all four destination lists.
- Verify: `rg -n 'pointer' AGENTS.md README.md cmd/gap/AGENTS.md global/` finds no `~/AGENTS.md` line, and `rg -c jcode AGENTS.md README.md cmd/gap/AGENTS.md global/AGENTS.md` is nonzero for each file.

### U5. Remove `~/AGENTS.md` (machine step after merge)
- Depends on: U4 merged and `gap sync` run, so the overlay exists before the old file goes
- Files: `~/AGENTS.md` (outside this repo)
- Change: run `cmp ~/AGENTS.md ~/.pi/agent/AGENTS.md`. If they are still identical, `trash ~/AGENTS.md`. If they differ, stop and show the diff.
- Done when: `~/AGENTS.md` does not exist, and a new Empryo tab no longer lists `global:AGENTS.md`.
- Verify: `test ! -e ~/AGENTS.md`, and the next Empryo session's instruction list shows only `~/.empryo/EMPRYO.md`.

### U6. Proof in jcode
- Depends on: U5
- Files: none
- Change: in a new jcode session with no tools allowed, ask it to quote the first policy sentence.
- Done when: jcode quotes "GAP is the global agent policy for coding agents.", and its prompt contains the policy once.
- Verify: the jcode session transcript under `~/.jcode/sessions/` contains the quote, and `grep -c 'gap:begin'` on the captured prompt (`jcode` debug prompt dump, or the session file) is 1.

## Out of scope

- Installing or deleting anything in `~/.jcode/skills`. Cleaning the 53 entries there is left to the user, guided by the warning.
- Project-level `./.jcode/prompt-overlay.md`, `preferred-tools.md`, `system-prompt.md`, and `swarm-prompt.md`.
- Writing `~/.jcode/config.toml`, which belongs to Echo.
- Publishing `global/commands/*.md` to jcode.
- Changing `~/.pi/agent/AGENTS.md`.

## Blockers

- jcode is not installed here, so U6 cannot run. Reinstall jcode, or accept U1's file-level tests as the proof and drop the live-session Success line.

## todo.txt
- Build jcode integration from the plan with do-work +gap
- Trash ~/AGENTS.md after the jcode overlay ships (U5) +gap
