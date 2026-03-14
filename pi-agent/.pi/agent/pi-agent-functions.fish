#!/usr/bin/env fish
# Pi Agent Evolution Functions for Fish Shell

set -g PI_AGENT_DIR "$HOME/.pi/agent"

# ============================================
# Profile Management
# ============================================

function pi-profile
    set -l profile $argv[1]
    set -l profile_path "$PI_AGENT_DIR/profiles/$profile.json"
    
    if not test -f "$profile_path"
        echo "✗ Profile '$profile' not found"
        echo "Available profiles:"
        ls -1 "$PI_AGENT_DIR/profiles/" | sed 's/\.json$//' | sed 's/^/  - /'
        return 1
    end
    
    # Update settings.json
    set -l settings_path "$PI_AGENT_DIR/settings.json"
    
    if not test -f "$settings_path"
        echo "{\"profile\": \"$profile\"}" > "$settings_path"
    else if command -sq jq
        jq --arg profile "$profile" '.profile = $profile' "$settings_path" > /tmp/pi-settings.json
        mv /tmp/pi-settings.json "$settings_path"
    else
        echo "{\"profile\": \"$profile\"}" > "$settings_path"
    end
    
    echo "✓ Switched to profile: $profile"
    echo "⚠️  Restart Pi to apply changes"
    
    # Log the switch
    echo (date '+%Y-%m-%d %H:%M:%S') "- Switched to $profile" >> "$PI_AGENT_DIR/profile-history.log"
end

# Quick aliases
function pi-core
    pi-profile core
end

function pi-code
    pi-profile coding
end

function pi-research
    pi-profile research
end

# ============================================
# Usage Analytics
# ============================================

function pi-usage
    set -l stats_file "$PI_AGENT_DIR/usage-stats.json"
    
    if not test -f "$stats_file"
        echo "No usage data yet. Start using Pi to collect stats."
        return 0
    end
    
    echo "=== Pi Component Usage Report ==="
    echo ""
    
    if command -sq jq
        echo "Top Skills:"
        jq -r '.skills | to_entries | sort_by(.value) | reverse | .[0:10] | .[] | "  \(.value)x \(.key)"' "$stats_file" 2>/dev/null; or echo "  No skill data"
        
        echo ""
        echo "Top Extensions:"
        jq -r '.extensions | to_entries | sort_by(.value) | reverse | .[0:10] | .[] | "  \(.value)x \(.key)"' "$stats_file" 2>/dev/null; or echo "  No extension data"
        
        echo ""
        set -l sessions (jq -r '.sessions // 0' "$stats_file")
        echo "Sessions completed: $sessions"
        
        echo ""
        echo "🔥 Promotion Candidates (used ≥ 10 times):"
        jq -r '(.skills // {}) + (.extensions // {}) | to_entries | map(select(.value >= 10)) | sort_by(.value) | reverse | .[] | "  ✓ \(.value)x \(.key)"' "$stats_file" 2>/dev/null; or echo "  None yet - keep using those components!"
    else
        echo "Install jq for better formatting. Raw stats:"
        cat "$stats_file"
    end
end

# ============================================
# Promotion Workflow
# ============================================

function pi-promote
    set -l component $argv[1]
    set -l from_profile $argv[2]
    
    if test -z "$from_profile"
        set from_profile "coding"
    end
    
    set -l core_config "$PI_AGENT_DIR/profiles/core.json"
    set -l source_config "$PI_AGENT_DIR/profiles/$from_profile.json"
    
    if not test -f "$core_config"
        echo "✗ Core profile not found at $core_config"
        return 1
    end
    
    # Detect if skill or extension
    set -l is_skill false
    set -l is_extension false
    
    if command -sq jq
        if jq -e ".skills | index(\"$component\")" "$source_config" >/dev/null 2>&1
            set is_skill true
        end
        if jq -e ".extensions | index(\"$component\")" "$source_config" >/dev/null 2>&1
            set is_extension true
        end
    end
    
    # Check if already in core
    if test "$is_skill" = false; and test "$is_extension" = false
        if command -sq jq
            if jq -e ".skills | index(\"$component\")" "$core_config" >/dev/null 2>&1
                echo "ℹ️  '$component' is already in core agent"
                return 0
            end
            if jq -e ".extensions | index(\"$component\")" "$core_config" >/dev/null 2>&1
                echo "ℹ️  '$component' is already in core agent"
                return 0
            end
        end
        echo "✗ Component '$component' not found in profile '$from_profile'"
        return 1
    end
    
    # Promote to core
    if test "$is_skill" = true
        if command -sq jq
            set -l tmp_file "$core_config.tmp"
            jq ".skills += [\"$component\"] | .skills |= unique" "$core_config" > "$tmp_file"
            mv "$tmp_file" "$core_config"
        end
        echo "✓ Promoted skill '$component' to core agent"
        echo (date '+%Y-%m-%d %H:%M:%S') "- PROMOTED skill: $component (from $from_profile)" >> "$PI_AGENT_DIR/promotions.log"
    end
    
    if test "$is_extension" = true
        if command -sq jq
            set -l tmp_file "$core_config.tmp"
            jq ".extensions += [\"$component\"] | .extensions |= unique" "$core_config" > "$tmp_file"
            mv "$tmp_file" "$core_config"
        end
        echo "✓ Promoted extension '$component' to core agent"
        echo (date '+%Y-%m-%d %H:%M:%S') "- PROMOTED extension: $component (from $from_profile)" >> "$PI_AGENT_DIR/promotions.log"
    end
    
    echo ""
    echo "Core agent now includes:"
    if command -sq jq
        jq -r '.skills | .[]' "$core_config" | sed 's/^/  skill: /'
        jq -r '.extensions | .[]' "$core_config" | sed 's/^/  ext:   /'
    end
    
    echo ""
    echo "⚠️  Restart Pi to apply changes"
