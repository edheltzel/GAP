import { execFileSync } from "node:child_process"
import { createHash } from "node:crypto"
import { lstat, mkdir, readFile, rename, unlink, writeFile } from "node:fs/promises"
import { homedir, tmpdir } from "node:os"
import { isAbsolute, dirname, join, relative, resolve } from "node:path"
import { createInterface } from "node:readline/promises"
import { stdin, stdout } from "node:process"
import { fileURLToPath } from "node:url"

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const home = homedir()
const docsDir = join(repoRoot, "global/docs/agents")
const GAP_START = "<!-- gap:start -->\n"
const GAP_END = "<!-- gap:end -->\n"

type Wrap = "none" | "cursor" | "codex"
type Decision = "write" | "foreign"

function adapt(source: string, wrap: Wrap): string {
  const body = source.replaceAll("](docs/agents/", `](${docsDir}/`)
  const text = body.endsWith("\n") ? body : `${body}\n`
  if (wrap === "cursor") return `---\nalwaysApply: true\n---\n\n${text}`
  return text
}

function digest(text: string): string {
  return createHash("sha256").update(text).digest("hex")
}

function decide(existing: string | undefined, payload: string, lastDigest?: string): Decision {
  if (existing === undefined || existing === payload) return "write"
  if (lastDigest !== undefined && digest(existing) === lastDigest) return "write"
  return "foreign"
}

function codexBlock(payload: string): string {
  return `${GAP_START}${payload}${GAP_END}`
}

function ledgerPrefixEnd(existing: string, lastDigest: string): number | undefined {
  let cut = 0
  while (cut < existing.length) {
    const nl = existing.indexOf("\n", cut)
    const next = nl === -1 ? existing.length : nl + 1
    if (digest(existing.slice(0, next)) === lastDigest) return next
    if (nl === -1) return undefined
    cut = next
  }
  return undefined
}

function spliceCodex(existing: string | undefined, payload: string, lastDigest?: string): string | "foreign" {
  const block = codexBlock(payload)
  if (existing === undefined) return block
  const start = existing.indexOf(GAP_START)
  const end = existing.indexOf(GAP_END, start + GAP_START.length)
  if (start !== -1 && end !== -1) {
    const inner = existing.slice(start + GAP_START.length, end)
    if (decide(inner, payload, lastDigest) === "foreign") return "foreign"
    return existing.slice(0, start) + block + existing.slice(end + GAP_END.length)
  }
  if (decide(existing, payload, lastDigest) === "write") return block
  if (existing.startsWith(payload)) return block + existing.slice(payload.length)
  if (lastDigest !== undefined) {
    const prefix = ledgerPrefixEnd(existing, lastDigest)
    if (prefix !== undefined) return block + existing.slice(prefix)
  }
  return "foreign"
}
function replaceGap(existing: string, payload: string): string {
  const start = existing.indexOf(GAP_START)
  const end = start === -1 ? -1 : existing.indexOf(GAP_END, start + GAP_START.length)
  if (start === -1 || end === -1) return codexBlock(payload)
  return existing.slice(0, start) + codexBlock(payload) + existing.slice(end + GAP_END.length)
}

function resolveCodexHome(envValue: string | undefined, homeDir: string): string {
  if (envValue === undefined || envValue === "") return join(homeDir, ".codex")
  if (!isAbsolute(envValue)) throw new Error(`CODEX_HOME must be an absolute path: ${envValue}`)
  return envValue
}

function refuseCheckout(branch: string, primary: string, gitDir: string, commonDir: string): string | undefined {
  if (branch !== primary) return `refusing to publish from ${branch}; checkout ${primary}`
  if (resolve(repoRoot, gitDir) !== resolve(repoRoot, commonDir)) return "refusing to publish from a worktree"
  return undefined
}


