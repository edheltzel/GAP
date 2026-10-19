#!/usr/bin/env bun

import {
  existsSync,
  mkdirSync,
  readFileSync,
  writeFileSync,
} from "node:fs";
import { dirname, join, resolve as resolvePath } from "node:path";
import { FEEDBACK_AGENT_SCAFFOLD } from "../templates/overlays.ts";
import { resolve, type Archetype } from "../templates/resolve.ts";
import {
  answersToConfig,
  type InterviewAnswers,
  type ScaffoldConfig,
} from "./scaffold-core.ts";

export interface ScaffoldArgs extends InterviewAnswers {
  owner: string;
  repo: string;
  repoPath?: string;
  title: string;
  sourceProject?: number;
  sourceOwner?: string;
  apply: boolean;
}

export type CommandExec = (command: string, args: string[], cwd?: string) => string;

const defaultExec: CommandExec = (command, args, cwd) => {
  const result = Bun.spawnSync([command, ...args], {
    cwd,
    stdout: "pipe",
    stderr: "pipe",
  });
  if (result.exitCode !== 0) {
    const detail = new TextDecoder().decode(result.stderr).trim();
    throw new Error(`${command} ${args.join(" ")} failed${detail ? `: ${detail}` : ""}`);
  }
  return new TextDecoder().decode(result.stdout).trim();
};

let commandExec: CommandExec = defaultExec;

export function __setCommandExec(exec: CommandExec | null): void {
  commandExec = exec ?? defaultExec;
}

function nextValue(argv: string[], index: number, flag: string): string {
  const value = argv[index + 1];
  if (!value || value.startsWith("--")) throw new Error(`${flag} requires a value`);
  return value;
}

function choice<T extends string>(flag: string, value: string, allowed: readonly T[]): T {
  if (!allowed.includes(value as T)) {
    throw new Error(`${flag} must be one of: ${allowed.join(", ")}`);
  }
  return value as T;
}

function yesNo(flag: string, value: string): boolean {
  return choice(flag, value, ["yes", "no"] as const) === "yes";
}

export function parseArgs(argv: string[]): ScaffoldArgs {
  const parsed: Partial<ScaffoldArgs> = { apply: false };
  for (let i = 0; i < argv.length; i++) {
    const flag = argv[i];
    const value = () => nextValue(argv, i, flag);
    switch (flag) {
      case "--owner": parsed.owner = value(); i++; break;
      case "--repo": parsed.repo = value(); i++; break;
      case "--repo-path": parsed.repoPath = value(); i++; break;
      case "--title": parsed.title = value(); i++; break;
      case "--work-kind": parsed.workKind = choice(flag, value(), ["software", "writing", "research", "personal", "open-source"] as const); i++; break;
      case "--visibility": parsed.visibility = choice(flag, value(), ["public", "private"] as const); i++; break;
      case "--contributors": parsed.contributors = choice(flag, value(), ["solo", "external"] as const); i++; break;
      case "--cadence": parsed.cadence = choice(flag, value(), ["continuous", "time-boxed"] as const); i++; break;
      case "--estimation": parsed.estimation = yesNo(flag, value()); i++; break;
      case "--roadmap": parsed.roadmap = yesNo(flag, value()); i++; break;
      case "--feedback": parsed.feedback = yesNo(flag, value()); i++; break;
      case "--priority": parsed.priority = choice(flag, value(), ["p0-p3", "high-medium-low", "none"] as const); i++; break;
      case "--source-project": parsed.sourceProject = Number(value()); i++; break;
      case "--source-owner": parsed.sourceOwner = value(); i++; break;
      case "--apply": parsed.apply = true; break;
      case "-h":
      case "--help":
        printUsage();
        process.exit(0);
      default: throw new Error(`unknown argument: ${flag}`);
    }
  }

  const required: Array<keyof ScaffoldArgs> = [
    "owner", "repo", "title", "workKind", "visibility", "contributors", "cadence",
    "estimation", "roadmap", "feedback", "priority",
  ];
  const missing = required.filter((key) => parsed[key] === undefined);
  if (missing.length) throw new Error(`missing required arguments: ${missing.map((key) => `--${String(key).replace(/[A-Z]/g, (letter) => `-${letter.toLowerCase()}`)}`).join(", ")}`);
  if (parsed.sourceProject !== undefined && (!Number.isInteger(parsed.sourceProject) || parsed.sourceProject < 1)) {
    throw new Error("--source-project must be a positive integer");
  }
  if (parsed.feedback && !parsed.repoPath) {
    throw new Error("--repo-path is required when --feedback yes installs the .github scaffold");
  }
  return parsed as ScaffoldArgs;
}

