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
  destructiveGate,
  isEmpty,
  specHash,
  optionLiteral,
  colorLiteral,
  normalizeLiveColor,
  type LiveOption,
  type LiveLabel,
  type LiveField,
  type Snapshot,
  type Plan,
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

describe("diffOptions — explicit id-preserving rename (F1, the blocker)", () => {
  test("an explicit rename carries the live id to the new name — no removal, no reset", () => {
    const live = [opt("Backlog"), opt("Done")];
    const desired = [want("Icebox"), want("Done")]; // spec now says Icebox, not Backlog
    const { union, removals, ops, warnings } = diffOptions(live, desired, {
      prune: true,
      renames: [{ from: "Backlog", to: "Icebox" }],
    });
    // The new name inherits Backlog's node id → GitHub preserves every assignment.
    expect(union.find((o) => o.name === "Icebox")?.id).toBe("id-Backlog");
    // Nothing is removed and nothing is destructive — this is the whole point.
    expect(removals).toEqual([]);
    expect(ops.some((o) => o.safety === "destructive")).toBe(false);
    const rename = ops.find((o) => o.target === "Icebox");
    expect(rename?.kind).toBe("update");
    expect(rename?.safety).toBe("safe");
    expect(rename?.detail).toContain("rename");
    // "Backlog" must NOT resurface as an extra/removal.
    expect(union.some((o) => o.name === "Backlog")).toBe(false);
    expect(warnings).toEqual([]);
  });

  test("without the rename mapping the SAME edit is a destructive remove+add (default unchanged)", () => {
    const live = [opt("Backlog"), opt("Done")];
    const desired = [want("Icebox"), want("Done")];
    const { removals, ops } = diffOptions(live, desired, { prune: true });
    expect(removals).toEqual(["Backlog"]);
    expect(ops.find((o) => o.target === "Backlog")?.safety).toBe("destructive");
  });

  test("a rename whose source is missing live is warned and skipped (not a silent create churn)", () => {
    const live = [opt("Done")];
    const desired = [want("Icebox"), want("Done")];
    const { union, warnings } = diffOptions(live, desired, {
      prune: false,
      renames: [{ from: "Backlog", to: "Icebox" }],
    });
    expect(warnings.some((w) => w.includes("Backlog") && w.includes("skipped"))).toBe(true);
    // Falls through to a normal create — no id carried (there was nothing to carry).
    expect(union.find((o) => o.name === "Icebox")?.id).toBeUndefined();
  });

  test("a rename whose target is not in the spec is warned and skipped", () => {
    const live = [opt("Backlog"), opt("Done")];
    const desired = [want("Backlog"), want("Done")]; // Icebox not desired
    const { warnings } = diffOptions(live, desired, {
      prune: false,
      renames: [{ from: "Backlog", to: "Icebox" }],
    });
    expect(warnings.some((w) => w.includes("Icebox") && w.includes("skipped"))).toBe(true);
  });

  test("rename onto an existing option is rejected before the union can drop that id", () => {
    const live = [opt("Todo", "id-todo"), opt("Backlog", "id-backlog")];
    const desired = [want("Backlog"), want("Done")];
    expect(() => diffOptions(live, desired, {
      prune: false,
      renames: [{ from: "Todo", to: "Backlog" }],
    })).toThrow(/collides with existing option "Backlog"/);
  });

  test("a non-bijective rename map is rejected before the union is built", () => {
    const live = [opt("Todo"), opt("Icebox"), opt("Done")];
    const desired = [want("Backlog"), want("Ready"), want("Done")];
    expect(() => diffOptions(live, desired, {
      prune: false,
      renames: [
        { from: "Todo", to: "Backlog" },
        { from: "Icebox", to: "Backlog" },
      ],
    })).toThrow(/not bijective/);
    expect(() => diffOptions(live, desired, {
      prune: false,
      renames: [
        { from: "Todo", to: "Backlog" },
        { from: "Todo", to: "Ready" },
      ],
    })).toThrow(/not bijective/);
  });

  test("buildPlan threads --rename per field and preserves assignments end to end", () => {
    const spec = { key: "k", title: "T", status: [want("Icebox"), want("Done", "GREEN")], fields: [], labels: [], views: [] };
    const snap: Snapshot = {
      projectId: "PVT",
      fields: [{ id: "status", name: "Status", dataType: "SINGLE_SELECT", options: [opt("Backlog"), opt("Done", "id-done", "GREEN")] }],
      labels: [],
    };
    const plan = buildPlan(snap, spec, { prune: true, renames: [{ field: "Status", from: "Backlog", to: "Icebox" }] });
    expect(hasDestructive(plan)).toBe(false); // a rename is safe
    const status = plan.selects.find((s) => s.fieldName === "Status");
    expect(status?.union.find((o) => o.name === "Icebox")?.id).toBe("id-Backlog");
    expect(status?.removals).toEqual([]);
  });

  test("buildPlan warns when --rename targets a non-single-select field", () => {
    const spec = { key: "k", title: "T", status: [want("Todo")], fields: [], labels: [], views: [] };
    const snap: Snapshot = { projectId: "PVT", fields: [], labels: [] };
    const plan = buildPlan(snap, spec, { prune: false, renames: [{ field: "Nope", from: "A", to: "B" }] });
    expect(plan.warnings.some((w) => w.includes("Nope") && w.includes("skipped"))).toBe(true);
  });
});

