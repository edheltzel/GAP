import {
  resolve,
  type Archetype,
  type PriorityScheme,
} from "../templates/resolve.ts";

export type WorkKind = "software" | "writing" | "research" | "personal" | "open-source";
export type Visibility = "public" | "private";
export type Contributors = "solo" | "external";
export type Cadence = "continuous" | "time-boxed";

export interface InterviewAnswers {
  workKind: WorkKind;
  visibility: Visibility;
  contributors: Contributors;
  cadence: Cadence;
  estimation: boolean;
  roadmap: boolean;
  feedback: boolean;
  priority: PriorityScheme;
}

export interface ScaffoldConfig {
  archetype: string;
  overlays: string[];
  visibility: Visibility;
  priority: PriorityScheme;
}

const ARCHETYPE_BY_WORK_KIND: Record<WorkKind, string> = {
  software: "solo-software",
  writing: "content-pipeline",
  research: "research-tracker",
  personal: "life-os-tasks",
  "open-source": "open-source-repo",
};

export function answersToConfig(answers: InterviewAnswers): ScaffoldConfig {
  const overlays: string[] = [];
  if (answers.contributors === "external") overlays.push("triage");
  if (answers.cadence === "time-boxed") overlays.push("iterations");
  if (answers.estimation) overlays.push("estimation");
  if (answers.roadmap) overlays.push("roadmap");
  if (answers.feedback) overlays.push("feedback-agent");

  return {
    archetype: ARCHETYPE_BY_WORK_KIND[answers.workKind],
    overlays,
    visibility: answers.visibility,
    priority: answers.priority,
  };
}

export function resolveInterview(answers: InterviewAnswers): Archetype {
  const config = answersToConfig(answers);
  return resolve(config.archetype, config.overlays, config.priority);
}
