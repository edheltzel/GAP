/**
 * Todo Extension — Markdown-based todo tracking with dependency support
 *
 * Tools (callable by LLM):
 *   todo_list   — Show all items with status, dependencies, blocked info
 *   todo_add    — Add one or more items, optionally with dependencies
 *   todo_toggle — Toggle completion (blocks if unmet dependencies)
 *   todo_remove — Remove an item (rewrites dependency indices)
 *
 * Command:
 *   /todos      — Interactive TUI view of the todo list
 *
 * State lives in tool result details for proper branch/fork support.
 * Format mirrors standard markdown checkboxes:
 *   - [ ] Do thing (depends: 1, 3)
 *   - [x] Already done
 */

import { StringEnum } from "@mariozechner/pi-ai";
import type { ExtensionAPI, ExtensionContext, Theme } from "@mariozechner/pi-coding-agent";
import { matchesKey, Text, truncateToWidth } from "@mariozechner/pi-tui";
import { Type } from "@sinclair/typebox";

// --- Types ---

interface TodoItem {
	text: string;
	done: boolean;
	depends: number[]; // one-based indices
}

interface TodoState {
	items: TodoItem[];
}

// --- Markdown parsing/formatting ---

function parseTodoMarkdown(md: string): TodoItem[] {
	const items: TodoItem[] = [];
	for (const line of md.split("\n")) {
		const match = line.match(/^- \[([ xX])\] (.+)$/);
		if (!match) continue;
		let text = match[2];
		let depends: number[] = [];
		const depMatch = text.match(/\s*\(depends:\s*([0-9,\s]+)\)\s*$/);
		if (depMatch) {
			depends = depMatch[1]
				.split(",")
				.map((s) => parseInt(s.trim(), 10))
				.filter((n) => !isNaN(n));
			text = text.slice(0, depMatch.index!).trimEnd();
		}
		items.push({ text, done: match[1] !== " ", depends });
	}
	return items;
}

function formatTodoMarkdown(items: TodoItem[]): string {
	return items
		.map((item, i) => {
			const checkbox = item.done ? "x" : " ";
			const dep = item.depends.length > 0 ? ` (depends: ${item.depends.join(", ")})` : "";
			return `- [${checkbox}] ${item.text}${dep}`;
		})
		.join("\n");
}

function getUnmetDeps(items: TodoItem[], index: number): number[] {
	const item = items[index];
	if (!item || item.depends.length === 0) return [];
	return item.depends.filter((d) => {
		const idx = d - 1;
		return idx >= 0 && idx < items.length && !items[idx].done;
	});
}

// --- Tool parameter schemas ---

const TodoAddParams = Type.Object({
	items: Type.Array(
		Type.Object({
			text: Type.String({ description: "Text of the todo item" }),
			depends: Type.Optional(
				Type.Array(Type.Number(), {
					description: "One-based indices of items this depends on. Cannot be completed until dependencies are done.",
				}),
			),
		}),
		{ description: "Items to add", minItems: 1 },
	),
});

const TodoToggleParams = Type.Object({
	index: Type.Number({ description: "One-based index of the item to toggle" }),
});

const TodoRemoveParams = Type.Object({
	index: Type.Number({ description: "One-based index of the item to remove" }),
});

// --- TUI Component for /todos ---

class TodoListView {
	private items: TodoItem[];
	private theme: Theme;
	private onClose: () => void;
	private cachedWidth?: number;
	private cachedLines?: string[];

	constructor(items: TodoItem[], theme: Theme, onClose: () => void) {
		this.items = items;
		this.theme = theme;
		this.onClose = onClose;
	}

	handleInput(data: string): void {
		if (matchesKey(data, "escape") || matchesKey(data, "ctrl+c")) {
			this.onClose();
		}
	}

