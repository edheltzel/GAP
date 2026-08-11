/**
 * Pairwise Comparison Grader
 * Compare output against a reference using explicitly supplied judge responses.
 */

import { BaseGrader, registerGrader, type GraderContext } from '../Base.ts';
import type { GraderResult, PairwiseComparisonParams } from '../../Types/index.ts';
import { existsSync, readFileSync } from 'node:fs';

type Comparison = {
  position: 'output_first' | 'reference_first';
  winner: 'A' | 'B' | 'tie';
  reasoning: string;
};

export class PairwiseComparisonGrader extends BaseGrader {
  type = 'pairwise_comparison' as const;
  category = 'model_based' as const;

  async grade(context: GraderContext): Promise<GraderResult> {
    const start = performance.now();
    const params = this.config.params as unknown as PairwiseComparisonParams;
    let reference = params.reference;
    if (existsSync(reference)) reference = readFileSync(reference, 'utf-8');

    if (!reference) {
      return this.createResult(0, false, performance.now() - start, {
        reasoning: 'No reference output available',
      });
    }

    const positionSwap = params.position_swap ?? true;
    const needed = positionSwap ? 2 : 1;
    if (!params.judge_responses || params.judge_responses.length < needed) {
      const prompts = [
        this.buildPrompt(context.output, reference, params.criteria),
        ...(positionSwap ? [this.buildPrompt(reference, context.output, params.criteria)] : []),
      ];
      return this.createResult(0, false, performance.now() - start, {
        reasoning: `Pairwise grader requires ${needed} explicit params.judge_responses value(s).`,
        details: { judge_prompts: prompts, adapter_called: false },
      });
    }

    const results: Comparison[] = [];
    results.push({ position: 'output_first', ...this.parseResponse(params.judge_responses[0]!) });

    if (positionSwap) {
      const swapped = this.parseResponse(params.judge_responses[1]!);
      results.push({
        position: 'reference_first',
        winner: swapped.winner === 'A' ? 'B' : swapped.winner === 'B' ? 'A' : 'tie',
        reasoning: swapped.reasoning,
      });
    }

    const outputWins = results.filter((result) => result.winner === 'A').length;
    const referenceWins = results.filter((result) => result.winner === 'B').length;
    const ties = results.filter((result) => result.winner === 'tie').length;
    const score = (outputWins + ties * 0.5) / results.length;
    const aggregateWinner = outputWins > referenceWins ? 'output' : referenceWins > outputWins ? 'reference' : 'tie';

    return this.createResult(score, score >= 0.5, performance.now() - start, {
      reasoning: `${aggregateWinner} wins (output: ${outputWins}, reference: ${referenceWins}, ties: ${ties})`,
      details: { results, position_swap: positionSwap, criteria: params.criteria, adapter_called: false },
    });
  }

  private buildPrompt(outputA: string, outputB: string, criteria?: string[]): string {
    const criteriaText = criteria?.length
      ? `Focus on these criteria:\n${criteria.map((criterion) => `- ${criterion}`).join('\n')}`
      : 'Consider overall quality, accuracy, clarity, and helpfulness.';
    return `${criteriaText}\n\nOutput A:\n${outputA}\n\nOutput B:\n${outputB}\n\nRespond with REASONING and WINNER: A, B, or TIE.`;
  }

  private parseResponse(text: string): { winner: 'A' | 'B' | 'tie'; reasoning: string } {
    const winnerMatch = text.match(/WINNER:\s*(A|B|TIE)/i);
    const reasoningMatch = text.match(/REASONING:\s*([\s\S]*?)(?=WINNER:|$)/i);
    const token = winnerMatch?.[1]?.toUpperCase();
    return {
      winner: token === 'A' ? 'A' : token === 'B' ? 'B' : 'tie',
      reasoning: reasoningMatch?.[1]?.trim() || text.trim(),
    };
  }
}

registerGrader('pairwise_comparison', PairwiseComparisonGrader);
