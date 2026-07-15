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

# Restow all packages and provision skills from skills.txt
[group('stow')]
update: prune-broken && skills-install
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

# Restow the omp package only
[group('stow')]
omp: (prune-broken "omp")
    #!/usr/bin/env bash
    set -eu
    {{ stow_cmd }} --restow omp
    echo -e "{{ green }}omp restowed{{ clr }}"

# Verify the local patched OMP branch and active source link
[group('omp-source')]
omp-patched-status:
    #!/usr/bin/env bash
    set -euo pipefail
    source_dir="${OMP_SOURCE_DIR:-$HOME/Developer/AI-Agents/omp}"
    worktree="${OMP_PATCH_WORKTREE:-$source_dir/.agents/worktrees/atlas-fixes}"
    expected_wrapper="$worktree/packages/coding-agent/scripts/omp"
    if [ ! -d "$worktree" ]; then
        echo -e "{{ red }}Patched OMP worktree not found: $worktree{{ clr }}" >&2
        exit 1
    fi
    branch="$(git -C "$worktree" branch --show-current)"
    if [ "$branch" != "atlas/fixes" ]; then
        echo -e "{{ red }}Expected atlas/fixes, found ${branch:-detached}{{ clr }}" >&2
        exit 1
    fi
    version="$(git -C "$worktree" describe --tags --abbrev=0)"
    head="$(git -C "$worktree" rev-parse --short HEAD)"
    executable="$(command -v omp || true)"
    resolved_executable=""
    resolved_expected=""
    if [ -n "$executable" ]; then
        resolved_executable="$(realpath "$executable")"
    fi
    if [ -e "$expected_wrapper" ]; then
        resolved_expected="$(realpath "$expected_wrapper")"
    fi
    echo -e "{{ cyan }}Patched branch:{{ clr }} $branch ($version + $head)"
    echo -e "{{ cyan }}Worktree:{{ clr }}      $worktree"
    echo -e "{{ cyan }}Active omp:{{ clr }}    ${executable:-not found}"
    echo -e "{{ cyan }}Resolved:{{ clr }}      ${resolved_executable:-not found}"
    if [ -n "$executable" ]; then
        echo -e "{{ cyan }}Version:{{ clr }}       $(omp --version)"
    fi
    link_active=true
    if [ "$resolved_executable" = "$resolved_expected" ] && [ -n "$resolved_expected" ]; then
        echo -e "{{ green }}Source link active{{ clr }}"
    else
        echo -e "{{ red }}Source link inactive; expected $resolved_expected{{ clr }}" >&2
        link_active=false
    fi
    if [ -n "$(git -C "$worktree" status --porcelain)" ]; then
        echo -e "{{ yellow }}Worktree has local changes{{ clr }}"
    else
        echo -e "{{ green }}Worktree clean{{ clr }}"
    fi
    if [ "$link_active" = false ]; then
        exit 1
    fi

