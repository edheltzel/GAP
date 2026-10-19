#!/usr/bin/env bash
# Skill catalog generator. Regenerates scope and group READMEs plus the
# catalog tables in the root README.md from SKILL.md frontmatters.
# Skills live under skills/<scope>/<group>/<skill>/SKILL.md, scope = global|project.
# Bundled copies under references/ are ignored.
# Usage: bash scripts/update-readme.sh
set -euo pipefail

ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
README="$ROOT/README.md"
SKILLS_DIR="$ROOT/skills"
GROUPINGS_FILE="$ROOT/skills.sh.json"
# skills.sh accepts owner/repo and a skills/<scope>/<group> subpath. A walk of
# skills/project (no SKILL.md at the top) also installs SKILL.md files bundled
# under references/. Catalogs install with gap sync, or one skill with -s.
REPO="edheltzel/agent-config"

# Render order. Group blurbs come from skills.sh.json ("<Scope>: <Group>"),
# their authoritative owner. Unlisted groups on disk are appended.
SCOPE_ORDER="global project"
GROUP_ORDER_global="core workflow operations content personal"
GROUP_ORDER_project="typescript frontend backend tooling swift product marketing personal"
# Groups without docs pages.
UNPROMOTED="global/operations global/personal project/personal project/marketing"

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

group_blurb() {
    node -e '
const fs = require("fs");
const [path, title] = process.argv.slice(1);
const config = JSON.parse(fs.readFileSync(path, "utf8"));
const group = config.groupings.find((candidate) => candidate.title === title);
process.stdout.write(group?.description ?? "");
' "$GROUPINGS_FILE" "$(scope_title "$1"): $(group_title "$2")"
}

is_promoted() {
    ! echo "$UNPROMOTED" | grep -qw "$1/$2"
}

write_install() {
    printf 'These skills live in this repo (`%s`, being renamed GAP). `gap sync` installs every skill listed in `global/skills.json`.\n\n' "$REPO"
    printf '```bash\ngap sync\n```\n\n'
    printf 'One skill by exact name:\n\n'
    printf '```bash\nnpx skills add %s -s <skill-name>\n```\n\n' "$REPO"
}
# Ordered groups for a scope: preferred order, then any others on disk.
ordered_groups() {
    scope="$1"
    found=$(find "$SKILLS_DIR/$scope" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort)
    preferred_var="GROUP_ORDER_$scope"
    preferred="${!preferred_var:-}"
    for g in $preferred; do
        echo "$found" | grep -qx "$g" && printf '%s ' "$g"
    done
    for g in $found; do
        echo "$preferred" | grep -qw "$g" || printf '%s ' "$g"
    done
}

group_file=$(mktemp)
root_section=$(mktemp)
trap 'rm -f "$group_file" "$root_section"' EXIT

# Parse name and description from a SKILL.md file.
# Handles inline values (quoted or unquoted) and block scalars (> and |).
# Outputs: <name><TAB><description>
parse_skill() {
    awk '
    BEGIN { cnt=0; name=""; desc=""; in_desc=0 }
    /^---[[:space:]]*$/ {
        cnt++
        if (cnt == 2) exit
        next
    }
    cnt != 1 { next }
    /^description:[[:space:]]*[>|]/ { in_desc=1; next }
    /^description:[[:space:]]/ {
        in_desc=0
        val=$0; sub(/^description:[[:space:]]*/, "", val)
        gsub(/^[[:space:]"]+|[[:space:]"]+$/, "", val)
        desc=val
        next
    }
    /^name:[[:space:]]/ {
        in_desc=0
        val=$0; sub(/^name:[[:space:]]*/, "", val)
        gsub(/^[[:space:]"]+|[[:space:]"]+$/, "", val)
        name=val
        next
    }
    /^[a-zA-Z]/ { in_desc=0; next }
    in_desc && /^[[:space:]]/ {
        line=$0; sub(/^[[:space:]]+/, "", line)
        desc=(desc=="" ? line : desc " " line)
        next
    }
    END { printf "%s\t%s\n", name, desc }
    ' "$1"
}

