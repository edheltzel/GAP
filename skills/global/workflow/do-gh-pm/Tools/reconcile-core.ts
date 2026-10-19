// Reconcile engine — PURE CORE (no `gh`, no filesystem, no Bun APIs).
//
// This is the highest-risk logic in the skill (skill plan §6). Everything here
// is a deterministic function of (live snapshot + desired spec) → typed plan of
// operations. Because it touches no I/O, the dangerous option-merge behaviour is
// unit-testable offline without ever mutating a real GitHub board.
//
// The §6 safety invariants this module enforces:
//   1. `updateProjectV2Field` is a FULL-LIST REPLACE. We always build the option
//      payload as the MERGED UNION of live ∪ desired, carrying each existing
//      option's node `id` forward, so a write never drops an option it didn't
//      mean to and never churns the IDs of options it keeps (preserving item
//      assignments). A subset payload would silently wipe the rest.
//   2. Adding an option / editing its colour or description is SAFE (the id is
//      preserved → no assignment loss).
//   3. Removing an option (and therefore renaming, which a name-keyed diff sees
//      as remove+add) is DESTRUCTIVE — it is only ever included under prune and
//      is reported with the count of item assignments it resets.
//   4. Fields and projects are never auto-deleted — extras are flagged and only
//      removed behind prune.

import type { Archetype, Field, Label, SelectOption, SelectColor, FieldType } from "../templates/types.ts";

// ──────────────────────────── Live snapshot ────────────────────────────
// The shape the thin IO shell collects from `gh api graphql` / `gh label list`.
// Names are the stable key; node IDs are ephemeral and re-resolved every run.

export interface LiveOption {
  id: string;
  name: string;
  color: string;
  description: string;
}

export interface LiveField {
  id: string;
  name: string;
  dataType: FieldType;
  // Present only for SINGLE_SELECT fields.
  options?: LiveOption[];
}

export interface LiveLabel {
  name: string;
  color: string;
  description: string;
}

export interface Snapshot {
  projectId: string;
  fields: LiveField[];
  labels: LiveLabel[];
}

// ──────────────────────────── Plan model ────────────────────────────

export type OpKind = "create" | "update" | "delete";
export type Safety = "safe" | "destructive";
export type Resource = "field" | "option" | "label";

export interface Op {
  kind: OpKind;
  resource: Resource;
  target: string; // e.g. `Status / In Review` for an option
  safety: Safety;
  detail: string;
}

// The exact id-preserving option list to send to `updateProjectV2Field`.
export interface OptionInput {
  id?: string; // present → option preserved in place (no churn); absent → created fresh
  name: string;
  color: SelectColor;
  description: string;
}

export interface SelectPlan {
  fieldName: string;
  builtin: boolean; // Status is built-in: reconciled in place, never created
  exists: boolean; // false → the field itself must be created before options apply
  fieldId?: string;
  union: OptionInput[]; // merged union to send (prune already applied)
  removals: string[]; // option names dropped this run (destructive; empty unless prune)
  ops: Op[]; // per-option create/update/delete classification
}

export interface Plan {
  fieldOps: Op[]; // non-select field create/delete
  selects: SelectPlan[];
  labelOps: Op[];
  warnings: string[];
  specHash: string;
}

// ──────────────────────────── Helpers ────────────────────────────

// GitHub's built-in project fields — never created, never flagged, never pruned.
const BUILTIN_FIELDS = new Set([
  "Title",
  "Assignees",
  "Status",
  "Labels",
  "Linked pull requests",
  "Milestone",
  "Repository",
  "Reviewers",
  "Parent issue",
  "Sub-issues progress",
  // Auto-managed system fields present on every board — never created/flagged/pruned.
  "Created",
  "Updated",
  "Closed",
]);

