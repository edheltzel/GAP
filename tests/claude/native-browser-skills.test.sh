#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
skills_root="$repo_root/claude/.claude/skills"
this_test="$repo_root/tests/claude/native-browser-skills.test.sh"

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

aphorisms="$skills_root/Aphorisms/Workflows/ResearchThinker.md"
bright_skill="$skills_root/BrightData/SKILL.md"
bright_scrape="$skills_root/BrightData/Workflows/FourTierScrape.md"
bright_crawl="$skills_root/BrightData/Workflows/Crawl.md"
browser_skill="$skills_root/Browser/SKILL.md"
browser_readme="$skills_root/Browser/README.md"
browser_form="$skills_root/Browser/Recipes/FormFill.md"
browser_recipes="$skills_root/Browser/Recipes/README.md"
browser_compare="$skills_root/Browser/Recipes/ScreenshotCompare.md"
browser_summary="$skills_root/Browser/Recipes/SummarizePage.md"
browser_automate="$skills_root/Browser/Workflows/Automate.md"
browser_stories="$skills_root/Browser/Workflows/ReviewStories.md"
browser_update="$skills_root/Browser/Workflows/Update.md"
extract_skill="$skills_root/ExtractWisdom/SKILL.md"
extract_workflow="$skills_root/ExtractWisdom/Workflows/Extract.md"
html_skill="$skills_root/HTML/SKILL.md"
html_workflow="$skills_root/HTML/Workflows/Render.md"
ideate_skill="$skills_root/Ideate/SKILL.md"
ideate_full="$skills_root/Ideate/Workflows/FullCycle.md"
ideate_quick="$skills_root/Ideate/Workflows/QuickCycle.md"
ideate_steal="$skills_root/Ideate/Workflows/Steal.md"
interceptor_skill="$skills_root/Interceptor/SKILL.md"
remotion_content="$skills_root/Remotion/Workflows/ContentToAnimation.md"
remotion_generated="$skills_root/Remotion/Workflows/GeneratedContentVideo.md"
remotion_patterns="$skills_root/Remotion/Patterns.md"
remotion_ai_pipeline="$skills_root/Remotion/Tools/Ref-ai-pipeline.md"
webdesign_skill="$skills_root/Webdesign/SKILL.md"
webdesign_readme="$skills_root/Webdesign/README.md"
webdesign_direct="$skills_root/Webdesign/Workflows/DirectDesign.md"
webdesign_philosophy="$skills_root/Webdesign/References/FrontendDesignPhilosophy.md"

for tool in /opt/homebrew/bin/chrome-devtools-axi /opt/homebrew/bin/lavish-axi; do
	[ -x "$tool" ] || fail "required executable is unavailable: $tool"
done

for tool in web_search source_check fetch_content; do
	assert_contains "$aphorisms" "$tool" "Aphorisms must use current research tool $tool"
done
assert_absent "$aphorisms" 'research_skill.parallel_research' 'Aphorisms still invokes the removed Research skill'
assert_absent "$aphorisms" 'fabric -y' 'Aphorisms still invokes missing fabric'

assert_contains "$bright_skill" 'fetch_content' 'BrightData Tier 1 must use current content fetching'
assert_contains "$bright_skill" '/opt/homebrew/bin/chrome-devtools-axi' 'BrightData Tier 3 must use installed Chrome tooling'
assert_contains "$bright_scrape" '/opt/homebrew/bin/chrome-devtools-axi' 'BrightData scrape workflow lacks current Chrome commands'
assert_contains "$bright_scrape" 'optional credentialed escalation' 'BrightData must label paid escalation as optional and credentialed'
assert_absent "$bright_skill" 'mcp__Brightdata__' 'BrightData names an unconfigured MCP tool'
assert_absent "$bright_scrape" 'mcp__Brightdata__' 'BrightData workflow names an unconfigured MCP tool'
assert_absent "$bright_crawl" 'mcp__Brightdata__' 'BrightData crawl names an unconfigured MCP tool'
assert_contains "$bright_scrape" 'curl -L' 'BrightData must preserve its public curl path'

assert_contains "$browser_skill" '/opt/homebrew/bin/chrome-devtools-axi' 'Browser must use the installed Chrome CLI'
assert_contains "$browser_skill" 'STALE_REF' 'Browser must preserve the chrome-devtools-axi ref contract'
assert_contains "$browser_skill" "fresh \`snapshot\`" 'Browser must verify state changes with a fresh snapshot'
assert_absent "$browser_skill" 'agent-browser open' 'Browser still publishes absent agent-browser commands'
assert_absent "$browser_readme" 'interceptor open' 'Browser README still publishes absent Interceptor commands'
assert_contains "$browser_stories" 'Independent Agent dispatch' 'Browser story review must use current Agent dispatch'
assert_absent "$browser_stories" 'Task(' 'Browser story review still publishes retired Task syntax'
assert_absent "$browser_stories" 'TeamCreate' 'Browser story review still publishes retired team syntax'

