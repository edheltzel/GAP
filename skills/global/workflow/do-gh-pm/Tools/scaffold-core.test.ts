import { describe, expect, test } from "bun:test";
import { resolve } from "../templates/resolve.ts";
import {
  answersToConfig,
  resolveInterview,
  type InterviewAnswers,
} from "./scaffold-core.ts";

const FULL_FEATURED_ANSWERS: InterviewAnswers = {
  workKind: "software",
  visibility: "private",
  contributors: "external",
  cadence: "time-boxed",
  estimation: true,
  roadmap: true,
  feedback: true,
  priority: "high-medium-low",
};

describe("answersToConfig", () => {
  test("maps all eight interview answers deterministically", () => {
    expect(answersToConfig(FULL_FEATURED_ANSWERS)).toEqual({
      archetype: "solo-software",
      overlays: ["triage", "iterations", "estimation", "roadmap", "feedback-agent"],
      visibility: "private",
      priority: "high-medium-low",
    });
  });

  test("maps each work kind to its intended archetype", () => {
    const expected = {
      software: "solo-software",
      writing: "content-pipeline",
      research: "research-tracker",
      personal: "life-os-tasks",
      "open-source": "open-source-repo",
    } as const;

    for (const [workKind, archetype] of Object.entries(expected)) {
      expect(
        answersToConfig({
          ...FULL_FEATURED_ANSWERS,
          workKind: workKind as InterviewAnswers["workKind"],
        }).archetype,
      ).toBe(archetype);
    }
  });
});

describe("resolveInterview", () => {
  test("replaces the default Priority vocabulary", () => {
    const spec = resolveInterview(FULL_FEATURED_ANSWERS);
    expect(spec.fields.find((field) => field.name === "Priority")?.options?.map((option) => option.name)).toEqual([
      "High",
      "Medium",
      "Low",
    ]);
  });

  test("removes Priority when the interview selects none", () => {
    const spec = resolveInterview({ ...FULL_FEATURED_ANSWERS, priority: "none" });
    expect(spec.fields.some((field) => field.name === "Priority")).toBe(false);
  });

  test("rejects inherited object keys as Priority schemes", () => {
    expect(() => resolve("solo-software", [], "toString" as never)).toThrow(
      'Unknown priority scheme "toString"',
    );
  });
});
