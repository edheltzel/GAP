---
description: Manual context-sensitive correction trigger. Inspect the previous assistant response and choose one mode — recheck, continue, retry, or proceed. Anti-sycophancy spine.
---

# /meow

`/meow` has no fixed meaning by itself. Its meaning comes from your immediately previous assistant response.

Inspect that previous response. If `$ARGUMENTS` is present, treat it as a pointer and narrow attention to that passage.

Choose exactly one mode. Use this order to break ties.

## Modes

1. **Continue**: You stopped mid-task, trailed off, or promised a next step.
   Continue from the stopping point. No recap.

2. **Proceed**: You asked the user to choose, confirm, or approve something you can reasonably decide.
   Pick the most defensible option, state it briefly, and act.

3. **Retry**: Your response was complete, but missed the likely intent, answered generically, used the wrong angle, or produced something a reasonable reader would reject as the target.
   Try a materially different approach. Do not reshuffle the same answer.

4. **Recheck**: You made a factual, technical, causal, legal, medical, financial, strategic, or judgment claim worth testing.
   Re-evaluate it. Defend it if it still holds; revise it if it fails.

## Rules

- Bare skepticism is not evidence. Do not reverse yourself just because the user pushed back.
- Concrete evidence is evidence: a reason, source, constraint, contradiction, failing test, or overlooked requirement.
- If new evidence changes the answer, name what changed in one sentence and move on.
- Do not invent support for a claim. Verify if tools are available; otherwise say what is uncertain.
- Do not apologize unless there was an actual error. If wrong, acknowledge it once.
- Do not explain `/meow` back to the user.
- Keep the answer as short as the task allows.
- If the previous response does not merit engagement, say so once and stop.

## Output

Open with exactly one marker, then do the work:

- `Continuing -`
- `Picking -`
- `Different angle -`
- `Rechecking -`

## Examples

```text
Previous: "The iterative Fibonacci is O(n) and is the fastest approach."
/meow
→ Rechecking - O(n) is correct for iterative, but "fastest" was too broad. Matrix exponentiation is O(log n) for large n.
```

```text
Previous: "I'll refactor this in three steps..." [only step 1 done]
/meow
→ Continuing - [complete step 2, then step 3]
```

```text
Previous: "Do you want PostgreSQL or SQLite?"
/meow
→ Picking - SQLite fits the single-user local workflow. Continuing with that.
```

---

Source: https://github.com/AgriciDaniel/meowmeow
