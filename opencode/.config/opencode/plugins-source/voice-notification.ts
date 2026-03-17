/**
 * OpenCode Voice Notification Plugin
 *
 * Integrates with the shared VoiceServer (port 8888) to provide:
 * - Session start/end notifications
 * - Task completion announcements
 * - Permission request alerts
 * - Test result notifications
 *
 * Uses shared voice config from ~/.claude/settings.json (Atlas manages both tools)
 */

import { readFileSync, existsSync } from "fs"
import { join } from "path"
import { homedir } from "os"
import { spawn } from "child_process"

// ============================================================================
// Configuration
// ============================================================================

const VOICE_SERVER_URL = 'http://localhost:8888'
const VOICE_SERVER_DIR = join(homedir(), '.claude', 'VoiceServer')
const SETTINGS_PATH = join(homedir(), '.claude', 'settings.json')

interface VoiceConfig {
  voiceId: string
  catchphrase: string
  principalName: string
  aiName: string
  voice?: {
    stability?: number
    similarity_boost?: number
    style?: number
    speed?: number
    use_speaker_boost?: boolean
  }
}

const DEFAULT_CONFIG: VoiceConfig = {
  voiceId: 'AXdMgz6evoL7OPd7eU12',
  catchphrase: 'OpenCode ready.',
  principalName: 'User',
  aiName: 'OpenCode'
}

// ============================================================================
// Voice Config Loader (reads from shared Claude Code settings)
// ============================================================================

function loadVoiceConfig(): VoiceConfig {
  if (!existsSync(SETTINGS_PATH)) {
    console.log('[voice] No settings.json found, using defaults')
    return DEFAULT_CONFIG
  }

  try {
    const settings = JSON.parse(readFileSync(SETTINGS_PATH, 'utf-8'))
    const daidentity = settings.daidentity || {}
    const principal = settings.principal || {}

    return {
      voiceId: daidentity.voiceId || DEFAULT_CONFIG.voiceId,
      catchphrase: daidentity.startupCatchphrase || DEFAULT_CONFIG.catchphrase,
      principalName: principal.name || DEFAULT_CONFIG.principalName,
      aiName: daidentity.name || DEFAULT_CONFIG.aiName,
      voice: daidentity.voice || {}
    }
  } catch (error) {
    console.error('[voice] Failed to load settings:', error)
    return DEFAULT_CONFIG
  }
}

// ============================================================================
// Voice Server Management
// ============================================================================

async function isServerRunning(): Promise<boolean> {
  try {
    const response = await fetch(`${VOICE_SERVER_URL}/health`, {
      signal: AbortSignal.timeout(2000)
    })
    return response.ok
  } catch {
    return false
  }
}

async function startServer(): Promise<boolean> {
  const startScript = join(VOICE_SERVER_DIR, 'start-server.sh')

  if (!existsSync(startScript)) {
    console.error('[voice] start-server.sh not found at:', startScript)
    return false
  }

  return new Promise((resolve) => {
    const proc = spawn('bash', [startScript, '--quiet'], {
      detached: true,
      stdio: 'ignore'
    })

    proc.unref()

    // Give it a moment to start
    setTimeout(async () => {
      const running = await isServerRunning()
      if (running) {
        console.log('[voice] Server started successfully')
      } else {
        console.log('[voice] Server failed to start')
      }
      resolve(running)
    }, 2000)
  })
}

async function ensureServerRunning(): Promise<boolean> {
  if (await isServerRunning()) {
    return true
  }

  console.log('[voice] Server not running, attempting to start...')
  return await startServer()
}

// ============================================================================
// Notification Functions
// ============================================================================

