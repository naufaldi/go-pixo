import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildBenchmarkSummary,
  formatDuration,
  percentile,
  summarizeStep,
} from './metrics.mjs';

test('percentile uses nearest-rank selection for sorted latency samples', () => {
  const samples = [30.9, 26.7, 47.4, 118.3];

  assert.equal(percentile(samples, 50), 30.9);
  assert.equal(percentile(samples, 95), 118.3);
  assert.equal(percentile(samples, 99), 118.3);
});

test('summarizeStep reports latency, pass/fail, and throughput for one step', () => {
  const summary = summarizeStep('1. Load App', [42, 38, 44], 2, 1, 1.5);

  assert.deepEqual(summary, {
    step: '1. Load App',
    type: 'Real',
    avg: 41.3,
    p50: 42,
    p95: 44,
    p99: 44,
    pass: 2,
    fail: 1,
    throughput: 1.33,
  });
});

test('buildBenchmarkSummary reports total E2E percentiles and success rate', () => {
  const summary = buildBenchmarkSummary([100, 140, 110], 2, 3, 2.5);

  assert.deepEqual(summary, {
    avg: 116.7,
    p50: 110,
    p95: 140,
    p99: 140,
    successRate: 66.7,
    passedIterations: 2,
    totalIterations: 3,
    submittedThroughput: 1.2,
    completedThroughput: 0.8,
  });
});

test('formatDuration keeps millisecond precision readable', () => {
  assert.equal(formatDuration(65.74), '65.7ms');
  assert.equal(formatDuration(238), '238.0ms');
});
