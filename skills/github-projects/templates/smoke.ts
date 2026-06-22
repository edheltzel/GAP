// Tiny resolve smoke-check / self-test. Run: `bun smoke.ts`
// Resolves the default archetype with two overlays and asserts the merged spec
// matches the skill plan's Phase 1 verify gate.

import { resolve } from "./resolve.ts";

function assert(cond: boolean, msg: string): void {
  if (!cond) {
    console.error(`FAIL: ${msg}`);
    throw new Error(msg);
  }
}

const spec = resolve("solo-software", ["estimation", "roadmap"]);

const statusNames = spec.status.map((s) => s.name);
const fieldNames = spec.fields.map((f) => f.name);
const labelNames = spec.labels.map((l) => l.name);

// Gate: Status = Backlog·Ready·In Progress·In Review·Done
assert(
  JSON.stringify(statusNames) ===
    JSON.stringify(["Backlog", "Ready", "In Progress", "In Review", "Done"]),
  `unexpected status vocabulary: ${statusNames.join(" · ")}`,
);

// Gate: fields include Phase, Estimate#, Area, Start/Target dates
for (const required of ["Phase", "Estimate", "Area", "Start Date", "Target Date"]) {
  assert(fieldNames.includes(required), `missing field "${required}"`);
}

// Gate: labels include type:*
const typeLabels = [
  "type:bug",
  "type:feature",
  "type:docs",
  "type:test",
  "type:refactor",
  "type:chore",
];
for (const t of typeLabels) {
  assert(labelNames.includes(t), `missing label "${t}"`);
}

// Determinism: resolving the same inputs twice yields identical output.
assert(
  JSON.stringify(resolve("solo-software", ["estimation", "roadmap"])) ===
    JSON.stringify(spec),
  "resolve is not deterministic",
);

console.log(`resolved: ${spec.key} + [estimation, roadmap]`);
console.log(`  Status: ${statusNames.join(" · ")}`);
console.log(`  Fields: ${fieldNames.join(", ")}`);
console.log(`  Labels: ${labelNames.length} total (incl. ${typeLabels.length} type:*)`);
console.log(`  Views : ${spec.views.map((v) => v.name).join(", ")}`);
console.log("\nALL ASSERTIONS PASSED");
