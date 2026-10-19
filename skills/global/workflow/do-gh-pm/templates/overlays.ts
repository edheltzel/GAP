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

// Feedback-agent overlay (§4.3, interview Q7): the board footprint is an intake
// view over needs:triage; the repo-side `.github/` scaffold (Issue Forms, router
// workflow, slash commands) lives in `feedback-agent/` and is installed by
// copying FEEDBACK_AGENT_SCAFFOLD into the target repo (see
// references/FeedbackAgent.md for placeholders, secrets, and verification).
const feedbackAgent: Overlay = (a) =>
  extend(a, {
    views: [{ name: "Feedback Intake", layout: "TABLE", filter: "label:needs:triage" }],
  });

// Scaffold manifest: template source (relative to this directory) → path in the
// target repo. The interview/scaffold flow installs these when Q7 toggles the
// feedback-agent overlay on.
export const FEEDBACK_AGENT_SCAFFOLD: ReadonlyArray<{ source: string; target: string }> = [
  {
    source: "feedback-agent/ISSUE_TEMPLATE/feedback.yml",
    target: ".github/ISSUE_TEMPLATE/feedback.yml",
  },
  {
    source: "feedback-agent/ISSUE_TEMPLATE/bug.yml",
    target: ".github/ISSUE_TEMPLATE/bug.yml",
  },
  {
    source: "feedback-agent/workflows/feedback-router.yml",
    target: ".github/workflows/feedback-router.yml",
  },
  {
    source: "feedback-agent/workflows/pm-commands.yml",
    target: ".github/workflows/pm-commands.yml",
  },
];

export const OVERLAYS: Record<string, Overlay> = {
  iterations,
  estimation,
  roadmap,
  triage,
  "feedback-agent": feedbackAgent,
};
