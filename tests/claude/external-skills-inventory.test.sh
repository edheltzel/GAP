#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
manifest="$repo_root/skills.txt"

expected=(
	agent-browser
	anti-slop
	ast-grep
	ast-grep-outline
	chrome-devtools-axi
	codegraph
	computer-use
	gh-axi
	git-guardrails-claude-code
	grill-me
	grill-with-docs
	handoff
	herdr
	herdr-browser
	impeccable
	improve-codebase-architecture
	karpathy-guidelines
	lavish
	migrate-to-shoehorn
	opensrc
	orca-cli
	orchestration
	pm-tools
	prototype
	quota-axi
	recall-add
	recall-doctor
	recall-dump
	recall-loa
	recall-recent
	recall-scout
	recall-search
	recall-stats
	recall-update
	scaffold-exercises
	setup-matt-pocock-skills
	setup-pre-commit
	slop-code
	slop-review
	slop-rewrite
	slop-verify
	tasks-axi
	tdd
	teach
	triage
	writing-beats
	writing-fragments
	writing-shape
)

declared="$({
	while IFS= read -r line || [ -n "$line" ]; do
		[[ "$line" =~ ^[[:space:]]*# ]] && continue
		line="${line%%[[:space:]]#*}"
		line="$(printf '%s' "$line" | tr -d '[:space:]')"
		[ -z "$line" ] && continue
		descriptor="${line#*=}"
		names="${descriptor%%|*}"
		IFS=',' read -ra entries <<<"$names"
		for name in "${entries[@]}"; do
			[ -n "$name" ] && printf '%s\n' "$name"
		done
	done <"$manifest"
} | LC_ALL=C sort)"
expected_sorted="$(printf '%s\n' "${expected[@]}" | LC_ALL=C sort)"

if [ "$declared" != "$expected_sorted" ]; then
	printf 'shared skill manifest differs from the reconciled historical inventory\n' >&2
	diff -u <(printf '%s\n' "$expected_sorted") <(printf '%s\n' "$declared") >&2 || true
	exit 1
fi

if [ "$(printf '%s\n' "$declared" | uniq -d | wc -l | tr -d ' ')" -ne 0 ]; then
	printf 'shared skill manifest contains duplicate names\n' >&2
	exit 1
fi

if [ -e "$repo_root/claude/.claude/skills/anti-slop" ] || [ -L "$repo_root/claude/.claude/skills/anti-slop" ]; then
	printf 'manifest-owned anti-slop must not return to the Claude Stow package\n' >&2
	exit 1
fi

anti_slop_commit='3a9f04d287e492e38ad9ac24c76c9760266225aa'
anti_slop_entries="$(grep -E "^local:\.\./PiExtensions#$anti_slop_commit=(anti-slop|slop-code|slop-review|slop-rewrite|slop-verify)\\|packages/pi-anti-slop/skills/[^|]+/SKILL\.md\\|[0-9a-f]{40}\\|agents=claude-code$" "$manifest" || true)"
if [ "$(printf '%s\n' "$anti_slop_entries" | sed '/^$/d' | wc -l | tr -d ' ')" -ne 5 ]; then
	printf 'anti-slop skills must use pinned local PiExtensions sources owned only by Claude\n' >&2
	exit 1
fi

if [ -d "$repo_root/.claude/skills" ] && find "$repo_root/.claude/skills" -mindepth 1 -print -quit | grep -q .; then
	printf 'shared skills must not leave project-scoped pointers under .claude/skills\n' >&2
	exit 1
fi

printf 'PASS: shared skill manifest covers recovered pointers with explicit harness ownership\n'
