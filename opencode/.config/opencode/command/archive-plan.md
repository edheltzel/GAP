---
description: "Move a plan from .atlas/plans/ to .atlas/plans/archive/ [plan filename]"
---

# Archive Plan

Move a completed plan from `.atlas/plans/` to `.atlas/plans/archive/`.

**Plan to archive:** $ARGUMENTS

## Steps

1. If no argument provided, list active plans and ask which to archive:
   ```
   ls -1 .atlas/plans/*.md 2>/dev/null
   ```

2. Verify the file exists in `.atlas/plans/`:
   ```
   ls .atlas/plans/<filename>
   ```

3. Move to archive:
   ```
   mv .atlas/plans/<filename> .atlas/plans/archive/<filename>
   ```

4. Confirm the move:
   ```
   Archived: .atlas/plans/<filename> → .atlas/plans/archive/<filename>
   ```
