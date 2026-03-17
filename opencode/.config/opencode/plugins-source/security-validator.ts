/**
 * SecurityValidator.ts - Security validation for tool execution (OpenCode Plugin)
 *
 * PURPOSE:
 * Validates Bash commands and file operations against security patterns before
 * execution. Prevents accidental or malicious operations that could damage the
 * system, expose secrets, or compromise security.
 *
 * TRIGGER: tool.execute.before (Bash, Edit, Write, Read)
 *
 * PORTED FROM: SecurityValidator.hook.ts (Claude Code)
 *
 * BEHAVIOR:
 * - Blocks catastrophic operations (exit with Error)
 * - Prompts for confirmation on risky operations (ask permission)
 * - Logs all security events to ~/.claude/MEMORY/SECURITY/
 */

import { readFileSync, existsSync, writeFileSync, mkdirSync } from 'fs';
import { join, dirname } from 'path';
import { homedir } from 'os';
import { parse as parseYaml } from 'yaml';

const CLAUDE_DIR = join(homedir(), '.claude');
const PATTERNS_PATH = join(CLAUDE_DIR, 'hooks', 'patterns.yaml');
const SECURITY_LOG_DIR = join(CLAUDE_DIR, 'MEMORY', 'SECURITY');

interface SecurityEvent {
  timestamp: string;
  tool: string;
  input: Record<string, unknown>;
  decision: 'allow' | 'block' | 'ask';
  pattern?: string;
  message?: string;
}

interface SecurityPatterns {
  bash?: {
    blocked?: string[];
    confirm?: string[];
    alert?: string[];
  };
  files?: {
    zeroAccess?: string[];
    readOnly?: string[];
    confirmWrite?: string[];
    noDelete?: string[];
  };
}

// Default patterns if patterns.yaml not found
const DEFAULT_PATTERNS: SecurityPatterns = {
  bash: {
    blocked: [
      'rm -rf /',
      'rm -rf /*',
      'mkfs',
      'dd if=',
      ':(){:|:&};:',
      'chmod -R 777 /',
      'chown -R',
      '>/dev/sda',
      'curl .* | bash',
      'wget .* | bash',
    ],
    confirm: [
      'git push --force',
      'git push -f',
      'npm publish',
      'yarn publish',
      'pnpm publish',
      'sudo',
      'su ',
      'chmod',
      'chown',
    ],
    alert: ['aws ', 'gcloud ', 'kubectl delete', 'terraform destroy'],
  },
  files: {
    zeroAccess: ['.ssh', '.gnupg', 'credentials', 'secrets', '.env'],
    readOnly: ['/etc/', '/usr/', '/System/'],
    confirmWrite: ['package.json', 'tsconfig.json', '.gitignore'],
    noDelete: ['.git/', 'node_modules/'],
  },
};

function loadPatterns(): SecurityPatterns {
  try {
    if (existsSync(PATTERNS_PATH)) {
      const content = readFileSync(PATTERNS_PATH, 'utf-8');
      return parseYaml(content) as SecurityPatterns;
    }
  } catch (error) {
    console.error('[SecurityValidator] Failed to load patterns.yaml, using defaults:', error);
  }
  return DEFAULT_PATTERNS;
}

function logSecurity(event: SecurityEvent): void {
  try {
    const now = new Date();
    const yearMonth = `${now.getFullYear()}/${String(now.getMonth() + 1).padStart(2, '0')}`;
    const logDir = join(SECURITY_LOG_DIR, yearMonth);
    
    if (!existsSync(logDir)) {
      mkdirSync(logDir, { recursive: true });
    }
    
    const timestamp = now.toISOString().replace(/[:.]/g, '-');
    const summary = event.message?.slice(0, 50).replace(/[^a-zA-Z0-9]/g, '-') || 'event';
    const logPath = join(logDir, `security-${summary}-${timestamp}.jsonl`);
    
    writeFileSync(logPath, JSON.stringify(event) + '\n', 'utf-8');
  } catch (error) {
    // Non-fatal - don't block operations for logging failures
    console.error('[SecurityValidator] Failed to log security event:', error);
  }
}

