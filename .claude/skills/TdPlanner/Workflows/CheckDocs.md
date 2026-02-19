# CheckDocs Workflow

Fetches the latest TD and Sidecar documentation, compares against the stored reference, and reports new commands, features, or changes.

## When to Run

- Periodically (monthly or before major plan decompositions)
- When TD or Sidecar releases a new version
- When user says "check td docs" or "update td knowledge"

## Process

### Step 1: Fetch Current Documentation

Fetch the latest docs from these sources:

1. **TD Core Workflow:** https://td.haplab.com/docs/core-workflow/
2. **TD GitHub:** https://github.com/marcus/td
3. **Sidecar Docs:** https://sidecar.haplab.com/docs/intro
4. **Sidecar TD Integration:** https://sidecar.haplab.com/docs/td

Use WebFetch or WebSearch to retrieve current content from each source.

### Step 2: Load Stored Reference

Read the current knowledge reference:
```bash
cat ~/.claude/MEMORY/LEARNING/SYSTEM/sidecar-td-reference.md
```

### Step 3: Compare and Identify Changes

Compare fetched docs against stored reference. Look for:

- **New commands** — TD commands not in our reference
- **Changed syntax** — Commands whose flags or arguments changed
- **New features** — Capabilities not previously documented
- **Deprecated items** — Commands or features marked deprecated
- **New Sidecar plugins** — Any new plugins beyond the five we know
- **New agent types** — Sidecar conversation support for new agents
- **New keyboard shortcuts** — Sidecar shortcuts not in our reference

### Step 4: Report Findings

Present a structured diff report:

```markdown
# TD/Sidecar Documentation Check — {date}

## Status: {UP TO DATE | CHANGES FOUND}

### New Commands
- `td [command]` — {description}

### Changed Commands
- `td [command]` — Was: {old}. Now: {new}.

### New Features
- {feature} — {description}

### Deprecated
- {item} — {reason if available}

### No Changes
- {section} — Still current as of {last checked date}
```

### Step 5: Update Reference (if changes found)

If changes were found:
1. Ask user: "Found N changes. Update the reference file?"
2. On approval, update `~/.claude/MEMORY/LEARNING/SYSTEM/sidecar-td-reference.md` with new information
3. Preserve the existing structure — add new content in the appropriate sections
4. Update the "Last updated" date at the top of the file

If no changes:
- Report "All clear — documentation matches our reference."
- Update the "Last updated" date to confirm the check was performed
