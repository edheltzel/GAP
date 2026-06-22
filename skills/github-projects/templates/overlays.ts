// Additive overlay modules (skill plan §5). Each is a pure function that
// merges its delta into an archetype; they compose by application. Toggled by
// the interview (§4.1) and applied in order by `resolve`.

import type { Overlay } from "./types.ts";
import { extend } from "./merge.ts";

// Sprint/time-box work: an Iteration field + a sprint view.
const iterations: Overlay = (a) =>
  extend(a, {
    fields: [{ name: "Iteration", type: "ITERATION" }],
    views: [{ name: "Current Sprint", layout: "TABLE", filter: "iteration:@current" }],
  });

// Effort estimation: a numeric Estimate and a T-shirt Size (§4.1 Q5).
const estimation: Overlay = (a) =>
  extend(a, {
    fields: [
      { name: "Estimate", type: "NUMBER" },
      {
        name: "Size",
        type: "SINGLE_SELECT",
        options: [
          { name: "XS", color: "GRAY" },
          { name: "S", color: "BLUE" },
          { name: "M", color: "GREEN" },
          { name: "L", color: "YELLOW" },
          { name: "XL", color: "RED" },
        ],
      },
    ],
  });

// Dated roadmap: Start/Target date fields + a Roadmap view.
const roadmap: Overlay = (a) =>
  extend(a, {
    fields: [
      { name: "Start Date", type: "DATE" },
      { name: "Target Date", type: "DATE" },
    ],
    views: [{ name: "Roadmap", layout: "ROADMAP" }],
  });

// External-contributor triage: a leading "Triage" status column + intake labels.
const triage: Overlay = (a) =>
  extend(a, {
    status: [{ name: "Triage", color: "RED" }],
    labels: [
      { name: "good first issue", color: "7057ff", description: "good for newcomers" },
      { name: "help wanted", color: "008672", description: "extra attention wanted" },
    ],
  });

// Feedback-agent board footprint: an intake view over needs:triage. The actual
// `.github/` scaffold (Issue Forms, router workflow, GitHub App) is Phase 5 and
// is installed separately — this overlay only contributes the board-visible view.
const feedbackAgent: Overlay = (a) =>
  extend(a, {
    views: [{ name: "Feedback Intake", layout: "TABLE", filter: "label:needs:triage" }],
  });

export const OVERLAYS: Record<string, Overlay> = {
  iterations,
  estimation,
  roadmap,
  triage,
  "feedback-agent": feedbackAgent,
};
