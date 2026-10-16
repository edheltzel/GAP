#!/usr/bin/env bun
// Reconcile engine — IO SHELL (skill plan §4.4 / Phase 2).
//
// A THIN wrapper around the pure core in `reconcile-core.ts`. Everything risky
// (diff, the §6 option-union, op classification) lives in the pure core and is
// unit-tested offline; this file only does I/O: `gh` / `gh api graphql` calls,
// the lockfile, and argument parsing.
//
// Modes:
//   (default)                 plan  — dry-run diff, no writes (like `terraform plan`)
//   --apply                   apply — execute SAFE ops only (creates + drift updates)
//   --apply --prune --confirm prune — also execute DESTRUCTIVE ops (option/field/label removals)
//
// Board reconcile:  --project <n> --owner <login> --archetype <key> [--overlay <k>]...
// Label sync:       --repo <owner/repo> [--repo ...] | --manifest <file>   (+ --archetype)
//
// §6 safety (NON-NEGOTIABLE), enforced here on top of the pure core:
//   • updateProjectV2Field is a full-list replace → we always send the merged
//     union the core computed (existing options carry their id forward).
//   • Destructive option/field removals require --prune --confirm, are preceded
//     by a "resets N item assignments" warning, and trigger a re-snapshot
//     (re-resolve names→IDs) into the lockfile afterward.

import { resolve } from "../templates/resolve.ts";
import type { Archetype, PriorityScheme } from "../templates/types.ts";
import {
  buildPlan,
  destructiveGate,
  formatPlan,
  isEmpty,
  optionLiteral,
  type Snapshot,
  type LiveField,
  type LiveLabel,
  type OptionInput,
  type FieldRename,
  type Plan,
} from "./reconcile-core.ts";

// ──────────────────────────── gh shell ────────────────────────────

const dec = new TextDecoder();

// The gh executor is a swappable seam so the orchestration — most importantly the
// destructive gate and the apply ordering — is unit-testable without spawning real
// `gh` (F4). Production uses `defaultExec`; tests inject a fake via `__setGhExec`.
export type GhExec = (args: string[]) => string;

const defaultExec: GhExec = (args) => {
  const r = Bun.spawnSync(["gh", ...args]);
  if (!r.success) {
    throw new Error(`gh ${args.join(" ")}\n${dec.decode(r.stderr).trim() || dec.decode(r.stdout).trim()}`);
  }
  return dec.decode(r.stdout);
};

let ghExec: GhExec = defaultExec;

/** Test seam: swap the gh executor; pass `null` to restore the real one. */
export function __setGhExec(fn: GhExec | null): void {
  ghExec = fn ?? defaultExec;
}

function gh(args: string[]): string {
  return ghExec(args);
}

function graphql(query: string, vars: Record<string, string> = {}): unknown {
  const args = ["api", "graphql", "-f", `query=${query}`];
  for (const [k, v] of Object.entries(vars)) args.push("-f", `${k}=${v}`);
  return JSON.parse(gh(args));
}

// ──────────────────────────── Snapshot ────────────────────────────

function resolveProjectId(owner: string, number: number): string {
  const out = gh(["project", "view", String(number), "--owner", owner, "--format", "json"]);
  const id = (JSON.parse(out) as { id?: string }).id;
  if (!id) throw new Error(`could not resolve project id for ${owner}/#${number}`);
  return id;
}

const SNAPSHOT_QUERY = `query($id: ID!) {
  node(id: $id) {
    ... on ProjectV2 {
      id
      fields(first: 50) {
        nodes {
          __typename
          ... on ProjectV2FieldCommon { id name dataType }
          ... on ProjectV2SingleSelectField {
            id name dataType
            options { id name color description }
          }
        }
      }
    }
  }
}`;

function snapshotBoard(projectId: string): Omit<Snapshot, "labels"> {
  const data = graphql(SNAPSHOT_QUERY, { id: projectId }) as {
    data: { node: { id: string; fields: { nodes: RawField[] } } };
  };
  const raw = data.data.node;
  const fields: LiveField[] = raw.fields.nodes
    .filter((n) => n.__typename) // skip nulls
    .map((n) => ({
      id: n.id,
      name: n.name,
      dataType: n.dataType,
      ...(n.options ? { options: n.options } : {}),
    }));
  return { projectId: raw.id, fields };
}

interface RawField {
  __typename: string;
  id: string;
  name: string;
  dataType: LiveField["dataType"];
  options?: { id: string; name: string; color: string; description: string }[];
}

