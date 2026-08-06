#!/usr/bin/env bash
# Validate manifest-owned and repository-owned skill installations without mutating them.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
manifest="${SKILLS_MANIFEST:-$repo_root/skills.txt}"
native_source="${NATIVE_SKILLS_SOURCE:-$repo_root/claude/.claude/skills}"
errors=0
checked=0

fail() {
	printf 'FAIL %s\n' "$1" >&2
	errors=$((errors + 1))
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
	line="$(printf '%s' "$line" | tr -d '[:space:]')"
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
	done < <(find "$root" -type l ! -exec test -e {} \; -print)
done

if [ "$errors" -ne 0 ]; then
	exit 1
fi

printf 'PASS installed skills: %s manifest/native checks and no dangling managed links\n' "$checked"
