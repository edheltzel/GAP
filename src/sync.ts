import { lstat, mkdir, readFile, unlink, writeFile } from "node:fs/promises"
import { homedir } from "node:os"
import { dirname, join, resolve } from "node:path"
import { fileURLToPath } from "node:url"

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const home = homedir()
const docsDir = join(repoRoot, "global/docs/agents")

const copies = [
  {
    source: "global/AGENTS.md",
    dest: join(home, ".claude/rules/global-agent-instructions.md"),
  },
  {
    source: "global/AGENTS.md",
    dest: join(home, ".pi/agent/APPEND_SYSTEM.md"),
  },
  {
    source: "global/AGENTS.md",
    dest: join(home, ".omp/agent/APPEND_SYSTEM.md"),
  },
] as const

async function publish(source: string, dest: string): Promise<void> {
  const raw = await readFile(join(repoRoot, source), "utf8")
  try {
    const current = await lstat(dest)
    if (current.isSymbolicLink()) await unlink(dest)
  } catch (err) {
    if (!(err instanceof Error && "code" in err && err.code === "ENOENT")) throw err
  }
  await mkdir(dirname(dest), { recursive: true })
  await writeFile(dest, raw.replaceAll("](docs/agents/", `](${docsDir}/`))
  console.log(dest)
}

for (const copy of copies) {
  await publish(copy.source, copy.dest)
}
