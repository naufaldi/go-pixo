import { describe, it, expect } from 'vitest';
import {
  INITIAL_PROGRESS,
  MAX_PREDICTED_PROGRESS,
  STAGE,
  advancePredictedProgress,
  applyRealProgress,
  clampProgress,
  mapPngPhaseToGlobal,
  phaseLabel,
  predictionRateForFileSize,
  type ProgressSnapshot,
} from './progress';

describe('progress', () => {
  it('maps PNG sub-phases into the encoding window', () => {
    expect(mapPngPhaseToGlobal('preprocess', 0)).toBe(STAGE.ENCODING.start);
    expect(mapPngPhaseToGlobal('deflate', 0)).toBeGreaterThan(STAGE.ENCODING.start);
    expect(mapPngPhaseToGlobal('deflate', 100)).toBeLessThanOrEqual(STAGE.ENCODING.end);
    expect(mapPngPhaseToGlobal('finalize', 100)).toBe(STAGE.ENCODING.end);
  });

  it('seeds progress above zero for smoother start', () => {
    expect(INITIAL_PROGRESS).toBeGreaterThan(0);
    expect(INITIAL_PROGRESS).toBeLessThan(STAGE.ENCODING.start);
  });

  it('keeps progress monotonic', () => {
    expect(clampProgress(50, 40)).toBe(50);
    expect(clampProgress(50, 60)).toBe(60);
    expect(clampProgress(97, 100)).toBe(MAX_PREDICTED_PROGRESS);
  });

  it('caps predicted progress below completion', () => {
    const snapshot: ProgressSnapshot = {
      progress: 50,
      phase: 'encoding',
      lastRealProgress: 50,
      lastRealUpdateTime: 1000,
      phaseStartTime: 1000,
      phaseTarget: 92,
      predictable: true,
      fileSize: 100_000,
    };
    const advanced = advancePredictedProgress(snapshot, 6000);
    expect(advanced.progress).toBeGreaterThan(50);
    expect(advanced.progress).toBeLessThan(MAX_PREDICTED_PROGRESS);
    expect(advanced.progress).toBeLessThan(snapshot.phaseTarget);
  });

  it('predicts faster for larger files', () => {
    const base: ProgressSnapshot = {
      progress: 12,
      phase: 'preprocess',
      lastRealProgress: 12,
      lastRealUpdateTime: 1000,
      phaseStartTime: 1000,
      phaseTarget: 20,
      predictable: true,
      fileSize: 0,
    };
    const small = advancePredictedProgress({ ...base, fileSize: 5_000 }, 3000);
    const large = advancePredictedProgress({ ...base, fileSize: 3_000_000 }, 3000);
    expect(predictionRateForFileSize(3_000_000)).toBeGreaterThan(predictionRateForFileSize(5_000));
    expect(large.progress).toBeGreaterThan(small.progress);
  });

  it('predicts during preprocess when marked predictable', () => {
    const snapshot: ProgressSnapshot = {
      progress: 12,
      phase: 'preprocess',
      lastRealProgress: 12,
      lastRealUpdateTime: 1000,
      phaseStartTime: 1000,
      phaseTarget: 20,
      predictable: true,
      fileSize: 50_000,
    };
    const advanced = advancePredictedProgress(snapshot, 4000);
    expect(advanced.progress).toBeGreaterThan(12);
    expect(advanced.progress).toBeLessThanOrEqual(19);
  });

  it('does not predict when phase is not predictable', () => {
    const snapshot: ProgressSnapshot = {
      progress: 50,
      phase: 'encoding',
      lastRealProgress: 50,
      lastRealUpdateTime: 1000,
      phaseStartTime: 1000,
      phaseTarget: 92,
      predictable: false,
      fileSize: 100_000,
    };
    expect(advancePredictedProgress(snapshot, 10000)).toEqual(snapshot);
  });

  it('real updates override predicted values', () => {
    const predicted: ProgressSnapshot = {
      progress: 55,
      phase: 'encoding',
      lastRealProgress: 50,
      lastRealUpdateTime: 1000,
      phaseStartTime: 1000,
      phaseTarget: 92,
      predictable: true,
      fileSize: 100_000,
    };
    const real = applyRealProgress(predicted, {
      phase: 'deflate',
      progress: 70,
      predictable: true,
      phaseTarget: 88,
    }, 2000);
    expect(real.progress).toBe(70);
    expect(real.lastRealProgress).toBe(70);
    expect(real.phase).toBe('deflate');
    expect(real.fileSize).toBe(100_000);
  });

  it('formats human-readable phase labels', () => {
    expect(phaseLabel('deflate')).toBe('Deflate');
    expect(phaseLabel('preparing')).toBe('Preparing');
    expect(phaseLabel('unknown-phase')).toBe('unknown-phase');
  });
});