for file in "$extract_skill" "$extract_workflow"; do
	assert_contains "$file" 'fetch_content' "ExtractWisdom must use current URL/video fetching in $file"
	assert_absent "$file" 'fabric -y' "ExtractWisdom still invokes missing fabric in $file"
done
assert_contains "$extract_workflow" 'optional Recall' 'ExtractWisdom must make Recall persistence optional'

assert_contains "$html_skill" 'Tools/Render.ts' 'HTML must retain its deterministic local renderer'
assert_contains "$html_workflow" '/opt/homebrew/bin/lavish-axi' 'HTML must deliver through installed lavish-axi'
assert_contains "$html_workflow" 'local HTML file' 'HTML must support direct local delivery'
for stale in artifact-design 'Artifact tool' Interceptor; do
	assert_absent "$html_workflow" "$stale" "HTML still depends on retired $stale"
done

for file in "$ideate_skill" "$ideate_full" "$ideate_quick" "$ideate_steal"; do
	assert_absent "$file" 'Skill("Research")' "Ideate still invokes the removed Research skill in $file"
	assert_absent "$file" 'LIFEOS/MEMORY/WORK' "Ideate still persists into removed LifeOS memory in $file"
done
assert_contains "$ideate_full" 'web_search' 'Ideate FullCycle must use current web research'
assert_contains "$ideate_full" 'fetch_content' 'Ideate FullCycle must use current content fetching'
assert_contains "$ideate_skill" '.agents/atlas/artifacts/' 'Ideate requested persistence must use the project artifact root'
assert_contains "$ideate_skill" 'only when the user requests persistence' 'Ideate persistence must be opt-in'

assert_contains "$interceptor_skill" 'Historical reference only' 'Interceptor must gate unavailable historical material'
assert_contains "$interceptor_skill" '/opt/homebrew/bin/chrome-devtools-axi' 'Interceptor current browser route must use installed Chrome tooling'
assert_contains "$interceptor_skill" "Do not execute any \`interceptor\` command below" 'Interceptor must fail closed around historical commands'

for file in "$remotion_content" "$remotion_generated"; do
	assert_absent "$file" 'Parser' "Remotion still depends on missing Parser in $file"
	assert_contains "$file" 'fetch_content' "Remotion must use current content fetching in $file"
done
assert_absent "$remotion_content" 'Remotion/theme' 'Remotion uses the invalid theme path'
assert_contains "$remotion_content" 'Remotion/Tools/Theme' 'Remotion must import the current theme module'
assert_absent "$remotion_patterns" 'Tools/Reference/' 'Remotion points at a missing reference directory'
assert_contains "$remotion_patterns" 'Tools/Ref-*.md' 'Remotion must point at current low-level references'
assert_absent "$remotion_ai_pipeline" 'Tools/Art.ts' 'Remotion points at a missing Art helper'
assert_contains "$remotion_ai_pipeline" 'skills/Art/Tools/Generate.ts' 'Remotion must point at the current Art generator'

assert_contains "$webdesign_skill" 'DirectDesign' 'Webdesign must retain DirectDesign as the supported route'
assert_contains "$webdesign_readme" 'DirectDesign' 'Webdesign README must document the supported route'
assert_contains "$webdesign_direct" '/opt/homebrew/bin/chrome-devtools-axi' 'DirectDesign must use current browser verification'
assert_contains "$webdesign_philosophy" 'chrome-devtools-axi' 'Webdesign philosophy must name current browser verification'
for stale in '/design-sync' '/design' Interceptor; do
	assert_absent "$webdesign_skill" "$stale" "Webdesign still routes through unsupported $stale"
done

em_dash=$'\xE2\x80\x94'
touched=(
	"$aphorisms"
	"$bright_skill"
	"$bright_scrape"
	"$bright_crawl"
	"$browser_skill"
	"$browser_readme"
	"$browser_form"
	"$browser_recipes"
	"$browser_compare"
	"$browser_summary"
	"$browser_automate"
	"$browser_stories"
	"$browser_update"
	"$extract_skill"
	"$extract_workflow"
	"$html_skill"
	"$html_workflow"
	"$ideate_skill"
	"$ideate_full"
	"$ideate_quick"
	"$ideate_steal"
	"$interceptor_skill"
	"$remotion_content"
	"$remotion_generated"
	"$remotion_patterns"
	"$remotion_ai_pipeline"
	"$webdesign_skill"
	"$webdesign_readme"
	"$webdesign_direct"
	"$webdesign_philosophy"
	"$this_test"
)
for file in "${touched[@]}"; do
	assert_absent "$file" "$em_dash" "em dash character found in $file"
done

printf 'PASS: native browser skills use available execution paths\n'