if (process.argv.includes("--check")) {
  const source = "See [x](docs/agents/dox.md).\n"
  const payload = adapt(source, "none")
  if (decide(payload, payload) !== "write") throw new Error("matching payload must be ours")
  if (decide(source, payload) !== "foreign") throw new Error("source bytes are not the publish")
  if (decide("other\n", payload) !== "foreign") throw new Error("other text is foreign")
  if (decide("old publish\n", "new publish\n", digest("old publish\n")) !== "write") {
    throw new Error("last publish must re-sync")
  }
  if (decide("hand edit\n", "new publish\n", digest("old publish\n")) !== "foreign") {
    throw new Error("hand edit is foreign")
  }
  if (decide(undefined, payload) !== "write") throw new Error("missing file is a write")
  const wrapped = adapt(source, "cursor")
  if (!wrapped.startsWith("---\nalwaysApply: true\n---\n\n")) throw new Error("cursor wrap missing")
  if (!wrapped.endsWith(payload)) throw new Error("cursor body drifted")
  if (resolveCodexHome(undefined, "/h") !== "/h/.codex") throw new Error("unset CODEX_HOME")
  if (resolveCodexHome("", "/h") !== "/h/.codex") throw new Error("empty CODEX_HOME")
  if (resolveCodexHome("/abs/codex", "/h") !== "/abs/codex") throw new Error("absolute CODEX_HOME")
  for (const bad of ["codex", "~/.codex", "rel/codex"]) {
    let threw = false
    try {
      resolveCodexHome(bad, "/h")
    } catch {
      threw = true
    }
    if (!threw) throw new Error(`relative CODEX_HOME must fail: ${bad}`)
  }
  if (refuseCheckout("side", "master", ".git", ".git") === undefined) throw new Error("non-primary must refuse")
  if (refuseCheckout("main", "master", ".git", ".git") === undefined) throw new Error("main is not primary here")
  if (refuseCheckout("master", "master", ".git", ".git") !== undefined) throw new Error("primary must publish")
  if (refuseCheckout("master", "master", ".git/worktrees/x", ".git") === undefined) throw new Error("worktree must refuse")
  const soul = "<!-- soulstack:start -->\nkeep\n<!-- soulstack:end -->\n<!-- gitbutler-agent-setup:start -->\nbut\n<!-- gitbutler-agent-setup:end -->\n"
  const old = "old policy\n"
  const legacy = `${old}${soul}`
  const migrated = spliceCodex(legacy, payload, digest(old))
  if (migrated === "foreign" || !migrated.startsWith(codexBlock(payload)) || !migrated.endsWith(soul)) {
    throw new Error("ledger prefix must replace the old unmarked body")
  }
  if (migrated.indexOf("<!-- gap:start -->") !== migrated.lastIndexOf("<!-- gap:start -->")) {
    throw new Error("duplicate gap block")
  }
  if (spliceCodex(legacy, payload) !== "foreign") throw new Error("unmarked mismatch must be foreign")
  if (spliceCodex(soul, payload) !== "foreign") throw new Error("unmarked co-writer file must be foreign")
  const owned = spliceCodex(migrated, "next\n", digest(payload))
  if (owned === "foreign" || !owned.includes("keep") || !owned.includes("but\n") || !owned.includes("next\n")) {
    throw new Error("resplice dropped co-writers")
  }
  const hand = spliceCodex(codexBlock("hand\n"), payload, digest("old\n"))
  if (hand !== "foreign") throw new Error("hand-edited gap block must be foreign")
  const relock = spliceCodex(codexBlock("old\n"), payload, digest("old\n"))
  if (relock === "foreign" || !relock.includes(payload)) throw new Error("ledger must own the gap block")
  const ledgerFile = join(tmpdir(), `gap-check-${process.pid}.json`)
  await writeLedger(ledgerFile, { a: "1" })
  const rawLedger = await readFile(ledgerFile, "utf8")
  if (JSON.parse(rawLedger).a !== "1") throw new Error("ledger did not persist")
  await unlink(ledgerFile)
  const probe = join(tmpdir(), `gap-probe-${process.pid}`)
  const foreignFile = join(probe, "foreign.md")
  const otherFile = join(probe, "other.md")
  await mkdir(probe, { recursive: true })
  await writeFile(foreignFile, "nope\n")
  const box: Record<string, string> = {}
  const skippedForeign = await publish(source, foreignFile, "none", box, ledgerFile)
  const wroteOther = await publish(source, otherFile, "none", box, ledgerFile)
  if (skippedForeign || !wroteOther) throw new Error("foreign skip must not block the next write")
  if ((await readFile(foreignFile, "utf8")) !== "nope\n") throw new Error("foreign file was replaced")
  if (!(await readFile(otherFile, "utf8")).includes(docsDir)) throw new Error("other destination was not published")
  await unlink(foreignFile)
  await unlink(otherFile)
  await unlink(ledgerFile)
  console.log("ok")
  process.exit(0)
}

function readCheckout(): { branch: string; primary: string; gitDir: string; commonDir: string } {
  const run = (args: string[]) => execFileSync("git", args, { cwd: repoRoot, encoding: "utf8" }).trim()
  const originHead = run(["symbolic-ref", "--short", "refs/remotes/origin/HEAD"])
  const primary = originHead.startsWith("origin/") ? originHead.slice("origin/".length) : originHead
  return {
    branch: run(["rev-parse", "--abbrev-ref", "HEAD"]),
    primary,
    gitDir: run(["rev-parse", "--git-dir"]),
    commonDir: run(["rev-parse", "--git-common-dir"]),
  }
}