	render(width: number): string[] {
		if (this.cachedLines && this.cachedWidth === width) return this.cachedLines;

		const lines: string[] = [];
		const th = this.theme;

		lines.push("");
		const title = th.fg("accent", " Todos ");
		lines.push(truncateToWidth(th.fg("borderMuted", "─".repeat(3)) + title + th.fg("borderMuted", "─".repeat(Math.max(0, width - 10))), width));
		lines.push("");

		if (this.items.length === 0) {
			lines.push(truncateToWidth(`  ${th.fg("dim", "No todos yet. Ask the agent to add some!")}`, width));
		} else {
			const done = this.items.filter((t) => t.done).length;
			const blocked = this.items.filter((t, i) => !t.done && getUnmetDeps(this.items, i).length > 0).length;
			let summary = `${done}/${this.items.length} completed`;
			if (blocked > 0) summary += `, ${blocked} blocked`;
			lines.push(truncateToWidth(`  ${th.fg("muted", summary)}`, width));
			lines.push("");

			this.items.forEach((item, i) => {
				const idx = i + 1;
				const unmet = getUnmetDeps(this.items, i);
				let check: string;
				if (item.done) {
					check = th.fg("success", "✓");
				} else if (unmet.length > 0) {
					check = th.fg("warning", "⊘");
				} else {
					check = th.fg("dim", "○");
				}
				const num = th.fg("accent", `${idx}.`);
				const text = item.done ? th.fg("dim", item.text) : th.fg("text", item.text);
				let line = `  ${check} ${num} ${text}`;
				if (item.depends.length > 0) {
					line += " " + th.fg("dim", `(depends: ${item.depends.join(", ")})`);
				}
				if (unmet.length > 0) {
					line += " " + th.fg("warning", `[blocked by ${unmet.join(", ")}]`);
				}
				lines.push(truncateToWidth(line, width));
			});
		}

		lines.push("");
		lines.push(truncateToWidth(`  ${th.fg("dim", "Press Escape to close")}`, width));
		lines.push("");

		this.cachedWidth = width;
		this.cachedLines = lines;
		return lines;
	}

	invalidate(): void {
		this.cachedWidth = undefined;
		this.cachedLines = undefined;
	}
}

// --- Extension ---

