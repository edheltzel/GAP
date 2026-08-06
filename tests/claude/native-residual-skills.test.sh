#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
skills_root="$repo_root/claude/.claude/skills"
apify="$skills_root/Apify"
bpe="$skills_root/BitterPillEngineering"
hardening="$skills_root/Hardening"
loop="$skills_root/Loop"
targets=("$apify" "$bpe" "$hardening" "$loop")

fail_matches() {
  local description="$1"
  local pattern="$2"
  shift 2

  if grep -RInE --include='*.md' --include='*.ts' --include='package.json' "$pattern" "$@"; then
    printf 'FAIL: %s\n' "$description" >&2
    exit 1
  fi
}

fail_matches \
  'removed LifeOS executable or documentation dependencies remain' \
  'LIFEOS/(TOOLS|RULES|DOCUMENTATION|ALGORITHM)|\$\{LIFEOS_DIR\}|filesystem-mcps/apify|LifeOS' \
  "${targets[@]}"

fail_matches \
  'Apify contains an unresolved generic actors import' \
  "from[[:space:]]+['\"]actors['\"]" \
  "$apify"

fail_matches \
  'BitterPillEngineering claims retired settings keys' \
  'loadAtStartup|postCompactRestore|dynamicContext' \
  "$bpe"

fail_matches \
  'Hardening links a workflow that is not present' \
  'Workflows/(MutationTest|CrapAnalysis|DryAnalysis|AcceptanceTestMutation)\.md' \
  "$hardening"

fail_matches \
  'Loop still depends on removed orchestration machinery or a missing Research skill' \
  'mode:[[:space:]]*loop|ISA frontmatter|LIFEOS/|skills/Research' \
  "$loop"

fail_matches \
  'an em dash remains in a touched skill tree' \
  $'\u2014' \
  "${targets[@]}"

if [ ! -f "$apify/bun.lock" ]; then
  printf 'FAIL: Apify bun.lock is missing\n' >&2
  exit 1
fi

if grep -q 'bun.lock' "$apify/.gitignore"; then
  printf 'FAIL: Apify bun.lock is ignored\n' >&2
  exit 1
fi

if ! grep -q '"apify-client": "2\.22\.3"' "$apify/package.json"; then
  printf 'FAIL: Apify dependency must be pinned exactly\n' >&2
  exit 1
fi

(
  cd "$apify"
  bun install --frozen-lockfile --ignore-scripts >/dev/null
  env -u APIFY_TOKEN -u APIFY_API_KEY bun examples/smoke-test.ts
)

printf 'PASS: native residual skills are self-contained and current\n'
