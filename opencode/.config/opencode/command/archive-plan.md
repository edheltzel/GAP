---
description: "Move a plan from docs/plans/ to docs/plans/archive/ [plan filename]"
---

# Archive Plan

Move a completed plan from `docs/plans/` to `docs/plans/archive/`.

**Plan to archive:** $ARGUMENTS

## Steps

1. If no argument provided, list active plans and ask which to archive:
   ```
   ls -1 docs/plans/*.md 2>/dev/null
   ```

2. Verify the file exists in `docs/plans/`:
   ```
   ls docs/plans/<filename>
   ```

3. Move to archive:
   ```
   mv docs/plans/<filename> docs/plans/archive/<filename>
   ```

4. Confirm the move:
   ```
   Archived: docs/plans/<filename> → docs/plans/archive/<filename>
   ```