export default function (pi: ExtensionAPI) {
	let items: TodoItem[] = [];

	const reconstructState = (ctx: ExtensionContext) => {
		items = [];
		for (const entry of ctx.sessionManager.getBranch()) {
			if (entry.type !== "message") continue;
			const msg = entry.message;
			if (msg.role !== "toolResult") continue;
			if (msg.toolName !== "todo_list" && msg.toolName !== "todo_add" && msg.toolName !== "todo_toggle" && msg.toolName !== "todo_remove") continue;
			const details = msg.details as TodoState | undefined;
			if (details?.items) {
				items = details.items.map((it) => ({ ...it }));
			}
		}
	};

	pi.on("session_start", async (_event, ctx) => reconstructState(ctx));
	pi.on("session_switch", async (_event, ctx) => reconstructState(ctx));
	pi.on("session_fork", async (_event, ctx) => reconstructState(ctx));
	pi.on("session_tree", async (_event, ctx) => reconstructState(ctx));

	const makeState = (): TodoState => ({ items: items.map((it) => ({ ...it, depends: [...it.depends] })) });

	// --- todo_list ---
	pi.registerTool({
		name: "todo_list",
		label: "List Todos",
		description: "List all todo items showing completion status, dependencies, and blocked items.",
		parameters: Type.Object({}),

		async execute() {
			if (items.length === 0) {
				return {
					content: [{ type: "text", text: "No todos yet. Use todo_add to create items." }],
					details: makeState(),
				};
			}
			const done = items.filter((t) => t.done).length;
			const blocked = items.filter((t, i) => !t.done && getUnmetDeps(items, i).length > 0).length;
			const header = `${done}/${items.length} completed${blocked > 0 ? `, ${blocked} blocked` : ""}\n\n`;
			return {
				content: [{ type: "text", text: header + formatTodoMarkdown(items) }],
				details: makeState(),
			};
		},

		renderCall(_args, theme) {
			return new Text(theme.fg("toolTitle", theme.bold("todo_list")), 0, 0);
		},

		renderResult(result, { expanded }, theme) {
			const state = result.details as TodoState | undefined;
			if (!state?.items?.length) return new Text(theme.fg("dim", "No todos"), 0, 0);

			const done = state.items.filter((t) => t.done).length;
			const blocked = state.items.filter((t, i) => !t.done && getUnmetDeps(state.items, i).length > 0).length;
			let text = theme.fg("muted", `${done}/${state.items.length} completed`);
			if (blocked > 0) text += theme.fg("warning", ` (${blocked} blocked)`);

			if (expanded) {
				state.items.forEach((item, i) => {
					const unmet = getUnmetDeps(state.items, i);
					const check = item.done ? theme.fg("success", "✓") : unmet.length > 0 ? theme.fg("warning", "⊘") : theme.fg("dim", "○");
					const itemText = item.done ? theme.fg("dim", item.text) : theme.fg("muted", item.text);
					let line = `\n${check} ${theme.fg("accent", `${i + 1}.`)} ${itemText}`;
					if (item.depends.length > 0) line += " " + theme.fg("dim", `(depends: ${item.depends.join(", ")})`);
					text += line;
				});
			}
			return new Text(text, 0, 0);
		},
	});

	// --- todo_add ---
	pi.registerTool({
		name: "todo_add",
		label: "Add Todos",
		description:
			"Add one or more items to the todo list. Each item can optionally depend on other items by their one-based index — the item cannot be completed until its dependencies are done.",
		parameters: TodoAddParams,

		async execute(_toolCallId, params: { items: Array<{ text: string; depends?: number[] }> }) {
			const newItems: TodoItem[] = params.items.map((p) => ({
				text: p.text,
				done: false,
				depends: p.depends ?? [],
			}));
			items.push(...newItems);
			return {
				content: [{ type: "text", text: `Added ${params.items.length} item(s). Total: ${items.length}.` }],
				details: makeState(),
			};
		},

		renderCall(args, theme) {
			const count = args.items?.length ?? 0;
			return new Text(
				theme.fg("toolTitle", theme.bold("todo_add ")) + theme.fg("muted", `${count} item(s)`),
				0,
				0,
			);
		},

		renderResult(result, _opts, theme) {
			const text = result.content[0];
			return new Text(theme.fg("success", "✓ ") + theme.fg("muted", text?.type === "text" ? text.text : ""), 0, 0);
		},
	});

	// --- todo_toggle ---
	pi.registerTool({
		name: "todo_toggle",
		label: "Toggle Todo",
		description:
			"Toggle completion status of a todo item by one-based index. Cannot mark complete if it has unmet dependencies.",
		parameters: TodoToggleParams,

		async execute(_toolCallId, params: { index: number }) {
			const idx = params.index - 1;
			if (idx < 0 || idx >= items.length) {
				return {
					content: [{ type: "text", text: `Invalid index ${params.index}. There are ${items.length} items.` }],
					details: makeState(),
					isError: true,
				};
			}

			if (!items[idx].done) {
				const unmet = getUnmetDeps(items, idx);
				if (unmet.length > 0) {
					const unmetTexts = unmet.map((d) => `${d}. ${items[d - 1].text}`).join(", ");
					return {
						content: [{ type: "text", text: `Cannot complete item ${params.index}: blocked by unfinished dependencies: ${unmetTexts}` }],
						details: makeState(),
						isError: true,
					};
				}
			}

			items[idx].done = !items[idx].done;
			const status = items[idx].done ? "completed" : "uncompleted";
			return {
				content: [{ type: "text", text: `Item ${params.index} marked as ${status}: "${items[idx].text}"` }],
				details: makeState(),
			};
		},

		renderCall(args, theme) {
			return new Text(
				theme.fg("toolTitle", theme.bold("todo_toggle ")) + theme.fg("accent", `#${args.index}`),
				0,
				0,
			);
		},

		renderResult(result, _opts, theme) {
			if (result.isError) {
				const text = result.content[0];
				return new Text(theme.fg("error", text?.type === "text" ? text.text : "Error"), 0, 0);
			}
			const text = result.content[0];
			return new Text(theme.fg("success", "✓ ") + theme.fg("muted", text?.type === "text" ? text.text : ""), 0, 0);
		},
	});

	// --- todo_remove ---
	pi.registerTool({
		name: "todo_remove",
		label: "Remove Todo",
		description: "Remove a todo item by one-based index. Dependency indices are automatically adjusted.",
		parameters: TodoRemoveParams,

		async execute(_toolCallId, params: { index: number }) {
			const idx = params.index - 1;
			if (idx < 0 || idx >= items.length) {
				return {
					content: [{ type: "text", text: `Invalid index ${params.index}. There are ${items.length} items.` }],
					details: makeState(),
					isError: true,
				};
			}

			const removed = items.splice(idx, 1)[0];
			const removedIndex = params.index; // one-based
			for (const item of items) {
				item.depends = item.depends
					.filter((d) => d !== removedIndex)
					.map((d) => (d > removedIndex ? d - 1 : d));
			}

			return {
				content: [{ type: "text", text: `Removed item ${params.index}: "${removed.text}". ${items.length} remaining.` }],
				details: makeState(),
			};
		},

		renderCall(args, theme) {
			return new Text(
				theme.fg("toolTitle", theme.bold("todo_remove ")) + theme.fg("accent", `#${args.index}`),
				0,
				0,
			);
		},

		renderResult(result, _opts, theme) {
			if (result.isError) {
				const text = result.content[0];
				return new Text(theme.fg("error", text?.type === "text" ? text.text : "Error"), 0, 0);
			}
			const text = result.content[0];
			return new Text(theme.fg("success", "✓ ") + theme.fg("muted", text?.type === "text" ? text.text : ""), 0, 0);
		},
	});

	// --- /todos command ---
	pi.registerCommand("todos", {
		description: "Show all todos with status, dependencies, and blocked items",
		handler: async (_args, ctx) => {
			if (!ctx.hasUI) {
				// Print mode fallback
				if (items.length === 0) {
					ctx.ui.notify("No todos", "info");
				} else {
					ctx.ui.notify(formatTodoMarkdown(items), "info");
				}
				return;
			}

			await ctx.ui.custom<void>((_tui, theme, _kb, done) => {
				return new TodoListView(items, theme, () => done());
			});
		},
	});
}
