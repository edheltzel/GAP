set shell := ["bash", "-cu"]

stow_dir := justfile_directory()
export ATLAS_STOW_DIR := stow_dir
no_folding_pkgs := "claude omp pi jcode"

yellow := '\033[33m'
green := '\033[32m'
white := '\033[37m'
cyan := '\033[36m'
red := '\033[31m'
clr := '\033[0m'

# List all available commands (runs when you type bare `just`)
default:
    @just --list

# Remove broken symlinks that stow refuses to replace (default: all packages)
[group('stow')]
prune-broken *packages:
    #!/usr/bin/env bash
    set -euo pipefail
    packages="{{ packages }}"
    if [ -z "$packages" ]; then packages="{{ no_folding_pkgs }}"; fi
    total=0
    for pkg in $packages; do
        flags=""
        case " {{ no_folding_pkgs }} " in *" $pkg "*) flags="--no-folding";; esac
        # Pruning a broken directory link can expose more beneath it, so repeat
        # until a pass finds nothing.
        while :; do
            removed=0
            while IFS= read -r rel; do
                link="$HOME/$rel"
                # Only ever remove a link that is BOTH stow-blocking and dangling.
                # Real files and valid links pointing elsewhere are left for stow
                # to report as genuine conflicts.
                if [ -L "$link" ] && [ ! -e "$link" ]; then
                    echo -e "{{ yellow }}  pruned{{ clr }} $link -> $(readlink "$link")"
                    if command -v trash >/dev/null 2>&1; then trash "$link"; else rm "$link"; fi
                    removed=$((removed + 1))
                fi
            done < <(stow -d "$ATLAS_STOW_DIR" -t "$HOME" $flags -n --restow "$pkg" 2>&1 \
                | sed -n 's/^ *\* existing target is not owned by stow: //p')
            total=$((total + removed))
            if [ "$removed" -eq 0 ]; then break; fi
        done
    done
    if [ "$total" -eq 0 ]; then
        echo -e "{{ green }}No broken links blocking stow{{ clr }}"
    else
        echo -e "{{ green }}Pruned $total broken link(s){{ clr }}"
    fi

# Remove dangling links hidden beneath ignored native Claude dependency paths
[group('stow')]
prune-native-dependency-links:
    @./prune-stale-native-dependency-links.sh

# Fail before mutation when any package has a genuine Stow ownership conflict
[group('stow')]
stow-preflight *packages:
    #!/usr/bin/env bash
    set -euo pipefail
    packages="{{ packages }}"
    if [ -z "$packages" ]; then packages="{{ no_folding_pkgs }}"; fi
    failed=0
    for pkg in $packages; do
        flags=""
        case " {{ no_folding_pkgs }} " in *" $pkg "*) flags="--no-folding";; esac
        if ! output="$(stow -d "$ATLAS_STOW_DIR" -t "$HOME" $flags -n --restow "$pkg" 2>&1)"; then
            echo -e "{{ red }}Stow preflight failed for $pkg{{ clr }}" >&2
            printf '%s\n' "$output" >&2
            failed=1
        fi
    done
    if [ "$failed" -ne 0 ]; then exit 1; fi
    echo -e "{{ green }}Stow preflight passed: $packages{{ clr }}"

# Restow all packages, then provision native tools, skills, and OMP plugins
[group('stow')]
update: jcode-filter prune-broken prune-native-dependency-links stow-preflight
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ no_folding_pkgs }}; do stow -d "$ATLAS_STOW_DIR" -t "$HOME" --no-folding --restow "$pkg"; done
    ./install-native-skill-tools.sh
    ./install-skills.sh
    ./install-omp-plugins.sh
    echo -e "{{ green }}All packages restowed and manifests provisioned{{ clr }}"

alias up := update

# Restow Claude and provision its repository-owned native tool dependencies
[group('stow')]
claude: (prune-broken "claude") prune-native-dependency-links (stow-preflight "claude")
    #!/usr/bin/env bash
    set -eu
    stow -d "$ATLAS_STOW_DIR" -t "$HOME" --no-folding --restow claude
    ./install-native-skill-tools.sh
    echo -e "{{ green }}claude restowed and native skill tools provisioned{{ clr }}"

# Restow the OMP package, then provision its marketplace plugins
[group('stow')]
omp: (prune-broken "omp") (stow-preflight "omp")
    #!/usr/bin/env bash
    set -eu
    stow -d "$ATLAS_STOW_DIR" -t "$HOME" --no-folding --restow omp
    ./install-omp-plugins.sh
    echo -e "{{ green }}omp restowed and plugins provisioned{{ clr }}"

