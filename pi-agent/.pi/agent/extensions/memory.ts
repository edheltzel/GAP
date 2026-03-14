/**
 * Memory Extension - Persistent memory across sessions
 *
 * Stores memories as JSON files:
 * - Global: ~/.pi/agent/memory/memories.json
 * - Project: <cwd>/.pi/memory/memories.json
 *
 * Memories are auto-injected into the system prompt on each agent start.
 */

import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { Type } from "@sinclair/typebox";
import { StringEnum } from "@mariozechner/pi-ai";
import { Text } from "@mariozechner/pi-tui";
import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import { join, dirname } from "node:path";
import { homedir } from "node:os";
import { randomUUID } from "node:crypto";

interface Memory {
	id: string;
	text: string;
	scope: "global" | "project";
	created: number;
	source?: string;
}

function loadMemories(filePath: string): Memory[] {
	try {
		if (!existsSync(filePath)) return [];
		const data = JSON.parse(readFileSync(filePath, "utf-8"));
		return data.memories ?? [];
	} catch {
		return [];
	}
}

function saveMemories(filePath: string, memories: Memory[]): void {
	const dir = dirname(filePath);
	if (!existsSync(dir)) {
		mkdirSync(dir, { recursive: true });
	}
	writeFileSync(filePath, JSON.stringify({ memories }, null, 2), "utf-8");
}

function getGlobalPath(): string {
	return join(homedir(), ".pi", "agent", "memory", "memories.json");
}

function getProjectPath(cwd: string): string {
	return join(cwd, ".pi", "memory", "memories.json");
}

function getAllMemories(cwd: string): Memory[] {
	const global = loadMemories(getGlobalPath()).map((m) => ({ ...m, scope: "global" as const }));
	const project = loadMemories(getProjectPath(cwd)).map((m) => ({ ...m, scope: "project" as const }));
	return [...global, ...project];
}

