#!/usr/bin/env bash
# Validate manifest-owned and repository-owned skill installations without mutating them.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
git_common_dir="$(git -C "$repo_root" rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
if [[ "$git_common_dir" == */.git ]]; then
	canonical_root="${git_common_dir%/.git}"
else
	canonical_root="$repo_root"
fi
manifest="${SKILLS_MANIFEST:-$repo_root/skills.txt}"
native_source="${NATIVE_SKILLS_SOURCE:-$repo_root/claude/.claude/skills}"
pi_settings="${PI_SETTINGS:-$repo_root/pi/.pi/agent/settings.json}"
pi_extensions_root="${PI_EXTENSIONS_ROOT:-$HOME/.pi/agent/git/github.com/edheltzel/pi-extensions}"
pi_extensions_registration="${PI_EXTENSIONS_REGISTRATION:-git:github.com/edheltzel/pi-extensions}"
pi_command="${PI_COMMAND:-pi}"
errors=0
checked=0

fail() {
	printf 'FAIL %s\n' "$1" >&2
	errors=$((errors + 1))
}

trim_whitespace() {
	local value="$1"
	value="${value#"${value%%[![:space:]]*}"}"
	value="${value%"${value##*[![:space:]]}"}"
	printf '%s' "$value"
}

check_skill() {
	local path="$1"
	local owner="$2"
	checked=$((checked + 1))
	if [ -L "$path" ] && [ ! -e "$path" ]; then
		fail "$owner: dangling skill link: $path"
	elif [ ! -f "$path/SKILL.md" ]; then
		fail "$owner: missing SKILL.md: $path/SKILL.md"
	fi
}

agent_skill_root() {
	case "$1" in
	claude-code) printf '%s\n' "$HOME/.claude/skills" ;;
	pi) printf '%s\n' "$HOME/.pi/agent/skills" ;;
	codex) printf '%s\n' "$HOME/.codex/skills" ;;
	*) return 1 ;;
	esac
}

if [ ! -f "$manifest" ]; then
	printf 'missing skills manifest: %s\n' "$manifest" >&2
	exit 2
fi

