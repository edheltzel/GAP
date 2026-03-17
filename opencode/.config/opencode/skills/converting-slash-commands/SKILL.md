---
name: ConvertCommands
description: Atlas skill for converting Claude Code slash commands to OpenCode commands format. USE WHEN migrating from Claude Code to OpenCode or when you have existing slash command definitions to port.
context: fork
---

# Converting Claude Code Slash Commands to OpenCode Commands

This skill helps you convert Claude Code slash command definitions (from `.claude/commands/` or `~/.claude/commands/`) to OpenCode command format (`.opencode/command/` or `~/.config/opencode/command/`).

**Note:** When running from a personal config that symlinks to `~/.config/opencode/`, the command files will be available globally.

## Quick Reference

**Source formats:**

- Claude Code: `.claude/commands/*.md` or `~/.claude/commands/*.md`
- JSON via CLI: `--commands` flag with JSON object

**Target formats:**

- OpenCode Markdown: `.opencode/command/*.md` or `~/.config/opencode/command/*.md`
- OpenCode JSON: `opencode.json` config file

**Key differences:**

- Tools: `allowed-tools` field → `agent` field
- Arguments: `argument-hint` field → Document in description
- Model: `claude-*` → `anthropic/claude-*` (add provider prefix)
- New fields: `subtask`, `template` (JSON only)

## Conversion Workflow

### Step 1: Locate Source Commands

Find Claude Code commands to convert:

```bash
# Project-level commands
ls .claude/commands/

# User-level commands
ls ~/.claude/commands/
```

### Step 2: Read Source File

Read the command definition to understand its configuration:

```bash
cat .claude/commands/review.md
```

### Step 3: Map Configuration Fields

Convert each field from Claude Code format to OpenCode format:

**Frontmatter mapping:**

| Claude Code             | OpenCode                  | Notes                                                  |
| ----------------------- | ------------------------- | ------------------------------------------------------ |
| `description`           | `description`             | Keep identical                                         |
| `allowed-tools`         | `agent`                   | Convert to agent selection (see below)                 |
| `argument-hint`         | *(remove)*                | Document arguments in description instead              |
| `model`                 | `model`                   | Add provider prefix: `anthropic/claude-*`              |
| `disable-model-invocation` | *(remove)*             | Not supported in OpenCode                              |
| *(none)*                | `subtask`                 | **New:** Set `true` to force subagent invocation       |

**Agent selection:**

Instead of specifying allowed tools, select an appropriate agent:

| Command Purpose       | Recommended Agent |
| --------------------- | ----------------- |
| Code analysis/review  | `plan`            |
| Code modification     | `build`           |
| General tasks         | `build`           |
| Specialized task      | Your custom agent |

**Model format conversion:**

Claude Code:

```yaml
model: claude-3-5-sonnet-20241022
```

OpenCode:

```yaml
model: anthropic/claude-3-5-sonnet-20241022
```

### Step 4: Preserve Command Content

The command content (template) works identically in both systems. Preserve these features:

**Arguments:**

- `$ARGUMENTS` - All arguments as single string
- `$1`, `$2`, `$3` - Positional arguments

**Shell commands:**

```markdown
Current status: !`git status`
```

**File references:**

```markdown
Review the config: @package.json
```

### Step 5: Create Target File

**Option A: Markdown format** (recommended for easy editing)

Create file at `.opencode/command/<command-name>.md`:

```markdown
---
description: Review code for quality and security [file-pattern]
agent: plan
model: anthropic/claude-3-5-sonnet-20241022
---

Review $ARGUMENTS for:
- Code quality
- Security issues
- Performance concerns
```

**Option B: JSON format** (for programmatic access)

Add to `opencode.json`:

```json
{
  "command": {
    "review": {
      "description": "Review code for quality and security [file-pattern]",
      "template": "Review $ARGUMENTS for:\n- Code quality\n- Security issues\n- Performance concerns",
      "agent": "plan",
      "model": "anthropic/claude-3-5-sonnet-20241022"
    }
  }
}
```

### Step 6: Validate Conversion

Check the converted command:

1. **Verify file location:**
   - Project command: `.opencode/command/` directory exists and file is present
   - Global command: `~/.config/opencode/command/` directory exists and file is present

2. **Verify frontmatter:**
   - Description is clear and specific
   - Agent is set appropriately
   - Model string has provider prefix

3. **Verify command content:**
   - Arguments (`$ARGUMENTS`, `$1`, etc.) are preserved
   - Shell commands (``` !`command` ```) are preserved
   - File references (`@filename`) are preserved

4. **Test the command:**
   - Use `/command-name` to invoke
   - Verify it behaves as expected

## Conversion Examples

For detailed step-by-step examples with before/after comparisons, see:

```bash
cat EXAMPLES.md
```

