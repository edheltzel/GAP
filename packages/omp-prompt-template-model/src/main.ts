import { existsSync, readFileSync, readdirSync, realpathSync, statSync } from "node:fs";
import { homedir } from "node:os";
import { basename, dirname, join, parse, resolve } from "node:path";
import { ThinkingLevel as CoreThinkingLevel, type ThinkingLevel } from "@oh-my-pi/pi-agent-core";
import type { Model } from "@oh-my-pi/pi-ai";
import {
	discoverSlashCommands,
	type ExtensionAPI,
	type ExtensionCommandContext,
	type ExtensionContext,
} from "@oh-my-pi/pi-coding-agent";
import { getAgentDir, getProjectPromptsDir, getPromptsDir, parseFrontmatter, prompt } from "@oh-my-pi/pi-utils";
import {
	appendPromptArgs,
	buildDelegationGuidance,
	normalizePromptFrontmatter,
	isValidSkillName,
	parseCommandArgs,
	renderPromptBody,
	templateUsesInlineArgPlaceholders,
	type PromptTemplateConfig,
	type PromptThinkingLevel,
} from "./config.ts";

interface EnhancedPrompt {
	name: string;
	path: string;
	body: string;
	config: PromptTemplateConfig;
}

interface PromptDiagnostic {
	path: string;
	readonly messages: readonly string[];
}

interface InvalidPrompt extends PromptDiagnostic {
	name: string;
}

interface ParsedPromptFile {
	frontmatter: Record<string, unknown>;
	body: string;
}

interface PendingContext {
	expectedPrompt: string;
	injected?: boolean;
	skill?: {
		name: string;
		path: string;
		content: string;
	};
	delegation?: string;
}

interface PendingRestore {
	model?: Model;
	thinking?: ThinkingLevel;
}

interface ModelAttempt {
	readonly model: Model;
	readonly error?: string;
}

type ModelSelection =
	| { readonly kind: "ready"; readonly changed: boolean }
	| { readonly kind: "no-match" }
	| { readonly kind: "switch-failed"; readonly attempts: readonly ModelAttempt[] };

const PROMPT_CONTEXT_TYPE = "omp-prompt-template-context";
const THINKING_BY_NAME: Record<PromptThinkingLevel, ThinkingLevel> = {
	off: CoreThinkingLevel.Off,
	minimal: CoreThinkingLevel.Minimal,
	low: CoreThinkingLevel.Low,
	medium: CoreThinkingLevel.Medium,
	high: CoreThinkingLevel.High,
	xhigh: CoreThinkingLevel.XHigh,
	max: CoreThinkingLevel.Max,
};

const OMP_RESERVED_COMMAND_NAMES = new Set([
	"settings",
	"setup",
	"providers",
	"plan",
	"plan-review",
	"vibe",
	"goal",
	"guided-goal",
	"loop",
	"queue",
	"model",
	"models",
	"switch",
	"fast",
	"computer",
	"prewalk",
	"advisor",
	"export",
	"dump",
	"share",
	"collab",
	"join",
	"leave",
	"browser",
	"copy",
	"todo",
	"session",
	"jobs",
	"usage",
	"stats",
	"changelog",
	"hotkeys",
	"tools",
	"context",
	"extensions",
	"status",
	"agents",
	"branch",
	"fork",
	"tree",
	"login",
	"logout",
	"mcp",
	"ssh",
	"new",
	"clear",
	"fresh",
	"drop",
	"compact",
	"shake",
	"handoff",
	"resume",
	"btw",
	"tan",
	"omfg",
	"retry",
	"debug",
	"memory",
	"rename",
	"move",
	"add-dir",
	"remove-dir",
	"dirs",
	"exit",
	"marketplace",
	"plugins",
	"reload-plugins",
	"force",
	"force:",
	"live",
	"pause",
	"quit",
	"q",
]);

function agentDirectory(): string {
	return process.env.PI_CODING_AGENT_DIR?.trim() || getAgentDir();
}

function errorMessage(error: unknown): string {
	return error instanceof Error ? error.message : String(error);
}