while IFS= read -r line || [ -n "$line" ]; do
	[[ "$line" =~ ^[[:space:]]*# ]] && continue
	if [[ "$line" =~ ^(.*[^[:space:]])[[:space:]]+#.*$ ]]; then
		line="${BASH_REMATCH[1]}"
	fi
	line="$(trim_whitespace "$line")"
	[ -z "$line" ] && continue

	descriptor="${line#*=}"
	names="${descriptor%%|*}"
	entry_agents=""
	IFS='|' read -ra descriptor_parts <<<"$descriptor"
	last_part="${descriptor_parts[${#descriptor_parts[@]} - 1]}"
	if [[ "$last_part" == agents=* ]]; then
		entry_agents="${last_part#agents=}"
	fi

	if [ -z "$names" ] || [ "$names" = "$line" ] || [ "$names" = "*" ]; then
		fail "manifest entry has no explicit skill inventory: $line"
		continue
	fi

	IFS=',' read -ra skill_names <<<"$names"
	for name in "${skill_names[@]}"; do
		[ -n "$name" ] || continue
		if [ -z "$entry_agents" ]; then
			check_skill "$HOME/.agents/skills/$name" "skills.txt"
			continue
		fi
		IFS=',' read -ra targets <<<"$entry_agents"
		for target in "${targets[@]}"; do
			if ! root="$(agent_skill_root "$target")"; then
				fail "skills.txt: unsupported agents= target '$target' for $name"
				continue
			fi
			check_skill "$root/$name" "skills.txt [$target]"
		done
	done
done <"$manifest"

anti_slop_root="$HOME/.claude/skills/anti-slop"
if [ -d "$anti_slop_root" ]; then
	anti_slop_resources=(
		"scripts/lint_voice.py"
		"scripts/scan_common.py"
		"scripts/scan_packages.py"
		"scripts/scan_placeholders.py"
		"scripts/scan_refs.py"
		"scripts/scan_residue.py"
		"scripts/score_substance.py"
		"references/code-markers.md"
		"references/false-positives.md"
		"references/markers-tier1.md"
		"references/markers-tier2.md"
		"references/markers-tier3.md"
		"references/structural-tests.md"
	)
	for resource in "${anti_slop_resources[@]}"; do
		if [ ! -f "$anti_slop_root/$resource" ]; then
			fail "Claude anti-slop support file is missing: $anti_slop_root/$resource"
		fi
	done
fi

pi_anti_slop_skills=(anti-slop slop-code slop-review slop-rewrite slop-verify)
if ! command -v jq >/dev/null 2>&1; then
	fail "Pi anti-slop registration: jq is required"
elif [ ! -f "$pi_settings" ]; then
	fail "Pi anti-slop registration: missing settings file: $pi_settings"
else
	direct_count="$(jq --arg registration "$pi_extensions_registration" \
		'[.packages[]? | select(. == $registration)] | length' "$pi_settings")"
	legacy_count="$(jq '[.packages[]? | select(. == "local-packages/pi-extensions")] | length' "$pi_settings")"
	if [ "$direct_count" -ne 1 ]; then
		fail "Pi anti-slop registration: expected exactly one $pi_extensions_registration entry in $pi_settings"
	fi
	if [ "$legacy_count" -ne 0 ]; then
		fail "Pi anti-slop registration: retired local Pi Extensions registration remains in $pi_settings"
	fi
fi
if ! command -v "$pi_command" >/dev/null 2>&1 && [ ! -x "$pi_command" ]; then
	fail "Pi anti-slop discovery: Pi command is unavailable: $pi_command"
elif ! pi_list_output="$("$pi_command" list 2>&1)"; then
	fail "Pi anti-slop discovery: pi list failed: $pi_list_output"
else
	discovered_registration_count=0
	discovered_root_count=0
	while IFS= read -r output_line; do
		output_line="$(trim_whitespace "$output_line")"
		[ "$output_line" = "$pi_extensions_registration" ] && discovered_registration_count=$((discovered_registration_count + 1))
		[ "$output_line" = "$pi_extensions_root" ] && discovered_root_count=$((discovered_root_count + 1))
	done <<<"$pi_list_output"
	if [ "$discovered_registration_count" -ne 1 ] || [ "$discovered_root_count" -ne 1 ]; then
		fail "Pi anti-slop discovery: pi list must resolve exactly one $pi_extensions_registration entry to $pi_extensions_root"
	fi
fi
for name in "${pi_anti_slop_skills[@]}"; do
	check_skill "$pi_extensions_root/extensions/pi-anti-slop/skills/$name" "Pi Extensions Git package"
	if [ -e "$HOME/.pi/agent/skills/$name" ] || [ -L "$HOME/.pi/agent/skills/$name" ]; then
		fail "Pi anti-slop discovery: duplicate physical skill must be removed: $HOME/.pi/agent/skills/$name"
	fi
done

if [ ! -d "$native_source" ]; then
	fail "native Claude source is missing: $native_source"
else
	while IFS= read -r source; do
		name="$(basename "$source")"
		check_skill "$HOME/.claude/skills/$name" "Atlas native Claude skill"
	done < <(find "$native_source" -mindepth 1 -maxdepth 1 \( -type d -o -type l \) -print | LC_ALL=C sort)
fi

roots=(
	"$HOME/.agents/skills"
	"$HOME/.claude/skills"
	"$HOME/.pi/agent/skills"
	"$HOME/.omp/agent/skills"
	"$HOME/.omp/skills"
	"$HOME/.jcode/skills"
	"$HOME/.codex/skills"
)
for root in "${roots[@]}"; do
	[ -d "$root" ] || [ -L "$root" ] || continue
	while IFS= read -r dangling; do
		fail "dangling link under managed skill root: $dangling"
	done < <(find -L "$root" -type l ! -exec test -e {} \; -print)
done

if [ "$errors" -ne 0 ]; then
	exit 1
fi

printf 'PASS installed skills: %s manifest/native checks and no dangling managed links\n' "$checked"
