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
local_source="$temp_home/local source"
local_skill_path="packages/demo/skills/only-claude/SKILL.md"
local_installed="$temp_home/home/.agents/skills/only-claude/SKILL.md"
local_manifest="$temp_home/local-skills.txt"

[ -n "$source_commit" ]

mkdir -p "$(dirname "$installed")" "$(dirname "$local_installed")" "$temp_home/bin" \
	"$local_source/$(dirname "$local_skill_path")"
printf '%s\n' '---' 'name: only-claude' 'description: local fixture' '---' >"$local_source/$local_skill_path"
git -C "$local_source" init -q
git -C "$local_source" config user.name 'Atlas Test'
git -C "$local_source" config user.email 'atlas-test@example.invalid'
git -C "$local_source" add "$local_skill_path"
git -C "$local_source" commit -qm 'Add local skill fixture'
local_source_real="$(cd "$local_source" && pwd -P)"
local_commit="$(git -C "$local_source" rev-parse HEAD)"
local_blob="$(git -C "$local_source" rev-parse "HEAD:$local_skill_path")"
cp "$local_source/$local_skill_path" "$local_installed"
printf 'local:%s#%s=only-claude|%s|%s|agents=claude-code\n' \
	"$local_source" "$local_commit" "$local_skill_path" "$local_blob" >"$local_manifest"
ln -s "$(command -v true)" "$temp_home/bin/skills"
git -C "$repo_root" show "$source_commit:.agents/skills/lavish/SKILL.md" >"$installed"
printf 'kunchenguid/lavish-axi#%s=lavish|skills/lavish/SKILL.md|%s # parser comment\n' "$commit" "$blob" >"$manifest"

default_output="$(env -u SKILLS_AGENTS HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" "$repo_root/install-skills.sh" --dry-run "$manifest")"
grep -F -e '-a claude-code -a codex -a pi -a grok' <<<"$default_output" >/dev/null

output="$(HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" SKILLS_AGENTS='claude-code,pi' "$repo_root/install-skills.sh" --dry-run "$manifest")"
grep -F "kunchenguid/lavish-axi#$commit" <<<"$output" >/dev/null
grep -F "checkout: https://github.com/kunchenguid/lavish-axi.git at $commit" <<<"$output" >/dev/null
grep -F "git-blob $blob" <<<"$output" >/dev/null
grep -F "integrity verified" <<<"$output" >/dev/null
grep -F -e "-a claude-code -a pi" <<<"$output" >/dev/null

local_output="$(HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" \
	SKILLS_AGENTS='claude-code,pi' "$repo_root/install-skills.sh" --dry-run "$local_manifest")"
grep -F "local:$local_source#$local_commit" <<<"$local_output" >/dev/null
grep -F "checkout: $local_source_real at $local_commit" <<<"$local_output" >/dev/null
grep -F '[agents claude-code]' <<<"$local_output" >/dev/null
grep -F -e '-a claude-code -s only-claude' <<<"$local_output" >/dev/null
if grep -F -e '-a pi' <<<"$local_output" >/dev/null; then
	printf 'per-entry agent selector leaked into Pi\n' >&2
	exit 1
fi
pi_only_output="$(HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" \
	SKILLS_AGENTS='pi' "$repo_root/install-skills.sh" --dry-run "$local_manifest")"
grep -F '[skipped: agents=claude-code outside SKILLS_AGENTS=pi]' <<<"$pi_only_output" >/dev/null

printf 'drift\n' >"$installed"
if output="$(HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" SKILLS_AGENTS='claude-code,pi' "$repo_root/install-skills.sh" --dry-run "$manifest" 2>&1)"; then
	printf 'expected integrity mismatch to fail\n' >&2
	exit 1
fi
grep -F "integrity mismatch" <<<"$output" >/dev/null

rm "$temp_home/bin/skills"
cat >"$temp_home/bin/skills" <<'SH'
#!/usr/bin/env bash
printf "%s\n" "$@" > "$SKILLS_ARGS_FILE"
SH
chmod +x "$temp_home/bin/skills"
args_file="$temp_home/skills.args"
HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" SKILLS_AGENTS='claude-code,pi' \
	SKILLS_ARGS_FILE="$args_file" "$repo_root/install-skills.sh" "$local_manifest" >/dev/null
grep -F 'atlas-pinned-skills' "$args_file" >/dev/null
grep -Fx -e 'claude-code' "$args_file" >/dev/null
if grep -Fx -e 'pi' "$args_file" >/dev/null; then
	printf 'local pinned install targeted Pi despite entry ownership\n' >&2
	exit 1
fi

printf 'owner/repo=demo\n' >"$temp_home/unpinned-skills.txt"
env -u SKILLS_AGENTS HOME="$temp_home/home" PATH="$temp_home/bin:$PATH" SKILLS_ARGS_FILE="$args_file" \
	"$repo_root/install-skills.sh" "$temp_home/unpinned-skills.txt" >/dev/null
if grep -Fx -e '*' "$args_file" >/dev/null; then
	printf 'default execution forwarded an explicit all-agent selector\n' >&2
	exit 1
fi
for default_agent in claude-code codex pi grok; do
	grep -Fx -e "$default_agent" "$args_file" >/dev/null
done
grep -Fx -e 'add' "$args_file" >/dev/null
grep -Fx -e 'owner/repo' "$args_file" >/dev/null
grep -Fx -e '-g' "$args_file" >/dev/null
grep -Fx -e '-s' "$args_file" >/dev/null
grep -Fx -e 'demo' "$args_file" >/dev/null

printf 'PASS: pinned installs are verified and the default targets claude-code,codex,pi,grok\n'