async function notify(message: string, options: {
  title?: string
  voiceEnabled?: boolean
} = {}): Promise<boolean> {
  const config = loadVoiceConfig()

  // Ensure server is running
  if (!(await ensureServerRunning())) {
    console.log('[voice] Server unavailable, skipping notification')
    return false
  }

  try {
    const response = await fetch(`${VOICE_SERVER_URL}/notify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        message,
        title: options.title || config.aiName,
        voice_enabled: options.voiceEnabled !== false
        // NO voice_id - VoiceServer uses identity from voices.json
      }),
      signal: AbortSignal.timeout(15000) // 15s timeout for speech
    })

    if (!response.ok) {
      console.error('[voice] Notification failed:', response.status)
      return false
    }

    return true
  } catch (error) {
    console.error('[voice] Notification error:', error)
    return false
  }
}

// ============================================================================
// Debouncing for session.idle
// ============================================================================

let lastIdleNotification = 0
const IDLE_DEBOUNCE_MS = 10000 // 10 seconds between idle notifications

function shouldNotifyIdle(): boolean {
  const now = Date.now()
  if (now - lastIdleNotification < IDLE_DEBOUNCE_MS) {
    return false
  }
  lastIdleNotification = now
  return true
}

// Track if we've sent startup notification this session
let startupNotificationSent = false

// ============================================================================
// Test Detection Patterns
// ============================================================================

const TEST_PATTERNS = [
  /\b(npm|yarn|pnpm|bun)\s+(run\s+)?test/i,
  /\bpytest\b/i,
  /\bjest\b/i,
  /\bvitest\b/i,
  /\bmocha\b/i,
  /\bcargo\s+test\b/i,
  /\bgo\s+test\b/i,
  /\bmake\s+test\b/i,
]

function isTestCommand(command: string): boolean {
  return TEST_PATTERNS.some(pattern => pattern.test(command))
}

// ============================================================================
// Summary Extraction
// ============================================================================

/**
 * Extract a speakable summary from the last AI message.
 * Looks for patterns like:
 * - "📋 SUMMARY: ..." (PAI format)
 * - "🗣️ Atlas: ..." (voice line)
 * - Last meaningful line of the response
 */
function extractSummary(text: string, maxLength: number = 150): string | null {
  if (!text || text.trim().length === 0) return null

  // Try to find PAI voice line: 🗣️ Atlas: ...
  const voiceMatch = text.match(/🗣️\s*\w+:\s*(.+?)(?:\n|$)/i)
  if (voiceMatch) {
    return voiceMatch[1].trim().substring(0, maxLength)
  }

  // Try to find PAI summary: 📋 SUMMARY: ...
  const summaryMatch = text.match(/📋\s*SUMMARY:\s*(.+?)(?:\n|$)/i)
  if (summaryMatch) {
    return summaryMatch[1].trim().substring(0, maxLength)
  }

  // Fallback: get last non-empty line that looks like a conclusion
  const lines = text.split('\n').filter(l => l.trim().length > 10)
  if (lines.length > 0) {
    const lastLine = lines[lines.length - 1].trim()
    // Skip lines that are just code or formatting
    if (!lastLine.startsWith('```') && !lastLine.startsWith('|') && !lastLine.startsWith('-')) {
      return lastLine.substring(0, maxLength)
    }
  }

  return null
}

// ============================================================================
// Plugin Export
// ============================================================================

export default async (ctx: any) => {
  const config = loadVoiceConfig()
  const client = ctx.client // SDK client for accessing messages

  console.log(`[voice] Plugin loaded - AI: ${config.aiName}, User: ${config.principalName}`)

  // Store current session ID when we get it
  let currentSessionId: string | null = null

  return {
    // ========================================================================
    // Event Handlers
    // ========================================================================
    event: async ({ event }: { event: { type: string; [key: string]: any } }) => {
      // Track session ID from events
      if (event.session?.id) {
        currentSessionId = event.session.id
      }

      switch (event.type) {
        // --------------------------------------------------------------------
        // Session Events
        // --------------------------------------------------------------------
        case 'session.created':
          if (!startupNotificationSent) {
            startupNotificationSent = true
            currentSessionId = event.session?.id || null
            // Personalize the catchphrase
            const greeting = config.catchphrase
              .replace('{USER}', config.principalName)
              .replace('{AI}', config.aiName)
            await notify(greeting, { title: 'Session Started' })
          }
          break

        case 'session.idle':
          if (shouldNotifyIdle()) {
            // Try to get a summary from the last AI message
            let summary: string | null = null

            if (client && currentSessionId) {
              try {
                const messages = await client.session.messages({
                  path: { id: currentSessionId }
                })

                // Find last assistant message
                if (messages && Array.isArray(messages)) {
                  for (let i = messages.length - 1; i >= 0; i--) {
                    const msg = messages[i]
                    if (msg?.info?.role === 'assistant' && msg?.parts) {
                      // Extract text from parts
                      const textParts = msg.parts
                        .filter((p: any) => p.type === 'text')
                        .map((p: any) => p.text)
                        .join('\n')

                      summary = extractSummary(textParts)
                      if (summary) break
                    }
                  }
                }
              } catch (err) {
                console.log('[voice] Could not fetch messages:', err)
              }
            }

            // Use summary or fallback to generic message
            const message = summary || 'Task completed.'
            await notify(message, { title: 'Ready' })
          }
          break

        case 'session.deleted':
          await notify('Session ended. Goodbye.', { title: 'Goodbye' })
          break

        // --------------------------------------------------------------------
        // Permission Events
        // --------------------------------------------------------------------
        case 'permission.asked':
          await notify(`${config.principalName}, waiting for your approval.`, {
            title: 'Approval Needed'
          })
          break

        // --------------------------------------------------------------------
        // Error Events (if available)
        // --------------------------------------------------------------------
        case 'session.error':
          await notify('An error occurred.', { title: 'Error' })
          break
      }
    },

    // ========================================================================
    // Tool Execution Hooks
    // ========================================================================
    tool: {
      after: async (
        tool: { name: string },
        args: { command?: string; [key: string]: any },
        result: { exitCode?: number; [key: string]: any }
      ) => {
        // Detect test command completions
        if (tool.name === 'bash' && args.command) {
          if (isTestCommand(args.command)) {
            const success = result?.exitCode === 0
            if (success) {
              await notify('Tests passed.', { title: 'Tests' })
            } else {
              await notify('Tests failed.', { title: 'Tests' })
            }
          }
        }
      }
    }
  }
}
