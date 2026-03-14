#!/bin/bash
# Pi Agent Evolution Functions
# Source this in your ~/.zshrc or ~/.bashrc

PI_AGENT_DIR="${HOME}/.pi/agent"

# ============================================
# Profile Management
# ============================================

pi-profile() {
  local profile=$1
  local profile_path="${PI_AGENT_DIR}/profiles/${profile}.json"
  
  if [[ ! -f "$profile_path" ]]; then
    echo "✗ Profile '$profile' not found"
    echo "Available profiles:"
    ls -1 "${PI_AGENT_DIR}/profiles/" | sed 's/\.json$//' | sed 's/^/  - /'
    return 1
  fi
  
  # Update settings.json to point to new profile
  local settings_path="${PI_AGENT_DIR}/settings.json"
  
  # Create settings if doesn't exist
  if [[ ! -f "$settings_path" ]]; then
    echo "{\"profile\": \"${profile}\"}" > "$settings_path"
  else
    # Update existing settings
    if command -v jq &> /dev/null; then
      jq --arg profile "$profile" '.profile = $profile' "$settings_path" > /tmp/pi-settings.json
      mv /tmp/pi-settings.json "$settings_path"
    else
      # Fallback without jq
      echo "{\"profile\": \"${profile}\"}" > "$settings_path"
    fi
  fi
  
  echo "✓ Switched to profile: $profile"
  echo "⚠️  Restart Pi to apply changes"
  
  # Log the switch
  echo "$(date '+%Y-%m-%d %H:%M:%S') - Switched to $profile" >> "${PI_AGENT_DIR}/profile-history.log"
}

# Quick aliases
alias pi-core='pi-profile core'
alias pi-code='pi-profile coding'
alias pi-research='pi-profile research'

# ============================================
# Usage Analytics
# ============================================

