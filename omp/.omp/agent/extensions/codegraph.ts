import { execFileSync, spawn, type ChildProcess } from "node:child_process";
import { existsSync } from "node:fs";
import { join } from "node:path";
import { createInterface } from "node:readline";
import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";

const SYSTEM_PROMPT_ADDITION = `
## CodeGraph

This project has a CodeGraph index — a tree-sitter-parsed knowledge graph of every symbol, edge, and file. Reads return structural information grep cannot. Use Semble search to find the right symbol, then navigate with CodeGraph for relevant context.

## Semble Search

**Step 0 — locate the code symbol (skip only if you already know the exact name):**

\`semble search "concept or description" .\`

Semble bridges vocabulary mismatches (e.g. \`"tool call limit"\` → \`MAX_TOOL_ROUNDS\`). Get the symbol name + file, then proceed to step 1.

### Tool Choice

- Structural questions: use CodeGraph. This includes definitions, signatures, callers/callees, impact, architecture, data flow, request construction, persistence, resume/load behavior, and "what happens/current behavior" questions.
- Literal questions: use native search/read only for exact strings, comments, logs, config text, or a small range already identified by CodeGraph.
- Do not use native search/read, codegraph_explore, large owner symbols, or broad symbol searches for read-only structural/behavior answers.

### Micro-Budget

- For read-only structural/behavior questions, spend at most 2 CodeGraph calls by default:
  1. codegraph_context(includeCode:false, maxNodes:8) to find decisive symbols.
  2. One codegraph_node(includeCode:true) for the single decisive boundary, or codegraph_trace if the answer is specifically a path.
- If the first call already shows the decisive type/signature/relationship, answer immediately without a second call.
- Use a third CodeGraph call only if the first two results conflict or the user explicitly asks for more proof.
- If uncertainty remains after the budget, state it as a caveat instead of investigating adjacent plumbing.

### Boundaries

- Prefer decisive boundary symbols over plumbing: public types/schemas, save-load functions, request builders, command/route handlers, adapters.
- Do not inspect parser helpers, option/message types, UI previews, store internals, callbacks, or owner components/classes unless the decisive boundary explicitly delegates there.
- Do not prove negatives by exhaustive search. If the boundary lacks a field/path, answer from that and mention the caveat.
- Avoid duplicate source retrieval. Never fetch source for the same symbol twice.
- Use codegraph_explore only during implementation work when several exact small symbols are needed.
- Do not run git diff / git status for read-only code questions.
`;

interface McpTool {
	name: string;
	description: string;
	inputSchema: Record<string, unknown>;
}

interface McpClient {
	listTools(): Promise<McpTool[]>;
	callTool(name: string, args: Record<string, unknown>): Promise<string>;
	destroy(): void;
}

const clientsByCwd = new Map<string, McpClient>();
let cleanupHooked = false;

function ensureProcessCleanup(): void {
	if (cleanupHooked) return;
	cleanupHooked = true;

	const killAll = () => {
		for (const client of clientsByCwd.values()) client.destroy();
	};

	process.once("exit", killAll);
	for (const signal of ["SIGTERM", "SIGHUP"] as const) {
		process.once(signal, () => {
			killAll();
			process.kill(process.pid, signal);
		});
	}
}

function rejectPending(pending: Map<number, PromiseWithResolvers<unknown>>, error: Error): void {
	for (const waiter of pending.values()) waiter.reject(error);
	pending.clear();
}

function descendantPids(rootPid: number): number[] {
	try {
		const out = execFileSync("ps", ["-axo", "pid=,ppid="], { encoding: "utf8" });
		const children = new Map<number, number[]>();

		for (const line of out.split("\n")) {
			const match = line.trim().match(/^(\d+)\s+(\d+)$/);
			if (!match) continue;

			const pid = Number(match[1]);
			const ppid = Number(match[2]);
			const siblings = children.get(ppid);
			if (siblings) siblings.push(pid);
			else children.set(ppid, [pid]);
		}

		const descendants: number[] = [];
		const stack = [rootPid];
		while (stack.length > 0) {
			const parent = stack.pop();
			if (parent === undefined) break;
			for (const child of children.get(parent) ?? []) {
				descendants.push(child);
				stack.push(child);
			}
		}
		return descendants;
	} catch {
		return [];
	}
}

function killProcessTree(proc: ChildProcess): void {
	const { pid } = proc;
	if (!pid) {
		proc.kill("SIGTERM");
		return;
	}

	if (process.platform === "win32") {
		spawn("taskkill", ["/pid", String(pid), "/T", "/F"], {
			stdio: "ignore",
			windowsHide: true,
		});
		return;
	}

	const targets = descendantPids(pid);
	try {
		process.kill(-pid, "SIGTERM");
	} catch {
		// Best effort cleanup.
	}
	for (const target of [...targets, pid]) {
		try {
			process.kill(target, "SIGTERM");
		} catch {
			// Best effort cleanup.
		}
	}
}

