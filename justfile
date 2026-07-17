set shell := ["bash", "-cu"]

stow_dir := justfile_directory()
stow_target := env("HOME")
stow_cmd := "stow -d " + stow_dir + " -t " + stow_target
pkgs := "claude omp"
pi_pkg := "pi"

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
    if [ -z "$packages" ]; then packages="{{ pkgs }} {{ pi_pkg }}"; fi
    total=0
    for pkg in $packages; do
        flags=""
        if [ "$pkg" = "{{ pi_pkg }}" ]; then flags="--no-folding"; fi
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

# Restow all packages and provision skills plus OMP marketplace plugins
[group('stow')]
update: prune-broken && skills-install omp-plugins-install
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ pkgs }}; do {{ stow_cmd }} --restow $pkg; done
    {{ stow_cmd }} --no-folding --restow {{ pi_pkg }}
    echo -e "{{ green }}All packages restowed{{ clr }}"

alias up := update

# Restow the claude package only
[group('stow')]
claude: (prune-broken "claude")
    #!/usr/bin/env bash
    set -eu
    {{ stow_cmd }} --restow claude
    echo -e "{{ green }}claude restowed{{ clr }}"

# Restow the OMP package and provision its marketplace plugins
[group('stow')]
omp: (prune-broken "omp") && omp-plugins-install
    #!/usr/bin/env bash
    set -eu
    {{ stow_cmd }} --restow omp
    echo -e "{{ green }}omp restowed{{ clr }}"

# Restow the pi package only (no-folding)
[group('stow')]
pi: (prune-broken pi_pkg)
    #!/usr/bin/env bash
    set -eu
    {{ stow_cmd }} --no-folding --restow {{ pi_pkg }}
    echo -e "{{ green }}pi restowed{{ clr }}"

# Create symlinks and provision skills plus OMP marketplace plugins
[group('stow')]
link: prune-broken && skills-install omp-plugins-install
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ pkgs }}; do {{ stow_cmd }} $pkg; done
    {{ stow_cmd }} --no-folding {{ pi_pkg }}
    echo -e "{{ green }}Packages linked: {{ pkgs }} {{ pi_pkg }}{{ clr }}"

# Remove all symlinks
[group('stow')]
unlink:
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ pkgs }}; do {{ stow_cmd }} --delete $pkg; done
    {{ stow_cmd }} --no-folding --delete {{ pi_pkg }}
    echo -e "{{ white }}All package symlinks removed{{ clr }}"

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
    report {{ pi_pkg }} --no-folding

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
