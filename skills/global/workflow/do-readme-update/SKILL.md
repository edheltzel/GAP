---
name: do-readme-update
description: Bring a repository's README back in line with the code. Use when a README is stale or wrong, after adding, removing, or renaming commands, scripts, flags, packages, config, env vars, or install steps, or when asked to update, refresh, sync, or fix a README.
---

# README Update

Make every claim in the README match the repo as it is now. Edit only what drifted; keep the author's structure, headings, and voice.

## 1. Find the README and anything that generates it

- The root `README.md`, plus nested package or workspace READMEs when the change touched them.
- Generated regions: marker comments (`<!-- ... start -->` / `<!-- ... end -->`), a header saying the file is generated, or a generator: `scripts/*readme*`, a `package.json` script, a Makefile, justfile, or Taskfile target, or docs tooling. Project `AGENTS.md` or `CONTRIBUTING.md` may name the command.
- Never hand-edit a generated region. Run its generator, then edit only the hand-written parts.

## 2. Collect the truth from the repo

Read sources, not memory or the README itself:

- **Install and setup:** manifests (`package.json`, `pyproject.toml`, `Cargo.toml`, `go.mod`, `Gemfile`, ...), the lockfile's package manager, runtime versions (`.nvmrc`, `.tool-versions`, `engines`, `rust-toolchain`).
- **Commands:** manifest scripts, Makefile/justfile/Taskfile targets, CLI `--help` output.
- **Config:** `.env.example`, config files and schemas, flags.
- **Structure:** the top-level directories the README describes.
- **What changed:** `git log --oneline $(git log -1 --format=%H -- README.md)..HEAD` and that range's diff show everything since the README was last touched.

## 3. Compare claims against the truth

Go section by section. List each claim that is now wrong (command, flag, path, version, step, example output), missing (a user-visible change with no mention), or dead (describes something removed). Check that every relative link and path resolves.

## 4. Edit

- Fix wrong claims, add missing user-visible ones, remove dead ones.
- Copy commands from the manifest; do not paraphrase them.
- Add a section only when a user-visible feature has no home. Leave license, attributions, and badges alone unless they are wrong.

## 5. Verify

- Run each command you added or changed when it is safe: local, non-destructive (`--help`, install, build, test). Never run deploy, publish, or release commands to check a README.
- Re-check links.
- Report what changed and anything you could not verify.

## Related skills

- A new doc page beyond the README: `do-docs`, if installed.