function checkBashCommand(command: string, patterns: SecurityPatterns): { 
  decision: 'allow' | 'block' | 'ask'; 
  pattern?: string; 
  message?: string;
} {
  const bashPatterns = patterns.bash || {};
  
  // Check blocked patterns
  for (const pattern of bashPatterns.blocked || []) {
    if (new RegExp(pattern, 'i').test(command)) {
      return {
        decision: 'block',
        pattern,
        message: `Blocked catastrophic command matching: ${pattern}`,
      };
    }
  }
  
  // Check confirm patterns
  for (const pattern of bashPatterns.confirm || []) {
    if (new RegExp(pattern, 'i').test(command)) {
      return {
        decision: 'ask',
        pattern,
        message: `Command requires confirmation: ${pattern}`,
      };
    }
  }
  
  // Check alert patterns
  for (const pattern of bashPatterns.alert || []) {
    if (new RegExp(pattern, 'i').test(command)) {
      // Alert patterns just log, don't block
      logSecurity({
        timestamp: new Date().toISOString(),
        tool: 'bash',
        input: { command },
        decision: 'allow',
        pattern,
        message: `Alert: command matched ${pattern}`,
      });
    }
  }
  
  return { decision: 'allow' };
}

function checkFilePath(filePath: string, patterns: SecurityPatterns): {
  decision: 'allow' | 'block' | 'ask';
  pattern?: string;
  message?: string;
} {
  const filePatterns = patterns.files || {};
  const normalizedPath = filePath.replace(/\\/g, '/');
  
  // Check zeroAccess patterns
  for (const pattern of filePatterns.zeroAccess || []) {
    if (normalizedPath.includes(pattern)) {
      return {
        decision: 'block',
        pattern,
        message: `Access denied to protected path: ${pattern}`,
      };
    }
  }
  
  // Check readOnly patterns
  for (const pattern of filePatterns.readOnly || []) {
    if (normalizedPath.includes(pattern)) {
      return {
        decision: 'ask',
        pattern,
        message: `Read-only path requires confirmation: ${pattern}`,
      };
    }
  }
  
  return { decision: 'allow' };
}

export default async function SecurityValidator({ 
  input, 
  output 
}: { 
  input: { tool: string; args?: Record<string, unknown> };
  output: { args: Record<string, unknown> };
}): Promise<void> {
  const patterns = loadPatterns();
  const tool = input.tool;
  
  // Check Bash commands
  if (tool === 'bash') {
    const command = output.args.command || '';
    const result = checkBashCommand(String(command), patterns);
    
    logSecurity({
      timestamp: new Date().toISOString(),
      tool: 'bash',
      input: { command: String(command) },
      decision: result.decision,
      pattern: result.pattern,
      message: result.message,
    });
    
    if (result.decision === 'block') {
      throw new Error(`[SecurityValidator] BLOCKED: ${result.message}`);
    }
    
    if (result.decision === 'ask') {
      // Return ask decision - OC will prompt user
      output.args._security_ask = result.message;
    }
  }
  
  // Check file operations
  if (['read', 'write', 'edit'].includes(tool)) {
    const filePath = output.args.file_path || output.args.path || '';
    
    if (filePath) {
      const result = checkFilePath(String(filePath), patterns);
      
      logSecurity({
        timestamp: new Date().toISOString(),
        tool,
        input: { file_path: String(filePath) },
        decision: result.decision,
        pattern: result.pattern,
        message: result.message,
      });
      
      if (result.decision === 'block') {
        throw new Error(`[SecurityValidator] BLOCKED: ${result.message}`);
      }
      
      if (result.decision === 'ask') {
        output.args._security_ask = result.message;
      }
    }
  }
}