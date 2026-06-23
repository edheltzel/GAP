// IO-shell tests for the reconcile orchestration. The gh executor is injected so
// the dangerous paths — the destructive gate and the apply ordering — are exercised
// WITHOUT spawning real `gh` (F4). The key safety claims: the gate throws BEFORE any
// mutation, and a select reaching apply with !exists throws before applyOptionUnion.
// Run: `bun test Reconcile.test.ts`

import { test, expect, describe, afterEach } from "bun:test";
import { reconcileBoard, __setGhExec, camelKey, parseRename, type GhExec } from "./Reconcile.ts";
import type { Archetype } from "../templates/types.ts";

afterEach(() => __setGhExec(null));

// A gh call that mutates the board (vs a read snapshot / item-list).
const isMutation = (call: string[]): boolean =>
  call.some((a) => a.includes("updateProjectV2Field")) ||
  (call[0] === "project" && (call[1] === "field-create" || call[1] === "field-delete")) ||
  (call[0] === "label" && (call[1] === "create" || call[1] === "delete"));

// Build a fake gh executor over a snapshot response. Records every call so a test
// can assert NO mutation happened before a throw.
function fakeGh(snapshotNodes: unknown[], calls: string[][]): GhExec {
  return (args) => {
    calls.push(args);
    if (args[0] === "project" && args[1] === "view") return JSON.stringify({ id: "PVT_test" });
    if (args[0] === "api" && args[1] === "graphql") {
      if (args.some((a) => a.includes("updateProjectV2Field"))) return JSON.stringify({ data: {} }); // mutation echo
      return JSON.stringify({ data: { node: { id: "PVT_test", fields: { nodes: snapshotNodes } } } });
    }
    if (args[0] === "project" && args[1] === "item-list") return JSON.stringify({ items: [] });
    throw new Error(`unexpected gh call in test: ${args.join(" ")}`);
  };
}

const baseArgs = (over: Record<string, unknown>) => ({
  overlays: [],
  repos: [],
  apply: false,
  prune: false,
  confirm: false,
  lockdir: ".agents/atlas/artifacts",
  renames: [],
  project: 1,
  owner: "tester",
  ...over,
}) as Parameters<typeof reconcileBoard>[0];

describe("reconcileBoard destructive gate — throws BEFORE any mutation (F4)", () => {
  // Status live has an extra option "Stale" not in the spec → a prune removal is destructive.
  const nodes = [
    {
      __typename: "ProjectV2SingleSelectField",
      id: "status",
      name: "Status",
      dataType: "SINGLE_SELECT",
      options: [
        { id: "id-todo", name: "Todo", color: "GRAY", description: "" },
        { id: "id-stale", name: "Stale", color: "GRAY", description: "" },
      ],
    },
  ];
  const spec: Archetype = { key: "k", title: "T", status: [{ name: "Todo", color: "GRAY" }], fields: [], labels: [], views: [] };

  test("apply + prune + !confirm rejects and never issues a mutation", async () => {
    const calls: string[][] = [];
    __setGhExec(fakeGh(nodes, calls));
    await expect(reconcileBoard(baseArgs({ apply: true, prune: true, confirm: false }), spec)).rejects.toThrow(/--confirm/);
    expect(calls.some(isMutation)).toBe(false); // the board was never touched
  });

  test("apply + prune + confirm is allowed past the gate (the union write is attempted)", async () => {
    const calls: string[][] = [];
    __setGhExec(fakeGh(nodes, calls));
    // Isolate the recovery artifact this destructive apply writes (then remove it).
    const tmp = `${import.meta.dir}/.test-lockdir`;
    await reconcileBoard(baseArgs({ apply: true, prune: true, confirm: true, lockdir: tmp }), spec);
    expect(calls.some((c) => c.some((a) => a.includes("updateProjectV2Field")))).toBe(true);
    Bun.spawnSync(["rm", "-rf", tmp]);
  });
});

describe("reconcileBoard apply guard — !s.exists throws before applyOptionUnion (F3/F4)", () => {
  // Priority exists live as TEXT while the spec wants SINGLE_SELECT. diffFields warns
  // but emits no create → the Priority select reaches apply with exists=false + ops.
  const nodes = [
    { __typename: "ProjectV2SingleSelectField", id: "status", name: "Status", dataType: "SINGLE_SELECT", options: [{ id: "id-todo", name: "Todo", color: "GRAY", description: "" }] },
    { __typename: "ProjectV2FieldCommon", id: "prio-text", name: "Priority", dataType: "TEXT" },
  ];
  const spec: Archetype = {
    key: "k",
    title: "T",
    status: [{ name: "Todo", color: "GRAY" }],
    fields: [{ name: "Priority", type: "SINGLE_SELECT", options: [{ name: "P0", color: "RED" }] }],
    labels: [],
    views: [],
  };

  test("apply throws 'not resolved' and never writes an option union", async () => {
    const calls: string[][] = [];
    __setGhExec(fakeGh(nodes, calls));
    await expect(reconcileBoard(baseArgs({ apply: true }), spec)).rejects.toThrow(/not resolved/);
    expect(calls.some((c) => c.some((a) => a.includes("updateProjectV2Field")))).toBe(false);
  });
});

describe("camelKey — matches gh's item-list field-key normalization", () => {
  test("single word lowercases", () => expect(camelKey("Status")).toBe("status"));
  test("multi-word camelCases (the F1b miss)", () => expect(camelKey("In Progress")).toBe("inProgress"));
  test("punctuation splits too", () => expect(camelKey("Sub-issues progress")).toBe("subIssuesProgress"));
});

describe("parseRename", () => {
  test("splits on the first ':' and first '=' (names may contain spaces)", () => {
    expect(parseRename("Status:Backlog=Icebox")).toEqual({ field: "Status", from: "Backlog", to: "Icebox" });
    expect(parseRename("Estimate:Big Job=Huge Job")).toEqual({ field: "Estimate", from: "Big Job", to: "Huge Job" });
  });
  test("rejects malformed specs", () => {
    expect(() => parseRename("nocolon")).toThrow();
    expect(() => parseRename("Status:noequals")).toThrow();
    expect(() => parseRename("Status:=Icebox")).toThrow();
    expect(() => parseRename("Status:Backlog=")).toThrow();
  });
});
