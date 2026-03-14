/**
 * Notifications Extension — System notifications with custom chime sound
 *
 * Supports:
 *   macOS  — osascript notifications + afplay for sound
 *   Linux  — notify-send + aplay/paplay/ffplay for sound
 *
 * Tools:
 *   notify   — Send a system notification with optional sound
 *   ask_user — Play chime + notification + prompt for input
 *
 * Command:
 *   /ping    — Test the chime sound
 *
 * Skips registration on unsupported platforms (no notification tool available).
 */

import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { Type } from "@sinclair/typebox";
import { Text } from "@mariozechner/pi-tui";
import { spawn, execSync } from "node:child_process";
import { join } from "node:path";
import { existsSync } from "node:fs";
import { platform } from "node:os";

function which(cmd: string): boolean {
  try {
    execSync(`which ${cmd}`, { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

const IS_MACOS = platform() === "darwin";
const IS_LINUX = platform() === "linux";

// Detect sound player
function detectSoundPlayer(): string[] | null {
  if (IS_MACOS) return ["afplay"];
  if (IS_LINUX) {
    if (which("paplay")) return ["paplay"];
    if (which("aplay")) return ["aplay"];
    if (which("ffplay")) return ["ffplay", "-nodisp", "-autoexit"];
    if (which("mpv")) return ["mpv", "--no-video"];
  }
  return null;
}

// Detect notification command
function detectNotifier(): boolean {
  if (IS_MACOS) return true; // osascript always available
  if (IS_LINUX) return which("notify-send");
  return false;
}

const SOUND_PLAYER = detectSoundPlayer();
const HAS_NOTIFIER = detectNotifier();

const SOUND_PATH = (() => {
  try {
    if (typeof __dirname !== "undefined") {
      const p = join(__dirname, "chime.mp3");
      if (existsSync(p)) return p;
    }
  } catch {}
  return "/Users/noah/.my-pi/extensions/notifications/chime.mp3";
})();

function playChime() {
  if (!SOUND_PLAYER || !existsSync(SOUND_PATH)) return;
  try {
    const [cmd, ...args] = SOUND_PLAYER;
    const child = spawn(cmd, [...args, SOUND_PATH], {
      detached: true,
      stdio: "ignore",
    });
    child.unref();
  } catch {}
}

function showNotification(message: string, title: string = "pi") {
  if (!HAS_NOTIFIER) return;
  try {
    if (IS_MACOS) {
      const safeMsg = message.replace(/"/g, '\\"').replace(/'/g, "'");
      const safeTitle = title.replace(/"/g, '\\"').replace(/'/g, "'");
      execSync(
        `osascript -e 'display notification "${safeMsg}" with title "${safeTitle}"'`
      );
    } else if (IS_LINUX) {
      execSync(`notify-send ${JSON.stringify(title)} ${JSON.stringify(message)}`);
    }
  } catch {}
}

export default function (pi: ExtensionAPI) {
  // Don't register anything if we have neither sound nor notifications
  if (!SOUND_PLAYER && !HAS_NOTIFIER) return;

  // /ping command - just plays the chime sound for testing
  pi.registerCommand("ping", {
    description: "Play notification chime sound",
    handler: async (_args, ctx) => {
      if (!SOUND_PLAYER) {
        ctx.ui.notify("No sound player available on this platform", "warning");
        return;
      }
      playChime();
      ctx.ui.notify("🔔 Chime played!", "info");
    },
  });

  // notify tool - send a system notification with optional sound
  pi.registerTool({
    name: "notify",
    label: "Notify",
    description:
      "Send a system notification to the user. Use this to alert the user about important events, completed tasks, or when you need their attention.",
    parameters: Type.Object({
      message: Type.String({ description: "Notification message body" }),
      title: Type.Optional(
        Type.String({ description: 'Notification title (default: "pi")' })
      ),
      sound: Type.Optional(
        Type.Boolean({
          description: "Play chime sound (default: true)",
        })
      ),
    }),

    async execute(_toolCallId, params) {
      const title = params.title ?? "pi";
      const sound = params.sound !== false;

      if (sound) playChime();
      showNotification(params.message, title);

      return {
        content: [
          {
            type: "text",
            text: `Notification sent: "${params.message}"`,
          },
        ],
        details: { message: params.message, title, sound },
      };
    },

    renderCall(args, theme) {
      let text = theme.fg("toolTitle", theme.bold("notify "));
      text += theme.fg("muted", `"${args.message}"`);
      if (args.title) text += theme.fg("dim", ` (${args.title})`);
      return new Text(text, 0, 0);
    },

    renderResult(result, _options, theme) {
      return new Text(theme.fg("success", "✓ Notification sent"), 0, 0);
    },
  });

  // ask_user tool - play chime + notification + prompt user for input
  pi.registerTool({
    name: "ask_user",
    label: "Ask User",
    description:
      "Play a notification chime, show a system notification, and prompt the user for input. Use this when you need the user's attention and a response. The user will hear a sound and see a notification, then can type their answer.",
    parameters: Type.Object({
      question: Type.String({
        description: "The question to ask the user",
      }),
    }),

    async execute(_toolCallId, params, _signal, _onUpdate, ctx) {
      playChime();
      showNotification(params.question, "pi needs your input");

      const answer = await ctx.ui.input(params.question, "");

      if (answer === undefined || answer === null) {
        return {
          content: [{ type: "text", text: "User dismissed the prompt without answering." }],
          details: { question: params.question, answer: null },
        };
      }

      return {
        content: [
          {
            type: "text",
            text: `User answered: ${answer}`,
          },
        ],
        details: { question: params.question, answer },
      };
    },

    renderCall(args, theme) {
      let text = theme.fg("toolTitle", theme.bold("ask_user "));
      text += theme.fg("muted", `"${args.question}"`);
      return new Text(text, 0, 0);
    },

    renderResult(result, _options, theme) {
      const answer = result.details?.answer;
      if (answer === null || answer === undefined) {
        return new Text(theme.fg("warning", "⚠ User dismissed prompt"), 0, 0);
      }
      return new Text(
        theme.fg("success", "✓ ") + theme.fg("dim", `"${answer}"`),
        0,
        0
      );
    },
  });
}
