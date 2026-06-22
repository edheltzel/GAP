// Pure-logic tests for the reconcile core. NO network, NO gh, NO filesystem —
// this is the §6 safety net proving the dangerous option-merge behaviour offline.
// Run: `bun test reconcile-core.test.ts`

import { test, expect, describe } from "bun:test";
import {
  diffOptions,
  diffLabels,
  diffFields,
  buildPlan,
  planOps,
  hasDestructive,
  isEmpty,
  specHash,
  type LiveOption,
  type LiveLabel,
  type LiveField,
  type Snapshot,
} from "./reconcile-core.ts";
import type { Archetype, SelectOption, Label } from "../templates/types.ts";

const opt = (name: string, id = `id-${name}`, color = "GRAY", description = ""): LiveOption => ({ id, name, color, description });
const want = (name: string, color: SelectOption["color"] = "GRAY", description?: string): SelectOption => ({ name, color, ...(description ? { description } : {}) });

describe("diffOptions — the §6 footgun", () => {
  test("union NEVER drops an existing option (non-prune keeps extras with their id)", () => {
    const live = [opt("Todo"), opt("In Progress"), opt("Done"), opt("Wontfix")];
    const desired = [want("Todo"), want("In Progress"), want("Done")]; // Wontfix not desired
    const { union, removals } = diffOptions(live, desired, { prune: false });
    const names = union.map((o) => o.name);
    expect(names).toContain("Wontfix"); // preserved — a subset payload would wipe it
    expect(removals).toEqual([]);
    // Every preserved option carries its node id forward (no churn → no orphaning).
    expect(union.find((o) => o.name === "Wontfix")?.id).toBe("id-Wontfix");
    expect(union.find((o) => o.name === "Todo")?.id).toBe("id-Todo");
  });

  test("adding an option is classified SAFE and has no id (created fresh)", () => {
    const live = [opt("Todo"), opt("Done")];
    const desired = [want("Todo"), want("Ready", "BLUE"), want("Done")];
    const { ops, union } = diffOptions(live, desired, { prune: false });
    const add = ops.find((o) => o.target === "Ready");
    expect(add?.kind).toBe("create");
    expect(add?.safety).toBe("safe");
    expect(union.find((o) => o.name === "Ready")?.id).toBeUndefined();
    // Existing options keep their ids.
    expect(union.find((o) => o.name === "Todo")?.id).toBe("id-Todo");
  });

  test("removing an option is DESTRUCTIVE and only happens under prune", () => {
    const live = [opt("Todo"), opt("Stale")];
    const desired = [want("Todo")];
    const noPrune = diffOptions(live, desired, { prune: false });
    expect(noPrune.removals).toEqual([]); // suppressed
    expect(noPrune.ops.some((o) => o.kind === "delete")).toBe(false);
    expect(noPrune.union.map((o) => o.name)).toContain("Stale"); // kept

    const pruned = diffOptions(live, desired, { prune: true });
    expect(pruned.removals).toEqual(["Stale"]);
    const del = pruned.ops.find((o) => o.kind === "delete");
    expect(del?.safety).toBe("destructive");
    expect(pruned.union.map((o) => o.name)).not.toContain("Stale"); // dropped from union
  });

  test("rename = remove + add: destructive removal of old, safe add of new", () => {
    const live = [opt("Backlog"), opt("Done")];
    const desired = [want("Icebox"), want("Done")]; // Backlog → Icebox
    const { ops } = diffOptions(live, desired, { prune: true });
    const oldRemoval = ops.find((o) => o.target === "Backlog");
    const newAdd = ops.find((o) => o.target === "Icebox");
    expect(oldRemoval?.kind).toBe("delete");
    expect(oldRemoval?.safety).toBe("destructive");
    expect(newAdd?.kind).toBe("create");
    expect(newAdd?.safety).toBe("safe");
  });

  test("colour/description drift is a SAFE update with id preserved", () => {
    const live = [opt("Todo", "id-Todo", "GRAY", "old")];
    const desired = [want("Todo", "BLUE", "new")];
    const { ops, union } = diffOptions(live, desired, { prune: false });
    const upd = ops.find((o) => o.target === "Todo");
    expect(upd?.kind).toBe("update");
    expect(upd?.safety).toBe("safe");
    expect(union[0].id).toBe("id-Todo"); // edited in place, no churn
  });

  test("converged options produce zero ops (idempotent), regardless of live order", () => {
    const live = [opt("Done"), opt("Todo")]; // different order than desired
    const desired = [want("Todo"), want("Done")];
    const { ops, removals } = diffOptions(live, desired, { prune: true });
    expect(ops).toEqual([]);
    expect(removals).toEqual([]);
  });
});

describe("diffLabels", () => {
  const live: LiveLabel[] = [
    { name: "type:bug", color: "d4c5f9", description: "" },
    { name: "stale", color: "ffffff", description: "old" },
  ];
  const desired: Label[] = [
    { name: "type:bug", color: "d4c5f9" },
    { name: "type:feature", color: "d4c5f9" },
  ];

  test("create missing, update drift, never delete without prune", () => {
    const ops = diffLabels(live, desired, { prune: false });
    expect(ops.find((o) => o.target === "type:feature")?.kind).toBe("create");
    expect(ops.some((o) => o.kind === "delete")).toBe(false); // 'stale' not deleted
  });

  test("colour normalisation: #-prefixed and case differences are not drift", () => {
    const ops = diffLabels(
      [{ name: "type:bug", color: "D4C5F9", description: "" }],
      [{ name: "type:bug", color: "#d4c5f9" }],
      { prune: false },
    );
    expect(ops).toEqual([]);
  });

  test("prune deletes extras as destructive", () => {
    const ops = diffLabels(live, desired, { prune: true });
    const del = ops.find((o) => o.target === "stale");
    expect(del?.kind).toBe("delete");
    expect(del?.safety).toBe("destructive");
  });
});

