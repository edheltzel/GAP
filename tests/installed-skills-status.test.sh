#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

home="$tmp/home"
manifest="$tmp/skills.txt"
native="$tmp/native"
pi_settings="$tmp/pi-settings.json"
pi_extensions="$tmp/PiExtensions"
pi_registration="test-pi-extensions-registration"
mkdir -p \
	"$home/.agents/skills/shared" \
	"$home/.claude/skills/anti-slop/scripts" \
	"$home/.claude/skills/anti-slop/references" \
	"$home/.claude/skills/Native" \
	"$native/Native"
for skill in anti-slop slop-code slop-review slop-rewrite slop-verify; do
	mkdir -p "$pi_extensions/packages/pi-anti-slop/skills/$skill"
	printf '%s\n' "# $skill" >"$pi_extensions/packages/pi-anti-slop/skills/$skill/SKILL.md"
done
cat >"$pi_settings" <<EOF
{"packages":["$pi_registration"]}
EOF
printf '%s\n' '# shared' >"$home/.agents/skills/shared/SKILL.md"
printf '%s\n' '# anti-slop' >"$home/.claude/skills/anti-slop/SKILL.md"
printf '%s\n' '# native runtime' >"$home/.claude/skills/Native/SKILL.md"
printf '%s\n' '# native source' >"$native/Native/SKILL.md"
for script in lint_voice scan_common scan_packages scan_placeholders scan_refs scan_residue score_substance; do
	printf '%s\n' '#!/usr/bin/env python3' >"$home/.claude/skills/anti-slop/scripts/$script.py"
done
for reference in code-markers false-positives markers-tier1 markers-tier2 markers-tier3 structural-tests; do
	printf '%s\n' "# $reference" >"$home/.claude/skills/anti-slop/references/$reference.md"
done
cat >"$manifest" <<'EOF'
example/repo=shared
local:../Other#0123456789012345678901234567890123456789=anti-slop|skills/anti-slop/SKILL.md|0123456789012345678901234567890123456789|agents=claude-code
EOF

run_check() {
	HOME="$home" SKILLS_MANIFEST="$manifest" NATIVE_SKILLS_SOURCE="$native" \
		PI_SETTINGS="$pi_settings" PI_EXTENSIONS_ROOT="$pi_extensions" \
		PI_EXTENSIONS_REGISTRATION="$pi_registration" \
		"$repo_root/check-installed-skills.sh"
}

run_check >/dev/null

rm "$home/.agents/skills/shared/SKILL.md"
if run_check >"$tmp/missing.out" 2>&1; then
	printf 'installed skill checker accepted a missing manifest skill\n' >&2
	exit 1
fi
grep -q 'missing SKILL.md' "$tmp/missing.out"
printf '%s\n' '# shared' >"$home/.agents/skills/shared/SKILL.md"

rm "$home/.claude/skills/anti-slop/references/false-positives.md"
if run_check >"$tmp/support.out" 2>&1; then
	printf 'installed skill checker accepted a missing anti-slop support file\n' >&2
	exit 1
fi
grep -q 'Claude anti-slop support file is missing' "$tmp/support.out"
printf '%s\n' '# false-positives' >"$home/.claude/skills/anti-slop/references/false-positives.md"

rm "$pi_extensions/packages/pi-anti-slop/skills/slop-verify/SKILL.md"
if run_check >"$tmp/pi-package.out" 2>&1; then
	printf 'installed skill checker accepted a missing Pi anti-slop package skill\n' >&2
	exit 1
fi
grep -q 'PiExtensions native package: missing SKILL.md' "$tmp/pi-package.out"
printf '%s\n' '# slop-verify' >"$pi_extensions/packages/pi-anti-slop/skills/slop-verify/SKILL.md"

cat >"$pi_settings" <<EOF
{"packages":["$pi_registration","local-packages/pi-extensions"]}
EOF
if run_check >"$tmp/pi-registration.out" 2>&1; then
	printf 'installed skill checker accepted retired PiExtensions registration\n' >&2
	exit 1
fi
grep -q 'retired local-packages/pi-extensions entry remains' "$tmp/pi-registration.out"
cat >"$pi_settings" <<EOF
{"packages":["$pi_registration"]}
EOF

mkdir -p "$home/.jcode/skills"
ln -s "$tmp/does-not-exist" "$home/.jcode/skills/broken"
if run_check >"$tmp/dangling.out" 2>&1; then
	printf 'installed skill checker accepted a dangling managed link\n' >&2
	exit 1
fi
grep -q 'dangling link under managed skill root' "$tmp/dangling.out"

printf 'PASS: installed skill checker rejects missing skills, package registrations, and dangling links\n'