describe("optionLiteral / colorLiteral — §6 wire payload (must emit id when present)", () => {
  test("emits `id:` IFF an id is present (a dropped id orphans every assignment)", () => {
    expect(optionLiteral({ name: "Todo", color: "GRAY", description: "" })).toBe(`{name:"Todo",color:GRAY,description:""}`);
    expect(optionLiteral({ id: "OPT_1", name: "Todo", color: "GRAY", description: "" })).toBe(
      `{id:"OPT_1",name:"Todo",color:GRAY,description:""}`,
    );
  });

  test("escapes quotes/backslashes in name and description (JSON.stringify)", () => {
    const lit = optionLiteral({ name: 'a"b', color: "BLUE", description: "x\\y" });
    expect(lit).toContain(`name:"a\\"b"`);
    expect(lit).toContain(`description:"x\\\\y"`);
    expect(lit).toContain("color:BLUE"); // colour stays a bare enum token, not quoted
  });

  test("colorLiteral returns the bare token for valid colours and throws for invalid", () => {
    expect(colorLiteral("PURPLE")).toBe("PURPLE");
    expect(() => colorLiteral("rebeccapurple")).toThrow();
  });
});

describe("normalizeLiveColor — robustness, not a hard throw (F5)", () => {
  test("a known colour (any case) passes with no warning", () => {
    expect(normalizeLiveColor("blue", "X")).toEqual({ color: "BLUE" });
    expect(normalizeLiveColor("GREEN", "X")).toEqual({ color: "GREEN" });
  });

  test("an unknown colour defaults to GRAY and warns — it does not throw", () => {
    const r = normalizeLiveColor("chartreuse", "Weird");
    expect(r.color).toBe("GRAY");
    expect(r.warning).toContain("chartreuse");
  });

  test("a preserved live option with an unknown colour yields a normalized union + warning (no throw mid-map)", () => {
    const live: LiveOption[] = [{ id: "id-x", name: "Legacy", color: "chartreuse", description: "" }];
    const { union, warnings } = diffOptions(live, [want("Todo")], { prune: false });
    const legacy = union.find((o) => o.name === "Legacy");
    expect(legacy?.color).toBe("GRAY"); // normalized, preserved (not dropped)
    expect(legacy?.id).toBe("id-x"); // id still carried
    expect(warnings.some((w) => w.includes("Legacy"))).toBe(true);
    // The normalized colour must survive serialization without throwing.
    expect(() => optionLiteral(legacy!)).not.toThrow();
  });
});

describe("destructiveGate — pure gate (catches a boolean inversion)", () => {
  const destructivePlan = (): Plan =>
    buildPlan(
      { projectId: "P", fields: [{ id: "status", name: "Status", dataType: "SINGLE_SELECT", options: [opt("Todo"), opt("Stale")] }], labels: [] },
      { key: "k", title: "T", status: [want("Todo")], fields: [], labels: [], views: [] },
      { prune: true },
    );

  test("destructive + prune + !confirm THROWS", () => {
    const plan = destructivePlan();
    expect(hasDestructive(plan)).toBe(true);
    expect(() => destructiveGate(plan, { prune: true, confirm: false })).toThrow(/--confirm/);
  });

  test("destructive + prune + confirm does NOT throw", () => {
    expect(() => destructiveGate(destructivePlan(), { prune: true, confirm: true })).not.toThrow();
  });

  test("a non-destructive plan never gates, even under prune", () => {
    const plan = buildPlan(
      { projectId: "P", fields: [{ id: "status", name: "Status", dataType: "SINGLE_SELECT", options: [opt("Todo")] }], labels: [] },
      { key: "k", title: "T", status: [want("Todo")], fields: [], labels: [], views: [] },
      { prune: true },
    );
    expect(() => destructiveGate(plan, { prune: true, confirm: false })).not.toThrow();
  });
});