function printUsage(): void {
  console.log(`Interview-driven GitHub Projects scaffold.

Usage:
  bun Tools/Scaffold.ts --owner OWNER --repo OWNER/REPO --title TITLE \\
    --work-kind software|writing|research|personal|open-source \\
    --visibility public|private --contributors solo|external \\
    --cadence continuous|time-boxed --estimation yes|no \\
    --roadmap yes|no --feedback yes|no \\
    --priority p0-p3|high-medium-low|none [flags]

Flags:
  --source-project N     copy a golden project instead of creating an empty board
  --source-owner OWNER   owner of the golden project (defaults to --owner)
  --repo-path PATH       local checkout receiving feedback-agent files (required with --feedback yes)
  --apply                execute the displayed plan; omitted means dry-run
  -h, --help             show this help`);
}

function assertGhVersion(output: string): void {
  const match = output.match(/gh version (\d+)\.(\d+)/);
  if (!match) throw new Error(`unable to parse gh version from: ${output}`);
  const [, majorText, minorText] = match;
  const major = Number(majorText);
  const minor = Number(minorText);
  if (major < 2 || (major === 2 && minor < 94)) {
    throw new Error(`gh >= 2.94 is required; found ${major}.${minor}`);
  }
}

function projectCommand(args: ScaffoldArgs): string[] {
  if (args.sourceProject !== undefined) {
    return [
      "project", "copy", String(args.sourceProject),
      "--source-owner", args.sourceOwner ?? args.owner,
      "--target-owner", args.owner,
      "--title", args.title,
      "--format", "json",
    ];
  }
  return ["project", "create", "--owner", args.owner, "--title", args.title, "--format", "json"];
}

function reconcileCommand(
  args: ScaffoldArgs,
  config: ScaffoldConfig,
  projectNumber: string,
): string[] {
  const command = [
    "--project", projectNumber,
    "--owner", args.owner,
    "--archetype", config.archetype,
    "--priority", config.priority,
  ];
  for (const overlay of config.overlays) command.push("--overlay", overlay);
  command.push("--repo", args.repo);
  return command;
}

interface FeedbackFile {
  target: string;
  content: string;
}

function feedbackSources(repoPath: string): FeedbackFile[] {
  return FEEDBACK_AGENT_SCAFFOLD.map((entry) => {
    const source = join(import.meta.dir, "../templates", entry.source);
    try {
      return {
        target: join(resolvePath(repoPath), entry.target),
        content: readFileSync(source, "utf8"),
      };
    } catch (error) {
      throw new Error(`feedback scaffold source is unreadable: ${source}`, { cause: error });
    }
  });
}

function preflightFeedback(repoPath: string): FeedbackFile[] {
  const files = feedbackSources(repoPath);
  for (const file of files) {
    if (existsSync(file.target) && readFileSync(file.target, "utf8") !== file.content) {
      throw new Error(`refusing to overwrite existing feedback scaffold file: ${file.target}`);
    }
  }
  return files;
}

function installFeedback(files: FeedbackFile[]): string[] {
  const installed: string[] = [];
  for (const file of files) {
    mkdirSync(dirname(file.target), { recursive: true });
    try {
      writeFileSync(file.target, file.content, { flag: "wx" });
      installed.push(file.target);
    } catch (error) {
      const code = error && typeof error === "object" && "code" in error ? error.code : undefined;
      if (code !== "EEXIST") throw error;
      if (readFileSync(file.target, "utf8") !== file.content) {
        throw new Error(`refusing to overwrite existing feedback scaffold file: ${file.target}`);
      }
    }
  }
  return installed;
}

function residualSteps(args: ScaffoldArgs, spec: Archetype): string[] {
  const steps = spec.views.map((view) =>
    `Create view "${view.name}" (${view.layout}${view.filter ? `; filter: ${view.filter}` : ""}).`
  );
  if (args.sourceProject !== undefined) {
    steps.push("Re-enable the auto-add workflow; project copies do not preserve it.");
  } else {
    steps.push("Enable auto-add, closed → Done, and Done-after-14-days auto-archive workflows.");
  }
  if (spec.fields.some((field) => field.type === "ITERATION")) {
    steps.push("Create the Iteration field in the web UI; the CLI cannot create it.");
  }
  if (args.feedback) {
    steps.push("Configure feedback-agent placeholders and ATLAS_APP_ID, ATLAS_APP_PRIVATE_KEY, and ATLAS_KEY_PAT secrets.");
  }
  return steps;
}