async function writeLedger(path: string, ledger: Record<string, string>): Promise<void> {
  await mkdir(dirname(path), { recursive: true })
  const tmp = `${path}.${process.pid}.tmp`
  await writeFile(tmp, `${JSON.stringify(ledger, null, 2)}\n`)
  await rename(tmp, path)
}

const ledgerPath = join(home, ".gap/published.json")

async function readLedger(): Promise<Record<string, string>> {
  try {
    return JSON.parse(await readFile(ledgerPath, "utf8")) as Record<string, string>
  } catch (err) {
    if (err instanceof Error && "code" in err && err.code === "ENOENT") return {}
    throw err
  }
}

const codexHome = resolveCodexHome(process.env.CODEX_HOME, home)
const copies = [
  { dest: join(home, ".claude/rules/global-agent-policy.md"), wrap: "none" as const },
  { dest: join(home, ".pi/agent/APPEND_SYSTEM.md"), wrap: "none" as const },
  { dest: join(home, ".omp/agent/APPEND_SYSTEM.md"), wrap: "none" as const },
  { dest: join(codexHome, "AGENTS.md"), wrap: "codex" as const },
  { dest: join(home, ".cursor/rules/global-agent-policy.mdc"), wrap: "cursor" as const },
]

function backupPath(dest: string): string {
  const stamp = new Date().toISOString().replaceAll(/[:.]/g, "-")
  const tail = relative(home, dest)
  const safe = tail.startsWith("..") ? dest.slice(1) : tail
  return join(home, ".gap/backups", stamp, safe)
}

async function confirmReplace(dest: string, backup: string, codex: boolean): Promise<boolean> {
  const ask = codex
    ? `${dest} has a foreign GAP block. Replace that block only? Other blocks stay. [y/N] `
    : `${dest} is not the GAP publish. Move it to ${backup} and replace it with GAP? Review the backup and add anything you want to keep into global/AGENTS.md. [y/N] `
  if (!stdin.isTTY) {
    console.error(`Refusing to replace foreign file (no prompt): ${dest}`)
    return false
  }
  const rl = createInterface({ input: stdin, output: stdout })
  const answer = await rl.question(ask)
  rl.close()
  return answer.trim().toLowerCase() === "y"
}

async function publish(
  source: string,
  dest: string,
  wrap: Wrap,
  ledger: Record<string, string>,
  ledgerFile: string,
): Promise<boolean> {
  const payload = adapt(source, wrap === "cursor" ? "cursor" : "none")
  let existing: string | undefined
  let symlink = false
  try {
    const current = await lstat(dest)
    if (current.isSymbolicLink()) symlink = true
    else if (current.isFile()) existing = await readFile(dest, "utf8")
    else throw new Error(`${dest} is not a file`)
  } catch (err) {
    if (!(err instanceof Error && "code" in err && err.code === "ENOENT")) throw err
  }
  let text = payload
  if (wrap === "codex" && !symlink) {
    const spliced = spliceCodex(existing, payload, ledger[dest])
    if (spliced === "foreign") {
      if (!existing?.includes(GAP_START) || !existing.includes(GAP_END)) {
        console.error(`Refusing unmarked foreign file: ${dest}`)
        return false
      }
      if (!(await confirmReplace(dest, backupPath(dest), true))) return false
      text = replaceGap(existing, payload)
    } else text = spliced
  } else if (symlink || decide(existing, payload, ledger[dest]) === "foreign") {
    const backup = backupPath(dest)
    if (!(await confirmReplace(dest, backup, false))) return false
    await mkdir(dirname(backup), { recursive: true })
    await rename(dest, backup)
    console.log(`Backed up to ${backup}`)
    console.log("Review that backup. Add anything you want to keep into global/AGENTS.md.")
    text = wrap === "codex" ? codexBlock(payload) : payload
  }
  await mkdir(dirname(dest), { recursive: true })
  await writeFile(dest, text)
  ledger[dest] = digest(payload)
  await writeLedger(ledgerFile, ledger)
  console.log(dest)
  return true
}

const checkout = readCheckout()
const refused = refuseCheckout(checkout.branch, checkout.primary, checkout.gitDir, checkout.commonDir)
if (refused) {
  console.error(refused)
  process.exit(1)
}

const raw = await readFile(process.env.GAP_SOURCE ?? join(repoRoot, "global/AGENTS.md"), "utf8")
const ledger = await readLedger()
let skipped = false
for (const copy of copies) {
  if (!(await publish(raw, copy.dest, copy.wrap, ledger, ledgerPath))) skipped = true
}
if (skipped) process.exitCode = 1