const norm = (s: string | undefined): string => (s ?? "").trim();
const normColor = (c: string | undefined): string => norm(c).replace(/^#/, "").toLowerCase();

function byName<T extends { name: string }>(items: T[] | undefined): Map<string, T> {
  const m = new Map<string, T>();
  for (const it of items ?? []) m.set(it.name, it);
  return m;
}

// ──────────────── Single-select colour + option serialization (pure) ────────────────
// These were in the IO shell; they are pure string/enum logic and the exact point
// where id-preservation becomes the wire payload, so they live and are tested in
// the core (a regression dropping `id:` is the §6 footgun).

const SELECT_COLORS = new Set<SelectColor>(["GRAY", "BLUE", "GREEN", "YELLOW", "ORANGE", "RED", "PINK", "PURPLE"]);

// Normalize a colour token coming off a LIVE board (untrusted) to a valid enum.
// Unknown tokens are surfaced as a warning and defaulted — never thrown, so one
// odd colour cannot abort a whole field reconcile mid-`.map()` (F5).
export function normalizeLiveColor(c: string, optName: string): { color: SelectColor; warning?: string } {
  const up = norm(c).toUpperCase();
  if (SELECT_COLORS.has(up as SelectColor)) return { color: up as SelectColor };
  return {
    color: "GRAY",
    warning: `option "${optName}": unrecognized live colour "${c}" — defaulted to GRAY in the preserved union (review before apply)`,
  };
}

export function colorLiteral(c: string): string {
  if (!SELECT_COLORS.has(c as SelectColor)) throw new Error(`invalid single-select colour "${c}" (not in the 8-value enum)`);
  return c; // bare enum token for the GraphQL mutation
}

export function optionLiteral(o: OptionInput): string {
  const parts: string[] = [];
  if (o.id) parts.push(`id:${JSON.stringify(o.id)}`);
  parts.push(`name:${JSON.stringify(o.name)}`);
  parts.push(`color:${colorLiteral(o.color)}`);
  parts.push(`description:${JSON.stringify(o.description)}`);
  return `{${parts.join(",")}}`;
}

// The single-select fields a board should have: the built-in Status (vocabulary
// from `archetype.status`) plus any SINGLE_SELECT field declared in the archetype.
export interface DesiredSelect {
  name: string;
  builtin: boolean;
  options: SelectOption[];
}

export function desiredSelects(spec: Archetype): DesiredSelect[] {
  const out: DesiredSelect[] = [{ name: "Status", builtin: true, options: spec.status }];
  for (const f of spec.fields) {
    if (f.type === "SINGLE_SELECT") {
      out.push({ name: f.name, builtin: false, options: f.options ?? [] });
    }
  }
  return out;
}

// Non-single-select fields that are create-if-missing (TEXT/NUMBER/DATE/ITERATION).
export function desiredPlainFields(spec: Archetype): Field[] {
  return spec.fields.filter((f) => f.type !== "SINGLE_SELECT");
}

// ──────────────────── §6 core: single-select option diff ────────────────────
//
// Produces the id-preserving union to send AND the op classification. This is
// the function that must never drop an existing assignment.

// An explicit, deliberate option rename. A name-keyed diff cannot tell a rename
// from a remove+add, so renames are NEVER inferred — only an explicit mapping
// triggers the id-preserving in-place path below. `from` is a live option name,
// `to` is the desired (new) name.
export interface OptionRename {
  from: string;
  to: string;
}

// A rename map must be a bijection, and a target that already exists live must
// itself be renamed away. Otherwise the full-list replace silently drops that
// option's id (Todo→Backlog while Backlog exists keeps Todo's id and loses
// id-backlog) and classifies the write as safe.
function rejectUnsafeRenames(renames: OptionRename[], live: Map<string, LiveOption>): void {
  const froms = new Set<string>();
  const tos = new Set<string>();
  for (const r of renames) {
    if (froms.has(r.from) || tos.has(r.to)) {
      throw new Error(
        `rename "${r.from}"→"${r.to}": rename map is not bijective — each live option can be renamed once, onto a distinct target`,
      );
    }
    froms.add(r.from);
    tos.add(r.to);
  }
  for (const r of renames) {
    if (live.has(r.to) && !froms.has(r.to)) {
      const id = live.get(r.to)!.id;
      throw new Error(
        `rename "${r.from}"→"${r.to}": target collides with existing option "${r.to}" (${id}); refusing to drop that id`,
      );
    }
  }
}

export function diffOptions(
  live: LiveOption[] | undefined,
  desired: SelectOption[],
  opts: { prune: boolean; renames?: OptionRename[] },
): { union: OptionInput[]; removals: string[]; ops: Op[]; warnings: string[] } {
  const liveMap = byName<LiveOption>(live);
  const desiredMap = byName<SelectOption>(desired);
  const ops: Op[] = [];
  const union: OptionInput[] = [];
  const removals: string[] = [];
  const warnings: string[] = [];

  // Explicit renames: index by new name (the desired option that inherits the
  // live id) and track which live names are consumed so they are not also seen
  // as removals. An unusable mapping is warned and otherwise ignored. Usable
  // mappings are rejected before the union is built when they would drop an id.
  const renames = opts.renames ?? [];
  const renameByTo = new Map<string, string>();
  const renameFroms = new Set<string>();
  const usable: OptionRename[] = [];
  for (const r of renames) {
    if (!liveMap.has(r.from)) {
      warnings.push(`rename "${r.from}"→"${r.to}": no live option named "${r.from}" — skipped`);
      continue;
    }
    if (!desiredMap.has(r.to)) {
      warnings.push(`rename "${r.from}"→"${r.to}": "${r.to}" is not in the desired spec — skipped`);
      continue;
    }
    usable.push(r);
  }
  rejectUnsafeRenames(usable, liveMap);
  for (const r of usable) {
    renameByTo.set(r.to, r.from);
    renameFroms.add(r.from);
  }

  // 1. Desired options, in desired order. An explicit rename carries the live
  //    id of `from` forward under the new name (in-place, assignment-preserving);
  //    otherwise a name match carries its id (in-place edit); otherwise create.
  for (const d of desired) {
    const renamedFrom = renameByTo.get(d.name);
    const renameSource = renamedFrom ? liveMap.get(renamedFrom) : undefined;
    const existing = renameSource ?? liveMap.get(d.name);
    const desc = norm(d.description);
    union.push({
      ...(existing ? { id: existing.id } : {}),
      name: d.name,
      color: d.color,
      description: desc,
    });
    if (renameSource) {
      // In-place rename: same node id, new name → GitHub keeps every assignment.
      ops.push({
        kind: "update",
        resource: "option",
        target: d.name,
        safety: "safe",
        detail: `rename option "${renamedFrom}" → "${d.name}" in place (id preserved, assignments kept)`,
      });
    } else if (!existing) {
      ops.push({
        kind: "create",
        resource: "option",
        target: d.name,
        safety: "safe",
        detail: `add option "${d.name}" (${d.color})`,
      });
    } else {
      const colourChanged = normColor(existing.color) !== normColor(d.color);
      const descChanged = norm(existing.description) !== desc;
      if (colourChanged || descChanged) {
        const changes = [
          ...(colourChanged ? [`colour ${existing.color}→${d.color}`] : []),
          ...(descChanged ? [`description changed`] : []),
        ];
        ops.push({
          kind: "update",
          resource: "option",
          target: d.name,
          safety: "safe",
          detail: `update "${d.name}": ${changes.join(", ")}`,
        });
      }
    }
  }

  // 2. Extra live options (not desired, not consumed by a rename). Preserved
  //    verbatim unless pruning. Removal is the ONLY destructive option op — it
  //    orphans item assignments.
  for (const l of live ?? []) {
    if (desiredMap.has(l.name) || renameFroms.has(l.name)) continue;
    if (opts.prune) {
      removals.push(l.name);
      ops.push({
        kind: "delete",
        resource: "option",
        target: l.name,
        safety: "destructive",
        detail: `remove option "${l.name}" — resets item assignments referencing it`,
      });
    } else {
      // Keep it in the union (id preserved) so the full-replace doesn't wipe it.
      // Live colours are untrusted → normalize, warning (not throwing) on unknowns.
      const { color, warning } = normalizeLiveColor(l.color, l.name);
      if (warning) warnings.push(warning);
      union.push({ id: l.id, name: l.name, color, description: norm(l.description) });
    }
  }

  return { union, removals, ops, warnings };
}

// ──────────────────────────── Label diff ────────────────────────────
// Labels have no assignment-orphan hazard: create/update are safe (idempotent
// via `gh label create --force`); deletes only happen under prune.

export function diffLabels(
  live: LiveLabel[],
  desired: Label[],
  opts: { prune: boolean },
): Op[] {
  const liveMap = byName<LiveLabel>(live);
  const desiredMap = byName<Label>(desired);
  const ops: Op[] = [];

  for (const d of desired) {
    const existing = liveMap.get(d.name);
    const desc = norm(d.description);
    if (!existing) {
      ops.push({ kind: "create", resource: "label", target: d.name, safety: "safe", detail: `create label "${d.name}" (#${normColor(d.color)})` });
    } else if (normColor(existing.color) !== normColor(d.color) || norm(existing.description) !== desc) {
      ops.push({ kind: "update", resource: "label", target: d.name, safety: "safe", detail: `update label "${d.name}" (colour/description drift)` });
    }
  }

  if (opts.prune) {
    for (const l of live) {
      if (!desiredMap.has(l.name)) {
        ops.push({ kind: "delete", resource: "label", target: l.name, safety: "destructive", detail: `delete label "${l.name}"` });
      }
    }
  }

  return ops;
}

// ──────────────────────────── Field diff ────────────────────────────
// Plain (non-select) fields: create-if-missing. Type conflicts and extras are
// flagged (warnings); extras are only deleted under prune. Built-ins are inert.

export function diffFields(
  liveFields: LiveField[],
  spec: Archetype,
  opts: { prune: boolean },
): { ops: Op[]; warnings: string[] } {
  const liveMap = byName<LiveField>(liveFields);
  const ops: Op[] = [];
  const warnings: string[] = [];

  // create-if-missing for plain fields; type-conflict warning if present-but-wrong.
  for (const f of desiredPlainFields(spec)) {
    const existing = liveMap.get(f.name);
    if (!existing) {
      ops.push({ kind: "create", resource: "field", target: f.name, safety: "safe", detail: `create ${f.type} field "${f.name}"` });
    } else if (existing.dataType !== f.type) {
      warnings.push(`field "${f.name}" exists as ${existing.dataType} but spec wants ${f.type} — the API cannot change a field's type; rename or delete it manually.`);
    }
  }

  // A single-select field declared in the spec but missing live must be created
  // before its options can apply (the option apply path checks SelectPlan.exists).
  for (const ds of desiredSelects(spec)) {
    if (ds.builtin) continue;
    const existing = liveMap.get(ds.name);
    if (!existing) {
      ops.push({ kind: "create", resource: "field", target: ds.name, safety: "safe", detail: `create SINGLE_SELECT field "${ds.name}"` });
    } else if (existing.dataType !== "SINGLE_SELECT") {
      warnings.push(`field "${ds.name}" exists as ${existing.dataType} but spec wants SINGLE_SELECT — resolve manually.`);
    }
  }

  // Extra live fields: never auto-deleted. Flagged always; deleted only on prune.
  const desiredNames = new Set<string>([
    ...desiredPlainFields(spec).map((f) => f.name),
    ...desiredSelects(spec).map((s) => s.name),
  ]);
  for (const l of liveFields) {
    if (BUILTIN_FIELDS.has(l.name) || desiredNames.has(l.name)) continue;
    if (opts.prune) {
      ops.push({ kind: "delete", resource: "field", target: l.name, safety: "destructive", detail: `delete field "${l.name}" and all its data` });
    } else {
      warnings.push(`extra field "${l.name}" is present live but not in the spec — left untouched (use --prune --confirm to remove).`);
    }
  }

  return { ops, warnings };
}

// ──────────────────────────── Plan assembly ────────────────────────────

// A rename scoped to a single-select field, as supplied by `--rename Field:Old=New`.
export interface FieldRename extends OptionRename {
  field: string;
}

export function buildPlan(snapshot: Snapshot, spec: Archetype, opts: { prune: boolean; renames?: FieldRename[] }): Plan {
  const liveFieldMap = byName<LiveField>(snapshot.fields);
  const renames = opts.renames ?? [];
  const selectNames = new Set(desiredSelects(spec).map((s) => s.name));
  const optionWarnings: string[] = [];

  // A rename targeting a field that is not a reconciled single-select can never apply.
  for (const r of renames) {
    if (!selectNames.has(r.field)) {
      optionWarnings.push(`rename "${r.field}:${r.from}=${r.to}": "${r.field}" is not a reconciled single-select field — skipped`);
    }
  }

  const selects: SelectPlan[] = [];
  for (const ds of desiredSelects(spec)) {
    const liveField = liveFieldMap.get(ds.name);
    const exists = !!liveField && liveField.dataType === "SINGLE_SELECT";
    const fieldRenames = renames.filter((r) => r.field === ds.name).map((r) => ({ from: r.from, to: r.to }));
    const { union, removals, ops, warnings } = diffOptions(liveField?.options, ds.options, { prune: opts.prune, renames: fieldRenames });
    optionWarnings.push(...warnings.map((w) => `${ds.name}: ${w}`));
    selects.push({
      fieldName: ds.name,
      builtin: ds.builtin,
      exists,
      fieldId: liveField?.id,
      union,
      removals,
      ops,
    });
  }

  const { ops: fieldOps, warnings } = diffFields(snapshot.fields, spec, opts);
  const labelOps = diffLabels(snapshot.labels, spec.labels, opts);

  return {
    fieldOps,
    selects,
    labelOps,
    warnings: [...warnings, ...optionWarnings],
    specHash: specHash(spec),
  };
}

// All mutating ops across the plan, in apply order (create fields → options → labels).
export function planOps(plan: Plan): Op[] {
  return [...plan.fieldOps, ...plan.selects.flatMap((s) => s.ops), ...plan.labelOps];
}

export function hasDestructive(plan: Plan): boolean {
  return planOps(plan).some((o) => o.safety === "destructive");
}

// The §6 safety gate, as a pure function so a boolean inversion is caught by a
// unit test rather than shipping a board-wiping engine. A destructive apply
// (--prune carrying any destructive op) requires --confirm; throws otherwise.
export function destructiveGate(plan: Plan, opts: { prune: boolean; confirm: boolean }): void {
  if (opts.prune && hasDestructive(plan) && !opts.confirm) {
    throw new Error("refusing destructive apply: --prune requires --confirm");
  }
}

export function isEmpty(plan: Plan): boolean {
  return planOps(plan).length === 0;
}

// ──────────────────────────── Spec hash (audit) ────────────────────────────
// Deterministic FNV-1a over the reconciled surface (status/fields/labels). Views
// are excluded — this engine does not reconcile views (API has no view mutation).

export function specHash(spec: Archetype): string {
  const canonical = JSON.stringify({
    status: spec.status.map((s) => [s.name, s.color, norm(s.description)]),
    fields: spec.fields.map((f) => [f.name, f.type, (f.options ?? []).map((o) => [o.name, o.color, norm(o.description)])]),
    labels: spec.labels.map((l) => [l.name, normColor(l.color), norm(l.description)]),
  });
  let h = 0x811c9dc5;
  for (let i = 0; i < canonical.length; i++) {
    h ^= canonical.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return (h >>> 0).toString(16).padStart(8, "0");
}

// ──────────────────────────── Plan formatting ────────────────────────────
// terraform-plan-style human output for plan mode. Pure string building.

const SIGIL: Record<OpKind, string> = { create: "+", update: "~", delete: "-" };

export function formatPlan(plan: Plan, ctx: { project: string; prune: boolean }): string {
  const lines: string[] = [];
  lines.push(`Reconcile plan for ${ctx.project}  (spec ${plan.specHash})`);
  lines.push("");

  const all = planOps(plan);
  if (all.length === 0) {
    lines.push("  No changes. Board matches the resolved spec (idempotent no-op).");
  } else {
    for (const s of plan.selects) {
      if (s.ops.length === 0) continue;
      const tag = s.exists ? "" : "  [field absent — will be created first]";
      lines.push(`  field ${s.fieldName}${tag}`);
      for (const op of s.ops) lines.push(`    ${SIGIL[op.kind]} ${op.detail}${op.safety === "destructive" ? "   ⚠ DESTRUCTIVE" : ""}`);
    }
    for (const op of plan.fieldOps) lines.push(`  ${SIGIL[op.kind]} field ${op.detail}${op.safety === "destructive" ? "   ⚠ DESTRUCTIVE" : ""}`);
    for (const op of plan.labelOps) lines.push(`  ${SIGIL[op.kind]} ${op.detail}${op.safety === "destructive" ? "   ⚠ DESTRUCTIVE" : ""}`);
  }

  const counts = all.reduce(
    (acc, o) => ((acc[o.kind] += 1), acc),
    { create: 0, update: 0, delete: 0 } as Record<OpKind, number>,
  );
  lines.push("");
  lines.push(`Plan: ${counts.create} to create, ${counts.update} to update, ${counts.delete} to delete.`);

  if (plan.warnings.length) {
    lines.push("");
    lines.push("Warnings:");
    for (const w of plan.warnings) lines.push(`  ! ${w}`);
  }

  const destructive = all.filter((o) => o.safety === "destructive");
  if (destructive.length && !ctx.prune) {
    lines.push("");
    lines.push(`${destructive.length} destructive change(s) suppressed — re-run with --prune --confirm to apply them.`);
  }

  return lines.join("\n");
}
