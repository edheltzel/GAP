#!/usr/bin/env bun
/**
 * Algorithm Bridge
 * Run an Evals suite against outputs supplied by the active harness.
 */

import type { AlgorithmEvalRequest, AlgorithmEvalResult, EvalRun, Task } from '../Types/index.ts';
import { loadSuite, checkSaturation } from './SuiteManager.ts';
import { TrialRunner } from './TrialRunner.ts';
import { createTranscript } from './TranscriptCapture.ts';
import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { parse as parseYaml } from 'yaml';
import { parseArgs } from 'node:util';

const EVALS_DIR = join(import.meta.dir, '..');

type SuppliedOutput = string | { output: string };
type SuppliedOutputs = Record<string, SuppliedOutput>;

function loadSuppliedOutputs(path: string): SuppliedOutputs {
  if (!existsSync(path)) throw new Error(`Output file not found: ${path}`);
  let parsed: unknown;
  try {
    parsed = JSON.parse(readFileSync(path, 'utf-8')) as unknown;
  } catch (error) {
    throw new Error(`Output file is not valid JSON: ${error instanceof Error ? error.message : String(error)}`);
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('Output file must be a JSON object keyed by task id.');
  }
  return parsed as SuppliedOutputs;
}

function outputForTask(outputs: SuppliedOutputs, taskId: string): string {
  const supplied = outputs[taskId];
  const output = typeof supplied === 'string' ? supplied : supplied?.output;
  if (!output?.trim()) throw new Error(`No supplied output for task: ${taskId}`);
  return output;
}

export async function runEvalForAlgorithm(request: AlgorithmEvalRequest): Promise<AlgorithmEvalResult> {
  if (!request.output_file) {
    return {
      isc_row: request.isc_row,
      suite: request.suite,
      passed: false,
      score: 0,
      summary: 'An explicit output_file is required; no agent execution is performed by this bridge.',
      run_id: 'error',
    };
  }

  const suite = loadSuite(request.suite);
  if (!suite) {
    return {
      isc_row: request.isc_row,
      suite: request.suite,
      passed: false,
      score: 0,
      summary: `Suite not found: ${request.suite}`,
      run_id: 'error',
    };
  }

  let suppliedOutputs: SuppliedOutputs;
  try {
    suppliedOutputs = loadSuppliedOutputs(request.output_file);
  } catch (error) {
    return {
      isc_row: request.isc_row,
      suite: request.suite,
      passed: false,
      score: 0,
      summary: error instanceof Error ? error.message : String(error),
      run_id: 'error',
    };
  }

  const tasks = suite.tasks
    .map((taskId) => findTaskFile(taskId))
    .filter((path): path is string => Boolean(path))
    .map((path) => parseYaml(readFileSync(path, 'utf-8')) as Task);

  if (tasks.length === 0) {
    return {
      isc_row: request.isc_row,
      suite: request.suite,
      passed: false,
      score: 0,
      summary: `No tasks found in suite: ${request.suite}`,
      run_id: 'error',
    };
  }

  const results: EvalRun[] = [];
  let totalScore = 0;
  let passedTasks = 0;

  try {
    for (const task of tasks) {
      const output = outputForTask(suppliedOutputs, task.id);
      const runner = new TrialRunner({
        task,
        executor: async (_task, trialNum) => ({
          output,
          transcript: createTranscript(task.id, `trial_${trialNum}`, {
            turns: [
              { role: 'system', content: task.description },
              { role: 'assistant', content: output },
            ],
            toolCalls: [],
          }),
        }),
      });
      const run = await runner.run();
      results.push(run);
      totalScore += run.mean_score;
      if (run.pass_rate >= (task.pass_threshold ?? 0.75)) passedTasks++;
    }
  } catch (error) {
    return {
      isc_row: request.isc_row,
      suite: request.suite,
      passed: false,
      score: 0,
      summary: error instanceof Error ? error.message : String(error),
      run_id: 'error',
    };
  }

  const overallScore = totalScore / tasks.length;
  const overallPassed = passedTasks === tasks.length || overallScore >= (suite.pass_threshold ?? 0.75);
  return {
    isc_row: request.isc_row,
    suite: request.suite,
    passed: overallPassed,
    score: overallScore,
    summary: `${passedTasks}/${tasks.length} tasks passed, score: ${(overallScore * 100).toFixed(1)}%`,
    run_id: results[0]?.id ?? 'aggregate',
  };
}

function findTaskFile(taskId: string): string | null {
  const useCasesDir = join(EVALS_DIR, 'UseCases');
  const candidates = [
    join(useCasesDir, `${taskId}.yaml`),
    join(useCasesDir, 'Regression', `${taskId}.yaml`),
    join(useCasesDir, 'Capability', `${taskId}.yaml`),
  ];
  return candidates.find((path) => existsSync(path)) ?? null;
}

if (import.meta.main) {
  const { values } = parseArgs({
    args: Bun.argv.slice(2),
    options: {
      suite: { type: 'string', short: 's' },
      'isc-row': { type: 'string', short: 'r' },
      'output-file': { type: 'string', short: 'o' },
      'show-saturation': { type: 'boolean' },
      help: { type: 'boolean', short: 'h' },
    },
    allowPositionals: true,
  });

  if (values.help || !values.suite) {
    console.log(`AlgorithmBridge - evaluate explicitly supplied harness outputs

Usage:
  bun run AlgorithmBridge.ts --suite <suite> --output-file <outputs.json> [--isc-row <row>]

Options:
  -s, --suite            Eval suite to run
  -o, --output-file      JSON object keyed by task id; required for evaluation
  -r, --isc-row          Optional row identifier copied into the result only
  --show-saturation      Show suite saturation status without running
  -h, --help             Show this help

This bridge does not execute an agent and does not mutate ISC state.`);
    process.exit(0);
  }

  if (values['show-saturation']) {
    console.log(JSON.stringify(checkSaturation(values.suite), null, 2));
    process.exit(0);
  }

  const result = await runEvalForAlgorithm({
    isc_row: values['isc-row'] ? parseInt(values['isc-row'], 10) : 0,
    suite: values.suite,
    output_file: values['output-file'],
  });
  console.log(JSON.stringify(result, null, 2));
  process.exit(result.passed ? 0 : 1);
}