export default function (pi: ExtensionAPI) {
	// Auto-inject memories into system prompt
	pi.on("before_agent_start", async (event, ctx) => {
		const all = getAllMemories(ctx.cwd);
		if (all.length === 0) return;

		// Take the 50 most recent if there are too many
		const sorted = all.sort((a, b) => b.created - a.created);
		const toInject = sorted.slice(0, 50);

		const lines = toInject.map((m) => `- [${m.scope}] ${m.text}`);
		const section = `\n\n## Memories\n\nThese are lessons and notes from previous sessions:\n\n${lines.join("\n")}`;

		return {
			systemPrompt: event.systemPrompt + section,
		};
	});

	// memory_save
	pi.registerTool({
		name: "memory_save",
		label: "Save Memory",
		description:
			"Save a memory that persists across sessions. Especially useful for saving corrections, lessons learned, user preferences, and important context that should be remembered.",
		parameters: Type.Object({
			text: Type.String({ description: "The memory content to save" }),
			scope: Type.Optional(StringEnum(["global", "project"] as const, { description: "Where to store: 'global' (all projects) or 'project' (current project only). Default: 'project'" })),
			source: Type.Optional(Type.String({ description: "How it was created, e.g. 'correction', 'manual', 'auto'" })),
		}),

		async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
			const scope = params.scope ?? "project";
			const memory: Memory = {
				id: randomUUID(),
				text: params.text,
				scope,
				created: Date.now(),
				source: params.source,
			};

			const filePath = scope === "global" ? getGlobalPath() : getProjectPath(ctx.cwd);
			const existing = loadMemories(filePath);
			existing.push(memory);
			saveMemories(filePath, existing);

			return {
				content: [{ type: "text", text: `Memory saved (${scope}, id: ${memory.id})` }],
				details: { memory },
			};
		},

		renderCall(args, theme) {
			let text = theme.fg("toolTitle", theme.bold("memory_save "));
			text += theme.fg("muted", args.scope ?? "project");
			if (args.source) text += theme.fg("dim", ` [${args.source}]`);
			text += "\n" + theme.fg("dim", `"${args.text}"`);
			return new Text(text, 0, 0);
		},

		renderResult(result, _options, theme) {
			const text = result.content[0];
			return new Text(theme.fg("success", "✓ ") + theme.fg("muted", text?.type === "text" ? text.text : ""), 0, 0);
		},
	});

	// memory_search
	pi.registerTool({
		name: "memory_search",
		label: "Search Memories",
		description: "Search through saved memories using text matching",
		parameters: Type.Object({
			query: Type.String({ description: "Search query" }),
			scope: Type.Optional(StringEnum(["all", "global", "project"] as const, { description: "Which memories to search. Default: 'all'" })),
		}),

		async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
			const scope = params.scope ?? "all";
			let memories: Memory[];

			if (scope === "global") {
				memories = loadMemories(getGlobalPath()).map((m) => ({ ...m, scope: "global" as const }));
			} else if (scope === "project") {
				memories = loadMemories(getProjectPath(ctx.cwd)).map((m) => ({ ...m, scope: "project" as const }));
			} else {
				memories = getAllMemories(ctx.cwd);
			}

			const queryWords = params.query.toLowerCase().split(/\s+/).filter(Boolean);

			const scored = memories.map((m) => {
				const lower = m.text.toLowerCase();
				const score = queryWords.filter((w) => lower.includes(w)).length;
				return { memory: m, score };
			});

			const matches = scored
				.filter((s) => s.score > 0)
				.sort((a, b) => b.score - a.score)
				.slice(0, 20);

			if (matches.length === 0) {
				return {
					content: [{ type: "text", text: "No matching memories found." }],
					details: { matches: [] },
				};
			}

			const lines = matches.map(
				(m) => `[${m.memory.scope}] (score: ${m.score}) ${m.memory.text} (id: ${m.memory.id})`
			);

			return {
				content: [{ type: "text", text: `Found ${matches.length} matching memories:\n\n${lines.join("\n")}` }],
				details: { matches: matches.map((m) => m.memory) },
			};
		},

		renderCall(args, theme) {
			let text = theme.fg("toolTitle", theme.bold("memory_search "));
			text += theme.fg("dim", `"${args.query}"`);
			if (args.scope && args.scope !== "all") text += " " + theme.fg("muted", args.scope);
			return new Text(text, 0, 0);
		},

		renderResult(result, _options, theme) {
			const details = result.details as { matches: Memory[] } | undefined;
			const count = details?.matches?.length ?? 0;
			if (count === 0) {
				return new Text(theme.fg("dim", "No matching memories found."), 0, 0);
			}
			let text = theme.fg("success", `✓ Found ${count} match(es)`);
			for (const m of details!.matches.slice(0, 10)) {
				text += "\n" + theme.fg("muted", `  [${m.scope}] `) + theme.fg("dim", m.text);
			}
			if (count > 10) {
				text += "\n" + theme.fg("dim", `  ... ${count - 10} more`);
			}
			return new Text(text, 0, 0);
		},
	});

	// memory_list
	pi.registerTool({
		name: "memory_list",
		label: "List Memories",
		description: "List all saved memories",
		parameters: Type.Object({
			scope: Type.Optional(StringEnum(["all", "global", "project"] as const, { description: "Which memories to list. Default: 'all'" })),
		}),

		async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
			const scope = params.scope ?? "all";
			let memories: Memory[];

			if (scope === "global") {
				memories = loadMemories(getGlobalPath()).map((m) => ({ ...m, scope: "global" as const }));
			} else if (scope === "project") {
				memories = loadMemories(getProjectPath(ctx.cwd)).map((m) => ({ ...m, scope: "project" as const }));
			} else {
				memories = getAllMemories(ctx.cwd);
			}

			if (memories.length === 0) {
				return {
					content: [{ type: "text", text: "No memories found." }],
					details: { memories: [] },
				};
			}

			const lines = memories.map((m) => {
				const date = new Date(m.created).toISOString().slice(0, 10);
				const src = m.source ? ` (${m.source})` : "";
				return `[${m.scope}] ${m.text} — ${date}${src} (id: ${m.id})`;
			});

			return {
				content: [{ type: "text", text: `${memories.length} memories:\n\n${lines.join("\n")}` }],
				details: { memories },
			};
		},

		renderCall(args, theme) {
			let text = theme.fg("toolTitle", theme.bold("memory_list"));
			if (args.scope && args.scope !== "all") text += " " + theme.fg("muted", args.scope);
			return new Text(text, 0, 0);
		},

		renderResult(result, { expanded }, theme) {
			const details = result.details as { memories: Memory[] } | undefined;
			const count = details?.memories?.length ?? 0;
			if (count === 0) {
				return new Text(theme.fg("dim", "No memories found."), 0, 0);
			}
			let text = theme.fg("success", `✓ ${count} memory(ies)`);
			const display = expanded ? details!.memories : details!.memories.slice(0, 10);
			for (const m of display) {
				const date = new Date(m.created).toISOString().slice(0, 10);
				text += "\n" + theme.fg("muted", `  [${m.scope}] `) + theme.fg("dim", m.text) + theme.fg("dim", ` — ${date}`);
			}
			if (!expanded && count > 10) {
				text += "\n" + theme.fg("dim", `  ... ${count - 10} more`);
			}
			return new Text(text, 0, 0);
		},
	});

	// memory_remove
	pi.registerTool({
		name: "memory_remove",
		label: "Remove Memory",
		description: "Remove a memory by its ID",
		parameters: Type.Object({
			id: Type.String({ description: "The memory ID to remove" }),
		}),

		async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
			// Try global first
			const globalPath = getGlobalPath();
			const globalMemories = loadMemories(globalPath);
			const globalIdx = globalMemories.findIndex((m) => m.id === params.id);

			if (globalIdx !== -1) {
				const removed = globalMemories.splice(globalIdx, 1)[0];
				saveMemories(globalPath, globalMemories);
				return {
					content: [{ type: "text", text: `Removed global memory: ${removed.text}` }],
					details: { removed },
				};
			}

			// Try project
			const projectPath = getProjectPath(ctx.cwd);
			const projectMemories = loadMemories(projectPath);
			const projectIdx = projectMemories.findIndex((m) => m.id === params.id);

			if (projectIdx !== -1) {
				const removed = projectMemories.splice(projectIdx, 1)[0];
				saveMemories(projectPath, projectMemories);
				return {
					content: [{ type: "text", text: `Removed project memory: ${removed.text}` }],
					details: { removed },
				};
			}

			return {
				content: [{ type: "text", text: `Memory with id ${params.id} not found.` }],
				details: { error: "not found" },
			};
		},

		renderCall(args, theme) {
			let text = theme.fg("toolTitle", theme.bold("memory_remove "));
			text += theme.fg("dim", args.id);
			return new Text(text, 0, 0);
		},

		renderResult(result, _options, theme) {
			const details = result.details as { removed?: Memory; error?: string } | undefined;
			if (details?.error) {
				return new Text(theme.fg("error", "✗ Memory not found"), 0, 0);
			}
			return new Text(
				theme.fg("success", "✓ Removed: ") + theme.fg("dim", details?.removed?.text ?? ""),
				0,
				0,
			);
		},
	});
}
