---
name: semaphore-as-mutex
description: A counting semaphore keyed on the protected resource ID is itself a mutex on that resource — no separate per-resource lock needed.
metadata:
  type: knowledge
---

# Counting Semaphore + Resource Key = Free Mutex

When a counting semaphore stores its slots in a table whose PK is the protected resource's identifier (e.g. `extraction_locks.conversation_path`), the same row that counts toward capacity also serves as the per-resource mutex.

INSERT OR IGNORE behavior:
- If no row exists for the resource and capacity is under limit → insert succeeds → slot taken AND lock acquired.
- If a row already exists for the resource → INSERT OR IGNORE returns 0 changes → caller knows another holder exists.
- If capacity is at limit → fails outside the INSERT step.

Implications:
- Don't build a separate "per-conv lock" table on top of a semaphore that already has resource-keyed PK.
- A child process spawned by a parent that already acquired the semaphore does NOT re-acquire — the row is the slot AND the lock.
- The release primitive deletes by resource ID, not by PID, so any process can release on completion.
- PID liveness sweep + TTL handles orphaned rows from crashed processes.

**Source incident:** atlas-recall SessionExtract refactor 2026-05-17 — initial design had parent call `acquireSemaphore` AND child call `childAcquireLock`. E2E failed at the child step because the parent's row prevented the child's INSERT OR IGNORE. Refuted at the second writer (`extraction_locks` table PK is `conversation_path`). Removed the redundant child-side acquire; kept it only for standalone CLI modes (`--reextract*`) where no parent ran.

**Where this applies:** any pattern where you want both (a) a global cap on concurrent operations and (b) a per-resource mutex. Pick a table keyed on the resource, count the rows for the cap, INSERT OR IGNORE for acquisition.
