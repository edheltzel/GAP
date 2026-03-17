# Humanize Workflow

Remove AI writing patterns from text to make it sound natural and human.

## Voice Notification

```bash
  -H "Content-Type: application/json" \
  -d '{"message": "Running the Humanize workflow in the Humanizer skill to remove AI writing patterns"}' \
  > /dev/null 2>&1 &
```

Running the **Humanize** workflow in the **Humanizer** skill to remove AI writing patterns...

## Step 1: Load References

Read these context files before starting:
- `Patterns.md` — All 24 AI writing pattern categories with before/after examples
- `SoulGuide.md` — How to add personality and voice (not just remove bad patterns)

## Step 2: Get the Text

Either:
- User pastes text directly -> work with the pasted content
- User points to a file -> Read the file
- User points to multiple files -> Process each one

## Step 3: Identify AI Patterns

Scan the text for all 24 pattern categories from `Patterns.md`:

**Content patterns (1-6):** Inflated significance, undue notability, superficial -ing analyses, promotional language, vague attributions, challenges/prospects sections

**Language patterns (7-12):** AI vocabulary words, copula avoidance, negative parallelisms, rule of three, elegant variation, false ranges

**Style patterns (13-18):** Em dash overuse, boldface overuse, inline-header lists, title case headings, emojis, curly quotes

**Communication patterns (19-21):** Collaborative artifacts, knowledge-cutoff disclaimers, sycophantic tone

**Filler patterns (22-24):** Filler phrases, excessive hedging, generic positive conclusions

## Step 4: Rewrite

For each identified pattern:
1. **Rewrite problematic sections** using the pattern's "After" examples as guidance
2. **Preserve meaning** — keep the core message intact
3. **Maintain voice** — match the intended tone (formal, casual, technical, etc.)
4. **Add soul** — follow SoulGuide.md: have opinions, vary rhythm, acknowledge complexity, use "I" when appropriate, let some mess in, be specific about feelings

## Step 5: Quality Check

Ensure the revised text:
- Sounds natural when read aloud
- Varies sentence structure naturally
- Uses specific details over vague claims
- Maintains appropriate tone for context
- Uses simple constructions (is/are/has) where appropriate
- Has personality — not just "clean" but alive

## Step 6: Output

Provide:
1. **The rewritten text** (or edit the file in place if working with files)
2. **A brief summary of changes** listing which pattern categories were found and fixed

## Working with Files

When editing files in place:
- Use the Edit tool to make targeted replacements
- For large rewrites, use Write to replace the full file
- Preserve file structure (headings, code blocks, etc.) — only humanize prose sections
- Do not modify code examples, configuration blocks, or quoted material unless they contain AI artifacts