end

# ============================================
# Agent Selection
# ============================================

function pi-agent
    set -l agent $argv[1]
    set -l agent_file "$PI_AGENT_DIR/agents/$agent.md"
    
    # Check in subdirectories
    if not test -f "$agent_file"
        set agent_file (find "$PI_AGENT_DIR/agents" -name "$agent.md" | head -1)
    end
    
    if test -z "$agent_file"; or not test -f "$agent_file"
        echo "✗ Agent '$agent' not found"
        echo "Available agents:"
        find "$PI_AGENT_DIR/agents" -name "*.md" | while read f
            set -l name (basename "$f" .md)
            set -l dir (basename (dirname "$f"))
            echo "  - $name ($dir)"
        end
        return 1
    end
    
    echo "✓ Selected agent: $agent"
    set -l desc (head -3 "$agent_file" | grep "^description:" | sed 's/description: //')
    echo "📄 $desc"
    
    # Export for potential use by Pi
    set -gx PI_CURRENT_AGENT "$agent"
end

# ============================================
# Status & Diagnostics
# ============================================

function pi-status
    echo "=== Pi Agent Status ==="
    echo ""
    
    # Current profile
    set -l settings_path "$PI_AGENT_DIR/settings.json"
    if test -f "$settings_path"
        set -l current_profile (jq -r '.profile // "not set"' "$settings_path" 2>/dev/null)
        echo "Current Profile: $current_profile"
    else
        echo "Current Profile: not configured"
    end
    echo ""
    
    # Core components
    set -l core_config "$PI_AGENT_DIR/profiles/core.json"
    if test -f "$core_config"
        echo "Core Agent Components:"
        set -l num_skills (jq '.skills | length' "$core_config" 2>/dev/null; or echo 0)
        set -l num_exts (jq '.extensions | length' "$core_config" 2>/dev/null; or echo 0)
        set -l num_agents (jq '.agents | length' "$core_config" 2>/dev/null; or echo 0)
        echo "  Skills: $num_skills"
        echo "  Extensions: $num_exts"
        echo "  Agents: $num_agents"
    end
    echo ""
    
    # Available profiles
    echo "Available Profiles:"
    ls -1 "$PI_AGENT_DIR/profiles/" 2>/dev/null | sed 's/\.json$//' | sed 's/^/  - /'
    echo ""
    
    # Recent promotions
    if test -f "$PI_AGENT_DIR/promotions.log"
        echo "Recent Promotions:"
        tail -5 "$PI_AGENT_DIR/promotions.log" | sed 's/^/  /'
    end
end

# ============================================
# Setup Check
# ============================================

function pi-setup-check
    echo "=== Pi Agent Evolution Setup Check ==="
    echo ""
    
    set -l all_good true
    
    # Check directories
    for dir in profiles agents extensions/custom
        if test -d "$PI_AGENT_DIR/$dir"
            echo "✓ Directory exists: $dir"
        else
            echo "✗ Missing directory: $dir"
            set all_good false
        end
    end
    echo ""
    
    # Check profiles
    for profile in core coding research
        if test -f "$PI_AGENT_DIR/profiles/$profile.json"
            echo "✓ Profile exists: $profile"
        else
            echo "✗ Missing profile: $profile"
            set all_good false
        end
    end
    echo ""
    
    # Check agents
    for agent in core/scout core/planner core/worker
        if test -f "$PI_AGENT_DIR/agents/$agent.md"
            echo "✓ Agent exists: $agent"
        else
            echo "✗ Missing agent: $agent"
            set all_good false
        end
    end
    echo ""
    
    # Check commands
    if command -sq jq
        echo "✓ jq installed"
    else
        echo "⚠️  jq not installed (promotion features limited)"
        echo "   Install: brew install jq"
    end
    
    echo ""
    if test "$all_good" = true
        echo "✓ Setup complete!"
    else
        echo "⚠️  Setup incomplete"
    end
end