pi-usage() {
  local stats_file="${PI_AGENT_DIR}/usage-stats.json"
  
  if [[ ! -f "$stats_file" ]]; then
    echo "No usage data yet. Start using Pi to collect stats."
    return 0
  fi
  
  echo "=== Pi Component Usage Report ==="
  echo ""
  
  if command -v jq &> /dev/null; then
    echo "Top Skills:"
    jq -r '.skills | to_entries | sort_by(.value) | reverse | .[0:10] | .[] | "  \(.value)x \(.key)"' "$stats_file" 2>/dev/null || echo "  No skill data"
    
    echo ""
    echo "Top Extensions:"
    jq -r '.extensions | to_entries | sort_by(.value) | reverse | .[0:10] | .[] | "  \(.value)x \(.key)"' "$stats_file" 2>/dev/null || echo "  No extension data"
    
    echo ""
    echo "Sessions completed: $(jq -r '.sessions // 0' "$stats_file")"
    
    echo ""
    echo "🔥 Promotion Candidates (used ≥ 10 times):"
    jq -r '
      (.skills // {}) + (.extensions // {}) | to_entries | 
      map(select(.value >= 10)) | 
      sort_by(.value) | reverse | 
      .[] | "  ✓ \(.value)x \(.key)"
    ' "$stats_file" 2>/dev/null || echo "  None yet - keep using those components!"
  else
    echo "Install jq for better formatting. Raw stats:"
    cat "$stats_file"
  fi
}

# ============================================
# Promotion Workflow
# ============================================

pi-promote() {
  local component=$1
  local from_profile=${2:-coding}  # Default to coding profile
  
  local core_config="${PI_AGENT_DIR}/profiles/core.json"
  local source_config="${PI_AGENT_DIR}/profiles/${from_profile}.json"
  
  if [[ ! -f "$core_config" ]]; then
    echo "✗ Core profile not found at $core_config"
    return 1
  fi
  
  # Detect if skill or extension
  local is_skill=false
  local is_extension=false
  
  if command -v jq &> /dev/null; then
    if jq -e ".skills | index(\"$component\")" "$source_config" > /dev/null 2>&1; then
      is_skill=true
    fi
    if jq -e ".extensions | index(\"$component\")" "$source_config" > /dev/null 2>&1; then
      is_extension=true
    fi
  fi
  
  # If not found in source, check if it's already in core
  if [[ "$is_skill" == "false" && "$is_extension" == "false" ]]; then
    if command -v jq &> /dev/null; then
      if jq -e ".skills | index(\"$component\")" "$core_config" > /dev/null 2>&1; then
        echo "ℹ️  '$component' is already in core agent"
        return 0
      fi
      if jq -e ".extensions | index(\"$component\")" "$core_config" > /dev/null 2>&1; then
        echo "ℹ️  '$component' is already in core agent"
        return 0
      fi
    fi
    echo "✗ Component '$component' not found in profile '$from_profile'"
    return 1
  fi
  
  # Promote to core
  if [[ "$is_skill" == "true" ]]; then
    if command -v jq &> /dev/null; then
      jq ".skills += [\"$component\"] | .skills |= unique" "$core_config" > "${core_config}.tmp"
      mv "${core_config}.tmp" "$core_config"
    fi
    echo "✓ Promoted skill '$component' to core agent"
    
    # Log promotion
    echo "$(date '+%Y-%m-%d %H:%M:%S') - PROMOTED skill: $component (from $from_profile)" >> "${PI_AGENT_DIR}/promotions.log"
  fi
  
  if [[ "$is_extension" == "true" ]]; then
    if command -v jq &> /dev/null; then
      jq ".extensions += [\"$component\"] | .extensions |= unique" "$core_config" > "${core_config}.tmp"
      mv "${core_config}.tmp" "$core_config"
    fi
    echo "✓ Promoted extension '$component' to core agent"
    
    # Log promotion
    echo "$(date '+%Y-%m-%d %H:%M:%S') - PROMOTED extension: $component (from $from_profile)" >> "${PI_AGENT_DIR}/promotions.log"
  fi
  
  echo ""
  echo "Core agent now includes:"
  if command -v jq &> /dev/null; then
    jq -r '.skills | .[]' "$core_config" | sed 's/^/  skill: /'
    jq -r '.extensions | .[]' "$core_config" | sed 's/^/  ext:   /'
  fi
  
  echo ""
  echo "⚠️  Restart Pi to apply changes"
}

# ============================================
# Agent Switching (for multi-agent workflow)
# ============================================

pi-agent() {
  local agent=$1
  local agent_file="${PI_AGENT_DIR}/agents/${agent}.md"
  
  # Check in subdirectories
  if [[ ! -f "$agent_file" ]]; then
    agent_file=$(find "${PI_AGENT_DIR}/agents" -name "${agent}.md" | head -1)
  fi
  
  if [[ -z "$agent_file" || ! -f "$agent_file" ]]; then
    echo "✗ Agent '$agent' not found"
    echo "Available agents:"
    find "${PI_AGENT_DIR}/agents" -name "*.md" | while read f; do
      local name=$(basename "$f" .md)
      local dir=$(basename $(dirname "$f"))
      echo "  - $name ($dir)"
    done
    return 1
  fi
  
  echo "✓ Selected agent: $agent"
  echo "📄 $(head -3 "$agent_file" | grep "^description:" | sed 's/description: //')"
  
  # Export for potential use by Pi
  export PI_CURRENT_AGENT="$agent"
}

# ============================================
# Status & Diagnostics
# ============================================

pi-status() {
  echo "=== Pi Agent Status ==="
  echo ""
  
  # Current profile
  local settings_path="${PI_AGENT_DIR}/settings.json"
  if [[ -f "$settings_path" ]]; then
    local current_profile=$(jq -r '.profile // "not set"' "$settings_path" 2>/dev/null)
    echo "Current Profile: $current_profile"
  else
    echo "Current Profile: not configured"
  fi
  echo ""
  
  # Core components
  local core_config="${PI_AGENT_DIR}/profiles/core.json"
  if [[ -f "$core_config" ]]; then
    echo "Core Agent Components:"
    echo "  Skills: $(jq '.skills | length' "$core_config" 2>/dev/null || echo 0)"
    echo "  Extensions: $(jq '.extensions | length' "$core_config" 2>/dev/null || echo 0)"
    echo "  Agents: $(jq '.agents | length' "$core_config" 2>/dev/null || echo 0)"
  fi
  echo ""
  
  # Available profiles
  echo "Available Profiles:"
  ls -1 "${PI_AGENT_DIR}/profiles/" 2>/dev/null | sed 's/\.json$//' | sed 's/^/  - /'
  echo ""
  
  # Recent promotions
  if [[ -f "${PI_AGENT_DIR}/promotions.log" ]]; then
    echo "Recent Promotions:"
    tail -5 "${PI_AGENT_DIR}/promotions.log" | sed 's/^/  /'
  fi
}

# ============================================
# Setup Check
# ============================================

pi-setup-check() {
  echo "=== Pi Agent Evolution Setup Check ==="
  echo ""
  
  local all_good=true
  
  # Check directories
  for dir in profiles agents extensions/custom; do
    if [[ -d "${PI_AGENT_DIR}/$dir" ]]; then
      echo "✓ Directory exists: $dir"
    else
      echo "✗ Missing directory: $dir"
      all_good=false
    fi
  done
  echo ""
  
  # Check profiles
  for profile in core coding research; do
    if [[ -f "${PI_AGENT_DIR}/profiles/${profile}.json" ]]; then
      echo "✓ Profile exists: $profile"
    else
      echo "✗ Missing profile: $profile"
      all_good=false
    fi
  done
  echo ""
  
  # Check agents
  for agent in core/scout core/planner core/worker; do
    if [[ -f "${PI_AGENT_DIR}/agents/${agent}.md" ]]; then
      echo "✓ Agent exists: $agent"
    else
      echo "✗ Missing agent: $agent"
      all_good=false
    fi
  done
  echo ""
  
  # Check commands
  if command -v jq &> /dev/null; then
    echo "✓ jq installed (required for promotion features)"
  else
    echo "⚠️  jq not installed (promotion features will be limited)"
    echo "   Install: brew install jq (macOS) or apt-get install jq (Linux)"
  fi
  
  echo ""
  if [[ "$all_good" == "true" ]]; then
    echo "✓ Setup complete! Ready to use evolutionary agents."
  else
    echo "⚠️  Setup incomplete. Run setup script to fix."
  fi
}