#!/usr/bin/env bun
/**
 * ExplicitAgentAdapter
 *
 * Scenario adapter for a caller-supplied current model function. This module
 * never selects or invokes a model on its own.
 */

import { AgentAdapter, AgentRole, type AgentInput, type AgentReturnTypes } from '@langwatch/scenario';

export interface ExplicitAgentAdapterOptions {
  respond: (input: { systemPrompt: string; userPrompt: string }) => Promise<string>;
  systemPrompt?: string;
  name?: string;
}

export class ExplicitAgentAdapter extends AgentAdapter {
  override role = AgentRole.AGENT;
  override name: string;
  private readonly respond: ExplicitAgentAdapterOptions['respond'];
  private readonly systemPrompt: string;

  constructor(options: ExplicitAgentAdapterOptions) {
    super();
    if (typeof options?.respond !== 'function') {
      throw new Error('ExplicitAgentAdapter requires a caller-supplied respond adapter.');
    }
    this.name = options.name ?? 'explicit-agent';
    this.respond = options.respond;
    this.systemPrompt = options.systemPrompt ?? 'You are a helpful assistant.';
  }

  override async call(input: AgentInput): Promise<AgentReturnTypes> {
    const output = await this.respond({
      systemPrompt: this.systemPrompt,
      userPrompt: this.renderMessages(input.messages),
    });
    if (!output.trim()) throw new Error('Caller-supplied respond adapter returned an empty result.');
    return output.trim();
  }

  private renderMessages(messages: AgentInput['messages']): string {
    return messages
      .map((message) => `[${message.role ?? 'user'}]: ${this.extractText(message.content)}`)
      .join('\n\n');
  }

  private extractText(content: unknown): string {
    if (typeof content === 'string') return content;
    if (Array.isArray(content)) {
      return content
        .map((part) => {
          if (typeof part === 'string') return part;
          if (part && typeof part === 'object' && 'text' in part) {
            return String((part as { text: unknown }).text);
          }
          return '';
        })
        .filter(Boolean)
        .join(' ');
    }
    return JSON.stringify(content);
  }
}

/** @deprecated Use ExplicitAgentAdapter and provide respond explicitly. */
export const LifeosAgentAdapter = ExplicitAgentAdapter;