function errorCode(error: unknown): string | undefined {
	if (!error || typeof error !== "object" || !("code" in error)) return undefined;
	return typeof error.code === "string" ? error.code : undefined;
}

function collectMarkdownFiles(root: string, diagnostics: PromptDiagnostic[]): string[] {
	if (!existsSync(root)) return [];
	const files: string[] = [];
	const visitedDirectories = new Set<string>();

	function visit(directory: string): void {
		let canonicalDirectory: string;
		try {
			canonicalDirectory = realpathSync(directory);
		} catch (error) {
			diagnostics.push({
				path: directory,
				messages: [`prompt directory could not be resolved: ${errorMessage(error)}`],
			});
			return;
		}
		if (visitedDirectories.has(canonicalDirectory)) return;
		visitedDirectories.add(canonicalDirectory);

		let entries;
		try {
			entries = readdirSync(directory, { withFileTypes: true }).sort((left, right) =>
				left.name.localeCompare(right.name),
			);
		} catch (error) {
			diagnostics.push({
				path: directory,
				messages: [`prompt directory could not be read: ${errorMessage(error)}`],
			});
			return;
		}

		for (const entry of entries) {
			const entryPath = join(directory, entry.name);
			try {
				const stat = statSync(entryPath);
				if (stat.isDirectory()) visit(entryPath);
				else if (stat.isFile() && entry.name.endsWith(".md")) files.push(entryPath);
			} catch (error) {
				if (errorCode(error) !== "ENOENT") {
					diagnostics.push({
						path: entryPath,
						messages: [`prompt entry could not be inspected: ${errorMessage(error)}`],
					});
				}
			}
		}
	}

	visit(root);
	return files;
}

function loadEnhancedPrompts(cwd: string): {
	prompts: Map<string, EnhancedPrompt>;
	invalidPrompts: Map<string, InvalidPrompt>;
	diagnostics: PromptDiagnostic[];
} {
	const prompts = new Map<string, EnhancedPrompt>();
	const invalidPrompts = new Map<string, InvalidPrompt>();
	const diagnostics: PromptDiagnostic[] = [];
	const claimedNames = new Set<string>();
	const roots = [getProjectPromptsDir(cwd), getPromptsDir(agentDirectory())];
	function recordInvalidPrompt(
		name: string,
		path: string,
		messages: readonly string[],
	): void {
		const diagnostic = { path, messages };
		diagnostics.push(diagnostic);
		invalidPrompts.set(name, { name, ...diagnostic });
	}
	for (const root of roots) {
		for (const filePath of collectMarkdownFiles(root, diagnostics)) {
			const name = basename(filePath, ".md");
			if (claimedNames.has(name)) continue;
			claimedNames.add(name);
			let parsed: ParsedPromptFile;
			try {
				const result = parseFrontmatter(readFileSync(filePath, "utf8"), { source: filePath });
				parsed = { frontmatter: result.frontmatter, body: result.body };
			} catch (error) {
				recordInvalidPrompt(name, filePath, [
					`prompt template could not be loaded: ${errorMessage(error)}`,
				]);
				continue;
			}

			const normalized = normalizePromptFrontmatter(parsed.frontmatter);
			if (normalized.kind === "invalid") {
				recordInvalidPrompt(name, filePath, normalized.diagnostics);
				continue;
			}
			if (normalized.kind === "native") continue;

			prompts.set(name, {
				name,
				path: filePath,
				body: parsed.body,
				config: normalized.config,
			});
		}
	}

	return { prompts, invalidPrompts, diagnostics };
}

function findRepositoryRoot(cwd: string): string | undefined {
	let current = resolve(cwd);
	while (true) {
		if (existsSync(join(current, ".git"))) return current;
		const parent = dirname(current);
		if (parent === current) return undefined;
		current = parent;
	}
}