# Return the first sentence of a string. A sentence ends at a . ! or ? that is
# followed by whitespace or end of line; a period inside a token (AGENTS.md)
# is not a boundary.
first_sentence() {
    awk '{
        if (match($0, /[.!?]([[:space:]]|$)/))
            print substr($0, 1, RSTART)
        else
            print $0
        exit
    }' <<< "$1"
}

escape_cell() {
    printf '%s' "$1" | sed 's/|/\\|/g'
}

# Fill $group_file with name<TAB>dir<TAB>short-desc for SKILL.md files under $1.
collect_skills() {
    : > "$group_file"
    while IFS= read -r skill_file; do
        [ -n "$skill_file" ] || continue
        result=$(parse_skill "$skill_file")
        name=$(printf '%s' "$result" | cut -f1)
        desc=$(printf '%s' "$result" | cut -f2-)
        [ -n "$name" ] && [ -n "$desc" ] || continue
        dir=$(basename "$(dirname "$skill_file")")
        printf '%s\t%s\t%s\n' "$name" "$dir" "$(first_sentence "$desc")" >> "$group_file"
    done <<EOF
$(find "$1" -name SKILL.md ! -path '*/references/*' ! -path '*/node_modules/*' 2>/dev/null | sort)
EOF
    [ -s "$group_file" ] || return 1
    sort -f -o "$group_file" "$group_file"
}

write_group_readme() {
    scope="$1"; group="$2"
    {
        printf '# %s\n\n' "$(group_title "$group")"
        blurb=$(group_blurb "$scope" "$group")
        [ -n "$blurb" ] && printf '%s\n\n' "$blurb"
        printf '## Installation\n\n'
        write_install
        printf '## Skills\n\n| Skill | Description |\n|-------|-------------|\n'
        while IFS=$'\t' read -r name dir desc; do
            printf '| [`%s`](./%s/) | %s |\n' "$name" "$dir" "$(escape_cell "$desc")"
        done < "$group_file"
        printf '\n## See Also\n\n'
        printf -- '- [%s skills](../README.md)\n' "$(scope_title "$scope")"
        printf -- '- [Skill catalog](../../../README.md)\n'
        if is_promoted "$scope" "$group"; then
            printf -- '- [Docs](../../../docs/%s/%s/) - human-facing pages for these skills\n' "$scope" "$group"
        fi
    } > "$SKILLS_DIR/$scope/$group/README.md"
}

total=0
group_count=0
: > "$root_section"
for scope in $SCOPE_ORDER; do
    [ -d "$SKILLS_DIR/$scope" ] || continue
    scope_rows=$(mktemp)
    for group in $(ordered_groups "$scope"); do
        collect_skills "$SKILLS_DIR/$scope/$group" || continue
        count=$(wc -l < "$group_file" | tr -d ' ')
        total=$((total + count))
        group_count=$((group_count + 1))
        write_group_readme "$scope" "$group"
        printf '| [%s](./%s/) | %s | %s |\n' "$(group_title "$group")" "$group" "$count" "$(escape_cell "$(group_blurb "$scope" "$group")")" >> "$scope_rows"
    done
    {
        printf '# %s Skills\n\n%s\n\n' "$(scope_title "$scope")" "$(scope_blurb "$scope")"
        printf '## Installation\n\n'
        write_install
        printf '## Groups\n\n| Group | Skills | Coverage |\n|-------|--------|----------|\n'
        cat "$scope_rows"
        printf '\n## See Also\n\n- [Skill catalog](../../README.md)\n'
    } > "$SKILLS_DIR/$scope/README.md"
    {
        printf '### [%s](./skills/%s/)\n\n%s\n\n' "$(scope_title "$scope")" "$scope" "$(scope_blurb "$scope")"
        printf '| Group | Skills | Coverage |\n|-------|--------|----------|\n'
        sed "s#](\./#](./skills/$scope/#" "$scope_rows"
        printf '\n'
    } >> "$root_section"
    rm -f "$scope_rows"
done

awk -v section="$root_section" '
    /<!-- skills-start -->/ { print; print ""; while ((getline line < section) > 0) print line; skip=1; next }
    /<!-- skills-end -->/ { skip=0 }
    !skip { print }
' "$README" > "$README.tmp" && mv "$README.tmp" "$README"

echo "✓ Updated README.md and $group_count group READMEs ($total skills)"
