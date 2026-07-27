import { describe, expect, test } from "bun:test";
import {
  appendPromptArgs,
  buildDelegationGuidance,
  normalizePromptFrontmatter,
  parseCommandArgs,
  renderPromptBody,
  templateUsesInlineArgPlaceholders,
} from "../src/config.ts";

describe("normalizePromptFrontmatter", () => {
  test("normalizes the OMP-native frontmatter contract", () => {
    const result = normalizePromptFrontmatter({
      description: "Review with a contrasting model",
      model: ["@slow", "openai-codex/gpt-5.6-sol"],
      thinking: "xhigh",
      skill: "ce-code-review",
      restore: false,
      subagent: "reviewer",
      inheritContext: true,
    });

    expect(result.kind).toBe("profiled");
    if (result.kind !== "profiled") throw new Error("expected profiled frontmatter");
    expect(result.config).toEqual({
      description: "Review with a contrasting model",
      modelSpecs: ["@slow", "openai-codex/gpt-5.6-sol"],
      thinking: "xhigh",
      skill: "ce-code-review",
      restore: false,
      subagent: "reviewer",
      inheritContext: true,
    });
  });

  test("leaves ordinary OMP prompt templates to the native loader", () => {
    expect(normalizePromptFrontmatter({ description: "Plain prompt" })).toEqual({
      kind: "native",
    });
  });

  test("splits comma-separated fallback models and defaults restoration", () => {
    const result = normalizePromptFrontmatter({
      model: "@fast, anthropic/claude-sonnet-4-6",
      subagent: true,
    });

    expect(result.kind).toBe("profiled");
    if (result.kind !== "profiled") throw new Error("expected profiled frontmatter");
    expect(result.config).toMatchObject({
      modelSpecs: ["@fast", "anthropic/claude-sonnet-4-6"],
      restore: true,
      subagent: true,
      inheritContext: false,
    });
  });

  test("fails closed for invalid execution fields", () => {
    const result = normalizePromptFrontmatter({
      thinking: "turbo",
      restore: "yes",
      inheritContext: true,
    });

    expect(result.kind).toBe("invalid");
    if (result.kind !== "invalid") throw new Error("expected invalid frontmatter");
    expect(result.diagnostics).toEqual([
      "thinking must be one of off, minimal, low, medium, high, xhigh, or max",
      "restore must be a boolean",
      "inheritContext requires subagent",
    ]);
  });

  test("rejects invalid normalized skill names and malformed descriptions", () => {
    const result = normalizePromptFrontmatter({
      model: "@slow",
      skill: "skill:",
      description: 42,
    });

    expect(result.kind).toBe("invalid");
    if (result.kind !== "invalid") throw new Error("expected invalid frontmatter");
    expect(result.diagnostics).toEqual([
      "skill must be a non-empty name containing only letters, numbers, dots, underscores, or hyphens",
      "description must be a non-empty string",
    ]);
  });

  test("rejects skill names that escape their directory", () => {
    const result = normalizePromptFrontmatter({ model: "@slow", skill: ".." });
    expect(result.kind).toBe("invalid");
  });
});

describe("prompt argument rendering", () => {
  test("parses quoted arguments", () => {
    expect(parseCommandArgs(`src/main.ts "error paths" 'integration tests'`)).toEqual([
      "src/main.ts",
      "error paths",
      "integration tests",
    ]);
  });

  test("supports OMP and Pi prompt placeholders without recursive replacement", () => {
    const args = ["$2", "second", "third", "fourth"];
    const rendered = renderPromptBody(
      "$1 | $2 | $@ | $ARGUMENTS | $@[3:2] | ${@:2} | ${@:2:2}",
      args,
    );

    expect(rendered).toBe(
      "$2 | second | $2 second third fourth | $2 second third fourth | third fourth | second third fourth | second third",
    );
  });

  test("appends fallback arguments only when no inline syntax is present", () => {
    const argsText = "src/main.ts carefully";
    const plain = "Review this change.";
    expect(
      appendPromptArgs(
        renderPromptBody(plain, ["src/main.ts", "carefully"]),
        argsText,
        templateUsesInlineArgPlaceholders(plain),
      ),
    ).toBe("Review this change.\n\nsrc/main.ts carefully");

    expect(renderPromptBody("Review $@[2:]", ["one", "two", "three"])).toBe("Review two three");
    expect(templateUsesInlineArgPlaceholders("Review {{arguments}}")).toBe(true);
    expect(appendPromptArgs("Review one two", "one two", true)).toBe("Review one two");
  });
});

describe("native task delegation guidance", () => {
  test("names the requested agent and context policy", () => {
    const guidance = buildDelegationGuidance("review-change", "reviewer", true);
    expect(guidance).toContain(
      'delegate through the native `task` tool to the `reviewer` agent',
    );
    expect(guidance).toContain("include the relevant current-session context");
  });

  test("uses the configured default agent for subagent true", () => {
    const guidance = buildDelegationGuidance("review-change", true, false);
    expect(guidance).toContain("use its configured default agent");
    expect(guidance).toContain("do not add prior conversation context");
  });
});