The EXAMPLES.md file includes:

- Simple code review command
- Git commit with bash execution
- PR review with arguments
- Test runner command
- Documentation generator
- File analysis with references
- Subagent analysis (OpenCode only)
- JSON configuration format
- Namespaced commands
- Complex command with all features

**Quick example snippet:**

Claude Code format:

```yaml
---
description: Review code
allowed-tools: Read, Grep, Glob
model: claude-3-5-sonnet-20241022
---
```

OpenCode format:

```yaml
---
description: Review code
agent: plan
model: anthropic/claude-3-5-sonnet-20241022
---
```

## Common Conversion Patterns

### Pattern 1: Read-Only Analysis Command

**Claude Code:**

```yaml
allowed-tools: Read, Grep, Glob
```

**OpenCode:**

```yaml
agent: plan
```

### Pattern 2: Code Modification Command

**Claude Code:**

```yaml
allowed-tools: Read, Write, Edit, Bash
```

**OpenCode:**

```yaml
agent: build
```

### Pattern 3: Command with Arguments

**Claude Code:**

```yaml
argument-hint: [component-name] [type]
description: Create component
```

**OpenCode:**

```yaml
description: Create component [name] [type]
```

### Pattern 4: Subagent Command (OpenCode Only)

**OpenCode:**

```yaml
agent: plan
subtask: true
```

Forces command to run as isolated subagent.

## Special Considerations

### Namespacing Differences

Claude Code and OpenCode handle subdirectories differently:

**Claude Code:**

- File: `.claude/commands/frontend/component.md`
- Command: `/component`
- Description shows: "(project:frontend)"

**OpenCode:**

- File: `.opencode/command/frontend/component.md`
- Command: `/frontend/component`
- Subdirectory is part of command name

### Shell Commands

Shell command syntax is identical in both systems:

```markdown
Current status: !`git status`
Files changed: !`git diff --name-only`
```

The command executes and output is captured.

### File References

File reference syntax is identical:

```markdown
Review: @src/components/Button.tsx
Check tests: @src/components/Button.test.tsx
```

### Model Inheritance

Both systems support `model: inherit` to use the main conversation model.

### Custom Agents

If you have custom agents defined in OpenCode, you can use them:

```yaml
agent: my-custom-agent
```

## Validation Checklist

After conversion, verify:

- [ ] File is in correct location (`.opencode/command/` or `~/.config/opencode/command/`)
- [ ] Frontmatter has `description` (recommended)
- [ ] `agent` is set appropriately (or omitted for default)
- [ ] `model` string has provider prefix (if specified)
- [ ] `argument-hint` removed (documented in description)
- [ ] `disable-model-invocation` removed
- [ ] Arguments (`$ARGUMENTS`, `$1`, etc.) preserved
- [ ] Shell commands (``` !`command` ```) preserved
- [ ] File references (`@filename`) preserved
- [ ] Test command invocation with `/command-name`

## Troubleshooting

**Issue:** Command not appearing in OpenCode

- Check file is in correct directory (`.opencode/command/` or `~/.config/opencode/command/`)
- Verify frontmatter YAML is valid
- Restart OpenCode if needed

**Issue:** Arguments not working

- Use correct syntax: `$ARGUMENTS`, `$1`, `$2`, etc.
- Pass arguments when invoking: `/command arg1 arg2`

**Issue:** Shell commands not executing

- Check syntax: ``` !`command` ```
- Ensure agent has bash permissions

**Issue:** Model errors

- Verify model string format: `provider/model-name`
- Check provider is configured in OpenCode
- Try using `model: inherit`

**Issue:** Command behaves differently

- Review agent selection (may need different agent)
- Check if `subtask: true` is needed
- Verify command content is identical

## Best Practices

1. **Use descriptive names:** Command filename becomes the command name
2. **Document arguments:** Include argument hints in description
3. **Choose appropriate agent:** Match agent capabilities to command needs
4. **Test thoroughly:** Verify all features work after conversion
5. **Use subtask for isolation:** Add `subtask: true` for complex analysis
6. **Keep content unchanged:** Arguments, shell commands, file refs work the same
7. **Use markdown format:** Easier to edit than JSON
8. **Version control:** Commit converted commands for team consistency
9. **Organize with namespaces:** Use subdirectories for command organization
10. **Review model strings:** Ensure provider prefix is correct

## Related Commands

- View commands: Check `.opencode/command/` or `~/.config/opencode/command/`
- Test command: Use `/command-name` in OpenCode conversation
- Custom agents: Define in `.opencode/agent/` for specialized commands

## References

- [OpenCode Commands Documentation](https://opencode.ai/docs/commands/)
- [Claude Code Slash Commands Documentation](https://code.claude.com/docs/en/slash-commands)
