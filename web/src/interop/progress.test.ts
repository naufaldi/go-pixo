import {describe, it, expect} from 'vitest';
import {advancePredictedProgress, applyRealProgress, createInitialSnapshot, createProgressThrottle, phaseLabel} from './progress';

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

describe('worker progress throttle', () => {
  const run = (calls: [number, string, number][]) => {
    let clock = 0;
    const allow = createProgressThrottle(() => clock, 80);
    return calls.filter(([at, phase, progress]) => { clock = at; return allow(phase, progress); });
  };

  it('limits interior updates within a phase to one per interval', () => {
    expect(run([[0, 'palette', 0], [10, 'palette', 20], [79, 'palette', 40], [80, 'palette', 60], [120, 'palette', 70], [170, 'palette', 80]]))
      .toEqual([[0, 'palette', 0], [80, 'palette', 60], [170, 'palette', 80]]);
  });

  it('always passes phase changes and boundaries, so a fast phase may report only its boundaries', () => {
    expect(run([[0, 'histogram', 0], [5, 'histogram', 50], [9, 'histogram', 100], [10, 'palette', 0], [12, 'palette', 50], [15, 'palette', 100], [16, 'quantize', 30]]))
      .toEqual([[0, 'histogram', 0], [9, 'histogram', 100], [10, 'palette', 0], [15, 'palette', 100], [16, 'quantize', 30]]);
  });
});
