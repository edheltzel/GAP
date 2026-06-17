# Review Workflow

Drive the `roughdraft` CLI to open a markdown file for review and (optionally) wait for the human's feedback.

## Voice Notification

```bash
curl -s -X POST http://localhost:31337/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running Review in Roughdraft"}' \
  > /dev/null 2>&1 &
```

Running **Review** in **Roughdraft**...

## Open a file

```bash
roughdraft open /absolute/path/to/file.md
```

`open` reuses a running server or auto-starts one (default `http://localhost:7373`). Report the served URL back to the user.

## Intent-to-Flag Mapping

| User Says | Command | Effect |
|-----------|---------|--------|
| "open X for review" (and wait) | `roughdraft open <file>` | Opens; **blocks until "Done Reviewing"** (default) |
| "open it and wait, give me parseable result" | `roughdraft open <file> --json` | Blocks (same as plain `open`) + prints event JSON |
| "just give me the URL, don't block" | `roughdraft open <file> --print-url --no-watch` | Prints URL, no browser, **returns immediately** (needs BOTH flags) |
| "open in browser but don't wait" | `roughdraft open <file> --no-watch` | Opens and returns immediately |
| "wait but cap it at N seconds" | `roughdraft open <file> --timeout <N>` | Bounds the default wait |
| "open without a browser, keep waiting" | `roughdraft open <file> --no-open` | Starts/reuses server, no browser, still waits |
| "is roughdraft running?" | `roughdraft status [--json]` | Server state + URL |
| "wait for done separately" | `roughdraft watch <file> [--json]` | Blocks on next "Done Reviewing" |
| "check it's healthy / validate md" | `roughdraft doctor [path]` | Non-blocking diagnostics |
| "stop roughdraft" | `roughdraft stop` | Stops the background server |
| "start the server only" | `roughdraft start` | Starts server, prints URL, exits |

**Key:** `open` blocks by default. `--no-watch` is the ONLY thing that makes it return immediately. `--print-url` and `--json` do **not** stop the wait on their own.

## Agent handoff (write → review → revise)

The core agent flow uses plain `open` (which blocks by default); add `--json` to parse the result:

```bash
roughdraft open ./draft.md --json
```

This opens the document, registers a fresh watcher, and **blocks until the human clicks "Done Reviewing"** (the `review.completed` event) — the blocking is `open`'s default behavior, not something `--json` adds. It then prints event JSON: document path, file version, feedback counts, and any `overallComment` submitted at handoff. Then read the CriticMarkup/comments from the file and revise.

- **No timeout by default** — it will wait indefinitely. Pass `--timeout <seconds>` to bound it. Pass `--no-watch` only if you do NOT want to wait.
- Overall comments are written to the markdown as document-level YAML endmatter before the event fires — the `.md` file stays the durable source of truth.

## Open by direct URL (server already running)

```text
http://localhost:7373/?path=/absolute/path/to/file.md
```

## Experimental: MCP

```bash
roughdraft mcp
```

Starts a stdio MCP server exposing tools to read the review index, list pending feedback, watch review events, append replies, and mark items resolved. CriticMarkup in the file remains the source of truth.
