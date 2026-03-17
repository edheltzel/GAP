# RedTeam Skill Conversion Plan: PAI → Pi

## Executive Summary

RedTeam is a sophisticated 32-agent adversarial analysis skill currently in PAI/Claude Code format. This plan maps the conversion to Pi format, preserving the core functionality while adapting to Pi's conventions and tool ecosystem.

---

## Current Structure (PAI/Claude Code)

```
RedTeam/
├── SKILL.md              # Main entry point
├── Philosophy.md         # Core philosophy and success criteria
├── Integration.md        # Skill integration patterns
└── Workflows/
    ├── ParallelAnalysis.md      # 5-phase protocol
    └── AdversarialValidation.md # 3-round competition

**Total:** 5 files, ~800 lines of documentation
```

---

## Key Differences: PAI vs Pi

| Aspect | PAI/Claude Code | Pi |
|--------|-----------------|-----|
| **Parallel agents** | Task tool | `subagent` tool |
| **Notifications** | VoiceServer (curl localhost:8888) | `notify` tool |
| **User customization** | `~/.claude/skills/PAI/USER/SKILLCUSTOMIZATIONS/` | `~/.pi/agent/` or project `.pi/` |
| **Skill references** | `FirstPrinciples/Deconstruct` | Would need Pi equivalents |
| **Directory structure** | Similar | Similar |
| **Frontmatter** | YAML name+description | Same |

---

## Conversion Decisions

### 1. VoiceNotification

**Current:** curl to VoiceServer on localhost:8888

**Pi equivalent:** Use `notify` tool

```yaml
# PAI format
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running ParallelAnalysis workflow..."}'

# Pi format
notify(message="Running the ParallelAnalysis workflow in the RedTeam skill to red team arguments", title="RedTeam")
```

**Decision:** Replace all VoiceServer calls with `notify` tool invocations.

### 2. Parallel Agent Dispatch

**Current:** Task tool with multiple parallel calls

**Pi equivalent:** `subagent` tool with `tasks` array for parallel execution

**Key adaptation:**
- Pi's `subagent` uses `tasks: [{agent, task}, ...]` array for parallel dispatch
- Need to define agent personas as subagent configurations
- May need `scout`, `worker` agents or custom personas

**Decision:** Convert 32-agent roster to use `subagent` with parallel tasks array. May need to create custom agent definitions for the 4 archetypes (Engineer, Architect, Pentester, Intern).

### 3. Skill Dependencies

**Current references to other PAI skills:**
- `FirstPrinciples/Deconstruct` (Phase 1)
- `FirstPrinciples/Challenge` (Phase 5)
- `storyexplanation` (for decomposition)
- `extractalpha` (for synthesis)
- `research` (for precedents)

**Pi status:**
- `brainstorming` skill exists (could replace some FirstPrinciples analysis)
- `think` tool with `first_principles` mode is available
- No direct equivalents for `storyexplanation`, `extractalpha`