function skillCandidates(skillName: string, cwd: string): string[] {
	const candidates = [
		join(cwd, ".omp", "skills", skillName, "SKILL.md"),
		join(cwd, ".omp", "skills", `${skillName}.md`),
	];
	const repositoryRoot = findRepositoryRoot(cwd);
	let current = resolve(cwd);
	while (true) {
		candidates.push(
			join(current, ".agents", "skills", skillName, "SKILL.md"),
			join(current, ".agents", "skills", `${skillName}.md`),
		);
		if (current === repositoryRoot || current === parse(current).root) break;
		current = dirname(current);
	}
	candidates.push(
		join(agentDirectory(), "skills", skillName, "SKILL.md"),
		join(agentDirectory(), "skills", `${skillName}.md`),
		join(homedir(), ".agents", "skills", skillName, "SKILL.md"),
		join(homedir(), ".agents", "skills", `${skillName}.md`),
	);
	return candidates;
}

function resolveSkill(
	pi: ExtensionAPI,
	skillName: string,
	cwd: string,
): PendingContext["skill"] | undefined {
	if (!isValidSkillName(skillName)) return undefined;
	const registered = pi.getCommands().find((command) =>
		command.source === "skill" &&
		(command.name === skillName || command.name === `skill:${skillName}`) &&
		typeof command.path === "string",
	);
	const skillPath = registered?.path ?? skillCandidates(skillName, cwd).find((candidate) => existsSync(candidate));
	if (!skillPath) return undefined;
	return { name: skillName, path: skillPath, content: readFileSync(skillPath, "utf8") };
}

function sameModel(left: Model | undefined, right: Model | undefined): boolean {
	return left?.provider === right?.provider && left?.id === right?.id;
}

function samePath(left: string, right: string): boolean {
	try {
		return realpathSync(left) === realpathSync(right);
	} catch {
		return resolve(left) === resolve(right);
	}
}

function promptDescription(template: EnhancedPrompt): string {
	const { config } = template;
	const task = config.subagent
		? `task: ${config.subagent === true ? "default" : config.subagent}`
		: undefined;

	const attributes = [
		config.modelSpecs.length > 0 ? `model: ${config.modelSpecs.join(" → ")}` : undefined,
		config.thinking ? `thinking: ${config.thinking}` : undefined,
		config.skill ? `skill: ${config.skill}` : undefined,
		task,
	].filter(Boolean);
	const suffix = attributes.length > 0 ? ` [${attributes.join(", ")}]` : "";
	return `${config.description || "Prompt template with an OMP execution profile"}${suffix}`;
}