function startMcpClient(cwd: string): Promise<McpClient> {
	const startup = Promise.withResolvers<McpClient>();
	const proc = spawn("codegraph", ["serve", "--mcp"], {
		cwd,
		stdio: ["pipe", "pipe", "pipe"],
		env: process.env,
		detached: process.platform !== "win32",
		windowsHide: true,
	});

	let nextId = 1;
	const pending = new Map<number, PromiseWithResolvers<unknown>>();
	let settled = false;
	let destroyed = false;

	const rl = createInterface({ input: proc.stdout });
	rl.on("line", (line) => {
		const trimmed = line.trim();
		if (!trimmed) return;

		let msg: { id?: number; result?: unknown; error?: { message: string } };
		try {
			msg = JSON.parse(trimmed);
		} catch {
			return;
		}

		if (msg.id === undefined) return;
		const waiter = pending.get(msg.id);
		if (!waiter) return;

		pending.delete(msg.id);
		if (msg.error) waiter.reject(new Error(msg.error.message));
		else waiter.resolve(msg.result);
	});

	function destroy(): void {
		if (destroyed) return;
		destroyed = true;
		rejectPending(pending, new Error("codegraph MCP client destroyed"));
		rl.close();
		proc.stdin.end();
		killProcessTree(proc);
	}

	function failStartup(error: Error): void {
		if (settled) return;
		settled = true;
		destroy();
		startup.reject(error);
	}

	proc.once("error", failStartup);
	proc.once("exit", (code, signal) => {
		const error = new Error(`codegraph MCP exited${code === null ? "" : ` with code ${code}`}${signal ? ` (${signal})` : ""}`);
		rejectPending(pending, error);
		if (!settled) {
			settled = true;
			startup.reject(error);
		}
	});

	function rpc<T>(method: string, params?: unknown): Promise<T> {
		const id = nextId;
		nextId += 1;

		const waiter = Promise.withResolvers<unknown>();
		pending.set(id, waiter);
		proc.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", id, method, params })}\n`);
		return waiter.promise as Promise<T>;
	}

	function notify(method: string, params?: unknown): void {
		proc.stdin.write(`${JSON.stringify({ jsonrpc: "2.0", method, params })}\n`);
	}

	rpc("initialize", {
		protocolVersion: "2024-11-05",
		capabilities: {},
		clientInfo: { name: "omp-codegraph", version: "1.0.0" },
	})
		.then(() => {
			if (destroyed) return;
			settled = true;
			notify("notifications/initialized");
			startup.resolve({
				listTools: () => rpc<{ tools: McpTool[] }>("tools/list").then((result) => result.tools),
				callTool: (name, args) =>
					rpc<{ content: Array<{ type: string; text?: string }> }>("tools/call", {
						name,
						arguments: args,
					}).then((result) =>
						result.content
							.filter((chunk) => chunk.type === "text")
							.map((chunk) => chunk.text ?? "")
							.join("\n"),
					),
				destroy,
			});
		})
		.catch((error) => failStartup(error instanceof Error ? error : new Error(String(error))));

	return startup.promise;
}

function systemPromptWithCodeGraph(systemPrompt: string[]): string[] {
	if (systemPrompt.some((fragment) => fragment.includes("## CodeGraph"))) return systemPrompt;
	return [...systemPrompt, SYSTEM_PROMPT_ADDITION];
}

export default async function codegraph(pi: ExtensionAPI): Promise<void> {
	const cwd = process.cwd();
	if (!existsSync(join(cwd, ".codegraph"))) return;

	clientsByCwd.get(cwd)?.destroy();
	clientsByCwd.delete(cwd);

	let client: McpClient;
	try {
		client = await startMcpClient(cwd);
		clientsByCwd.set(cwd, client);
		ensureProcessCleanup();
	} catch {
		return;
	}

	let tools: McpTool[];
	try {
		tools = await client.listTools();
	} catch {
		client.destroy();
		clientsByCwd.delete(cwd);
		return;
	}

	for (const tool of tools) {
		const toolName = tool.name;
		pi.registerTool({
			name: toolName,
			label: toolName.replace("codegraph_", "").replace(/_/g, " "),
			description: tool.description,
			parameters: tool.inputSchema,
			execute: async (_id, params) => {
				try {
					const text = await client.callTool(toolName, params as Record<string, unknown>);
					return { content: [{ type: "text" as const, text }], details: {} };
				} catch (error) {
					const message = error instanceof Error ? error.message : String(error);
					return {
						content: [{ type: "text" as const, text: `codegraph error: ${message}` }],
						details: {},
					};
				}
			},
		});
	}

	pi.on("before_agent_start", (event) => ({
		systemPrompt: systemPromptWithCodeGraph(event.systemPrompt),
	}));

	pi.on("session_shutdown", () => {
		client.destroy();
		clientsByCwd.delete(cwd);
	});
}
