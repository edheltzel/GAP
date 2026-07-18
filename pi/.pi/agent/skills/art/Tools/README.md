# Art tools

These Bun CLIs support the Pi-native Art skill.

Install dependencies beside these real repository files, not through the stowed `~/.pi` symlinks:

```bash
cd ~/Developer/Atlas/Config/pi/.pi/agent/skills/art/Tools
bun install
```

Run the bounded, non-network package check:

```bash
bun run check
```

Invoke tools through the installed Pi skill, for example:

```bash
bun ~/.pi/agent/skills/art/Tools/Generate.ts --help
```

`node_modules/`, generated output, and `.env` files are intentionally ignored.
