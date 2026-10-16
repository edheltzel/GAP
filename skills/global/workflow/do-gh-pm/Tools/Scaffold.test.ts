import { afterEach, describe, expect, test } from "bun:test";
import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  __setCommandExec,
  parseArgs,
  runScaffold,
  type CommandExec,
  type ScaffoldArgs,
} from "./Scaffold.ts";

const tempDirs: string[] = [];

afterEach(() => {
  __setCommandExec(null);
  for (const dir of tempDirs.splice(0)) rmSync(dir, { recursive: true, force: true });
});

function args(overrides: Partial<ScaffoldArgs> = {}): ScaffoldArgs {
  return {
    owner: "edheltzel",
    repo: "edheltzel/example",
    title: "Example board",
    workKind: "software",
    visibility: "private",
    contributors: "solo",
    cadence: "continuous",
    estimation: false,
    roadmap: false,
    feedback: false,
    priority: "p0-p3",
    apply: true,
    ...overrides,
  };
}

function fakeExec(records: string[][], projectNumber = 99): CommandExec {
  return (command, commandArgs) => {
    records.push([command, ...commandArgs]);
    if (command === "gh" && commandArgs[0] === "--version") return "gh version 2.95.0 (2026-06-17)";
    if (command === "gh" && ["create", "copy"].includes(commandArgs[1] ?? "")) {
      return JSON.stringify({ number: projectNumber, title: "Example board" });
    }
    return "";
  };
}

describe("parseArgs", () => {
  test("requires explicit answers for all eight interview questions", () => {
    expect(() => parseArgs(["--owner", "edheltzel"])).toThrow("missing required arguments");
  });

  test("parses the complete interview payload", () => {
    expect(
      parseArgs([
        "--owner", "edheltzel",
        "--repo", "edheltzel/example",
        "--title", "Example board",
        "--work-kind", "research",
        "--visibility", "public",
        "--contributors", "external",
        "--cadence", "time-boxed",
        "--estimation", "yes",
        "--roadmap", "no",
        "--feedback", "no",
        "--priority", "none",
      ]),
    ).toMatchObject({
      workKind: "research",
      visibility: "public",
      contributors: "external",
      cadence: "time-boxed",
      estimation: true,
      roadmap: false,
      feedback: false,
      priority: "none",
      apply: false,
    });
  });
});

describe("runScaffold", () => {
  test("dry-run returns a complete plan without executing commands", async () => {
    const records: string[][] = [];
    __setCommandExec(fakeExec(records));

    const result = await runScaffold(args({ apply: false }));

    expect(result.projectNumber).toBeUndefined();
    expect(result.config.archetype).toBe("solo-software");
    expect(result.manualSteps.some((step) => step.includes("Board by Status"))).toBe(true);
    expect(records).toEqual([]);
  });

  test("requires a local repository path before planning feedback installation", async () => {
    await expect(runScaffold(args({ apply: false, feedback: true }))).rejects.toThrow(
      "--repo-path is required",
    );
  });

  test("lists every feedback destination in dry-run mode", async () => {
    const records: string[][] = [];
    const repoPath = mkdtempSync(join(tmpdir(), "do-gh-pm-scaffold-"));
    tempDirs.push(repoPath);
    __setCommandExec(fakeExec(records));

    const result = await runScaffold(args({ apply: false, feedback: true, repoPath }));

    expect(result.feedbackTargets).toHaveLength(4);
    expect(result.feedbackTargets).toContain(
      join(repoPath, ".github/workflows/feedback-router.yml"),
    );
    expect(records).toEqual([]);
  });

  test("creates, links, plans, and safely reconciles a board", async () => {
    const records: string[][] = [];
    __setCommandExec(fakeExec(records, 42));

    const result = await runScaffold(args());

    expect(result.projectNumber).toBe(42);
    expect(records.map((record) => record.slice(0, 3))).toEqual([
      ["gh", "--version"],
      ["gh", "project", "create"],
      ["gh", "project", "edit"],
      ["gh", "project", "link"],
      ["bun", expect.stringContaining("Reconcile.ts"), "--project"],
      ["bun", expect.stringContaining("Reconcile.ts"), "--project"],
    ]);
    expect(records[4]).toContain("--priority");
    expect(records[5]).toContain("--apply");
  });

  test("copies a golden project when a source is supplied", async () => {
    const records: string[][] = [];
    __setCommandExec(fakeExec(records, 43));

    await runScaffold(args({ sourceProject: 7, sourceOwner: "edheltzel" }));

    expect(records[1]).toEqual([
      "gh", "project", "copy", "7",
      "--source-owner", "edheltzel",
      "--target-owner", "edheltzel",
      "--title", "Example board",
      "--format", "json",
    ]);
  });

  test("installs the feedback-agent manifest without overwriting files", async () => {
    const records: string[][] = [];
    const repoPath = mkdtempSync(join(tmpdir(), "do-gh-pm-scaffold-"));
    tempDirs.push(repoPath);
    __setCommandExec(fakeExec(records));

    await runScaffold(args({ feedback: true, repoPath }));

    const target = join(repoPath, ".github/ISSUE_TEMPLATE/feedback.yml");
    expect(existsSync(target)).toBe(true);
    expect(readFileSync(target, "utf8")).toContain("name: Feedback");

    writeFileSync(target, "user-owned\n");
    await expect(runScaffold(args({ feedback: true, repoPath }))).rejects.toThrow("refusing to overwrite");
  });

  test("runs Reconcile in the target checkout, not the process directory", async () => {
    const seen: { command: string; cwd?: string }[] = [];
    const repoPath = mkdtempSync(join(tmpdir(), "do-gh-pm-checkout-"));
    tempDirs.push(repoPath);
    __setCommandExec((command, commandArgs, cwd) => {
      seen.push({ command, cwd });
      return fakeExec([])(command, commandArgs, cwd);
    });

    await runScaffold(args({ repoPath }));

    const reconcile = seen.filter((call) => call.command === "bun");
    expect(reconcile).toHaveLength(2);
    expect(reconcile.every((call) => call.cwd === repoPath)).toBe(true);
  });

  test("rejects unsupported gh versions before creating a project", async () => {
    const records: string[][] = [];
    __setCommandExec((command, commandArgs) => {
      records.push([command, ...commandArgs]);
      return "gh version 2.93.0";
    });

    await expect(runScaffold(args())).rejects.toThrow("gh >= 2.94");
    expect(records).toHaveLength(1);
  });
});
