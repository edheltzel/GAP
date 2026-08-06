#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
agents_root="$repo_root/claude/.claude/skills/Agents"
delegation="$repo_root/claude/.claude/skills/Delegation/SKILL.md"
test_idea="$repo_root/claude/.claude/skills/WorldThreatModel/Workflows/TestIdea.md"
compose_agent="$agents_root/Tools/ComposeAgent.ts"
spawn_team="$agents_root/Workflows/SpawnTeam.md"
spawn_parallel="$agents_root/Workflows/SpawnParallelAgents.md"
this_test="$repo_root/tests/claude/native-orchestration-skills.test.sh"

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

assert_absent() {
  local file="$1"
  local text="$2"
  local message="$3"
  if grep -Fq "$text" "$file"; then
    fail "$message"
  fi
}

assert_contains \
  "$compose_agent" \
  "const USER_TRAITS_PATH = \`\${HOME}/.claude/LIFEOS/USER/SKILLCUSTOMIZATIONS/Agents/Traits.yaml\`;" \
  'ComposeAgent must define the retained optional user traits path'

if command -v bun >/dev/null 2>&1 \
  && [ -d "$agents_root/Tools/node_modules/yaml" ] \
  && [ -d "$agents_root/Tools/node_modules/handlebars" ]; then
  temp_home="$(mktemp -d "${TMPDIR:-/tmp}/atlas-compose-agent.XXXXXX")"
  trap 'rm -rf "$temp_home"' EXIT
  mkdir -p "$temp_home/.claude/skills"
  ln -s "$agents_root" "$temp_home/.claude/skills/Agents"

  base_output="$(HOME="$temp_home" bun run "$compose_agent" --list 2>&1)"
  grep -Fq 'AVAILABLE TRAITS' <<<"$base_output" || {
    printf '%s\n' "$base_output" >&2
    fail 'ComposeAgent --list did not load base traits'
  }
  if grep -Fq 'ReferenceError' <<<"$base_output"; then
    printf '%s\n' "$base_output" >&2
    fail 'ComposeAgent --list raised a ReferenceError'
  fi

  customization_dir="$temp_home/.claude/LIFEOS/USER/SKILLCUSTOMIZATIONS/Agents"
  mkdir -p "$customization_dir"
  cat > "$customization_dir/Traits.yaml" <<'YAML'
personality:
  skeptical:
    name: "Verified Custom Skeptic"
YAML
  custom_output="$(HOME="$temp_home" bun run "$compose_agent" --list 2>&1)"
  grep -Fq 'Verified Custom Skeptic' <<<"$custom_output" || {
    printf '%s\n' "$custom_output" >&2
    fail 'ComposeAgent did not merge the retained user customization file'
  }
fi

assert_absent "$spawn_team" 'Data/Teams/' 'SpawnTeam must not require missing team definitions'
for team in Engineering Architecture Marketing Design Security Research Content Strategy; do
  assert_contains "$spawn_team" "**$team**" "SpawnTeam is missing the $team brief"
done
assert_contains "$spawn_team" 'self-contained' 'SpawnTeam must state that its role catalog is self-contained'
assert_contains "$spawn_team" 'get_subagent_result' 'SpawnTeam must document Pi result controls'
assert_contains "$spawn_team" '/tasks' 'SpawnTeam must document Claude background inspection'

orchestration_docs=(
  "$spawn_team"
  "$spawn_parallel"
  "$delegation"
  "$test_idea"
)
for workflow in "${orchestration_docs[@]}"; do
  if grep -Eq '(^|[^[:alnum:]_])Task\(' "$workflow"; then
    fail "retired Task invocation remains in $workflow"
  fi
  assert_contains "$workflow" "\`Agent\`" "current Agent tool is not documented in $workflow"
done

for retired in TeamCreate TaskCreate TaskUpdate SendMessage; do
  assert_absent "$delegation" "$retired" "retired orchestration tool $retired remains in Delegation"
done
for stale in CLAUDE_CODE_FORK_SUBAGENT TaskOutput output_file haiku sonnet opus hook; do
  assert_absent "$delegation" "$stale" "stale Delegation claim remains: $stale"
done
assert_contains "$delegation" 'get_subagent_result' 'Delegation must document Pi background results'
assert_contains "$delegation" 'steer_subagent' 'Delegation must document Pi steering'
assert_contains "$delegation" '/tasks' 'Delegation must document Claude task inspection'
assert_contains "$delegation" 'TaskStop' 'Delegation must document Claude cancellation'
assert_contains "$delegation" 'fresh agent' 'Delegation must state the portable fresh-agent continuation rule'

assert_contains "$spawn_parallel" '/tasks' 'SpawnParallelAgents must document background inspection'
assert_contains "$spawn_parallel" "\`TaskStop\`" 'SpawnParallelAgents must document background cancellation'
assert_contains "$spawn_parallel" 'get_subagent_result' 'SpawnParallelAgents must document Pi result controls'
assert_contains "$spawn_parallel" 'Do not write JavaScript or TypeScript' 'SpawnParallelAgents must reject pseudo API examples'

required_models=(
  6-month.md
  1-year.md
  2-year.md
  3-year.md
  5-year.md
  7-year.md
  10-year.md
  15-year.md
  20-year.md
  30-year.md
  50-year.md
)
for model in "${required_models[@]}"; do
  assert_contains "$test_idea" "$model" "TestIdea preflight is missing $model"
done
for framework in FirstPrinciples RedTeam Council; do
  assert_contains "$test_idea" "$framework" "TestIdea is missing framework prerequisite $framework"
done
for research_tool in web_search source_check fetch_content; do
  assert_contains "$test_idea" "$research_tool" "TestIdea is missing current research interface $research_tool"
done
assert_absent "$test_idea" 'Research skill' 'TestIdea must not require the removed Research skill'
assert_contains "$test_idea" 'STOP. Do not run any analysis' 'TestIdea must fail closed when a model is absent'
assert_contains "$test_idea" 'Missing required framework: fail preflight and stop.' 'TestIdea must fail closed when a framework is absent'
assert_contains "$test_idea" 'get_subagent_result' 'TestIdea must document Pi result controls'
assert_contains "$test_idea" '/tasks' 'TestIdea must document Claude background inspection'
assert_absent "$test_idea" 'degrade gracefully' 'TestIdea must not claim graceful degradation'
assert_absent "$test_idea" 'Task tool' 'TestIdea must not name the retired Task tool'

em_dash=$'\xE2\x80\x94'
touched=(
  "$compose_agent"
  "$spawn_team"
  "$spawn_parallel"
  "$delegation"
  "$test_idea"
  "$this_test"
)
for file in "${touched[@]}"; do
  assert_absent "$file" "$em_dash" "em dash character found in $file"
done

printf 'PASS: native orchestration skills use current bounded Agent workflows\n'
