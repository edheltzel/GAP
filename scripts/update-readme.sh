#!/usr/bin/env bash
# Skill catalog generator. Regenerates the catalog tables in the root
# README.md from skills.sh.json and global/components.lock.
# Do skills live in Do-Skills at the pinned ref; Gap-native skills, if any,
# still live under skills/<scope>/<group>/<skill>/SKILL.md.
# Usage: bash scripts/update-readme.sh
set -euo pipefail

ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
README="$ROOT/README.md"
GROUPINGS_FILE="$ROOT/skills.sh.json"
LOCK_FILE="$ROOT/global/components.lock"
SKILLS_DIR="$ROOT/skills"

SCOPE_ORDER="global project"
GROUP_ORDER_global="core workflow operations content personal"
GROUP_ORDER_project="typescript frontend backend tooling swift product marketing personal"

lock_field() {
    node -e '
const fs = require("fs");
const lock = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
const c = lock.components["do-skills"];
if (!c?.ref) process.exit(1);
process.stdout.write(c[process.argv[2]] ?? "");
' "$LOCK_FILE" "$1"
}

PIN="$(lock_field ref)"
SOURCE="$(lock_field source)"
if [ -z "$SOURCE" ]; then
    SOURCE="edheltzel/Do-Skills"
fi
TREE_BASE="https://github.com/${SOURCE}/tree/${PIN}/skills"

scope_title() {
    case "$1" in
        global)  echo "Global" ;;
        project) echo "Project" ;;
        *)       echo "$1" ;;
    esac
}

scope_blurb() {
    case "$1" in
        global)  echo "Useful in any repo, or none." ;;
        project) echo "Pay off only when a project uses that stack or product." ;;
    esac
}

group_title() {
    case "$1" in
        core)       echo "Core" ;;
        workflow)   echo "Workflow" ;;
        operations) echo "Operations" ;;
        content)    echo "Content" ;;
        personal)   echo "Personal" ;;
        typescript) echo "TypeScript" ;;
        frontend)   echo "Frontend" ;;
        swift)      echo "Swift" ;;
        backend)    echo "Backend" ;;
        tooling)    echo "Tooling" ;;
        product)    echo "Product" ;;
        marketing)  echo "Marketing" ;;
        *)          echo "$1" ;;
    esac
}

group_meta() {
    node -e '
const fs = require("fs");
const [path, title] = process.argv.slice(1);
const config = JSON.parse(fs.readFileSync(path, "utf8"));
const group = config.groupings.find((candidate) => candidate.title === title);
if (!group) process.exit(0);
process.stdout.write((group.skills?.length ?? 0) + "\t" + (group.description ?? ""));
' "$GROUPINGS_FILE" "$(scope_title "$1"): $(group_title "$2")"
}

escape_cell() {
    printf '%s' "$1" | sed 's/|/\\|/g'
}

root_section=$(mktemp)
trap 'rm -f "$root_section"' EXIT
: > "$root_section"

total=0
group_count=0
for scope in $SCOPE_ORDER; do
    {
        printf '### [%s](%s/%s/)\n\n%s\n\n' "$(scope_title "$scope")" "$TREE_BASE" "$scope" "$(scope_blurb "$scope")"
        printf '| Group | Skills | Coverage |\n|-------|--------|----------|\n'
        preferred_var="GROUP_ORDER_$scope"
        for group in ${!preferred_var}; do
            meta=$(group_meta "$scope" "$group")
            [ -n "$meta" ] || continue
            count=$(printf '%s' "$meta" | cut -f1)
            desc=$(printf '%s' "$meta" | cut -f2-)
            total=$((total + count))
            group_count=$((group_count + 1))
            printf '| [%s](%s/%s/%s/) | %s | %s |\n' "$(group_title "$group")" "$TREE_BASE" "$scope" "$group" "$count" "$(escape_cell "$desc")"
        done
        printf '\n'
    } >> "$root_section"
done

native=$(find "$SKILLS_DIR" -name SKILL.md ! -path '*/references/*' ! -path '*/node_modules/*' 2>/dev/null | wc -l | tr -d ' ')
if [ "$native" != 0 ]; then
    printf '### Gap-native\n\nThis checkout also has %s skill(s) under `skills/`.\n\n' "$native" >> "$root_section"
fi

awk -v section="$root_section" '
    /<!-- skills-start -->/ { print; print ""; while ((getline line < section) > 0) print line; skip=1; next }
    /<!-- skills-end -->/ { skip=0 }
    !skip { print }
' "$README" > "$README.tmp" && mv "$README.tmp" "$README"

echo "✓ Updated README.md catalog ($group_count groups, $total Do skills at $PIN)"