function snapshotLabels(repo: string): LiveLabel[] {
  const out = gh(["label", "list", "--repo", repo, "--json", "name,color,description", "--limit", "500"]);
  return JSON.parse(out) as LiveLabel[];
}

// ──────────────────────────── Option union → GraphQL ────────────────────────────
// `optionLiteral`/`colorLiteral` live in the pure core (id-preservation is the
// §6 wire payload — tested there). Full-list replace with the core's merged union;
// the id-preservation is what
// keeps this non-destructive for everything except the removals the core dropped.
function applyOptionUnion(fieldId: string, union: OptionInput[]): void {
  const opts = union.map(optionLiteral).join(",");
  const mutation = `mutation {
    updateProjectV2Field(input: { fieldId: ${JSON.stringify(fieldId)}, singleSelectOptions: [${opts}] }) {
      projectV2Field { ... on ProjectV2SingleSelectField { id } }
    }
  }`;
  graphql(mutation);
}

// ──────────────────────────── Field create / delete ────────────────────────────

function createField(owner: string, number: number, name: string, type: string, optionNames: string[]): void {
  if (type === "ITERATION") {
    console.warn(`  ! skipping ITERATION field "${name}" — not creatable via gh CLI (GraphQL/UI only). Create it manually.`);
    return;
  }
  const args = ["project", "field-create", String(number), "--owner", owner, "--name", name, "--data-type", type];
  if (type === "SINGLE_SELECT") {
    // CLI sets option NAMES only; the option-union pass below recolours them.
    args.push("--single-select-options", optionNames.join(","));
  }
  gh(args);
}

function deleteField(fieldId: string): void {
  gh(["project", "field-delete", "--id", fieldId]);
}

// ──────────────────────────── Labels ────────────────────────────

