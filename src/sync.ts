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
const TRUST_LINE = "Trust the GAP hook in Codex with /hooks. Keep the legacy GAP text in Codex AGENTS.md: SessionStart does not reach Codex subagents, so do not run --remove-codex-legacy yet."

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

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function insertGapHook(text: string, policyPath: string): string | "present" | "malformed" {
  let parsed: unknown
  try {
    parsed = JSON.parse(text)
  } catch {
    return "malformed"
  }
  if (!isRecord(parsed) || !isRecord(parsed.hooks)) return "malformed"
  const current = parsed.hooks.SessionStart
  const session = Array.isArray(current) ? [...current] : []
  if (!Array.isArray(current) && current !== undefined) return "malformed"
  const command = hookCommand(policyPath)
  const present = session.some((group) => {
    if (!isRecord(group) || group.matcher !== "startup|resume|clear|compact") return false
    const hooks = group.hooks
    return Array.isArray(hooks) && hooks.some((hook) => {
      return isRecord(hook) && hook.type === "command" && hook.command === command && hook.additionalContextLimit === 0 && hook.async !== true
    })
  })
  if (present) return "present"
  session.push({
    matcher: "startup|resume|clear|compact",
    hooks: [{ type: "command", command, timeout: 10, additionalContextLimit: 0 }],
  })
  parsed.hooks.SessionStart = session
  return `${JSON.stringify(parsed, null, 2)}\n`
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
  const policyPath = "/h/.codex/gap/global-agent-policy.md"
  const missing = insertGapHook('{\n  "hooks": {}\n}\n', policyPath)
  if (missing === "present" || missing === "malformed") throw new Error("missing hooks file failed")
  JSON.parse(missing)
  if (!missing.includes("startup|resume|clear|compact")) throw new Error("matcher missing")
  const compactIn = '{"hooks":{"Stop":[]}}'
  const compact = insertGapHook(compactIn, policyPath)
  if (compact === "present" || compact === "malformed") throw new Error("compact hooks failed")
  const compactParsed = JSON.parse(compact) as { hooks: { Stop: unknown[] } }
  if (JSON.stringify(compactParsed.hooks.Stop) !== "[]") throw new Error("compact Stop entry changed")
  const liveShape = '{"hooks":{"SessionStart":[{"hooks":[{"command":"keep","timeout":10,"type":"command"}],"matcher":""}],"Stop":[{"hooks":[{"command":"stop","type":"command"}]}]}}'
  const liveOnce = insertGapHook(liveShape, policyPath)
  if (liveOnce === "present" || liveOnce === "malformed") throw new Error("live-shaped hooks failed")
  const liveParsed = JSON.parse(liveOnce) as { hooks: { SessionStart: Array<{ hooks: Array<{ command: string }> }>; Stop: unknown } }
  if (liveParsed.hooks.SessionStart[0]?.hooks[0]?.command !== "keep") throw new Error("live-shaped hook value changed")
  if (JSON.stringify(liveParsed.hooks.Stop) !== JSON.stringify(JSON.parse(liveShape).hooks.Stop)) throw new Error("live-shaped Stop changed")
  if (insertGapHook(liveOnce, policyPath) !== "present") throw new Error("hook insert was not idempotent")
  if (insertGapHook("{", policyPath) !== "malformed") throw new Error("bad JSON must refuse")
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
  const codexDir = join(probe, "codex")
  const policy = join(codexDir, "gap/global-agent-policy.md")
  const agents = join(codexDir, "AGENTS.md")
  const hooks = join(codexDir, "hooks.json")
  await mkdir(dirname(policy), { recursive: true })
  await writeFile(policy, "foreign policy\n")
  await writeFile(agents, "keep agents\n")
  await writeFile(hooks, '{"hooks":{"Stop":[]}}\n')
  const wrotePolicy = await publish(source, policy, "none", {}, ledgerFile, false, probe)
  if (wrotePolicy) throw new Error("foreign policy must be refused")
  if (!(await afterCodexPublish(wrotePolicy, policy, payload, codexDir, probe))) {
    throw new Error("refused policy must not fail the hook gate")
  }
  if ((await readFile(agents, "utf8")) !== "keep agents\n") throw new Error("AGENTS.md changed")
  if ((await readFile(hooks, "utf8")) !== '{"hooks":{"Stop":[]}}\n') throw new Error("hooks.json changed")
  if (insertGapHook("null", policy) !== "malformed") throw new Error("null hooks root must refuse")
  if (insertGapHook('{"hooks":null}', policy) !== "malformed") throw new Error("null hooks object must refuse")
  const legacyFile = "old policy\n\n<!-- soulstack:start -->\nkeep\n<!-- soulstack:end -->\n<!-- gitbutler-agent-setup:start -->\nbut\n<!-- gitbutler-agent-setup:end -->\n"
  await writeFile(policy, payload)
  await writeFile(agents, legacyFile)
  await writeFile(hooks, '{"hooks":{}}\n')
  const noHook = { [agents]: digest("old policy\n") }
  if (await cleanupLegacyCodex(codexDir, policy, payload, noHook, probe, ledgerFile)) {
    throw new Error("flag must refuse when the hook is missing")
  }
  if ((await readFile(agents, "utf8")) !== legacyFile) throw new Error("missing hook changed AGENTS.md")
  if (!(await afterCodexPublish(true, policy, payload, codexDir, probe))) throw new Error("default publish failed")
  if ((await readFile(agents, "utf8")) !== legacyFile) throw new Error("default run changed AGENTS.md")
  await writeFile(policy, "foreign policy\n")
  if (await cleanupLegacyCodex(codexDir, policy, payload, noHook, probe, ledgerFile)) {
    throw new Error("flag must refuse when the policy file is foreign")
  }
  if ((await readFile(agents, "utf8")) !== legacyFile) throw new Error("foreign policy changed AGENTS.md")
  await writeFile(policy, payload)
  if (!(await cleanupLegacyCodex(codexDir, policy, payload, noHook, probe, ledgerFile))) throw new Error("flag cleanup failed")
  const cleaned = await readFile(agents, "utf8")
  if (cleaned.includes("old policy") || !cleaned.startsWith("\n<!-- soulstack:start -->") || !cleaned.endsWith("<!-- gitbutler-agent-setup:end -->\n")) {
    throw new Error("flag cleanup did not keep the tool blocks")
  }
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

async function ensureGapHook(codexDir: string, policyPath: string, backupRoot: string): Promise<boolean> {
  const hooksPath = join(codexDir, "hooks.json")
  let text = '{\n  "hooks": {}\n}\n'
  let existed = false
  try {
    text = await readFile(hooksPath, "utf8")
    existed = true
  } catch (err) {
    if (!(err instanceof Error && "code" in err && err.code === "ENOENT")) throw err
  }
  const next = insertGapHook(text, policyPath)
  if (next === "present") return true
  if (next === "malformed") {
    console.error(`Refusing to edit hooks.json; it is not JSON: ${hooksPath}`)
    return false
  }
  JSON.parse(next)
  if (existed) {
    const backup = backupPath(hooksPath, backupRoot)
    await mkdir(dirname(backup), { recursive: true })
    await rename(hooksPath, backup)
    console.log(`Backed up to ${backup}`)
  }
  await writeAtomic(hooksPath, next)
  console.log(hooksPath)
  return true
}

async function cleanupLegacyCodex(
  codexDir: string,
  policyPath: string,
  payload: string,
  ledger: Record<string, string>,
  backupRoot: string,
  ledgerFile: string,
): Promise<boolean> {
  const dest = join(codexDir, "AGENTS.md")
  let policy: string | undefined
  try {
    policy = await readFile(policyPath, "utf8")
  } catch {}
  if (policy !== payload) {
    console.error(`Refusing to strip unmarked Codex policy; the GAP policy file is missing or not this publish: ${policyPath}`)
    return false
  }
  const hooksPath = join(codexDir, "hooks.json")
  let hooksText = ""
  try {
    hooksText = await readFile(hooksPath, "utf8")
  } catch {
    console.error(`Refusing to strip unmarked Codex policy; the GAP hook is not installed: ${hooksPath}`)
    return false
  }
  if (insertGapHook(hooksText, policyPath) !== "present") {
    console.error(`Refusing to strip unmarked Codex policy; the GAP hook is not installed: ${hooksPath}`)
    return false
  }
  const last = ledger[dest]
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
      await writeLedger(ledgerFile, ledger)
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
  await writeLedger(ledgerFile, ledger)
  console.log(`Backed up to ${backup}`)
  console.log(dest)
  return true
}

async function afterCodexPublish(
  wrote: boolean,
  policyPath: string,
  payload: string,
  codexDir: string,
  backupRoot: string,
): Promise<boolean> {
  if (!wrote) return true
  let onDisk: string
  try {
    onDisk = await readFile(policyPath, "utf8")
  } catch {
    console.error(`Refusing to install the Codex hook; policy file could not be reread: ${policyPath}`)
    return false
  }
  if (onDisk !== payload) {
    console.error(`Refusing to install the Codex hook; policy file does not match the payload: ${policyPath}`)
    return false
  }
  if (!(await ensureGapHook(codexDir, policyPath, backupRoot))) return false
  console.log(TRUST_LINE)
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
let codexWrote = false
for (const copy of copies) {
  const wrote = await publish(raw, copy.dest, copy.wrap, ledger, ledgerPath)
  if (!wrote) skipped = true
  if (copy.dest === codexPolicy && wrote) codexWrote = true
}
// Cleanup runs before this run's hook install, so a hook added just now (not yet trusted in /hooks) never counts.
if (process.argv.includes("--remove-codex-legacy")) {
  if (!(await cleanupLegacyCodex(codexHome, codexPolicy, adapt(raw, "none"), ledger, home, ledgerPath))) skipped = true
}
if (!(await afterCodexPublish(codexWrote, codexPolicy, adapt(raw, "none"), codexHome, home))) skipped = true
if (skipped) process.exitCode = 1