export interface ScaffoldResult {
  config: ScaffoldConfig;
  projectNumber?: number;
  plannedCommands: string[][];
  manualSteps: string[];
  reconcileCwd: string;
  reconcilePlan?: string;
  reconcileApply?: string;
  feedbackTargets: string[];
}

export async function runScaffold(args: ScaffoldArgs): Promise<ScaffoldResult> {
  if (args.feedback && !args.repoPath) {
    throw new Error("--repo-path is required when feedback installs the .github scaffold");
  }
  const config = answersToConfig(args);
  const spec = resolve(config.archetype, config.overlays, config.priority);
  const projectArgs = projectCommand(args);
  const feedbackTargets = args.feedback
    ? FEEDBACK_AGENT_SCAFFOLD.map((entry) => join(resolvePath(args.repoPath!), entry.target))
    : [];
  const reconcilePath = join(import.meta.dir, "Reconcile.ts");
  const checkout = resolvePath(args.repoPath ?? process.cwd());
  const placeholder = "<new-project-number>";
  const plannedReconcileArgs = reconcileCommand(args, config, placeholder);
  const plannedCommands = [
    ["gh", ...projectArgs],
    ["gh", "project", "edit", placeholder, "--owner", args.owner, "--visibility", args.visibility.toUpperCase()],
    ["gh", "project", "link", placeholder, "--owner", args.owner, "--repo", args.repo],
    ["bun", reconcilePath, ...plannedReconcileArgs],
    ["bun", reconcilePath, ...plannedReconcileArgs, "--apply"],
  ];
  const result: ScaffoldResult = {
    config,
    plannedCommands,
    manualSteps: residualSteps(args, spec),
    installedFiles: [],
    feedbackTargets,
    reconcileCwd: checkout,
  };
  if (!args.apply) return result;

  const feedbackFiles = args.feedback ? preflightFeedback(args.repoPath!) : [];
  assertGhVersion(commandExec("gh", ["--version"]));

  const created = JSON.parse(commandExec("gh", projectArgs)) as { number?: number };
  if (!Number.isInteger(created.number) || created.number! < 1) {
    throw new Error("gh project create/copy did not return a project number");
  }
  const projectNumber = created.number!;
  try {
    commandExec("gh", ["project", "edit", String(projectNumber), "--owner", args.owner, "--visibility", args.visibility.toUpperCase()]);
    commandExec("gh", ["project", "link", String(projectNumber), "--owner", args.owner, "--repo", args.repo]);
    const reconcileArgs = reconcileCommand(args, config, String(projectNumber));
    result.reconcilePlan = commandExec("bun", [reconcilePath, ...reconcileArgs], checkout);
    result.reconcileApply = commandExec("bun", [reconcilePath, ...reconcileArgs, "--apply"], checkout);
    result.installedFiles = installFeedback(feedbackFiles);
    result.projectNumber = projectNumber;
    return result;
  } catch (error) {
    throw new Error(`Project #${projectNumber} was created but bootstrap failed: ${error instanceof Error ? error.message : String(error)}`);
  }
}

async function main(): Promise<void> {
  const args = parseArgs(process.argv.slice(2));
  const result = await runScaffold(args);
  console.log(JSON.stringify({
    mode: args.apply ? "apply" : "plan",
    config: result.config,
    projectNumber: result.projectNumber,
    plannedCommands: result.plannedCommands,
    reconcileCwd: result.reconcileCwd,
    installedFiles: result.installedFiles,
    feedbackTargets: result.feedbackTargets,
    manualSteps: result.manualSteps,
  }, null, 2));
  if (result.reconcilePlan) console.log(`\nReconcile plan:\n${result.reconcilePlan}`);
  if (result.reconcileApply) console.log(`\nReconcile apply:\n${result.reconcileApply}`);
}

if (import.meta.main) {
  main().catch((error) => {
    console.error(`\n✗ ${error instanceof Error ? error.message : String(error)}`);
    process.exit(1);
  });
}