**Decision:** 
- Replace `FirstPrinciples/Deconstruct` → `think` tool with `mode: "first_principles"`
- Replace `FirstPrinciples/Challenge` → same `think` tool
- `storyexplanation` → Inline the methodology (it's about 24-claim decomposition, can be documented)
- `extractalpha` → Remove reference, keep the synthesis step as manual process

### 4. Skill Customization

**Current:** Check for customizations at `~/.claude/skills/PAI/USER/SKILLCUSTOMIZATIONS/RedTeam/`

**Pi equivalent:** Pi doesn't have a built-in customization overlay system.

**Decision:** Convert to standard Pi convention:
- Project-level: `.pi/agent/AGENTS.md` or `.pi/skills/`
- User-level: `~/.pi/agent/` configuration

Remove the customization check section from SKILL.md.

### 5. Directory Structure

**Current:**
```
RedTeam/
├── SKILL.md
├── Philosophy.md
├── Integration.md
└── Workflows/
    ├── ParallelAnalysis.md
    └── AdversarialValidation.md
```

**Pi convention:** Pi skills use flat supporting files or subdirectories.

**Decision:** Keep current structure - Pi supports subdirectories. The workflow files are substantial enough to warrant separate files.

---

## Implementation Plan

### Phase 1: Core SKILL.md Conversion

**Changes:**
1. Update frontmatter `description` to Pi format (third-person, "Use when...")
2. Remove VoiceServer notification section → Replace with `notify` tool
3. Remove customization directory check
4. Update skill references to Pi equivalents
5. Update "Last Updated" dates

**New frontmatter:**
```yaml
---
name: red-team
description: Use when red team analysis, attack ideas, find counterarguments, critique proposals, stress-test plans, or need adversarial analysis from 32 expert perspectives
---
```

### Phase 2: Workflow Files

**ParallelAnalysis.md:**
1. Replace VoiceServer curl with `notify` tool
2. Update Task tool references to `subagent` tool
3. Convert 32-agent roster to `subagent` task array format
4. Replace FirstPrinciples references with `think` tool
5. Update skill references

**AdversarialValidation.md:**
1. Replace VoiceServer curl with `notify` tool
2. Update Task tool references to `subagent` tool
3. Keep the 3-round protocol structure

### Phase 3: Supporting Files

**Philosophy.md:**
- No changes needed (pure documentation)

**Integration.md:**
- Update skill references to Pi equivalents
- Remove references to skills that don't exist in Pi

### Phase 4: New Files

**Create:** `agents.md` (new file)
- Document the 32 agent archetypes
- Provide templates for subagent dispatch
- Map PAI "Task" calls to Pi `subagent` format

---

## Pi Subagent Format for RedTeam

### Challenge: 32 Parallel Agents

Pi's `subagent` tool supports parallel execution via `tasks` array. However, dispatching32 agents requires:

1. **Agent definitions:** The 32 archetypes need to be defined
2. **Parallel dispatch:** Use `mode: "parallel"` with tasks array

### Proposed Solution

Create a subagent configuration approach:

```markdown
# Agent Archetypes (8 each of 4 types)

## Engineers (technical/logical rigor)
EN-1 through EN-8 with specific personalities...

## Architects (structural/systemic issues)
AR-1 through AR-8...

## Pentesters (adversarial/security thinking)
PT-1 through PT-8...

## Interns (fresh perspectives)
IN-1 through IN-8...
```

Each archetype becomes a task:
```json
{
  "agent": "worker",
  "task": "BALANCED ANALYSIS - EN-1: The Skeptical Systems Thinker..."
}
```

All 32 dispatched in parallel using `subagent` with `tasks` array.

---

## Testing Strategy

Per Pi's `writing-skills` skill, follow TDD for skills:

### RED Phase
1. Create pressure scenarios WITHOUT the skill
2. Test that agents don't naturally do adversarial analysis
3. Document baseline behavior

### GREEN Phase
1. Write the converted skill
2. Test that agents now follow the 5-phase protocol
3. Verify subagent dispatch works correctly

### REFACTOR Phase
1. Identify loopholes where agents skip phases
2. Add explicit counters
3. Re-test

---

## Files to Create/Modify

| File | Action | Changes |
|------|--------|---------|
| `SKILL.md` | Modify | Update frontmatter, remove VoiceServer, update references |
| `Workflows/ParallelAnalysis.md` | Modify | Replace Task→subagent, update notifications |
| `Workflows/AdversarialValidation.md` | Modify | Replace Task→subagent, update notifications |
| `Philosophy.md` | Keep | No changes needed |
| `Integration.md` | Modify | Update skill references |
| `agents.md` | Create | Document 32archetypes for subagent use |

---

## Success Criteria

- [ ] SKILL.md has Pi-compatible frontmatter
- [ ] All VoiceServer calls replaced with `notify` tool
- [ ] All Task tool calls replaced with `subagent` tool
- [ ] FirstPrinciples references converted to `think` tool
- [ ] Customization directory check removed
- [ ] skill references point to Pi equivalents or inlined
- [ ] Directory structure works with Pi's skill loader
- [ ] skill is discoverable via description triggers
- [ ] TDD process followed (baseline → skill → verify)

---

## Questions for User

1. **Agent dispatch scope:** Should the skill actually dispatch 32 subagents, or should it guide the main agent to simulate the 32 perspectives iteratively? 32 parallel subagents could be expensive/slow.

2. **FirstPrinciples integration:** Pi has `think` tool with `first_principles` mode. Should we use that directly, or create a FirstPrinciples skill as a dependency?

3. **Voice notifications:** Is the `notify` tool sufficient, or do you want to integrate with Pi's notification system differently?

4. **Skill dependencies:** Should we create stub skills for missing dependencies (storyexplanation, extractalpha) or inline their methodology into RedTeam?

5. **Testing approach:** Wouldyou like me to follow the TDD-for-skills process, or proceed directly with conversion?

---

## Timeline Estimate

- Phase 1 (SKILL.md): 30 minutes
- Phase 2 (Workflows): 1 hour
- Phase 3 (Supporting files): 20 minutes
- Phase 4 (agents.md): 30 minutes
- Testing: 1 hour

**Total:** ~3.5 hours for complete conversion + testing

---

*Last Updated: 2025-03-12*