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

# Restow all packages and provision skills from skills.txt
[group('stow')]
update: && skills-install
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ pkgs }}; do {{ stow_cmd }} --restow $pkg; done
    {{ stow_cmd }} --no-folding --restow {{ pi_pkg }}
    echo -e "{{ green }}All packages restowed{{ clr }}"

alias up := update

# Restow the claude package only
[group('stow')]
claude:
    #!/usr/bin/env bash
    set -eu
    {{ stow_cmd }} --restow claude
    echo -e "{{ green }}claude restowed{{ clr }}"

# Restow the omp package only
[group('stow')]
omp:
    #!/usr/bin/env bash
    set -eu
    {{ stow_cmd }} --restow omp
    echo -e "{{ green }}omp restowed{{ clr }}"

# Restow the pi package only (no-folding)
[group('stow')]
pi:
    #!/usr/bin/env bash
    set -eu
    {{ stow_cmd }} --no-folding --restow {{ pi_pkg }}
    echo -e "{{ green }}pi restowed{{ clr }}"

# Create symlinks and provision skills from skills.txt
[group('stow')]
link: && skills-install
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

# Show what stow would do (dry run)
[group('stow')]
status:
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ pkgs }}; do
        echo -e "{{ cyan }}Package: $pkg{{ clr }}"
        {{ stow_cmd }} --restow -n -v $pkg 2>&1 || true
        echo ""
    done
    echo -e "{{ cyan }}Package: {{ pi_pkg }} (no-folding){{ clr }}"
    {{ stow_cmd }} --no-folding --restow -n -v {{ pi_pkg }} 2>&1 || true
    echo ""

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
