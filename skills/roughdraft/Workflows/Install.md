# Install Workflow

Install the published `roughdraft` CLI so the `roughdraft` command is available globally.

## Voice Notification

```bash
curl -s -X POST http://localhost:31337/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running Install in Roughdraft"}' \
  > /dev/null 2>&1 &
```

Running **Install** in **Roughdraft**...

## Install

`roughdraft` is a published npm package with a `roughdraft` bin. Pick the runner that matches the environment:

| Environment | Command |
|-------------|---------|
| npm (README default) | `npm i -g roughdraft` |
| bun-first toolchain | `bun add -g roughdraft` |
| no global install (one-off) | `bunx roughdraft ...` / `npx roughdraft ...` |

## Verify

```bash
roughdraft status        # should report no server yet, but the command resolves
roughdraft --help        # confirm the CLI is installed and on PATH
```

If `roughdraft` is not found after a global install, ensure your global bin directory is on `PATH` (e.g. the npm/bun global bin dir).

## Next

Use `Workflows/Review.md` to open and review a markdown file.

## Note

This installs the **published** CLI. Building from the source repo (a pnpm monorepo) is a separate concern and out of scope for this skill — this skill drives the published `roughdraft` command only.