describe("diffFields", () => {
  const spec: Archetype = {
    key: "t",
    title: "T",
    status: [want("Todo")],
    fields: [
      { name: "Priority", type: "SINGLE_SELECT", options: [want("P0", "RED")] },
      { name: "Estimate", type: "NUMBER" },
    ],
    labels: [],
    views: [],
  };

  test("create-if-missing for plain and single-select fields", () => {
    const ops = diffFields([], spec, { prune: false }).ops;
    expect(ops.find((o) => o.target === "Estimate")?.kind).toBe("create");
    expect(ops.find((o) => o.target === "Priority")?.kind).toBe("create");
  });

  test("type conflict is a warning, not an auto-fix", () => {
    const live: LiveField[] = [{ id: "f1", name: "Estimate", dataType: "TEXT" }];
    const { ops, warnings } = diffFields(live, spec, { prune: false });
    expect(ops.some((o) => o.target === "Estimate")).toBe(false);
    expect(warnings.some((w) => w.includes("Estimate") && w.includes("TEXT"))).toBe(true);
  });

  test("built-in fields are never flagged or deleted", () => {
    const live: LiveField[] = [{ id: "s", name: "Status", dataType: "SINGLE_SELECT" }, { id: "t", name: "Title", dataType: "TEXT" }];
    const { ops, warnings } = diffFields(live, spec, { prune: true });
    expect(ops.some((o) => o.target === "Status" || o.target === "Title")).toBe(false);
    expect(warnings.some((w) => w.includes("Status") || w.includes("Title"))).toBe(false);
  });

  test("extra field flagged without prune, deleted (destructive) under prune", () => {
    const live: LiveField[] = [{ id: "x", name: "Legacy", dataType: "TEXT" }];
    expect(diffFields(live, spec, { prune: false }).warnings.some((w) => w.includes("Legacy"))).toBe(true);
    const del = diffFields(live, spec, { prune: true }).ops.find((o) => o.target === "Legacy");
    expect(del?.kind).toBe("delete");
    expect(del?.safety).toBe("destructive");
  });
});

describe("buildPlan — end to end on a snapshot", () => {
  const spec: Archetype = {
    key: "solo",
    title: "Solo",
    status: [want("Backlog"), want("In Progress", "YELLOW"), want("Done", "GREEN")],
    fields: [{ name: "Priority", type: "SINGLE_SELECT", options: [want("P0", "RED"), want("P1", "ORANGE")] }],
    labels: [{ name: "type:bug", color: "d4c5f9" }],
    views: [],
  };

  const converged: Snapshot = {
    projectId: "PVT_1",
    fields: [
      { id: "status", name: "Status", dataType: "SINGLE_SELECT", options: [opt("Backlog"), opt("In Progress", "id-ip", "YELLOW"), opt("Done", "id-done", "GREEN")] },
      { id: "prio", name: "Priority", dataType: "SINGLE_SELECT", options: [opt("P0", "id-p0", "RED"), opt("P1", "id-p1", "ORANGE")] },
    ],
    labels: [{ name: "type:bug", color: "d4c5f9", description: "" }],
  };

  test("re-apply of an already-converged spec yields an EMPTY plan (idempotent)", () => {
    const plan = buildPlan(converged, spec, { prune: true });
    expect(isEmpty(plan)).toBe(true);
    expect(hasDestructive(plan)).toBe(false);
    expect(planOps(plan)).toEqual([]);
  });

  test("fresh board: every field/option/label is a safe create", () => {
    const empty: Snapshot = {
      projectId: "PVT_2",
      fields: [{ id: "status", name: "Status", dataType: "SINGLE_SELECT", options: [opt("Todo"), opt("In Progress"), opt("Done")] }],
    labels: [],
    };
    const plan = buildPlan(empty, spec, { prune: false });
    expect(isEmpty(plan)).toBe(false);
    expect(hasDestructive(plan)).toBe(false); // creates only
    // Priority field must be created before its options apply.
    const prioSelect = plan.selects.find((s) => s.fieldName === "Priority");
    expect(prioSelect?.exists).toBe(false);
    expect(plan.fieldOps.find((o) => o.target === "Priority")?.kind).toBe("create");
  });

  test("Status field is built-in and never appears in field create ops", () => {
    const plan = buildPlan(converged, spec, { prune: false });
    expect(plan.fieldOps.some((o) => o.target === "Status")).toBe(false);
    expect(plan.selects.find((s) => s.fieldName === "Status")?.builtin).toBe(true);
  });
});

describe("specHash", () => {
  const a: Archetype = { key: "k", title: "T", status: [want("Todo")], fields: [], labels: [], views: [] };
  test("deterministic across calls", () => {
    expect(specHash(a)).toBe(specHash(structuredClone(a)));
  });
  test("changes when the reconciled surface changes", () => {
    const b = structuredClone(a);
    b.status.push(want("Done"));
    expect(specHash(b)).not.toBe(specHash(a));
  });
  test("ignores views (this engine does not reconcile views)", () => {
    const b = structuredClone(a);
    b.views.push({ name: "Roadmap", layout: "ROADMAP" });
    expect(specHash(b)).toBe(specHash(a));
  });
});
