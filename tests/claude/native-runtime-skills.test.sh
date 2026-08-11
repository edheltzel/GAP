#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SKILLS="$ROOT/claude/.claude/skills"
BUN="$(command -v bun)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

expect_fail_contains() {
  local needle="$1"
  shift
  local output status
  set +e
  output="$("$@" 2>&1)"
  status=$?
  set -e
  [ "$status" -ne 0 ] || fail "command unexpectedly succeeded: $*"
  printf '%s' "$output" | grep -Fq -e "$needle" || fail "missing failure text '$needle' from: $*"
}

# Art must reject an unavailable local dependency before touching the input.
printf 'not-an-image' > "$TMP/input.png"
expect_fail_contains "rembg not found" env REMBG_BIN="$TMP/missing-rembg" "$BUN" "$SKILLS/Art/Tools/RemoveBg.ts" "$TMP/input.png"
[ "$(cat "$TMP/input.png")" = "not-an-image" ] || fail "Art modified input before dependency validation"

# AudioEditor takes classification as explicit input, not a hidden model dependency.
cat > "$TMP/audio.transcript.json" <<'JSON'
{"text":" hello world","chunks":[{"text":" hello","timestamp":[0,0.5]},{"text":" world","timestamp":[7,7.5]}]}
JSON
cat > "$TMP/model-edits.json" <<'JSON'
[{"type":"CUT_FILLER","start":0.1,"end":0.3,"reason":"inline classification","context":"hello","confidence":0.9}]
JSON
expect_fail_contains "--model-edits" "$BUN" "$SKILLS/AudioEditor/Tools/Analyze.ts" "$TMP/audio.transcript.json"
"$BUN" "$SKILLS/AudioEditor/Tools/Analyze.ts" "$TMP/audio.transcript.json" --model-edits "$TMP/model-edits.json" --output "$TMP/edits.json" >/dev/null
"$BUN" -e 'const e=await Bun.file(process.argv[1]).json(); if(e.length!==2 || !e.some((x)=>x.type==="CUT_DEAD_AIR") || !e.some((x)=>x.type==="CUT_FILLER")) process.exit(1)' "$TMP/edits.json" || fail "AudioEditor did not validate and merge explicit edits"
expect_fail_contains "No supported Whisper CLI found" env PATH="/usr/bin:/bin" "$BUN" "$SKILLS/AudioEditor/Tools/Transcribe.ts" "$TMP/input.png"

# ContextSearch is now only a project-scoped Recall compatibility guide.
expect_fail_contains "recall_memory_memory_hybrid_search" "$BUN" "$SKILLS/ContextSearch/Tools/ContextSearch.ts" "runtime skills"
expect_fail_contains '"project"' "$BUN" "$SKILLS/ContextSearch/Tools/ContextSearch.ts" "runtime skills" --json

# CreateCLI templates prefer process environment and the current Claude env file.
grep -Fq -e "process.env.API_KEY || fileValue" "$SKILLS/CreateCLI/Patterns.md" || fail "CreateCLI does not prefer the process environment"
grep -Fq -e "join(homedir(), '.claude', '.env')" "$SKILLS/CreateCLI/Workflows/CreateCli.md" || fail "CreateCLI does not use ~/.claude/.env"

# Evals model graders load without removed modules and fail closed without supplied judgments.
"$BUN" -e 'await Promise.all([import(process.argv[1]),import(process.argv[2]),import(process.argv[3])])' \
  "$SKILLS/Evals/Graders/ModelBased/LLMRubric.ts" \
  "$SKILLS/Evals/Graders/ModelBased/NaturalLanguageAssert.ts" \
  "$SKILLS/Evals/Graders/ModelBased/PairwiseComparison.ts"
"$BUN" -e '
  const { LLMRubricGrader } = await import(process.argv[1]);
  const grader = new LLMRubricGrader({type:"llm_rubric",params:{rubric:"Be correct"}});
  const result = await grader.grade({task_id:"t",trial_id:"1",output:"x",transcript:{task_id:"t",trial_id:"1",started_at:"now",turns:[],tool_calls:[],metrics:{n_turns:0,n_tool_calls:0,total_tokens:0,input_tokens:0,output_tokens:0,wall_time_ms:0}}});
  if(result.passed || !result.reasoning?.includes("judge_response")) process.exit(1);
' "$SKILLS/Evals/Graders/ModelBased/LLMRubric.ts" || fail "Evals model grader did not fail closed"
bridge_help="$($BUN "$SKILLS/Evals/Tools/AlgorithmBridge.ts" --help)"
printf '%s' "$bridge_help" | grep -Fq -e "--output-file" || fail "AlgorithmBridge lacks explicit output input"
if printf '%s' "$bridge_help" | grep -Fq -e "--update-isc"; then fail "AlgorithmBridge still advertises retired ISC mutation"; fi
expect_fail_contains "caller-supplied respond adapter" "$BUN" -e '
  const { ExplicitAgentAdapter } = await import(process.argv[1]);
  new ExplicitAgentAdapter({});
