// Tiny resolve smoke-check / self-test. Run: `bun smoke.ts`
// Resolves the default archetype with two overlays and asserts the merged spec
// matches the skill plan's Phase 1 verify gate.

import { existsSync, readFileSync } from "node:fs";
import { resolve, ARCHETYPES, OVERLAYS } from "./resolve.ts";
import { FEEDBACK_AGENT_SCAFFOLD } from "./overlays.ts";
import { CANON_LABELS } from "./archetypes.ts";

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

// Coverage: every archetype resolves cleanly with the shared base in place.
const archetypeKeys = Object.keys(ARCHETYPES);
assert(archetypeKeys.length === 5, `expected 5 archetypes, got ${archetypeKeys.length}`);
for (const key of archetypeKeys) {
  const a = resolve(key);
  assert(a.status.length > 0, `${key}: empty status vocabulary`);
  assert(a.fields.some((f) => f.name === "Priority"), `${key}: missing shared Priority field`);
  assert(a.labels.length > 0, `${key}: empty labels`);
}

// Coverage: every overlay composes additively onto the default archetype.
const overlayKeys = Object.keys(OVERLAYS);
assert(overlayKeys.length === 5, `expected 5 overlays, got ${overlayKeys.length}`);
const footprint = (s: ReturnType<typeof resolve>) =>
  s.status.length + s.fields.length + s.labels.length + s.views.length;
const baseSolo = resolve("solo-software");
for (const key of overlayKeys) {
  assert(
    footprint(resolve("solo-software", [key])) > footprint(baseSolo),
    `overlay "${key}" composed but added nothing`,
  );
}

// triage is the one asymmetric merge: it PREPENDS a single "Triage" column.
const triaged = resolve("solo-software", ["triage"]);
assert(triaged.status[0].name === "Triage", "triage did not prepend Triage");
assert(
  triaged.status.filter((s) => s.name === "Triage").length === 1,
  "triage produced a duplicate Triage column",
);

// F5: resolved specs are detached — mutating one must not leak into the next.
const probe = resolve("solo-software", ["estimation"]);
probe.fields.push({ name: "__leak__", type: "TEXT" });
probe.status.push({ name: "__leak__", color: "GRAY" });
const fresh = resolve("solo-software", ["estimation"]);
assert(
  !fresh.fields.some((f) => f.name === "__leak__") &&
    !fresh.status.some((s) => s.name === "__leak__"),
  "resolved spec shares mutable refs with module constants (F5 regression)",
);

// Phase 5: the feedback-agent scaffold manifest points at real template files
// (the four .github/ artifacts) with targets under the repo's .github/.
assert(
  FEEDBACK_AGENT_SCAFFOLD.length === 4,
  `expected 4 scaffold entries, got ${FEEDBACK_AGENT_SCAFFOLD.length}`,
);
for (const { source, target } of FEEDBACK_AGENT_SCAFFOLD) {
  assert(
    existsSync(`${import.meta.dir}/${source}`),
    `scaffold source missing: ${source}`,
  );
  assert(target.startsWith(".github/"), `scaffold target not under .github/: ${target}`);
}

// Exact manifest target set — a renamed or duplicated target is a defect.
assert(
  JSON.stringify(FEEDBACK_AGENT_SCAFFOLD.map((e) => e.target).sort()) ===
    JSON.stringify([
      ".github/ISSUE_TEMPLATE/bug.yml",
      ".github/ISSUE_TEMPLATE/feedback.yml",
      ".github/workflows/feedback-router.yml",
      ".github/workflows/pm-commands.yml",
    ]),
  "unexpected scaffold target set",
);

// Parse the templates and enforce the scaffold's form/workflow invariants.
const parseYaml = (rel: string) =>
  Bun.YAML.parse(readFileSync(`${import.meta.dir}/${rel}`, "utf8"));

// Issue Forms: canon labels only, needs:triage intake (the router's gate and
// the overlay view key on it), and the ORG/PROJECT_NUMBER placeholder.
const canon = new Set(CANON_LABELS.map((l) => l.name));
for (const rel of [
  "feedback-agent/ISSUE_TEMPLATE/feedback.yml",
  "feedback-agent/ISSUE_TEMPLATE/bug.yml",
]) {
  const form = parseYaml(rel);
  for (const label of form.labels) {
    assert(canon.has(label), `${rel}: label "${label}" not in CANON_LABELS`);
  }
  assert(form.labels.includes("needs:triage"), `${rel}: missing needs:triage intake label`);
  assert(
    JSON.stringify(form.projects) === JSON.stringify(["ORG/PROJECT_NUMBER"]),
    `${rel}: projects key must be the ORG/PROJECT_NUMBER placeholder`,
  );
}

