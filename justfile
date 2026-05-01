set shell := ["bash", "-cu"]

stow_dir := justfile_directory()
stow_target := env("HOME")
stow_cmd := "stow -d " + stow_dir + " -t " + stow_target
pkgs := "claudecode pi"

yellow := '\033[33m'
green := '\033[32m'
white := '\033[37m'
cyan := '\033[36m'
clr := '\033[0m'

# Restow all packages (use after adding/removing files)
[group('stow')]
update:
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ pkgs }}; do {{ stow_cmd }} --restow $pkg; done
    echo -e "{{ green }}All packages restowed{{ clr }}"

alias up := update

# Create symlinks (first time setup)
[group('stow')]
link:
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ pkgs }}; do {{ stow_cmd }} $pkg; done
    echo -e "{{ green }}Packages linked: {{ pkgs }}{{ clr }}"

# Remove all symlinks
[group('stow')]
unlink:
    #!/usr/bin/env bash
    set -eu
    for pkg in {{ pkgs }}; do {{ stow_cmd }} --delete $pkg; done
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
