import {describe, it, expect} from 'vitest';
import {advancePredictedProgress, applyRealProgress, createInitialSnapshot, phaseLabel} from './progress';

describe('measured progress', () => {
  it('never infers work from elapsed time or file size, including legacy predictable snapshots', () => {
    for (const fileSize of [5_000, 5_000_000]) {
      const snapshot = {...createInitialSnapshot('quantize', 12, 1000, fileSize), predictable: true, phasePercent: 25};
      expect(advancePredictedProgress(snapshot, 99_000)).toEqual(snapshot);
    }
  });
  it('resets the phase percentage on transitions while preserving legacy callback fields', () => {
    const initial = createInitialSnapshot('decoding', 0, 1000, 5_000_000);
    const mapped = applyRealProgress(initial, {phase: 'quantize', progress: 12, phasePercent: 75}, 2000);
    expect(mapped.phasePercent).toBe(75);
    const next = applyRealProgress(mapped, {phase: 'filtering', progress: 10, phasePercent: 0}, 3000);
    expect(next.phasePercent).toBe(0);
    expect(next.phaseStartTime).toBe(3000);
    expect(next.fileSize).toBe(5_000_000);
    const opaque = applyRealProgress(next, {phase: 'encoding-jpeg', progress: 12}, 4000);
    expect(opaque.phasePercent).toBeUndefined();
  });
  it('clamps phase units and keeps absent/nonfinite units indeterminate', () => {
    for (const [value, expected] of [[-1, 0], [101, 100], [NaN, undefined], [Infinity, undefined]] as const) {
      expect(applyRealProgress(null, {phase: 'quantize', progress: 12, phasePercent: value}, 1).phasePercent).toBe(expected);
    }
  });
  it('labels the counted work and retains unknown legacy phase names', () => {
    expect(phaseLabel('histogram')).toBe('Counting pixel colors');
    expect(phaseLabel('quantize')).toBe('Mapping pixels to palette');
    expect(phaseLabel('legacy-phase')).toBe('legacy-phase');
  });
});
