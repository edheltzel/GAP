import { afterEach, describe, expect, test } from "bun:test";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import type { ExtensionAPI, ExtensionCommandContext, ExtensionContext } from "@oh-my-pi/pi-coding-agent";
import promptTemplateModelExtension from "../src/main.ts";

interface RegisteredCommand {
	description?: string;
	handler: (args: string, ctx: ExtensionCommandContext) => Promise<void>;
}

const temporaryDirectories: string[] = [];

afterEach(() => {
	for (const directory of temporaryDirectories.splice(0)) rmSync(directory, { recursive: true, force: true });
});

describe("OMP prompt-template runtime", () => {
	test("switches execution profile, injects skill and task guidance, then restores", async () => {
		const root = mkdtempSync(join(tmpdir(), "omp-prompt-template-model-"));
		temporaryDirectories.push(root);
		const projectPrompts = join(root, ".omp", "prompts");
		const projectSkill = join(root, ".omp", "skills", "reviewing");
		const projectCommands = join(root, ".omp", "commands");
		const emptyAgentDir = join(root, "agent");
		const globalPrompts = join(emptyAgentDir, "prompts");
		mkdirSync(projectPrompts, { recursive: true });
		mkdirSync(projectCommands, { recursive: true });
		mkdirSync(projectSkill, { recursive: true });
		mkdirSync(emptyAgentDir, { recursive: true });
		mkdirSync(globalPrompts, { recursive: true });
		const projectPromptPath = join(projectPrompts, "review-change.md");
		const invalidPromptPath = join(projectPrompts, "invalid-profile.md");
		const blockedPromptPath = join(projectPrompts, "blocked-profile.md");
		const fallbackPromptPath = join(projectPrompts, "fallback-on-error.md");
		const modelOnlyPromptPath = join(projectPrompts, "model-only.md");
		const reservedPromptPath = join(projectPrompts, "model.md");
		const fileCollisionPromptPath = join(projectPrompts, "file-collision.md");
		const bootstrapPromptPath = join(projectPrompts, "bootstrap-profile.md");
		writeFileSync(
			projectPromptPath,
			`---\ndescription: Review with a specialist\nmodel: [openai-codex/target, openai-codex/original]\nthinking: high\nskill: reviewing\nrestore: true\nsubagent: reviewer\ninheritContext: true\n---\nReview $1 with context: $ARGUMENTS`,
		);
		writeFileSync(
			join(globalPrompts, "review-change.md"),
			`---\ndescription: Global fallback\nmodel: openai-codex/original\n---\nGlobal prompt`,
		);
		writeFileSync(
			invalidPromptPath,
			`---\nthinking: turbo\n---\nThis must never run`,
		);
		writeFileSync(
			blockedPromptPath,
			`---\nmodel: openai-codex/target\n---\nThis must not shadow another extension`,
		);
		writeFileSync(
			fallbackPromptPath,
			`---\nmodel: [openai-codex/broken, openai-codex/original]\n---\nFallback prompt`,
		);
		writeFileSync(
			modelOnlyPromptPath,
			`---\nmodel: openai-codex/target\nrestore: true\n---\nModel-only prompt`,
		);
		writeFileSync(
			reservedPromptPath,
			`---\nmodel: openai-codex/target\n---\nReserved prompt`,
		);
		writeFileSync(
			fileCollisionPromptPath,
			`---\nmodel: openai-codex/target\n---\nProfile collision prompt`,
		);
		writeFileSync(
			bootstrapPromptPath,
			`---\nmodel: openai-codex/target\nrestore: false\n---\nBootstrap prompt`,
		);
		writeFileSync(
			join(projectCommands, "file-collision.md"),
			`---\ndescription: Existing file command\n---\nExisting file command`,
		);
		const skillPath = join(projectSkill, "SKILL.md");
		writeFileSync(skillPath, "Review correctness before style.");

		const previousAgentDir = process.env.PI_CODING_AGENT_DIR;
		process.env.PI_CODING_AGENT_DIR = emptyAgentDir;
		try {
			const commands = new Map<string, RegisteredCommand>();
			const handlers = new Map<string, (event: unknown, ctx: ExtensionContext) => unknown>();
			const sentPrompts: string[] = [];
			const originalModel = { provider: "openai-codex", id: "original" };
			const targetModel = { provider: "openai-codex", id: "target" };
			const brokenModel = { provider: "openai-codex", id: "broken" };
			let currentModel: typeof originalModel | undefined = originalModel;
			let thinking = "medium";
			let dropNextSend = false;
			let failModelRestore = false;

			const pi = {
				on(event: string, handler: (event: unknown, ctx: ExtensionContext) => unknown) {
					handlers.set(event, handler);
				},
				registerCommand(name: string, command: RegisteredCommand) {
					commands.set(name, command);
				},
				getCommands: () => [
					{ name: "reviewing", source: "skill", path: skillPath },
					{ name: "review-change", source: "prompt", path: projectPromptPath },
					{ name: "invalid-profile", source: "prompt", path: invalidPromptPath },
					{ name: "blocked-profile", source: "extension", path: "/other/extension.ts" },
					{ name: "fallback-on-error", source: "prompt", path: fallbackPromptPath },
					{ name: "model-only", source: "prompt", path: modelOnlyPromptPath },
					{ name: "model", source: "prompt", path: reservedPromptPath },
					{ name: "file-collision", source: "prompt", path: fileCollisionPromptPath },
					{ name: "bootstrap-profile", source: "prompt", path: bootstrapPromptPath },
				],
				getActiveTools: () => ["task"],
				getThinkingLevel: () => thinking,
				setThinkingLevel(level: string) {
					thinking = level;
				},
				async setModel(model: typeof originalModel) {
					if (failModelRestore && model === originalModel) return false;
					currentModel = model;
					thinking = model === targetModel ? "minimal" : "low";
					if (model === brokenModel) throw new Error("activation failed");
					return true;
				},
				sendUserMessage(content: string) {
					if (dropNextSend) {
						dropNextSend = false;
						return;
					}
					sentPrompts.push(content);
				},
				logger: { warn() {} },
			} as unknown as ExtensionAPI;

			promptTemplateModelExtension(pi);
			const baseContext = {
				cwd: root,
				hasUI: false,
				models: {
					current: () => currentModel,
					resolve: (spec: string) => {
						if (spec === "openai-codex/target") return targetModel;
						if (spec === "openai-codex/original") return originalModel;
						if (spec === "openai-codex/broken") return brokenModel;
						return undefined;
					},
				},
			} as unknown as ExtensionContext;
			const commandContext = {
				...baseContext,
				async waitForIdle() {},
			} as unknown as ExtensionCommandContext;
			const agentStartEvent = {
				prompt: "Review src/main.ts with context: src/main.ts carefully",
				systemPrompt: ["base system"],
			};
			await handlers.get("session_start")?.({}, baseContext);

			expect(commands.has("blocked-profile")).toBe(false);
			expect(commands.has("model")).toBe(false);
			expect(commands.has("file-collision")).toBe(false);
			const invalidCommand = commands.get("invalid-profile");
			expect(invalidCommand?.description).toContain("Invalid OMP prompt profile");
			await invalidCommand?.handler("", commandContext);
			expect(sentPrompts).toEqual([]);

			const fallbackCommand = commands.get("fallback-on-error");
			await fallbackCommand?.handler("", commandContext);
			expect(currentModel).toBe(originalModel);
			expect(sentPrompts).toEqual(["Fallback prompt"]);
			expect(thinking).toBe("medium");
			sentPrompts.length = 0;

			const command = commands.get("review-change");
			expect(command?.description).toContain("model: openai-codex/target → openai-codex/original");
			await command?.handler("src/main.ts carefully", commandContext);

			expect(currentModel).toBe(targetModel);
			expect(thinking).toBe("high");
			expect(sentPrompts).toEqual(["Review src/main.ts with context: src/main.ts carefully"]);

			const startResult = await handlers.get("before_agent_start")?.(
				agentStartEvent,
				baseContext,
			) as { systemPrompt: string[]; message: { content: string; details: { skillPath: string } } };
			const systemPrompt = startResult.systemPrompt.join("\n");
			expect(systemPrompt).toContain("native `task` tool to the `reviewer` agent");
			expect(systemPrompt).toContain("include the relevant current-session context");
			expect(startResult.message.content).toContain("Review correctness before style.");
			expect(startResult.message.details.skillPath).toBe(skillPath);

			await handlers.get("agent_end")?.({}, baseContext);
			expect(currentModel).toBe(originalModel);
			expect(thinking).toBe("medium");

			const modelOnlyCommand = commands.get("model-only");
			await modelOnlyCommand?.handler("", commandContext);
			expect(currentModel).toBe(targetModel);
			expect(thinking).toBe("minimal");
			await handlers.get("before_agent_start")?.(
				{ prompt: "Model-only prompt", systemPrompt: ["base system"] },
				baseContext,
			);
			await handlers.get("agent_end")?.({}, baseContext);
			expect(currentModel).toBe(originalModel);
			expect(thinking).toBe("medium");

			await command?.handler("src/main.ts carefully", commandContext);
			await handlers.get("before_agent_start")?.(agentStartEvent, baseContext);
			expect(currentModel).toBe(targetModel);
			const preAgentRecovery = await handlers.get("input")?.({ text: "next prompt" }, baseContext);
			expect(preAgentRecovery).toBeUndefined();
			expect(currentModel).toBe(originalModel);
			expect(thinking).toBe("medium");

			dropNextSend = true;
			await command?.handler("src/main.ts carefully", commandContext);
			expect(currentModel).toBe(targetModel);
			expect(thinking).toBe("high");
			expect(sentPrompts).toHaveLength(3);
			const inputResult = await handlers.get("input")?.({ text: "unrelated prompt" }, baseContext);
			expect(inputResult).toBeUndefined();
			expect(currentModel).toBe(originalModel);
			expect(thinking).toBe("medium");

			await command?.handler("src/main.ts carefully", commandContext);
			expect(currentModel).toBe(targetModel);
			expect(thinking).toBe("high");
			expect(sentPrompts).toHaveLength(4);
			await handlers.get("before_agent_start")?.(agentStartEvent, baseContext);
			failModelRestore = true;
			await handlers.get("agent_end")?.({}, baseContext);
			expect(currentModel).toBe(targetModel);
			expect(thinking).toBe("medium");

			failModelRestore = false;
			await handlers.get("session_start")?.({}, baseContext);
			expect(currentModel).toBe(originalModel);
			expect(thinking).toBe("medium");
			await command?.handler("src/main.ts carefully", commandContext);
			expect(sentPrompts).toHaveLength(5);
			expect(currentModel).toBe(targetModel);
			await handlers.get("before_agent_start")?.(agentStartEvent, baseContext);
			await handlers.get("agent_end")?.({}, baseContext);
			expect(currentModel).toBe(originalModel);

			currentModel = undefined;
			thinking = "medium";
			const bootstrapCommand = commands.get("bootstrap-profile");
			await bootstrapCommand?.handler("", commandContext);
			expect(currentModel).toBe(targetModel);
			expect(thinking).toBe("minimal");
			expect(sentPrompts).toHaveLength(6);
			await handlers.get("before_agent_start")?.(
				{ prompt: "Bootstrap prompt", systemPrompt: ["base system"] },
				baseContext,
			);
			await handlers.get("agent_start")?.({}, baseContext);
			await handlers.get("agent_end")?.({}, baseContext);
			expect(currentModel).toBe(targetModel);
			expect(thinking).toBe("minimal");
		} finally {
			if (previousAgentDir === undefined) delete process.env.PI_CODING_AGENT_DIR;
			else process.env.PI_CODING_AGENT_DIR = previousAgentDir;
		}
	});
});