// Router: trigger, intake gate, action names, token plumbing, ack dedupe.
const router = parseYaml("feedback-agent/workflows/feedback-router.yml");
assert(
  JSON.stringify(router.on?.issues?.types) === JSON.stringify(["opened", "labeled"]),
  "router trigger must be issues: [opened, labeled]",
);
assert(
  String(router.jobs.route.if ?? "").includes(
    "contains(github.event.issue.labels.*.name, 'needs:triage')",
  ),
  "router job must be gated on the needs:triage intake label",
);
assert(
  router.concurrency?.group === "feedback-${{ github.event.issue.number }}",
  "router must serialize per-issue runs via the feedback concurrency group",
);
const routerSteps: any[] = router.jobs.route.steps;
const mint = routerSteps.find((s) => s.uses?.startsWith("actions/create-github-app-token@"));
assert(mint !== undefined, "router must mint an App token via actions/create-github-app-token");
// The minted token must stay narrowed, not inherit the installation's full set.
assert(
  mint.with["permission-organization-projects"] === "write" &&
    mint.with["permission-issues"] === "read" &&
    mint.with["permission-metadata"] === "read",
  "App token mint must carry the three permission-narrowing inputs",
);
const addToProject = routerSteps.find((s) => s.uses?.startsWith("actions/add-to-project@"));
assert(addToProject !== undefined, "router must use actions/add-to-project");
assert(
  addToProject.with["project-url"] === "https://github.com/orgs/ORG/projects/PROJECT_NUMBER",
  "router project-url must carry the ORG/PROJECT_NUMBER placeholder",
);
assert(
  addToProject.with["github-token"] === "${{ steps.app-token.outputs.token }}",
  "add-to-project must use the App-token step output, not GITHUB_TOKEN",
);
const ack = routerSteps.find((s) => s.name?.startsWith("Acknowledge reporter"));
const ackScript: string = ack?.with?.script ?? "";
assert(
  ackScript.includes("<!-- pm-feedback-ack -->") && ackScript.includes("listComments"),
  "ack comment must dedupe via the pm-feedback-ack marker before posting",
);

// Slash commands: full add/remove transition map (mutually exclusive states)
// plus the collaborator write gate.
const pmCommands = parseYaml("feedback-agent/workflows/pm-commands.yml");
const pmScript: string = pmCommands.jobs["pm-command"].steps[0].with.script;
for (const [cmd, transition] of [
  ["/pm:triage", '{ add: ["needs:triage"], remove: ["needs:info", "agent:ready"] }'],
  ["/pm:needs-info", '{ add: ["needs:info"],   remove: ["needs:triage", "agent:ready"] }'],
  ["/pm:ready", '{ add: ["agent:ready"],  remove: ["needs:triage", "needs:info"] }'],
] as const) {
  assert(pmScript.includes(`"${cmd}"`), `pm-commands script missing handler ${cmd}`);
  assert(pmScript.includes(transition), `pm-commands script: ${cmd} must map to ${transition}`);
}
assert(
  pmScript.includes("getCollaboratorPermissionLevel") &&
    pmScript.includes('["write", "admin"].includes(perm.permission)'),
  "pm-commands must gate on a real collaborator write-permission check",
);
assert(
  pmCommands.concurrency?.group === "pm-${{ github.event.issue.number }}" &&
    pmCommands.concurrency?.["cancel-in-progress"] === false,
  "pm-commands must serialize per-issue runs and not cancel the in-progress command",
);

console.log(`resolved: ${spec.key} + [estimation, roadmap]`);
console.log(`  Status: ${statusNames.join(" · ")}`);
console.log(`  Fields: ${fieldNames.join(", ")}`);
console.log(`  Labels: ${labelNames.length} total (incl. ${typeLabels.length} type:*)`);
console.log(`  Views : ${spec.views.map((v) => v.name).join(", ")}`);
console.log(
  `coverage: ${archetypeKeys.length}/5 archetypes resolve · ${overlayKeys.length}/5 overlays compose · triage prepend + F5 detachment checked`,
);
console.log("\nALL ASSERTIONS PASSED");
