// powerbar-pwd.ts
// Adds a "pwd" segment to pi-powerbar, shown before the git-branch segment.
// Auto-discovered from ~/.pi/agent/extensions/ — no settings.json wiring needed.

import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import * as os from "os";
import * as path from "path";

function shortenPath(p: string): string {
	const home = os.homedir();
	return p.startsWith(home) ? "~" + p.slice(home.length) : p;
}

export default function (pi: ExtensionAPI) {
	// Register the segment so it appears in /extension-settings
	pi.events.emit("powerbar:register-segment", {
		id: "pwd",
		label: "Working Directory",
	});

	function update() {
		pi.events.emit("powerbar:update", {
			id: "pwd",
			text: shortenPath(process.cwd()),
				color: "muted",
		});
	}

	// Emit on session start
	pi.on("session_start", () => update());

	// Re-emit after every bash tool result (cwd may have changed)
	pi.on("tool_result", (_event) => update());
}
