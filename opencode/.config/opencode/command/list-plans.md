---
description: "List all plans (active and archive count)"
---

# List Plans

Show all plans in `docs/plans/`.

## Steps

1. **Active plans** — list files in `docs/plans/`:
   ```
   ls -1 docs/plans/*.md 2>/dev/null || echo "No active plans."
   ```

2. **Archived plans** — show count only:
   ```
   ls docs/plans/archive/*.md 2>/dev/null | wc -l | xargs echo "Archived plans:"
   ```

3. **Format output** as a concise summary:
   ```
   ## Active Plans
   - <filename>

   ## Archive
   <N> archived plans
   ```
