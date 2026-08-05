set shell := ["bash", "-cu"]

stow_dir := justfile_directory()
stow_target := env("HOME")
stow_cmd := "stow -d " + stow_dir + " -t " + stow_target
pkgs := "claude"
no_folding_pkgs := "omp pi jcode"

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
    if [ -z "$packages" ]; then packages="{{ pkgs }} {{ no_folding_pkgs }}"; fi
    total=0
    for pkg in $packages; do
        flags=""
        case " {{ no_folding_pkgs }} " in *" $pkg "*) flags="--no-folding";; esac
        # Pruning a broken directory link can expose more beneath it, so repeat
        # until a pass finds nothing.
        for _ in 1 2 3; do
            removed=0
            while IFS= read -r rel; do
                link="{{ stow_target }}/$rel"
                # Only ever remove a link that is BOTH stow-blocking and dangling.
                # Real files and valid links pointing elsewhere are left for stow
                # to report as genuine conflicts.
                if [ -L "$link" ] && [ ! -e "$link" ]; then
                    echo -e "{{ yellow }}  pruned{{ clr }} $link -> $(readlink "$link")"
                    if command -v trash >/dev/null 2>&1; then trash "$link"; else rm "$link"; fi
                    removed=$((removed + 1))
                fi
            done < <({{ stow_cmd }} $flags -n --restow "$pkg" 2>&1 \
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

# Restow all packages, reconcile shared agent instructions, and provision manifests
[group('stow')]
update: jcode-filter prune-broken && skills-install omp-plugins-install
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ pkgs }}; do {{ stow_cmd }} --restow $pkg; done
    for pkg in {{ no_folding_pkgs }}; do {{ stow_cmd }} --no-folding --restow $pkg; done
    python3 ./manage-agents.py sync
    echo -e "{{ green }}All packages restowed{{ clr }}"

alias up := update

# Restow the claude package only
[group('stow')]
claude: (prune-broken "claude")
    #!/usr/bin/env bash
    set -eu
    {{ stow_cmd }} --restow claude
    echo -e "{{ green }}claude restowed{{ clr }}"

# Restow the OMP package without folding and provision its marketplace plugins
[group('stow')]
omp: (prune-broken "omp") && omp-plugins-install
    #!/usr/bin/env bash
    set -eu
    {{ stow_cmd }} --no-folding --restow omp
    echo -e "{{ green }}omp restowed{{ clr }}"

# Restow the pi package only (no-folding)
[group('stow')]
pi: (prune-broken "pi")
    #!/usr/bin/env bash
    set -eu
    {{ stow_cmd }} --no-folding --restow pi
    echo -e "{{ green }}pi restowed{{ clr }}"

# Restow the Jcode package only (no-folding)
[group('stow')]
jcode: jcode-filter (prune-broken "jcode")
    #!/usr/bin/env bash
    set -eu
    {{ stow_cmd }} --no-folding --restow jcode
    echo -e "{{ green }}jcode restowed{{ clr }}"

# Create symlinks, reconcile shared agent instructions, and provision manifests
[group('stow')]
link: jcode-filter prune-broken && skills-install omp-plugins-install
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ pkgs }}; do {{ stow_cmd }} $pkg; done
    for pkg in {{ no_folding_pkgs }}; do {{ stow_cmd }} --no-folding $pkg; done
    python3 ./manage-agents.py sync
    echo -e "{{ green }}Packages linked: {{ pkgs }} {{ no_folding_pkgs }}{{ clr }}"

# Remove all symlinks
[group('stow')]
unlink:
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ pkgs }}; do {{ stow_cmd }} --delete $pkg; done
    for pkg in {{ no_folding_pkgs }}; do {{ stow_cmd }} --no-folding --delete $pkg; done
    python3 ./manage-agents.py unlink
    echo -e "{{ white }}All package symlinks removed{{ clr }}"

# Reconcile the shared AGENTS.md, harness entry points, and local RULES.md indexes
[group('agents')]
agents *args:
    @python3 ./manage-agents.py {{ args }}

# Verify every managed global, harness, and local-rules link
[group('agents')]
agents-check:
    @python3 ./manage-agents.py check

# Show the state of every managed symlink (ACTIVE / BROKEN / NOT ACTIVE)
[group('stow')]
status:
    #!/usr/bin/env bash
    set -eu
    g='{{ green }}'; r='{{ red }}'; w='{{ white }}'; c='{{ cyan }}'; z='{{ clr }}'
    report() {
        local pkg="$1"; shift
        local note=""; [ $# -gt 0 ] && note=" (no-folding)"
        echo -e "${c}Package: $pkg$note${z}"
        # Enumerate the real link nodes stow manages (LINK: lines from a restow
        # dry-run), then check each one's actual state on disk.
        {{ stow_cmd }} "$@" --restow -n -v "$pkg" 2>&1 \
            | grep '^LINK: ' \
            | while read -r line; do
                rest="${line#LINK: }"
                path="${rest%% => *}"
                target="$HOME/$path"
                if [ -L "$target" ]; then
                    if [ -e "$target" ]; then
                        printf "  ${g}ACTIVE${z}      %s\n" "$path"
                    else
                        printf "  ${r}BROKEN${z}      %s\n" "$path"
                    fi
                else
                    printf "  ${w}NOT ACTIVE${z}  %s\n" "$path"
                fi
            done
        echo ""
    }
    for pkg in {{ pkgs }}; do report "$pkg"; done
    for pkg in {{ no_folding_pkgs }}; do report "$pkg" --no-folding; done

# List declared skills (skills.txt) grouped by repo with install status
[group('skills')]
skills:
    #!/usr/bin/env bash
    set -eu
    manifest="{{ stow_dir }}/skills.txt"
    store="$HOME/.agents/skills"
    echo -e "{{ cyan }}Global (from skills.txt):{{ clr }}"
    echo ""
    while IFS= read -r line || [ -n "$line" ]; do
        line="${line%%#*}"
        line="$(printf '%s' "$line" | tr -d '[:space:]')"
        [ -z "$line" ] && continue
        source="${line%%=*}"
        names="${line#*=}"
        echo -e "{{ yellow }}$source{{ clr }}"
        if [ "$names" = "$line" ] || [ -z "$names" ] || [ "$names" = "*" ]; then
            echo "  (all skills)"
            echo ""
            continue
        fi
        IFS=',' read -ra arr <<< "$names"
        for n in "${arr[@]}"; do
            [ -z "$n" ] && continue
            if [ -e "$store/$n" ]; then
                echo -e "  {{ green }}✓{{ clr }} $n"
            else
                echo -e "  {{ red }}✗{{ clr }} $n {{ white }}(not installed){{ clr }}"
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
# Configure the repository-local clean filter for writable Jcode config
[group('stow')]
jcode-filter:
    #!/usr/bin/env bash
    set -eu
    git config filter.jcode-config.clean "python3 {{ stow_dir }}/filter-jcode-config.py"
    git config filter.jcode-config.smudge cat
    git config filter.jcode-config.required true
