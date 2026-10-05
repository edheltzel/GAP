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
const HOOK_MARK = "gap/global-agent-policy.md"

type Wrap = "none" | "cursor"
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

function shellQuote(value: string): string {
  return `'${value.replaceAll("'", `'\\''`)}'`
}

function hookCommand(policyPath: string): string {
  return `cat ${shellQuote(policyPath)}`
}

function insertGapHook(text: string, policyPath: string): string | "present" | "malformed" {
  if (text.includes(HOOK_MARK)) return "present"
  const entry = [
    "",
    "      {",
    '        "matcher": "startup|resume|clear|compact",',
    '        "hooks": [',
    "          {",
    '            "type": "command",',
    `            "command": ${JSON.stringify(hookCommand(policyPath))},`,
    '            "timeout": 10,',
    '            "additionalContextLimit": 0',
    "          }",
    "        ]",
    "      }",
  ].join("\n")
  const session = '"SessionStart": ['
  const at = text.indexOf(session)
  if (at !== -1) {
    const insertAt = at + session.length
    const rest = text.slice(insertAt)
    const comma = /^\s*]/.test(rest) ? "" : ","
    return text.slice(0, insertAt) + entry + comma + rest
  }
  const hooksKey = '"hooks": {'
  const hooksAt = text.indexOf(hooksKey)
  if (hooksAt === -1) return "malformed"
  const block = `\n    "SessionStart": [${entry}\n    ],`
  return text.slice(0, hooksAt + hooksKey.length) + block + text.slice(hooksAt + hooksKey.length)
}

function promptAllowed(allowPrompt: boolean): boolean {
  return allowPrompt && stdin.isTTY
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
  if (promptAllowed(false)) throw new Error("check must not prompt")
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
  const old = "old policy\n"
  const suffix = "\n<!-- soulstack:start -->\nkeep\n<!-- soulstack:end -->\n<!-- gitbutler-agent-setup:start -->\nbut\n<!-- gitbutler-agent-setup:end -->\n"
  const legacy = old + suffix
  if (ledgerPrefixEnd(legacy, digest(old)) !== old.length) throw new Error("ledger cut missed the prefix")
  if (legacy.slice(old.length) !== suffix) throw new Error("suffix was not kept")
  if (ledgerPrefixEnd(legacy, digest("nope\n")) !== undefined) throw new Error("unmatched prefix must refuse")
  const before = '{\n  "hooks": {\n    "SessionStart": [\n      {"hooks":[{"command":"keep","type":"command"}]}\n    ]\n  }\n}\n'
  const policyPath = "/h/.codex/gap/global-agent-policy.md"
  const once = insertGapHook(before, policyPath)
  if (once === "present" || once === "malformed") throw new Error("hook insert failed")
  if (!once.includes('{"hooks":[{"command":"keep","type":"command"}]}')) throw new Error("other hook bytes changed")
  if (!once.includes('"matcher": "startup|resume|clear|compact"')) throw new Error("matcher missing")
  JSON.parse(once)
  if (insertGapHook(once, policyPath) !== "present") throw new Error("hook insert was not idempotent")
  const ledgerFile = join(tmpdir(), `gap-check-${process.pid}.json`)
  await writeLedger(ledgerFile, { a: "1" })
  if (JSON.parse(await readFile(ledgerFile, "utf8")).a !== "1") throw new Error("ledger did not persist")
  await unlink(ledgerFile)
  const probe = join(tmpdir(), `gap-probe-${process.pid}`)
  const foreignFile = join(probe, "foreign.md")
  const otherFile = join(probe, "other.md")
  await mkdir(probe, { recursive: true })
  await writeFile(foreignFile, "nope\n")
  const box: Record<string, string> = {}
  const skippedForeign = await publish(source, foreignFile, "none", box, ledgerFile, false, probe)
  const wroteOther = await publish(source, otherFile, "none", box, ledgerFile, false, probe)
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

async function writeAtomic(path: string, text: string): Promise<void> {
  await mkdir(dirname(path), { recursive: true })
  const tmp = `${path}.${process.pid}.tmp`
  await writeFile(tmp, text)
  await rename(tmp, path)
}

async function writeLedger(path: string, ledger: Record<string, string>): Promise<void> {
  await writeAtomic(path, `${JSON.stringify(ledger, null, 2)}\n`)
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
const codexPolicy = join(codexHome, "gap/global-agent-policy.md")
const copies = [
  { dest: join(home, ".claude/rules/global-agent-policy.md"), wrap: "none" as const },
  { dest: join(home, ".pi/agent/APPEND_SYSTEM.md"), wrap: "none" as const },
  { dest: join(home, ".omp/agent/APPEND_SYSTEM.md"), wrap: "none" as const },
  { dest: codexPolicy, wrap: "none" as const },
  { dest: join(home, ".cursor/rules/global-agent-policy.mdc"), wrap: "cursor" as const },
]

function backupPath(dest: string, root: string): string {
  const stamp = new Date().toISOString().replaceAll(/[:.]/g, "-")
  const tail = relative(root, dest)
  const safe = tail.startsWith("..") ? dest.slice(1) : tail
  return join(root, ".gap/backups", stamp, safe)
}

async function confirmReplace(dest: string, backup: string): Promise<boolean> {
  if (!promptAllowed(true)) {
    console.error(`Refusing to replace foreign file (no prompt): ${dest}`)
    return false
  }
  const rl = createInterface({ input: stdin, output: stdout })
  const answer = await rl.question(
    `${dest} is not the GAP publish. Move it to ${backup} and replace it with GAP? Review the backup and add anything you want to keep into global/AGENTS.md. [y/N] `,
  )
  rl.close()
  return answer.trim().toLowerCase() === "y"
}

async function publish(
  source: string,
  dest: string,
  wrap: Wrap,
  ledger: Record<string, string>,
  ledgerFile: string,
  allowPrompt = true,
  backupRoot = home,
): Promise<boolean> {
  const payload = adapt(source, wrap)
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
  if (symlink || decide(existing, payload, ledger[dest]) === "foreign") {
    const backup = backupPath(dest, backupRoot)
    const yes = allowPrompt ? await confirmReplace(dest, backup) : false
    if (!allowPrompt) console.error(`Refusing to replace foreign file (no prompt): ${dest}`)
    if (!yes) return false
    await mkdir(dirname(backup), { recursive: true })
    await rename(dest, backup)
    console.log(`Backed up to ${backup}`)
    console.log("Review that backup. Add anything you want to keep into global/AGENTS.md.")
  }
  await writeAtomic(dest, payload)
  ledger[dest] = digest(payload)
  await writeLedger(ledgerFile, ledger)
  console.log(dest)
  return true
}

async function ensureGapHook(policyPath: string, backupRoot: string): Promise<void> {
  const hooksPath = join(codexHome, "hooks.json")
  let text = '{\n  "hooks": {\n  }\n}\n'
  let existed = false
  try {
    text = await readFile(hooksPath, "utf8")
    existed = true
  } catch (err) {
    if (!(err instanceof Error && "code" in err && err.code === "ENOENT")) throw err
  }
  const next = insertGapHook(text, policyPath)
  if (next === "present") return
  if (next === "malformed") throw new Error(`${hooksPath} has no hooks object`)
  if (existed) {
    const backup = backupPath(hooksPath, backupRoot)
    await mkdir(dirname(backup), { recursive: true })
    await rename(hooksPath, backup)
    console.log(`Backed up to ${backup}`)
  }
  await writeAtomic(hooksPath, next.endsWith("\n") ? next : `${next}\n`)
  console.log(hooksPath)
}

async function cleanupLegacyCodex(ledger: Record<string, string>, backupRoot: string): Promise<boolean> {
  const dest = join(codexHome, "AGENTS.md")
  const last = ledger[dest]
  if (last === undefined) return true
  let existing: string
  try {
    const current = await lstat(dest)
    if (!current.isFile() || current.isSymbolicLink()) {
      console.error(`Refusing to strip unmarked Codex policy; not a regular file: ${dest}`)
      return false
    }
    existing = await readFile(dest, "utf8")
  } catch (err) {
    if (err instanceof Error && "code" in err && err.code === "ENOENT") {
      delete ledger[dest]
      return true
    }
    throw err
  }
  const cut = ledgerPrefixEnd(existing, last)
  if (cut === undefined) {
    console.error(`Refusing to strip unmarked Codex policy; prefix does not match the ledger: ${dest}`)
    return false
  }
  const backup = backupPath(dest, backupRoot)
  await mkdir(dirname(backup), { recursive: true })
  await rename(dest, backup)
  await writeAtomic(dest, existing.slice(cut))
  delete ledger[dest]
  await writeLedger(ledgerPath, ledger)
  console.log(`Backed up to ${backup}`)
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
await ensureGapHook(codexPolicy, home)
if (!(await cleanupLegacyCodex(ledger, home))) skipped = true
if (skipped) process.exitCode = 1
