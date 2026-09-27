import { lstat, mkdir, readFile, rename, writeFile } from "node:fs/promises"
import { homedir } from "node:os"
import { dirname, join, relative, resolve } from "node:path"
import { createInterface } from "node:readline/promises"
import { stdin, stdout } from "node:process"
import { fileURLToPath } from "node:url"

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const home = homedir()
const docsDir = join(repoRoot, "global/docs/agents")

type Wrap = "none" | "cursor"
type Decision = "write" | "foreign"

function adapt(source: string, wrap: Wrap): string {
  const body = source.replaceAll("](global/docs/agents/", `](${docsDir}/`)
  const text = body.endsWith("\n") ? body : `${body}\n`
  if (wrap === "none") return text
  return `---\nalwaysApply: true\n---\n\n${text}`
}

function decide(existing: string | undefined, payload: string): Decision {
  if (existing === undefined || existing === payload) return "write"
  return "foreign"
}

if (process.argv.includes("--check")) {
  const source = "See [x](global/docs/agents/dox.md).\n"
  const payload = adapt(source, "none")
  if (decide(payload, payload) !== "write") throw new Error("matching payload must be ours")
  if (decide(source, payload) !== "foreign") throw new Error("source bytes are not the publish")
  if (decide("other\n", payload) !== "foreign") throw new Error("other text is foreign")
  if (decide(undefined, payload) !== "write") throw new Error("missing file is a write")
  const wrapped = adapt(source, "cursor")
  if (!wrapped.startsWith("---\nalwaysApply: true\n---\n\n")) throw new Error("cursor wrap missing")
  if (!wrapped.endsWith(payload)) throw new Error("cursor body drifted")
  console.log("ok")
  process.exit(0)
}

const codexHome = process.env.CODEX_HOME ?? join(home, ".codex")
const copies = [
  { dest: join(home, ".claude/rules/global-agent-policy.md"), wrap: "none" as const },
  { dest: join(home, ".pi/agent/APPEND_SYSTEM.md"), wrap: "none" as const },
  { dest: join(home, ".omp/agent/APPEND_SYSTEM.md"), wrap: "none" as const },
  { dest: join(codexHome, "AGENTS.md"), wrap: "none" as const },
  { dest: join(home, ".cursor/rules/global-agent-policy.mdc"), wrap: "cursor" as const },
]

function backupPath(dest: string): string {
  const stamp = new Date().toISOString().replaceAll(/[:.]/g, "-")
  const tail = relative(home, dest)
  const safe = tail.startsWith("..") ? dest.slice(1) : tail
  return join(home, ".gap/backups", stamp, safe)
}

async function confirmReplace(dest: string, backup: string): Promise<boolean> {
  if (!stdin.isTTY) {
    console.error(`Refusing to replace foreign file (no prompt): ${dest}`)
    process.exitCode = 1
    return false
  }
  const rl = createInterface({ input: stdin, output: stdout })
  const answer = await rl.question(
    `${dest} is not the GAP publish. Move it to ${backup} and replace it with GAP? Review the backup and add anything you want to keep into global/AGENTS.md. [y/N] `,
  )
  rl.close()
  const yes = answer.trim().toLowerCase() === "y"
  if (!yes) process.exitCode = 1
  return yes
}

async function publish(source: string, dest: string, wrap: Wrap): Promise<void> {
  const payload = adapt(source, wrap)
  let existing: string | undefined
  try {
    const current = await lstat(dest)
    if (current.isSymbolicLink()) existing = "\0symlink"
    else if (current.isFile()) existing = await readFile(dest, "utf8")
    else throw new Error(`${dest} is not a file`)
  } catch (err) {
    if (!(err instanceof Error && "code" in err && err.code === "ENOENT")) throw err
  }
  if (decide(existing, payload) === "foreign") {
    const backup = backupPath(dest)
    if (!(await confirmReplace(dest, backup))) return
    await mkdir(dirname(backup), { recursive: true })
    await rename(dest, backup)
    console.log(`Backed up to ${backup}`)
    console.log("Review that backup. Add anything you want to keep into global/AGENTS.md.")
  }
  await mkdir(dirname(dest), { recursive: true })
  await writeFile(dest, payload)
  console.log(dest)
}

const raw = await readFile(join(repoRoot, "global/AGENTS.md"), "utf8")
for (const copy of copies) {
  await publish(raw, copy.dest, copy.wrap)
}
