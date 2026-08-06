#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temp_home="$(mktemp -d "${TMPDIR:-/tmp}/atlas-skills-test.XXXXXX")"
trap 'rm -rf "$temp_home"' EXIT

commit="55045850f61e7b23c8dbf8cc30d92bd6e31649d2"
blob="1e792b5bfc34127ca6f00bbb44fea306ce77ea52"
manifest="$temp_home/skills.txt"
installed="$temp_home/home/.agents/skills/lavish/SKILL.md"
source_commit="$(git -C "$repo_root" log -1 --format=%H --all --diff-filter=A ':(top).agents/skills/lavish/SKILL.md')"

[ -n "$source_commit" ]

mkdir -p "$(dirname "$installed")" "$temp_home/bin"
ln -s "$(command -v true)" "$temp_home/bin/skills"
git -C "$repo_root" show "$source_commit:.agents/skills/lavish/SKILL.md" > "$installed"
printf 'kunchenguid/lavish-axi#%s=lavish|skills/lavish/SKILL.md|%s # parser comment\n' "$commit" "$blob" > "$manifest"

default_output="$(env -u SKILLS_AGENTS HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" "$repo_root/install-skills.sh" --dry-run "$manifest")"
if grep -Eq '(^|[[:space:]])-a([[:space:]]|$)' <<<"$default_output"; then
  printf 'default install should let the skills CLI detect agents\n' >&2
  exit 1
fi

output="$(HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" SKILLS_AGENTS='claude-code,pi' "$repo_root/install-skills.sh" --dry-run "$manifest")"
grep -F "kunchenguid/lavish-axi#$commit" <<<"$output" >/dev/null
grep -F "checkout: https://github.com/kunchenguid/lavish-axi.git at $commit" <<<"$output" >/dev/null
grep -F "git-blob $blob" <<<"$output" >/dev/null
grep -F "integrity verified" <<<"$output" >/dev/null
grep -F -e "-a claude-code -a pi" <<<"$output" >/dev/null

printf 'drift\n' > "$installed"
if output="$(HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" SKILLS_AGENTS='claude-code,pi' "$repo_root/install-skills.sh" --dry-run "$manifest" 2>&1)"; then
  printf 'expected integrity mismatch to fail\n' >&2
  exit 1
fi
grep -F "integrity mismatch" <<<"$output" >/dev/null

rm "$temp_home/bin/skills"
printf '%s\n' '#!/usr/bin/env bash' 'printf "%s\n" "$@" > "$SKILLS_ARGS_FILE"' > "$temp_home/bin/skills"
chmod +x "$temp_home/bin/skills"
printf 'owner/repo=demo\n' > "$temp_home/unpinned-skills.txt"
args_file="$temp_home/skills.args"
env -u SKILLS_AGENTS HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" SKILLS_ARGS_FILE="$args_file" \
  "$repo_root/install-skills.sh" "$temp_home/unpinned-skills.txt" >/dev/null
if grep -Fx -e '-a' "$args_file" >/dev/null || grep -Fx -e '*' "$args_file" >/dev/null; then
  printf 'default execution forwarded an explicit all-agent selector\n' >&2
  exit 1
fi
grep -Fx -e 'add' "$args_file" >/dev/null
grep -Fx -e 'owner/repo' "$args_file" >/dev/null
grep -Fx -e '-g' "$args_file" >/dev/null
grep -Fx -e '-s' "$args_file" >/dev/null
grep -Fx -e 'demo' "$args_file" >/dev/null

printf 'PASS: pinned installs are verified and default agent detection stays CLI-owned\n'
