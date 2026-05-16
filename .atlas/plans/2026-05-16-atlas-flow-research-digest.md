# Atlas Flow Research Digest

Date: 2026-05-16

## Goal

Keep Atlas Pi configuration minimal globally while supporting on-demand project flow profiles for marketing, SSG/SSR, video, UI/UX consulting, and infra/IPsec/pentesting consulting.

## Current conclusion

Atlas should not install a LazyPi-sized global package set. Use a tiny global baseline plus project-local/on-demand flow manifests managed by an Atlas-owned `/atlas-flow` command.

## Recommended implementation order

Use GitHub issue #11 as the controlling roadmap:

1. #9 Define JSON flow manifest schema and fixtures.
2. #10 Add tests/smoke checks for manifest validation and install planning.
3. #2 Build non-mutating `/atlas-flow` skeleton: `list`, `show`, `diff`, `status`, `doctor`.
4. #8 Split current Pi settings into minimal baseline plus flow manifests.
5. #3-#7 Add domain flow manifests.
6. Enable guarded real `install/remove` only after dry-run planning is tested.

## Minimal global baseline

Recommended global core:

- local custom extensions:
  - `recall-compaction.ts`
  - `recall-extract.ts`
  - `superset-hooks.ts`
  - future `atlas-flow.ts`
- `npm:pi-ask-user`
- `npm:pi-subagents`
- `npm:pi-prompt-template-model`

Optional global only if used every session:

- `npm:pi-mcp-adapter`
- `npm:pi-web-access`
- `npm:pi-claude-cli`

Move everything else into flow/profile manifests before removing from global settings.

## Manifest placement and shape

- Flow manifests: `pi/.pi/agent/flows/*.json`
- Installer extension: `pi/.pi/agent/extensions/atlas-flow.ts`
- Domain skills: `pi/.pi/agent/skills/atlas-*/SKILL.md`

Start with dependency-free JSON. YAML can be added later only if justified.

Core manifest fields:

- `id`, `description`, `version`
- `installDefaults`
- `packages[]`
- `skills[]`
- `prompts[]`
- `externalDeps[]`
- `safety{}`
- `verification{}`

Package metadata should include:

- `source` / `pinnedSource`
- `filters`
- `scopeDefault`
- `invasiveness`
- `trustLevel`
- `reason`
- `registersTools`
- `interceptsWriteEdit`
- `autoInstallsBinaries`

## Safety rules

- Default installs to project-local (`pi install -l` / `.pi/settings.json`).
- Require explicit `--global` for global installs.
- Refuse unpinned third-party sources unless explicitly overridden.
- Show risk summary and require confirmation for packages that register tools, intercept writes, auto-install binaries, or access browser/auth state.
- Do not remove global packages until equivalent flow manifests exist.
- Tests must not run real `pi install`, touch real settings, or require network.

## Package findings

Strong flow candidates:

- `pi-agent-browser-native`: UI/UX, marketing QA, SSG/SSR QA, video workflows; requires `agent-browser`, optional `ffmpeg`.
- `pi-markdown-preview`: marketing/blogging/SSG docs; requires `pandoc`, Chromium for terminal previews.
- `pi-web-access`: useful for research/content flows, but probably optional global only if used constantly.

Use carefully / flow-only:

- `pi-lens`: popular but high-invasiveness; intercepts write/edit, autoformats/autofixes, starts LSP/tool pipelines, may auto-install tools.
- `pi-depo`: promising but too broad/invasive as required global dependency; can be optional export/import later.
- Heart of Gold Toolkit: useful content/media skills, but full install adds 32 skills + commands; selectively review/borrow patterns.

## Domain strategy

The ecosystem has few mature, directly domain-specific Pi packages for marketing/video/UIUX/infra-security. Prefer Atlas-authored skills/prompts plus generic tools.

Suggested Atlas-authored skills:

- `atlas-seo-audit`
- `atlas-content-calendar`
- `atlas-ssg-launch-checklist`
- `atlas-video-script-storyboard`
- `atlas-video-edit-review`
- `atlas-uiux-heuristic-audit`
- `atlas-ipsec-consulting-checklist`
- `atlas-pentest-roe-report`

## GitHub tracking

Milestone: `Atlas Flow: minimal Pi profiles`

Issues:

- #2 `/atlas-flow` command
- #3 Marketing flow
- #4 SSG/SSR flow
- #5 Video flow
- #6 UI/UX consulting flow
- #7 Infra/IPsec/pentesting flow
- #8 Baseline split
- #9 Manifest schema
- #10 Tests/smoke checks
- #11 Implementation sequencing

Labels created/applied: `area:pi`, `atlas-flow`, `flow`, `schema`, `testing`, `safety`, `roadmap`.
