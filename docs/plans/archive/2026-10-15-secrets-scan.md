# Secrets scan (U2), 2026-10-15

Tool: gitleaks 8.30.1, run as `gitleaks git --log-opts="--all"`. Secrets were classified by machine and never printed.

| Repo | Commits scanned | Findings |
| --- | --- | --- |
| `edheltzel/Do-Skills` | 167 | 0 |
| `edheltzel/agent-config` | 485 | 737 |

## agent-config findings

| Group | Count | Verdict |
| --- | --- | --- |
| `sourcegraph-access-token` in plugin catalogs and pi session transcripts | 697 | False positive: all 697 are bare 40-character hex git SHAs |
| `curl-auth-header`, `jwt` and `generic-api-key` in PAI, Fabric and WebAssessment docs, and in the vendored libghostty workflows and `Binding.zig` | 37 | Likely placeholder or example text in docs, plus vendored upstream code. Not verified one by one |
| `APP_CLIENT_SECRET` in `claude/.claude/Atlas/skills/LifeOS/install/LIFEOS/TOOLS/healthsync/eightsleep.ts` (commit `98ee113c`) | 1 | Real-looking: 64 characters, not a placeholder. Needs your call |
| `authToken` in `…/ide/17330.lock` (commit `ca1449ea`) | 1 | Real-looking: a 36-character UUID. It's a Claude IDE lock-file token, probably session-scoped. Needs your call |

## Not secrets, but public with the history

- `pi/.pi/agent/sessions/*.jsonl`: full agent session transcripts from May 2026.
- The February-August harness config era, including the PAI profile and statusline scripts.

## Sign-off needed before U8

U2 passes only when every real-looking finding is removed or rotated, with your sign-off.
