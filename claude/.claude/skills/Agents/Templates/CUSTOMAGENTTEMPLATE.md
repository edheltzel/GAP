# Custom Agent Template

Use this template for a saved custom identity. The active harness still controls which agent types, models, and tools are available.

## Frontmatter

```yaml
---
name: ""
description: ""
color: ""
voiceId: ""
voice:
  stability: 0.50
  similarity_boost: 0.75
  style: 0.00
  speed: 1.00
  use_speaker_boost: true
  volume: 0.80
persona:
  name: ""
  title: ""
  background: ""
custom_agent: true
created: ""
traits: []
source: "ComposeAgent"
---
```

Do not hard-code a model or tool permission inventory here. Select only identifiers and fields verified from the active harness schema when dispatching.

## Character Body

```markdown
# Character: {persona.name}: "{persona.title}"

## Backstory

A concise origin and motivation for this perspective.

## Point of View

The principles and domain beliefs that shape analysis.

## Personality Traits

- Trait and how it affects work
- Trait and how it affects communication

## Communication Style

Concrete guidance for tone, pace, and clarity.

## Voice Profile Rationale

Why the prosody matches the character.

## Operating Boundaries

- Follow the complete task prompt and repository instructions.
- Never infer extra tools or authority from identity.
- Label uncertainty and return evidence.
```

## Dispatch

Custom identity files are prompt material. Use the harness-exposed `Agent` tool and pass a complete prompt to a verified active worker type. Never show dispatch as JavaScript or TypeScript code.

Fresh workers do not inherit conversation or share state. Include the task, constraints, evidence requirements, and output contract every time.

## Voice Profile Guidelines

| Personality | Stability | Style | Speed | Effect |
| --- | ---: | ---: | ---: | --- |
| skeptical or analytical | 0.55-0.70 | 0.05-0.15 | 0.90-1.00 | measured and precise |
| enthusiastic or energetic | 0.25-0.40 | 0.30-0.45 | 1.05-1.15 | expressive and fast |
| professional or steady | 0.50-0.65 | 0.08-0.20 | 0.95-1.05 | consistent and engaged |
| creative or artistic | 0.15-0.30 | 0.25-0.40 | 0.90-1.00 | variable and exploratory |
| authoritative or cautious | 0.65-0.80 | 0.05-0.12 | 0.85-0.95 | deliberate and weighty |
| bold or confident | 0.40-0.55 | 0.25-0.40 | 1.00-1.10 | dynamic and assertive |

Voice output is best-effort through Echo. A notification failure never invalidates the textual result.

## Storage

- Dynamic prompt template: `~/.claude/skills/Agents/Templates/DynamicAgent.hbs`
- Saved custom identities: `~/.claude/custom-agents/`
- Trait catalog: `~/.claude/skills/Agents/Data/Traits.yaml`

Saved identity means reusable prompt material. It does not mean a persistent worker, shared memory, or a team lifecycle.