# Restow the pi package only (no-folding)
[group('stow')]
pi: (prune-broken "pi") (stow-preflight "pi")
    #!/usr/bin/env bash
    set -eu
    stow -d "$ATLAS_STOW_DIR" -t "$HOME" --no-folding --restow pi
    echo -e "{{ green }}pi restowed{{ clr }}"

# Restow the Jcode package only (no-folding)
[group('stow')]
jcode: jcode-filter (prune-broken "jcode") (stow-preflight "jcode")
    #!/usr/bin/env bash
    set -eu
    stow -d "$ATLAS_STOW_DIR" -t "$HOME" --no-folding --restow jcode
    echo -e "{{ green }}jcode restowed{{ clr }}"

# Create package links, then provision native tools, skills, and OMP plugins
[group('stow')]
link: jcode-filter prune-broken prune-native-dependency-links stow-preflight
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ no_folding_pkgs }}; do stow -d "$ATLAS_STOW_DIR" -t "$HOME" --no-folding "$pkg"; done
    ./install-native-skill-tools.sh
    ./install-skills.sh
    ./install-omp-plugins.sh
    echo -e "{{ green }}Packages linked and manifests provisioned: {{ no_folding_pkgs }}{{ clr }}"

# Remove all symlinks
[group('stow')]
unlink:
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ no_folding_pkgs }}; do stow -d "$ATLAS_STOW_DIR" -t "$HOME" --no-folding --delete "$pkg"; done
    echo -e "{{ white }}All package symlinks removed{{ clr }}"

# Show every managed symlink and fail loudly on Stow ownership conflicts
[group('stow')]
status:
    #!/usr/bin/env bash
    set -euo pipefail
    g='{{ green }}'; r='{{ red }}'; w='{{ white }}'; c='{{ cyan }}'; z='{{ clr }}'
    failed=0
    report() {
        local pkg="$1"; shift
        local note="" output
        [ $# -gt 0 ] && note=" (no-folding)"
        echo -e "${c}Package: $pkg$note${z}"
        if ! output="$(stow -d "$ATLAS_STOW_DIR" -t "$HOME" "$@" --restow -n -v "$pkg" 2>&1)"; then
            printf "  ${r}CONFLICT${z}    Stow rejected package %s\n" "$pkg" >&2
            printf '%s\n' "$output" >&2
            failed=1
            echo ""
            return
        fi
        while IFS= read -r line; do
            rest="${line#LINK: }"
            path="${rest%% => *}"
            target="$HOME/$path"
            if [ -L "$target" ]; then
                if [ -e "$target" ]; then
                    printf "  ${g}ACTIVE${z}      %s\n" "$path"
                else
                    printf "  ${r}BROKEN${z}      %s\n" "$path"
                    failed=1
                fi
            else
                printf "  ${w}NOT ACTIVE${z}  %s\n" "$path"
            fi
        done < <(printf '%s\n' "$output" | grep '^LINK: ' || true)
        echo ""
    }
    for pkg in {{ no_folding_pkgs }}; do report $pkg --no-folding; done
    if [ "$failed" -ne 0 ]; then exit 1; fi

# Link the globally installed harness modules needed for local type checking
[group('checks')]
dev-modules:
    #!/usr/bin/env bash
    set -eu
    ./link-agent-dev-modules.sh

# Install locked dependencies for repository-owned native Claude skill tools
[group('skills')]
native-skill-tools:
    #!/usr/bin/env bash
    set -euo pipefail
    ./install-native-skill-tools.sh

# Validate native Claude skill tool locks and installed declared dependencies
[group('checks')]
native-skill-tools-status:
    #!/usr/bin/env bash
    set -euo pipefail
    ./install-native-skill-tools.sh --check

# Validate config, manifests, native skills, and status-line behavior
[group('checks')]
tooling-preflight: dev-modules native-skill-tools-status
    #!/usr/bin/env bash
    set -euo pipefail
    jq empty pi/.pi/agent/settings.json
    bun -e 'Bun.YAML.parse(await Bun.file("omp/.omp/agent/config.yml").text())'
    ./link-agent-dev-modules.sh --check
    ./node_modules/typescript/bin/tsc -p tsconfig.json
    ./install-skills.sh --dry-run >/dev/null
    ./install-omp-plugins.sh --dry-run >/dev/null
    tests/install-skills.test.sh
    tests/installed-skills-status.test.sh
    tests/ownership-status.test.sh
    tests/native-skill-tools.test.sh
    tests/claude/external-skills-inventory.test.sh
    tests/claude/native-skills-snapshot.test.sh
    tests/claude/native-orchestration-skills.test.sh
    tests/claude/native-residual-skills.test.sh
    tests/claude/native-runtime-skills.test.sh
    tests/claude/native-browser-skills.test.sh
    tests/claude/native-continuity-skills.test.sh
    tests/statusline-recovery.test.sh
    tests/statusline-runtime.test.sh
    tests/stow-safety.test.sh
    git diff --check

# Validate installed manifest/native skills and managed skill symlinks
[group('checks')]
installed-skills-status:
    #!/usr/bin/env bash
    set -euo pipefail
    ./check-installed-skills.sh

# Require the external Firstmate and app-owned tldraw skill stores
[group('checks')]
ownership-status:
    #!/usr/bin/env bash
    set -euo pipefail
    ./check-owned-skills.sh --strict

# Run Stow, tooling, installed-skill, and external-ownership checks
[group('checks')]
preflight: status tooling-preflight installed-skills-status ownership-status

# List declared skills (skills.txt) grouped by repo with install status
[group('skills')]
skills:
    #!/usr/bin/env bash
    set -eu
    manifest="$ATLAS_STOW_DIR/skills.txt"
    store="$HOME/.agents/skills"
    echo -e "{{ cyan }}Global (from skills.txt):{{ clr }}"
    echo ""
    while IFS= read -r line || [ -n "$line" ]; do
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        if [[ "$line" =~ ^(.*[^[:space:]])[[:space:]]+#.*$ ]]; then line="${BASH_REMATCH[1]}"; fi
        line="$(printf '%s' "$line" | tr -d '[:space:]')"
        [ -z "$line" ] && continue
        source="${line%%=*}"
        descriptor="${line#*=}"
        names="${descriptor%%|*}"
        entry_agents=""
        IFS='|' read -ra descriptor_parts <<< "$descriptor"
        last_part="${descriptor_parts[${#descriptor_parts[@]}-1]}"
        if [[ "$last_part" == agents=* ]]; then entry_agents="${last_part#agents=}"; fi
        echo -e "{{ yellow }}$source{{ clr }}"
        if [ "$names" = "$line" ] || [ -z "$names" ] || [ "$names" = "*" ]; then
            echo "  (all skills)"
            echo ""
            continue
        fi
        IFS=',' read -ra arr <<< "$names"
        for n in "${arr[@]}"; do
            [ -z "$n" ] && continue
            installed=1
            target_note=""
            if [ -n "$entry_agents" ]; then
                target_note=" [$entry_agents]"
                IFS=',' read -ra targets <<< "$entry_agents"
                for target in "${targets[@]}"; do
                    case "$target" in
                        claude-code) path="$HOME/.claude/skills/$n" ;;
                        pi) path="$HOME/.pi/agent/skills/$n" ;;
                        codex) path="$HOME/.codex/skills/$n" ;;
                        *) installed=0; continue ;;
                    esac
                    [ -e "$path" ] || installed=0
                done
            else
                [ -e "$store/$n" ] || installed=0
            fi
            if [ "$installed" -eq 1 ]; then
                echo -e "  {{ green }}✓{{ clr }} $n$target_note"
            else
                echo -e "  {{ red }}✗{{ clr }} $n$target_note {{ white }}(not installed){{ clr }}"
            fi
        done
        echo ""
    done < "$manifest"
    echo -e "{{ cyan }}Project:{{ clr }}"
    proj="$(skills list -p 2>/dev/null || true)"
    if printf '%s' "$proj" | grep -qi 'no project skills'; then
        echo "  (none)"
    else
        printf '%s\n' "$proj"
    fi

