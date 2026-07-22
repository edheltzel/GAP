#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temp_home="$(mktemp -d "${TMPDIR:-/tmp}/atlas-skills-test.XXXXXX")"
trap 'rm -rf "$temp_home"' EXIT

commit="55045850f61e7b23c8dbf8cc30d92bd6e31649d2"
blob="1e792b5bfc34127ca6f00bbb44fea306ce77ea52"
manifest="$temp_home/skills.txt"
installed="$temp_home/home/.agents/skills/lavish/SKILL.md"

mkdir -p "$(dirname "$installed")" "$temp_home/bin"
ln -s "$(command -v true)" "$temp_home/bin/skills"
cp "$repo_root/.agents/skills/lavish/SKILL.md" "$installed"
printf 'kunchenguid/lavish-axi#%s=lavish|skills/lavish/SKILL.md|%s # parser comment\n' "$commit" "$blob" > "$manifest"

output="$(HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" SKILLS_AGENTS='claude-code,pi' "$repo_root/install-skills.sh" --dry-run "$manifest")"
grep -F "kunchenguid/lavish-axi#$commit" <<<"$output" >/dev/null
grep -F "git-blob $blob" <<<"$output" >/dev/null
grep -F "integrity verified" <<<"$output" >/dev/null
grep -F -- "-a claude-code -a pi" <<<"$output" >/dev/null

printf 'drift\n' > "$installed"
if output="$(HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" SKILLS_AGENTS='claude-code,pi' "$repo_root/install-skills.sh" --dry-run "$manifest" 2>&1)"; then
  printf 'expected integrity mismatch to fail\n' >&2
  exit 1
fi
grep -F "integrity mismatch" <<<"$output" >/dev/null

printf 'PASS: pinned ref survives parsing and installed Git blob drift is detected\n'
