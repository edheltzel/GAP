# Harness detection

How a skill or script decides it runs inside Empryo. Skills that switch to an Empryo-native feature cite this page instead of restating it.

## Active, not installed

Empryo sets no environment variable for its shell children. Use one of these signals:

1. **Agent.** Empryo-only tools are in your toolset: `final_response`, `genome_query`, `code_script`. All three present means Empryo.
2. **Script.** An ancestor process is `~/.empryo/bin/empryo`. Linux names it through `/proc/<pid>/exe`; macOS `ps -o comm=` prints the full path:

```sh
in_empryo() {
  pid=$$
  while [ "${pid:-1}" -gt 1 ]; do
    exe=$(readlink "/proc/$pid/exe" 2>/dev/null || ps -o comm= -p "$pid")
    [ "$exe" = "$HOME/.empryo/bin/empryo" ] && return 0
    pid=$(ps -o ppid= -p "$pid" | tr -d ' ')
  done
  return 1
}
```

These only mean Empryo is installed, not that it runs this session: `~/.empryo` exists, `~/.empryo/bin` is on `PATH`, `empryo` resolves with `command -v`.

## When unsure

Custom agents, dispatch workers, and read-only modes can run with fewer tools, so the tool signal can miss Empryo. That false negative is expected and safe. Use the portable path. An Empryo branch must be an optional fast path, never the only path. Outside Empryo the skill behaves exactly as it did before the branch existed.