# Install skills listed in skills.txt using skills.sh
[group('skills')]
skills-install *args:
    #!/usr/bin/env bash
    set -eu
    ./install-skills.sh {{ args }}

# Dry-run skills installation commands from skills.txt
[group('skills')]
skills-status:
    #!/usr/bin/env bash
    set -eu
    ./install-skills.sh --dry-run

# Install user-scoped OMP marketplace plugins from omp-plugins.txt
[group('plugins')]
omp-plugins-install *args:
    #!/usr/bin/env bash
    set -eu
    ./install-omp-plugins.sh {{ args }}

# Dry-run OMP marketplace plugin provisioning
[group('plugins')]
omp-plugins-status:
    #!/usr/bin/env bash
    set -eu
    ./install-omp-plugins.sh --dry-run

# Commit and push config changes
[group('git')]
push:
    #!/usr/bin/env bash
    set -eu
    git add -A
    if git diff --cached --quiet; then
        echo -e "{{ yellow }}Nothing to commit{{ clr }}"
    else
        git commit -m "Update config"
        git push origin master
        echo -e "{{ green }}Pushed to origin{{ clr }}"
    fi

# Configure repository-local drivers for writable and machine-local config
[group('stow')]
jcode-filter:
    #!/usr/bin/env bash
    set -eu
    git config filter.jcode-config.clean "python3 $ATLAS_STOW_DIR/filter-jcode-config.py"
    git config filter.jcode-config.smudge cat
    git config filter.jcode-config.required true
    git config merge.ours.driver true
