set shell := ["bash", "-cu"]

stow_dir := justfile_directory()
stow_target := env("HOME")
stow_cmd := "stow -d " + stow_dir + " -t " + stow_target
pkgs := "claude codex omp"
pi_pkg := "pi"

yellow := '\033[33m'
green := '\033[32m'
white := '\033[37m'
cyan := '\033[36m'
clr := '\033[0m'

# Restow all packages (use after adding/removing files)
[group('stow')]
update: && skills-link
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ pkgs }}; do {{ stow_cmd }} --restow $pkg; done
    {{ stow_cmd }} --no-folding --restow {{ pi_pkg }}
    echo -e "{{ green }}All packages restowed{{ clr }}"

alias up := update

# Create symlinks (first time setup)
[group('stow')]
link: && skills-link
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

# Link ~/.agents/skills to the repo-backed skills.sh store (idempotent)
[group('skills')]
skills-link:
    #!/usr/bin/env bash
    set -eu
    store="{{ stow_dir }}/skills"
    target="$HOME/.agents/skills"
    if [ -L "$target" ]; then
        echo -e "{{ green }}Already linked: $target → $(readlink "$target"){{ clr }}"
    elif [ -e "$target" ]; then
        echo -e "{{ yellow }}$target exists and is not a symlink — move its contents into $store first{{ clr }}"
        exit 1
    else
        ln -s "$store" "$target"
        echo -e "{{ green }}Linked $target → $store{{ clr }}"
    fi

# Review and commit changes to the skills store
[group('skills')]
skills-sync:
    #!/usr/bin/env bash
    set -eu
    if [ ! -L "$HOME/.agents/skills" ]; then
        echo -e "{{ yellow }}~/.agents/skills is not a symlink — store has diverged from the repo. Run 'just skills-link'{{ clr }}"
        exit 1
    fi
    cd "{{ stow_dir }}"
    if git status --porcelain skills/ | grep -q .; then
        git status --short skills/
        git add skills/
        git commit -m "skills: sync store"
        echo -e "{{ green }}Skills store synced{{ clr }}"
    else
        echo -e "{{ yellow }}Nothing to sync{{ clr }}"
    fi

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
