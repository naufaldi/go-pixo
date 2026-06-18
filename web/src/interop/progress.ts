export const MAX_PREDICTED_PROGRESS = 98;
export const INITIAL_PROGRESS = 2;

export const STAGE = {
  PREPARING: { start: 0, end: 3, label: 'preparing' },
  RESIZING: { start: 3, end: 12, label: 'resizing' },
  ENCODING: { start: 12, end: 92, label: 'encoding' },
  FINALIZING: { start: 92, end: MAX_PREDICTED_PROGRESS, label: 'finalizing' },
} as const;

export type ProgressSnapshot = {
  progress: number;
  phase: string;
  lastRealProgress: number;
  lastRealUpdateTime: number;
  phaseStartTime: number;
  phaseTarget: number;
  predictable: boolean;
  fileSize: number;
};

export type RealProgressUpdate = {
  phase: string;
  progress: number;
  predictable?: boolean;
  phaseTarget?: number;
};

const PHASE_LABELS: Record<string, string> = {
  preparing: 'Preparing',
  resizing: 'Resizing',
  encoding: 'Encoding',
  finalizing: 'Finalizing',
  preprocess: 'Preprocessing',
  filtering: 'Filtering',
  deflate: 'Deflate',
  finalize: 'Writing file',
};

function legacyPngGlobal(phase: string, subProgress: number): number {
  switch (phase) {
    case 'preprocess':
      return (subProgress * 10) / 100;
    case 'filtering':
      return 10 + (subProgress * 40) / 100;
    case 'deflate':
      return 50 + (subProgress * 45) / 100;
    case 'finalize':
      return 95 + (subProgress * 5) / 100;
    default:
      return subProgress;
  }
}

export function mapPngPhaseToGlobal(phase: string, subProgress: number): number {
  const legacy = legacyPngGlobal(phase, subProgress);
  const span = STAGE.ENCODING.end - STAGE.ENCODING.start;
  return Math.round(STAGE.ENCODING.start + (legacy / 100) * span);
}

export function clampProgress(previous: number, next: number): number {
  return Math.max(previous, Math.min(MAX_PREDICTED_PROGRESS, Math.round(next)));
}

export function phaseLabel(phase: string): string {
  return PHASE_LABELS[phase] ?? phase;
}

export function predictionRateForFileSize(fileSizeBytes: number): number {
  if (fileSizeBytes >= 2_000_000) return 8;
  if (fileSizeBytes >= 500_000) return 5;
  if (fileSizeBytes >= 100_000) return 3;
  if (fileSizeBytes >= 10_000) return 2;
  return 1;
}

export function advancePredictedProgress(
  snapshot: ProgressSnapshot,
  now: number,
): ProgressSnapshot {
  if (!snapshot.predictable) {
    return snapshot;
  }

  const ceiling = Math.min(snapshot.phaseTarget, MAX_PREDICTED_PROGRESS - 1);
  if (snapshot.progress >= ceiling) {
    return snapshot;
  }

  const elapsedMs = now - snapshot.lastRealUpdateTime;
  const ratePerSecond = predictionRateForFileSize(snapshot.fileSize);
  const predicted =
    snapshot.lastRealProgress + (elapsedMs / 1000) * ratePerSecond;
  const next = clampProgress(snapshot.progress, Math.min(predicted, ceiling));

  if (next <= snapshot.progress) {
    return snapshot;
  }

  return { ...snapshot, progress: next };
}

export function applyRealProgress(
  snapshot: ProgressSnapshot | null,
  update: RealProgressUpdate,
  now: number,
): ProgressSnapshot {
  const phaseTarget = update.phaseTarget ?? update.progress;
  const predictable = update.predictable ?? false;
  const previous = snapshot?.progress ?? 0;
  const progress = clampProgress(previous, update.progress);
  const phaseChanged = snapshot?.phase !== update.phase;

  return {
    progress,
    phase: update.phase,
    lastRealProgress: update.progress,
    lastRealUpdateTime: now,
    phaseStartTime: phaseChanged ? now : (snapshot?.phaseStartTime ?? now),
    phaseTarget,
    predictable,
    fileSize: snapshot?.fileSize ?? 0,
  };
}

export function createInitialSnapshot(
  phase: string,
  progress: number,
  now: number,
  fileSize = 0,
  options?: { predictable?: boolean; phaseTarget?: number },
): ProgressSnapshot {
  const snapshot = applyRealProgress(null, {
    phase,
    progress,
    predictable: options?.predictable ?? true,
    phaseTarget: options?.phaseTarget ?? STAGE.RESIZING.end,
  }, now);
  return { ...snapshot, fileSize };
}

export function applyRealProgressWrapper(
  snapshot: ProgressSnapshot | null,
  phase: string,
  progress: number,
  predictable: boolean,
  phaseTarget: number,
  now: number,
  fileSize: number,
): ProgressSnapshot {
  const result = applyRealProgress(snapshot, { phase, progress, predictable, phaseTarget }, now);
  return {
    ...result,
    fileSize: fileSize > 0 ? fileSize : (snapshot?.fileSize ?? 0),
  };
}

export function advancePredictedProgressWrapper(
  snapshot: ProgressSnapshot,
  now: number,
): ProgressSnapshot {
  return advancePredictedProgress(snapshot, now);
}