# Stage the newest stable OMP release, verify it, then atomically promote both links
[group('omp-source')]
omp-patched-update:
    #!/usr/bin/env bash
    set -euo pipefail
    source_dir="${OMP_SOURCE_DIR:-$HOME/Developer/AI-Agents/omp}"
    worktree="${OMP_PATCH_WORKTREE:-$source_dir/.agents/worktrees/atlas-fixes}"
    upstream_remote="${OMP_UPSTREAM_REMOTE:-origin}"
    branch="atlas/fixes"
    upstream_namespace="refs/atlas-omp-upstream/tags"
    if [ ! -d "$source_dir" ] || [ ! -d "$worktree" ]; then
        echo -e "{{ red }}OMP source checkout or patched worktree not found{{ clr }}" >&2
        exit 1
    fi
    if [ ! -L "$worktree" ]; then
        echo -e "{{ red }}Canonical patched worktree must be a symlink: $worktree{{ clr }}" >&2
        exit 1
    fi
    active_worktree="$(realpath "$worktree")"
    old_wrapper="$active_worktree/packages/coding-agent/scripts/omp"
    executable_link="$(command -v omp || true)"
    if [ -z "$executable_link" ] || [ ! -e "$old_wrapper" ] || [ "$(realpath "$executable_link")" != "$(realpath "$old_wrapper")" ]; then
        echo -e "{{ red }}Patched OMP source link is not active; run just omp-patched-status{{ clr }}" >&2
        exit 1
    fi
    source_common="$(git -C "$source_dir" rev-parse --path-format=absolute --git-common-dir)"
    worktree_common="$(git -C "$active_worktree" rev-parse --path-format=absolute --git-common-dir)"
    if [ "$source_common" != "$worktree_common" ]; then
        echo -e "{{ red }}OMP source and patched worktree belong to different repositories{{ clr }}" >&2
        exit 1
    fi
    active_branch="$(git -C "$active_worktree" branch --show-current)"
    if [ "$active_branch" != "$branch" ]; then
        echo -e "{{ red }}Expected $branch, found ${active_branch:-detached}{{ clr }}" >&2
        exit 1
    fi
    if [ -n "$(git -C "$active_worktree" status --porcelain)" ]; then
        echo -e "{{ red }}Patched OMP worktree must be clean before updating{{ clr }}" >&2
        git -C "$active_worktree" status --short
        exit 1
    fi
    git -C "$source_dir" fetch --prune "$upstream_remote" "+refs/tags/*:${upstream_namespace}/*"
    latest_tag=""
    while IFS= read -r tag; do
        if [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            latest_tag="$tag"
            break
        fi
    done < <(git -C "$source_dir" for-each-ref --sort=-version:refname --format='%(refname:strip=3)' "$upstream_namespace")
    if [ -z "$latest_tag" ]; then
        echo -e "{{ red }}No stable OMP release tag found on $upstream_remote{{ clr }}" >&2
        exit 1
    fi
    upstream_ref="$upstream_namespace/$latest_tag"
    old_head="$(git -C "$active_worktree" rev-parse HEAD)"
    candidate="$source_dir/.agents/worktrees/.atlas-fixes-candidate-$$"
    candidate_branch="atlas/fixes-candidate-$$"
    candidate_wrapper="$candidate/packages/coding-agent/scripts/omp"
    canonical_link_tmp="${worktree}.next.$$"
    executable_link_tmp="${executable_link}.next.$$"
    old_worktree_detached=false
    candidate_branch_promoted=false
    canonical_link_swapped=false
    executable_link_swapped=false
    promotion_committed=false
    rollback_uncommitted_promotion() {
        local exit_status=$?
        local rollback_ok=true
        local rollback_link_tmp
        trap - EXIT
        set +e
        if [ "$promotion_committed" = false ]; then
            if [ -d "$candidate" ]; then
                git -C "$candidate" rebase --abort >/dev/null 2>&1
            fi
            if [ "$executable_link_swapped" = true ]; then
                rollback_link_tmp="${executable_link}.rollback.$$"
                if ! ln -s "$old_wrapper" "$rollback_link_tmp" || ! mv -h -f "$rollback_link_tmp" "$executable_link"; then
                    echo -e "{{ red }}Failed to restore the previous OMP executable link{{ clr }}" >&2
                    rollback_ok=false
                fi
            fi
            if [ "$canonical_link_swapped" = true ]; then
                rollback_link_tmp="${worktree}.rollback.$$"
                if ! ln -s "$active_worktree" "$rollback_link_tmp" || ! mv -h -f "$rollback_link_tmp" "$worktree"; then
                    echo -e "{{ red }}Failed to restore the previous canonical worktree link{{ clr }}" >&2
                    rollback_ok=false
                fi
            fi
            if [ "$candidate_branch_promoted" = true ]; then
                if ! git -C "$candidate" switch --detach >/dev/null 2>&1; then
                    echo -e "{{ red }}Failed to detach the candidate during rollback{{ clr }}" >&2
                    rollback_ok=false
                fi
                if ! git -C "$source_dir" branch -f "$branch" "$old_head" >/dev/null 2>&1; then
                    echo -e "{{ red }}Failed to restore $branch to $old_head{{ clr }}" >&2
                    rollback_ok=false
                fi
            fi
            if [ "$old_worktree_detached" = true ] && ! git -C "$active_worktree" switch "$branch" >/dev/null 2>&1; then
                echo -e "{{ red }}Failed to restore $branch in the previous worktree{{ clr }}" >&2
                rollback_ok=false
            fi
            if [ -d "$candidate" ] && ! git -C "$source_dir" worktree remove --force "$candidate" >/dev/null 2>&1; then
                echo -e "{{ red }}Failed to remove candidate worktree $candidate{{ clr }}" >&2
                rollback_ok=false
            fi
            if git -C "$source_dir" show-ref --verify --quiet "refs/heads/$candidate_branch" && ! git -C "$source_dir" branch -D "$candidate_branch" >/dev/null 2>&1; then
                echo -e "{{ red }}Failed to remove candidate branch $candidate_branch{{ clr }}" >&2
                rollback_ok=false
            fi
            for pending_link in "$canonical_link_tmp" "$executable_link_tmp"; do
                if [ -L "$pending_link" ] && ! unlink "$pending_link"; then
                    echo -e "{{ red }}Failed to remove temporary link $pending_link{{ clr }}" >&2
                    rollback_ok=false
                fi
            done
            if [ "$(realpath "$executable_link" 2>/dev/null)" != "$(realpath "$old_wrapper" 2>/dev/null)" ]; then
                echo -e "{{ red }}Rollback left the OMP executable link inconsistent{{ clr }}" >&2
                rollback_ok=false
            fi
            if [ "$(realpath "$worktree" 2>/dev/null)" != "$active_worktree" ]; then
                echo -e "{{ red }}Rollback left the canonical worktree link inconsistent{{ clr }}" >&2
                rollback_ok=false
            fi
        fi
        if [ "$rollback_ok" = false ]; then
            exit 1
        fi
        exit "$exit_status"
    }
    trap rollback_uncommitted_promotion EXIT
    echo -e "{{ cyan }}Staging atlas/fixes on $latest_tag{{ clr }}"
    git -C "$source_dir" worktree add -b "$candidate_branch" "$candidate" "$branch"
    git -C "$candidate" rebase "$upstream_ref"
    (
        cd "$candidate"
        bun install --frozen-lockfile
        bun run build:native
        bun test packages/ai/test/openai-responses-stream-terminal.test.ts packages/ai/test/openai-codex-stream.test.ts --test-name-pattern "empty terminal content preserves streamed text"
        bun run ci:test:smoke
    )
    ln -s "$candidate" "$canonical_link_tmp"
    ln -s "$candidate_wrapper" "$executable_link_tmp"
    git -C "$active_worktree" switch --detach "$old_head"
    old_worktree_detached=true
    git -C "$candidate" branch -M "$branch"
    candidate_branch_promoted=true
    mv -h -f "$canonical_link_tmp" "$worktree"
    canonical_link_swapped=true
    mv -h -f "$executable_link_tmp" "$executable_link"
    executable_link_swapped=true
    resolved_executable="$(realpath "$executable_link")"
    resolved_expected="$(realpath "$worktree/packages/coding-agent/scripts/omp")"
    if [ "$resolved_executable" != "$resolved_expected" ] || [ "$resolved_expected" != "$(realpath "$candidate_wrapper")" ]; then
        echo -e "{{ red }}Patched OMP promotion verification failed{{ clr }}" >&2
        exit 1
    fi
    promotion_committed=true
    trap - EXIT
    if ! git -C "$source_dir" worktree remove "$active_worktree"; then
        echo -e "{{ yellow }}Patched OMP is active, but old worktree cleanup failed: $active_worktree{{ clr }}" >&2
    fi
    echo -e "{{ green }}Patched OMP installed:{{ clr }} $(omp --version)"

# Restow the pi package only (no-folding)
[group('stow')]
pi: (prune-broken pi_pkg)
    #!/usr/bin/env bash
    set -eu
    {{ stow_cmd }} --no-folding --restow {{ pi_pkg }}
    echo -e "{{ green }}pi restowed{{ clr }}"

# Create symlinks and provision skills from skills.txt
[group('stow')]
link: prune-broken && skills-install
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
