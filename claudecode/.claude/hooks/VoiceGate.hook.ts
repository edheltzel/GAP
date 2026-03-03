#!/usr/bin/env bun
/**
 * VoiceGate.hook.ts - Block Voice Curls from Subagents (PreToolUse)
 *
 * PURPOSE:
 * Prevents background agents / subagents from sending voice notifications.
 * Only the main terminal session (identified by absence of CLAUDE_CODE_AGENT_TASK_ID)
 * is allowed to curl the voice server at localhost:8888.
 *
 * ROOT CAUSE THIS FIXES:
 * Subagents inherit full PAI context (CLAUDE.md → SKILL.md → Algorithm),
 * which mandates voice curls at every phase. Without this gate, every
 * spawned agent triggers voice announcements — flooding the voice server.
 *
 * TRIGGER: PreToolUse (matcher: Bash)
 *
 * DECISION LOGIC:
 * 1. Command doesn't contain "localhost:8888" → PASS (not a voice curl)
 * 2. Command contains "localhost:8888" AND is main session → PASS
 * 3. Command contains "localhost:8888" AND is NOT main session → BLOCK
 *
 * PERFORMANCE: <5ms. Fast-path exit for non-voice commands.
 */

interface HookInput {
  tool_name: string;
  tool_input: {
    command?: string;
  };
  session_id: string;
}

function isMainSession(): boolean {
  // Subagents always have CLAUDE_CODE_AGENT_TASK_ID set by Claude Code.
  // Main interactive sessions do not. This is reliable regardless of terminal
  // emulator (works on WezTerm, Kitty, iTerm2, etc.) — unlike the previous
  // kitty-sessions file check which only worked with Kitty.
  return !process.env.CLAUDE_CODE_AGENT_TASK_ID;
}

async function main() {
  let input: HookInput;
  try {
    const reader = Bun.stdin.stream().getReader();
    let raw = '';
    const read = (async () => {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        raw += new TextDecoder().decode(value, { stream: true });
      }
    })();
    await Promise.race([read, new Promise<void>(r => setTimeout(r, 200))]);
    if (!raw.trim()) {
      console.log(JSON.stringify({ continue: true }));
      return;
    }
    input = JSON.parse(raw);
  } catch {
    console.log(JSON.stringify({ continue: true }));
    return;
  }

  const command = input.tool_input?.command || '';

  // Fast path: not a voice curl → allow immediately
  if (!command.includes('localhost:8888')) {
    console.log(JSON.stringify({ continue: true }));
    return;
  }

  // It's a voice curl — check if main session
  if (isMainSession()) {
    console.log(JSON.stringify({ continue: true }));
    return;
  }

  // Subagent trying to send voice → block silently
  // Return a fake success so the agent thinks it worked and moves on
  console.log(JSON.stringify({
    decision: "block",
    reason: "Voice notifications are only sent from the main session. Subagent voice curls are suppressed."
  }));
}

main().catch(() => {
  console.log(JSON.stringify({ continue: true }));
});