function applyLabel(repo: string, name: string, color: string, description: string): void {
  gh(["label", "create", name, "--repo", repo, "--color", color.replace(/^#/, ""), "--description", description, "--force"]);
}

function deleteLabel(repo: string, name: string): void {
  gh(["label", "delete", name, "--repo", repo, "--yes"]);
}

// ──────────────────────────── §6: capture item assignments ────────────────────────────
// Before a destructive option removal we capture WHICH items hold the option (not
// just a count), so a reset is manually recoverable (F2) and the warning is exact.
//
// `gh project item-list --format json` keys a custom field by gh's camelCase of the
// display name (e.g. "Sub-issues progress" → "subIssuesProgress"), NOT the lowercased
// name — so the old `it[name.toLowerCase()]` missed multi-word fields and silently
// returned 0 (F1b). We resolve the real key and, if no candidate key is present on
// any item, return `resolved:false` → the warning says "unknown" rather than a false 0.

// gh's field-name → JSON-key normalization: split on non-alphanumerics, lower the
// first word, capitalize the rest. Exported for unit testing.
export function camelKey(name: string): string {
  const words = name.split(/[^A-Za-z0-9]+/).filter(Boolean);
  return words.map((w, i) => (i === 0 ? w.toLowerCase() : w[0].toUpperCase() + w.slice(1).toLowerCase())).join("");
}

interface CapturedItem {
  id: string;
  title: string;
  option: string;
}

function captureAssignments(owner: string, number: number, fieldName: string, optionNames: string[]): { resolved: boolean; items: CapturedItem[] } {
  if (optionNames.length === 0) return { resolved: true, items: [] };
  let items: Record<string, unknown>[];
  try {
    const out = gh(["project", "item-list", String(number), "--owner", owner, "--format", "json", "--limit", "5000"]);
    items = (JSON.parse(out) as { items?: Record<string, unknown>[] }).items ?? [];
  } catch {
    return { resolved: false, items: [] }; // could not query → unknown, never a false 0
  }
  if (items.length === 0) return { resolved: true, items: [] }; // genuinely no items → honest 0
  const candidates = [camelKey(fieldName), fieldName.toLowerCase(), fieldName];
  const key = candidates.find((k) => items.some((it) => k in it));
  if (!key) return { resolved: false, items: [] }; // field's item-list key not found → unknown, not 0
  const targets = new Set(optionNames);
  const captured: CapturedItem[] = [];
  for (const it of items) {
    const v = it[key];
    if (typeof v === "string" && targets.has(v)) {
      captured.push({ id: String(it.id ?? ""), title: String(it.title ?? ""), option: v });
    }
  }
  return { resolved: true, items: captured };
}

interface RemovalCapture {
  count: number; // -1 when unresolved (unknown)
  resolved: boolean;
  items: CapturedItem[];
}

// Persist the item→option assignments a destructive removal is about to reset, so
// they can be re-applied by hand. Named `*.lock.json` so it lands under the same
// gitignore rule as the lockfile (default lockdir is .agents/atlas/artifacts).
async function writeRecoveryArtifact(dir: string, slug: string, caps: Map<string, RemovalCapture>): Promise<string> {
  Bun.spawnSync(["mkdir", "-p", dir]);
  const stamp = new Date().toISOString().replace(/[:.]/g, "-");
  const fields = [...caps.entries()].map(([field, c]) => ({ field, resolved: c.resolved, count: c.count, items: c.items }));
  const body = {
    slug,
    capturedAt: new Date().toISOString(),
    note: "Item→option assignments reset by a destructive single-select option removal. Re-assign manually to recover.",
    fields,
  };
  const path = `${dir}/${slug}-recovery-${stamp}.lock.json`;
  await Bun.write(path, JSON.stringify(body, null, 2) + "\n");
  return path;
}

// ──────────────────────────── Lockfile ────────────────────────────

interface Lockfile {
  slug: string;
  projectId: string;
  specHash: string;
  updatedAt: string;
  fields: Record<string, string>; // field name → node id
  options: Record<string, Record<string, string>>; // field name → (option name → id)
}

// Resolve the lockfile directory. A relative --lockdir (incl. the default) is
// anchored to the git repo root, NOT the CWD — otherwise a stray, un-gitignored
// `.agents/` is created wherever the tool happens to be invoked from.
function resolveLockdir(dir: string): string {
  if (dir.startsWith("/")) return dir;
  const r = Bun.spawnSync(["git", "rev-parse", "--show-toplevel"]);
  if (!r.success) return dir; // not a git repo — fall back to CWD-relative
  const root = dec.decode(r.stdout).trim();
  return `${root}/${dir}`;
}

function lockfilePath(dir: string, slug: string): string {
  return `${dir}/${slug}.lock.json`;
}

async function writeLockfile(dir: string, slug: string, projectId: string, specHash: string, snap: Omit<Snapshot, "labels">): Promise<string> {
  Bun.spawnSync(["mkdir", "-p", dir]);
  const fields: Record<string, string> = {};
  const options: Record<string, Record<string, string>> = {};
  for (const f of snap.fields) {
    fields[f.name] = f.id;
    if (f.options) {
      options[f.name] = Object.fromEntries(f.options.map((o) => [o.name, o.id]));
    }
  }
  const lock: Lockfile = { slug, projectId, specHash, updatedAt: new Date().toISOString(), fields, options };
  const path = lockfilePath(dir, slug);
  await Bun.write(path, JSON.stringify(lock, null, 2) + "\n");
  return path;
}

// ──────────────────────────── Arg parsing ────────────────────────────

interface Args {
  project?: number;
  owner?: string;
  archetype?: string;
  overlays: string[];
  priority: PriorityScheme;
  repos: string[];
  manifest?: string;
  apply: boolean;
  prune: boolean;
  confirm: boolean;
  lockdir: string;
  renames: FieldRename[];
}

// Parse `--rename "Field:Old=New"` → {field, from, to}. Split on the FIRST ':' and
// then the FIRST '=' so field/option names may contain later separators.
export function parseRename(spec: string): FieldRename {
  const colon = spec.indexOf(":");
  if (colon < 1) throw new Error(`--rename "${spec}": expected "Field:Old=New"`);
  const field = spec.slice(0, colon);
  const rest = spec.slice(colon + 1);
  const eq = rest.indexOf("=");
  if (eq < 1) throw new Error(`--rename "${spec}": expected "Field:Old=New"`);
  const from = rest.slice(0, eq);
  const to = rest.slice(eq + 1);
  if (!from || !to) throw new Error(`--rename "${spec}": both old and new option names are required`);
  return { field, from, to };
}

function parseArgs(argv: string[]): Args {
  const a: Args = { overlays: [], priority: "p0-p3", repos: [], apply: false, prune: false, confirm: false, lockdir: ".agents/atlas/artifacts", renames: [] };
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    const next = () => argv[++i];
    switch (arg) {
      case "--project": a.project = Number(next()); break;
      case "--owner": a.owner = next(); break;
      case "--archetype": a.archetype = next(); break;
      case "--overlay": a.overlays.push(next()); break;
      case "--priority": a.priority = next() as PriorityScheme; break;
      case "--repo": a.repos.push(next()); break;
      case "--manifest": a.manifest = next(); break;
      case "--lockdir": a.lockdir = next(); break;
      case "--rename": a.renames.push(parseRename(next())); break;
      case "--apply": a.apply = true; break;
      case "--prune": a.prune = true; break;
      case "--confirm": a.confirm = true; break;
      case "-h": case "--help": printUsage(); process.exit(0);
      default: throw new Error(`unknown argument: ${arg}`);
    }
  }
  return a;
}

function printUsage(): void {
  console.log(`Reconcile a GitHub Projects v2 board's fields/options and repo labels to an archetype.

Usage:
  bun Reconcile.ts --project <n> --owner <login> --archetype <key> [--overlay <k>]... [--repo <owner/repo>]... [flags]
  bun Reconcile.ts --archetype <key> --manifest <file>            # label sync across repos
  bun Reconcile.ts --archetype <key> --repo <owner/repo>...       # label sync (no board)

Modes (default is dry-run plan):
  --apply                   execute SAFE changes only (creates + colour/description drift)
  --prune                   include destructive removals in the plan (preview)
  --apply --prune --confirm execute destructive removals too (option/field/label deletes)

Other flags:
  --rename <Field:Old=New>  in-place, assignment-preserving option rename (repeatable; needs --apply)
  --priority <scheme>        Priority options: p0-p3 (default), high-medium-low, or none
  --lockdir <dir>           lockfile directory (default .agents/atlas/artifacts; *.lock.json is gitignored)
  -h, --help                this help

Manifest JSON: { "repos": ["owner/a", ...], "archetype"?: "<key>", "overlays"?: ["<k>"], "priority"?: "<scheme>" }`);
}

// ──────────────────────────── Orchestration ────────────────────────────

async function resolveSpecAndRepos(a: Args): Promise<{ spec: Archetype; repos: string[] }> {
  let archetypeKey = a.archetype;
  let overlays = a.overlays;
  let priority = a.priority;
  let repos = [...a.repos];
  if (a.manifest) {
    const m = JSON.parse(await Bun.file(a.manifest).text()) as { repos?: string[]; archetype?: string; overlays?: string[]; priority?: PriorityScheme };
    repos = [...new Set([...(m.repos ?? []), ...repos])];
    archetypeKey ??= m.archetype;
    if (a.overlays.length === 0 && m.overlays) overlays = m.overlays;
    if (m.priority) priority = m.priority;
  }
  if (!archetypeKey) throw new Error("--archetype is required (directly or via --manifest)");
  return { spec: resolve(archetypeKey, overlays, priority), repos };
}

export async function reconcileBoard(a: Args, spec: Archetype): Promise<{ plan: Plan; projectId: string }> {
  const owner = a.owner!;
  const number = a.project!;
  const slug = `${owner}-project-${number}`;
  const projectId = resolveProjectId(owner, number);
  let snap: Snapshot = { ...snapshotBoard(projectId), labels: [] };
  // Board mode reconciles fields + options only; labels are repo-scoped and
  // handled by reconcileLabels per --repo. specHash stays over the full spec.
  let plan = buildPlan(snap, spec, { prune: a.prune, renames: a.renames });
  plan.labelOps = [];

  console.log(formatPlan(plan, { project: `${owner}/#${number}`, prune: a.prune }));

  if (!a.apply) return { plan, projectId };
  // §6 gate (pure, unit-tested): destructive prune needs --confirm. Throws before
  // any mutation below.
  destructiveGate(plan, { prune: a.prune, confirm: a.confirm });

  console.log("\n── applying ──");

  // 1. Create missing fields, then re-snapshot so new field/option ids are known.
  const fieldCreates = plan.fieldOps.filter((o) => o.kind === "create");
  for (const op of fieldCreates) {
    const ds = spec.fields.find((f) => f.name === op.target);
    const type = ds?.type ?? "TEXT";
    const optionNames = (ds?.options ?? []).map((o) => o.name);
    console.log(`  + create field "${op.target}" (${type})`);
    createField(owner, number, op.target, type, optionNames);
  }
  if (fieldCreates.length) {
    snap = { ...snapshotBoard(projectId), labels: [] };
    plan = buildPlan(snap, spec, { prune: a.prune, renames: a.renames });
    plan.labelOps = [];
  }

  // 2. Capture item→option assignments for any destructive removals BEFORE mutating,
  //    so a reset is manually recoverable and the warning is exact (F2/F1b).
  const removalCaps = new Map<string, RemovalCapture>();
  for (const s of plan.selects) {
    if (s.ops.length === 0 || s.removals.length === 0) continue;
    const cap = captureAssignments(owner, number, s.fieldName, s.removals);
    removalCaps.set(s.fieldName, { count: cap.resolved ? cap.items.length : -1, resolved: cap.resolved, items: cap.items });
  }
  if (removalCaps.size) {
    const path = await writeRecoveryArtifact(resolveLockdir(a.lockdir), slug, removalCaps);
    console.warn(`  ⚠ recovery snapshot (item→option assignments about to reset): ${path}`);
  }

  // 3. Apply option unions (full-list replace, id-preserving) for selects with ops.
  for (const s of plan.selects) {
    if (s.ops.length === 0) continue;
    // The engine enforces its own invariant rather than leaning on the API to
    // reject a wrong-type / undefined-id write (F3): a select with ops must be a
    // resolved SINGLE_SELECT field by now (created + re-snapshotted).
    if (!s.exists || !s.fieldId) {
      throw new Error(`field "${s.fieldName}" not resolved after create/re-snapshot — re-run`);
    }
    if (s.removals.length) {
      const cap = removalCaps.get(s.fieldName);
      const count = !cap || cap.count < 0 ? "an unknown number of" : String(cap.count);
      console.warn(`  ⚠ ${s.fieldName}: removing [${s.removals.join(", ")}] resets ${count} item assignment(s)`);
    }
    console.log(`  ~ ${s.fieldName}: writing ${s.union.length}-option union (${s.ops.length} change(s))`);
    applyOptionUnion(s.fieldId, s.union);
  }

  // 4. Destructive field deletes (prune only).
  for (const op of plan.fieldOps.filter((o) => o.kind === "delete")) {
    const live = snap.fields.find((f) => f.name === op.target);
    if (live) {
      console.warn(`  - deleting field "${op.target}" and its data`);
      deleteField(live.id);
    }
  }

  return { plan, projectId };
}

function reconcileLabels(a: Args, spec: Archetype, repos: string[]): void {
  for (const repo of repos) {
    const live = snapshotLabels(repo);
    const plan = buildPlan({ projectId: "", fields: [], labels: live }, spec, { prune: a.prune });
    console.log(`\n${formatPlan({ ...plan, selects: [], fieldOps: [], warnings: [] }, { project: `labels ${repo}`, prune: a.prune })}`);
    if (!a.apply) continue;
    if (a.prune && plan.labelOps.some((o) => o.safety === "destructive") && !a.confirm) {
      throw new Error(`refusing destructive label prune on ${repo}: --prune requires --confirm`);
    }
    const desiredByName = new Map(spec.labels.map((l) => [l.name, l]));
    for (const op of plan.labelOps) {
      if (op.kind === "delete") {
        console.warn(`  - ${repo}: delete label "${op.target}"`);
        deleteLabel(repo, op.target);
      } else {
        const l = desiredByName.get(op.target)!;
        console.log(`  ${op.kind === "create" ? "+" : "~"} ${repo}: ${op.target}`);
        applyLabel(repo, l.name, l.color, l.description ?? "");
      }
    }
  }
}

async function main(): Promise<void> {
  const a = parseArgs(process.argv.slice(2));
  const { spec, repos } = await resolveSpecAndRepos(a);

  const boardMode = a.project !== undefined && a.owner !== undefined;
  if (!boardMode && repos.length === 0) {
    throw new Error("nothing to do: provide --project+--owner (board) and/or --repo/--manifest (labels)");
  }

  let projectId: string | undefined;
  let boardPlan: Plan | undefined;
  if (boardMode) {
    const r = await reconcileBoard(a, spec);
    projectId = r.projectId;
    boardPlan = r.plan;
  }

  if (repos.length) reconcileLabels(a, spec, repos);

  // After any board apply, re-resolve live state into the gitignored lockfile
  // (names→IDs cache + spec hash). Live state is the ground truth; this is audit.
  if (boardMode && a.apply && projectId) {
    const fresh = snapshotBoard(projectId);
    const slug = `${a.owner}-project-${a.project}`;
    const path = await writeLockfile(resolveLockdir(a.lockdir), slug, projectId, boardPlan!.specHash, fresh);
    console.log(`\nlockfile: ${path} (spec ${boardPlan!.specHash})`);
  }

  if (a.apply) {
    console.log("\n✓ reconcile complete");
  } else if (boardPlan && isEmpty(boardPlan) && !repos.length) {
    console.log("\n(dry run — board already converged)");
  } else {
    console.log("\n(dry run — re-run with --apply to execute)");
  }
}

// Only run as a CLI when invoked directly — importing this module (e.g. from the
// orchestration tests) must not execute `main()` against the test runner's argv.
if (import.meta.main) {
  main().catch((e) => {
    console.error(`\n✗ ${e instanceof Error ? e.message : String(e)}`);
    process.exit(1);
  });
}