' "$SKILLS/Evals/Tools/LifeosAgentAdapter.ts"
"$BUN" -e '
  const { PairwiseComparisonGrader } = await import(process.argv[1]);
  const grader = new PairwiseComparisonGrader({type:"pairwise_comparison",params:{reference:"reference",position_swap:false,judge_responses:["REASONING: better\nWINNER: A"]}});
  const result = await grader.grade({task_id:"t",trial_id:"1",output:"candidate",transcript:{task_id:"t",trial_id:"1",started_at:"now",turns:[],tool_calls:[],metrics:{n_turns:0,n_tool_calls:0,total_tokens:0,input_tokens:0,output_tokens:0,wall_time_ms:0}}});
  if(!result.passed || result.score!==1) process.exit(1);
' "$SKILLS/Evals/Graders/ModelBased/PairwiseComparison.ts" || fail "Evals pairwise grader did not consume explicit judgment"

# Fabric keeps local patterns usable without mandatory persistence or CLI claims.
grep -Fq -e "Optional Recall Persistence" "$SKILLS/Fabric/Workflows/ExecutePattern.md" || fail "Fabric lacks optional Recall persistence guidance"
grep -Fq -e "command -v fabric" "$SKILLS/Fabric/Workflows/ExecutePattern.md" || fail "Fabric lacks an honest CLI availability gate"

# Prompting and Science route only to included current surfaces.
grep -Fq -e "Fabric/Workflows/UpdatePatterns.md" "$SKILLS/Prompting/Standards.md" || fail "Prompting does not point to the included Fabric updater workflow"
grep -Fq -e "Evals/Tools/AlgorithmBridge.ts" "$SKILLS/Science/Workflows/DesignExperiment.md" || fail "Science does not point to the current Evals runner"

# Telos personal-data mutation and hidden inference are disabled.
mkdir -p "$TMP/home/.claude/LIFEOS/USER/TELOS"
printf '# Books\n' > "$TMP/home/.claude/LIFEOS/USER/TELOS/BOOKS.md"
expect_fail_contains "read-only" env HOME="$TMP/home" "$BUN" "$SKILLS/Telos/Tools/UpdateTelos.ts" BOOKS.md '- New book' 'test'
[ "$(cat "$TMP/home/.claude/LIFEOS/USER/TELOS/BOOKS.md")" = "# Books" ] || fail "Telos fail-closed tool mutated personal data"
grep -Fq -e "adapter_called: false" "$SKILLS/Telos/DashboardTemplate/App/api/chat/route.ts" || fail "Telos dashboard chat does not fail closed"
if grep -Fq -e "Inference.ts" "$SKILLS/Telos/DashboardTemplate/App/api/chat/route.ts"; then fail "Telos dashboard still invokes retired inference"; fi

# USMetrics callers must choose and validate the data root.
expect_fail_contains "US_METRICS_DATA_ROOT" env FRED_API_KEY=test "$BUN" "$SKILLS/USMetrics/Tools/UpdateSubstrateMetrics.ts" --data-root "$TMP/missing-data-root"

# Reject load-bearing stale references in this bounded cluster.
scoped=(Art AudioEditor ContextSearch CreateCLI Evals Fabric Prompting Science Telos USMetrics)
for skill in "${scoped[@]}"; do
  path="$SKILLS/$skill"
  [ -d "$path" ] || fail "missing native skill inventory entry: $skill"
  if grep -R -n -E \
    'LIFEOS_SKILL_DIR|LIFEOS/TOOLS/Inference\.ts|THEALGORITHM/Tools/ISCManager\.ts|~/.claude/Templates/Tools/RenderTemplate\.ts|Evals/EvalServer|_HARVEST/Tools/harvest\.ts|Fabric/Tools/update-patterns\.sh|~/Projects/Substrate/Data/US-Common-Metrics|\.claude/LifeOS/\.env' \
    "$path" >/dev/null; then
    fail "stale load-bearing reference remains in $skill"
  fi
done

[ -f "$SKILLS/Prompting/Templates/Tools/RenderTemplate.ts" ] || fail "included template renderer missing"
[ -f "$SKILLS/Fabric/Workflows/UpdatePatterns.md" ] || fail "included Fabric updater workflow missing"
[ -f "$SKILLS/Telos/DashboardTemplate/package.json" ] || fail "included Telos dashboard template missing"

printf 'PASS: native runtime skill cluster uses bounded, fail-closed current contracts\n'
