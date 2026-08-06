#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
aperture="$repo_root/claude/.claude/skills/ApertureOscillation"
iterative="$repo_root/claude/.claude/skills/IterativeDepth"
story="$repo_root/claude/.claude/skills/WriteStory"

fail() {
	printf '%s\n' "$1" >&2
	exit 1
}

assert_contains() {
	local file="$1"
	local text="$2"
	local message="$3"
	grep -Fq "$text" "$file" || fail "$message"
}

assert_tree_absent() {
	local root="$1"
	local text="$2"
	local message="$3"
	if grep -RFn "$text" "$root" >/dev/null; then
		grep -RFn "$text" "$root" >&2
		fail "$message"
	fi
}

assert_contains "$iterative/Workflows/Explore.md" 'plain Markdown' \
	'IterativeDepth must return plain Markdown'
assert_contains "$iterative/Workflows/Explore.md" 'maximum of 12 criteria' \
	'IterativeDepth must bound its criteria set'
for contract in 'C-01' 'Status' 'Evidence expectation' 'Refined criteria' 'Anti-criteria'; do
	assert_contains "$iterative/Workflows/Explore.md" "$contract" \
		"IterativeDepth output contract is missing: $contract"
done

for contract in 'Problem or decision' 'Tactical lens' 'Strategic lens' 'self-contained'; do
	assert_contains "$aperture/Workflows/Oscillate.md" "$contract" \
		"ApertureOscillation contract is missing: $contract"
done
assert_contains "$aperture/Workflows/Oscillate.md" 'ALIGNED | TENSIONED | DIVERGENT' \
	'ApertureOscillation must provide a stable coherence verdict'

assert_contains "$story/SKILL.md" 'user-selected project directory' \
	'WriteStory must require a user-selected project directory'
assert_contains "$story/SKILL.md" 'Never guess or create a project directory' \
	'WriteStory must fail closed instead of guessing a project path'
for artifact in interview.md story-bible.md; do
	assert_contains "$story/SKILL.md" "$artifact" \
		"WriteStory must document $artifact"
	assert_contains "$story/Workflows/Interview.md" "$artifact" \
		"Interview workflow must document $artifact"
	assert_contains "$story/Workflows/BuildBible.md" "$artifact" \
		"BuildBible workflow must document $artifact"
done
for workflow in Explore Revise WriteChapter; do
	assert_contains "$story/Workflows/$workflow.md" 'story-bible.md' \
		"$workflow workflow must use story-bible.md for continuity"
done
assert_contains "$story/Workflows/BuildBible.md" 'Read the existing file in full before updating it' \
	'BuildBible must preserve existing writer material before updates'
assert_contains "$story/Workflows/BuildBible.md" '## Story checklist' \
	'BuildBible must store normal Markdown checklists'

roots=("$aperture" "$iterative" "$story")
retired_dependencies=(
	'Algorithm OBSERVE'
	'Algorithm THINK'
	'Capability Audit'
	'ISC Output:'
	'ISA.md'
	'project ISA'
	'Skill("ISA"'
	'TaskCreate'
	'TaskUpdate'
	'TaskList'
	'TaskOutput'
	'TeamCreate'
	'TeamUpdate'
	'SendMessage'
	'LIFEOS/'
	'filesystem MCP'
	'mcp__filesystem'
)
for root in "${roots[@]}"; do
	for retired in "${retired_dependencies[@]}"; do
		assert_tree_absent "$root" "$retired" \
			"retired dependency remains under ${root##*/}: $retired"
	done
done

operational_docs=(
	"$aperture/SKILL.md"
	"$aperture/Workflows/Oscillate.md"
	"$iterative/SKILL.md"
	"$iterative/Workflows/Explore.md"
	"$story/SKILL.md"
	"$story/Workflows/Interview.md"
	"$story/Workflows/BuildBible.md"
	"$story/Workflows/Explore.md"
	"$story/Workflows/Revise.md"
	"$story/Workflows/WriteChapter.md"
	"$repo_root/tests/claude/native-continuity-skills.test.sh"
)
em_dash=$'\xE2\x80\x94'
for file in "${operational_docs[@]}"; do
	if grep -Fq "$em_dash" "$file"; then
		fail "em dash character found in $file"
	fi
done

printf 'PASS: native continuity skills use standalone current contracts\n'