export default function promptTemplateModelExtension(pi: ExtensionAPI): void {
	let enhancedPrompts = new Map<string, EnhancedPrompt>();
	let invalidPrompts = new Map<string, InvalidPrompt>();
	let pendingContext: PendingContext | undefined;
	let pendingRestore: PendingRestore | undefined;
	let diagnosticsFingerprint = "";

	const registeredCommands = new Set<string>();

	const reportDiagnostics = (diagnostics: PromptDiagnostic[], ctx: ExtensionContext): void => {
		const fingerprint = diagnostics
			.map((diagnostic) => `${diagnostic.path}:${diagnostic.messages.join("|")}`)
			.join("\n");
		if (!fingerprint) {
			diagnosticsFingerprint = "";
			return;
		}
		if (fingerprint === diagnosticsFingerprint) return;
		diagnosticsFingerprint = fingerprint;
		for (const diagnostic of diagnostics) {
			pi.logger.warn("Prompt template frontmatter ignored", {
				path: diagnostic.path,
				messages: diagnostic.messages,
			});
		}
		if (ctx.hasUI) {
			ctx.ui.notify(
				`${diagnostics.length} prompt template${diagnostics.length === 1 ? "" : "s"} had invalid OMP frontmatter`,
				"warning",
			);
		}
	};

	const reportRuntimeIssue = (
		ctx: ExtensionContext,
		message: string,
		level: "error" | "warning" = "error",
		error?: unknown,
	): void => {
		const detail = error === undefined ? message : `${message}: ${errorMessage(error)}`;
		pi.logger.warn("Prompt template runtime issue", {
			message,
			...(error === undefined ? {} : { error: errorMessage(error) }),
		});
		if (ctx.hasUI) ctx.ui.notify(detail, level);
	};

	const restoreSession = async (
		ctx: ExtensionContext,
		restore: PendingRestore | undefined,
	): Promise<PendingRestore | undefined> => {
		if (!restore) return undefined;
		const remaining: PendingRestore = {};

		if (restore.model && !sameModel(ctx.models.current(), restore.model)) {
			try {
				const restored = await pi.setModel(restore.model);
				if (!restored) {
					remaining.model = restore.model;
					reportRuntimeIssue(
						ctx,
						`Could not restore model ${restore.model.provider}/${restore.model.id}`,
						"warning",
					);
				}
			} catch (error) {
				remaining.model = restore.model;
				reportRuntimeIssue(
					ctx,
					`Could not restore model ${restore.model.provider}/${restore.model.id}`,
					"warning",
					error,
				);
			}
		}

		if (restore.thinking !== undefined && pi.getThinkingLevel() !== restore.thinking) {
			try {
				pi.setThinkingLevel(restore.thinking);
			} catch (error) {
				remaining.thinking = restore.thinking;
				reportRuntimeIssue(ctx, `Could not restore thinking level ${restore.thinking}`, "warning", error);
			}
		}

		if (remaining.model && restore.thinking !== undefined) {
			remaining.thinking = restore.thinking;
		}

		return remaining.model || remaining.thinking !== undefined ? remaining : undefined;
	};

	const restorePending = async (ctx: ExtensionContext): Promise<boolean> => {
		pendingRestore = await restoreSession(ctx, pendingRestore);
		return pendingRestore === undefined;
	};

	const selectModel = async (
		modelSpecs: readonly string[],
		ctx: ExtensionContext,
	): Promise<ModelSelection> => {
		const initialModel = ctx.models.current();
		if (modelSpecs.length === 0) return { kind: "ready", changed: false };

		const attempts: ModelAttempt[] = [];
		for (const spec of modelSpecs) {
			const candidate = ctx.models.resolve(spec);
			if (!candidate) continue;

			const attemptModel = ctx.models.current();
			const attemptThinking = pi.getThinkingLevel();
			if (sameModel(candidate, attemptModel)) {
				return {
					kind: "ready",
					changed: !sameModel(candidate, initialModel),
				};
			}

			let activationError: string | undefined;
			try {
				const activated = await pi.setModel(candidate);
				if (activated && sameModel(ctx.models.current(), candidate)) {
					return {
						kind: "ready",
						changed: !sameModel(candidate, initialModel),
					};
				}
				if (activated) activationError = "activation returned success without selecting the model";
			} catch (error) {
				activationError = errorMessage(error);
			}

			const modelDrifted = !sameModel(ctx.models.current(), attemptModel);
			const thinkingDrifted = pi.getThinkingLevel() !== attemptThinking;
			if (modelDrifted || thinkingDrifted) {
				if (!attemptModel) {
					throw new Error(`Model activation for ${candidate.provider}/${candidate.id} changed an empty model state`);
				}
				const remaining = await restoreSession(ctx, {
					...(modelDrifted ? { model: attemptModel } : {}),
					...(thinkingDrifted ? { thinking: attemptThinking } : {}),
				});
				if (remaining) {
					throw new Error(
						`Could not roll back failed activation for ${candidate.provider}/${candidate.id}`,
					);
				}
			}
			attempts.push({ model: candidate, ...(activationError ? { error: activationError } : {}) });
		}
		return attempts.length === 0
			? { kind: "no-match" }
			: { kind: "switch-failed", attempts };
	};

	const runPrompt = async (name: string, argsText: string, ctx: ExtensionCommandContext): Promise<void> => {
		try {
			await ctx.waitForIdle();
		} catch (error) {
			reportRuntimeIssue(ctx, `Prompt template /${name} could not wait for the active turn`, "error", error);
			return;
		}

		const refreshed = loadEnhancedPrompts(ctx.cwd);
		enhancedPrompts = refreshed.prompts;
		invalidPrompts = refreshed.invalidPrompts;
		reportDiagnostics(refreshed.diagnostics, ctx);
		const invalidPrompt = invalidPrompts.get(name);
		if (invalidPrompt) {
			reportRuntimeIssue(
				ctx,
				`Prompt template /${name} has invalid OMP frontmatter: ${invalidPrompt.messages.join("; ")}`,
			);
			return;
		}

		const template = enhancedPrompts.get(name);
		if (!template) {
			reportRuntimeIssue(ctx, `Prompt template /${name} no longer exists`);
			return;
		}
		pendingContext = undefined;
		if (pendingRestore && !(await restorePending(ctx))) {
			reportRuntimeIssue(ctx, "The previous prompt profile could not be restored; refusing to start another");
			return;
		}
		if (template.config.subagent && !pi.getActiveTools().includes("task")) {
			reportRuntimeIssue(ctx, `Prompt template /${name} requires the native task tool`);
			return;
		}

		let skill: PendingContext["skill"];
		if (template.config.skill) {
			try {
				skill = resolveSkill(pi, template.config.skill, ctx.cwd);
			} catch (error) {
				reportRuntimeIssue(ctx, `Failed to load skill ${template.config.skill}`, "error", error);
				return;
			}
			if (!skill) {
				reportRuntimeIssue(ctx, `Skill ${template.config.skill} was not found`);
				return;
			}
		}

		const originalModel = ctx.models.current();
		const originalThinking = pi.getThinkingLevel();
		if (template.config.restore && template.config.modelSpecs.length > 0 && !originalModel) {
			reportRuntimeIssue(ctx, `Prompt template /${name} cannot restore because no original model is active`);
			return;
		}

		try {
			const selected = await selectModel(template.config.modelSpecs, ctx);
			if (selected.kind === "no-match") {
				reportRuntimeIssue(ctx, `No available model matched: ${template.config.modelSpecs.join(", ")}`);
				return;
			}
			if (selected.kind === "switch-failed") {
				const attempted = selected.attempts
					.map((attempt) => {
						const model = `${attempt.model.provider}/${attempt.model.id}`;
						return attempt.error ? `${model} (${attempt.error})` : model;
					})
					.join(", ");
				reportRuntimeIssue(ctx, `Resolved models could not be activated: ${attempted}`);
				return;
			}

			const requestedThinking = template.config.thinking
				? THINKING_BY_NAME[template.config.thinking]
				: undefined;
			const thinkingChanged = requestedThinking !== undefined && requestedThinking !== pi.getThinkingLevel();
			if (requestedThinking !== undefined) pi.setThinkingLevel(requestedThinking);

			const args = parseCommandArgs(argsText);
			const joinedArgs = args.join(" ");
			const usesInlineArgs = templateUsesInlineArgPlaceholders(template.body);
			const substituted = renderPromptBody(template.body, args);
			const renderedTemplate = prompt.render(substituted, {
				args,
				ARGUMENTS: joinedArgs,
				arguments: joinedArgs,
			});
			const rendered = appendPromptArgs(renderedTemplate, joinedArgs, usesInlineArgs);
			const delegation = template.config.subagent
				? buildDelegationGuidance(name, template.config.subagent, template.config.inheritContext)
				: undefined;
			pendingContext = {
				expectedPrompt: rendered,
				...(skill ? { skill } : {}),
				...(delegation ? { delegation } : {}),
			};
			if (template.config.restore) {
				const restoreModel = selected.changed ? originalModel : undefined;
				const restoreThinking = selected.changed || thinkingChanged ? originalThinking : undefined;
				pendingRestore = restoreModel || restoreThinking !== undefined
					? {
						...(restoreModel ? { model: restoreModel } : {}),
						...(restoreThinking !== undefined ? { thinking: restoreThinking } : {}),
					}
					: undefined;
			}
			pi.sendUserMessage(rendered);
		} catch (error) {
			pendingContext = undefined;
			pendingRestore = undefined;
			const rollback: PendingRestore = {
				...(!sameModel(ctx.models.current(), originalModel) && originalModel ? { model: originalModel } : {}),
				...(pi.getThinkingLevel() !== originalThinking ? { thinking: originalThinking } : {}),
			};
			pendingRestore = await restoreSession(ctx, rollback);
			reportRuntimeIssue(ctx, `Prompt template /${name} failed`, "error", error);
		}
	};

	const refreshPrompts = async (ctx: ExtensionContext): Promise<void> => {
		const loaded = loadEnhancedPrompts(ctx.cwd);
		enhancedPrompts = loaded.prompts;
		invalidPrompts = loaded.invalidPrompts;
		reportDiagnostics(loaded.diagnostics, ctx);

		let fileCommandNames: Set<string>;
		try {
			fileCommandNames = new Set((await discoverSlashCommands({ cwd: ctx.cwd })).map((command) => command.name));
		} catch (error) {
			reportRuntimeIssue(ctx, "Could not inspect OMP file commands; prompt profiles were not registered", "error", error);
			return;
		}

		const definitions: Array<EnhancedPrompt | InvalidPrompt> = [
			...enhancedPrompts.values(),
			...invalidPrompts.values(),
		];
		for (const definition of definitions) {
			const { name } = definition;
			if (registeredCommands.has(name)) continue;
			const collision = pi.getCommands().find((command) => command.name === name);
			const matchingNativePrompt = collision?.source === "prompt" &&
				typeof collision.path === "string" &&
				samePath(collision.path, definition.path);
			const reservedCollision = OMP_RESERVED_COMMAND_NAMES.has(name);
			const fileCollision = fileCommandNames.has(name);
			if (reservedCollision || fileCollision || (collision && !matchingNativePrompt)) {
				let existingSource: string | undefined = collision?.source;
				if (reservedCollision) existingSource = "builtin";
				else if (fileCollision) existingSource = "file";
				pi.logger.warn("Prompt template command collision", {
					name,
					path: definition.path,
					existingSource,
					existingPath: collision?.path,
				});
				continue;
			}

			try {
				pi.registerCommand(name, {
					description: "config" in definition
						? promptDescription(definition)
						: `Invalid OMP prompt profile: ${definition.messages.join("; ")}`,
					handler: async (args, commandContext) => runPrompt(name, args, commandContext),
				});
				registeredCommands.add(name);
			} catch (error) {
				reportRuntimeIssue(ctx, `Could not register prompt template /${name}`, "error", error);
			}
		}
	};

	pi.on("session_start", async (_event, ctx) => {
		pendingContext = undefined;
		await restorePending(ctx);
		await refreshPrompts(ctx);
	});

	pi.on("session_before_switch", async (_event, ctx) => {
		pendingContext = undefined;
		return await restorePending(ctx) ? undefined : { cancel: true };
	});

	pi.on("session_shutdown", async (_event, ctx) => {
		pendingContext = undefined;
		await restorePending(ctx);
	});

	pi.on("input", async (_event, ctx) => {
		if (!pendingContext) return undefined;
		pendingContext = undefined;
		if (await restorePending(ctx)) return undefined;
		reportRuntimeIssue(ctx, "The previous prompt profile could not be restored; input was blocked");
		return { handled: true };
	});

	pi.on("before_agent_start", async (event, ctx) => {
		const context = pendingContext;
		if (!context) return undefined;
		if (event.prompt !== context.expectedPrompt) {
			pendingContext = undefined;
			if (!(await restorePending(ctx))) {
				reportRuntimeIssue(ctx, "The previous prompt profile could not be restored; aborting the next turn");
				ctx.abort();
			}
			return undefined;
		}
		if (context.injected) return undefined;
		context.injected = true;
		return {
			...(context.delegation ? { systemPrompt: [...event.systemPrompt, context.delegation] } : {}),
			...(context.skill
				? {
					message: {
						customType: PROMPT_CONTEXT_TYPE,
						content: `<skill name="${context.skill.name}">\n${context.skill.content}\n</skill>`,
						display: true,
						details: { skillName: context.skill.name, skillPath: context.skill.path },
					},
				}
				: {}),
		};
	});

	pi.on("agent_start", async () => {
		pendingContext = undefined;
	});

	pi.on("agent_end", async (_event, ctx) => {
		pendingContext = undefined;
		await restorePending(ctx);
	});
}
