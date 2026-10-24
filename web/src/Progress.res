type snapshot = {
  progress: int,
  phasePercent: option<int>,
  phase: string,
  lastRealProgress: int,
  lastRealUpdateTime: float,
  phaseStartTime: float,
  phaseTarget: int,
  predictable: bool,
  fileSize: int,
}

@module("./interop/progress.ts")
external createInitialSnapshot: (string, int, float, int) => snapshot = "createInitialSnapshot"

@module("./interop/progress.ts")
external applyRealProgressWrapper: (
  Nullable.t<snapshot>,
  string,
  int,
  bool,
  int,
  float,
  int,
  option<int>,
) => snapshot = "applyRealProgressWrapper"

@module("./interop/progress.ts")
external advancePredictedProgressWrapper: (snapshot, float) => snapshot = "advancePredictedProgressWrapper"

@module("./interop/progress.ts")
external phaseLabel: string => string = "phaseLabel"

let toCompressionProgress = (
  itemId: string,
  fileSize: int,
  startTime: float,
  s: snapshot,
): Types.compressionProgress => {
  {
    isActive: true,
    itemId,
    phase: s.phase,
    progress: s.progress,
    phasePercent: s.phasePercent,
    startTime,
    fileSize,
    lastRealProgress: s.lastRealProgress,
    lastRealUpdateTime: s.lastRealUpdateTime,
    phaseStartTime: s.phaseStartTime,
    phaseTarget: s.phaseTarget,
    predictable: s.predictable,
  }
}

let fromCompressionProgress = (p: Types.compressionProgress): snapshot => {
  {
    progress: p.progress,
    phasePercent: p.phasePercent,
    phase: p.phase,
    lastRealProgress: p.lastRealProgress,
    lastRealUpdateTime: p.lastRealUpdateTime,
    phaseStartTime: p.phaseStartTime,
    phaseTarget: p.phaseTarget,
    predictable: p.predictable,
    fileSize: p.fileSize,
  }
}

let seedProgress = (itemId: string, fileSize: int, now: float): Types.compressionProgress => {
  let snapshot = createInitialSnapshot("decoding", 0, now, fileSize)
  toCompressionProgress(itemId, fileSize, now, snapshot)
}

let applyWorkerUpdate = (
  previous: option<Types.compressionProgress>,
  phase: string,
  progress: int,
  predictable: bool,
  phaseTarget: int,
  now: float,
  phasePercent: option<int>,
): Types.compressionProgress => {
  switch previous {
  | Some(p) =>
    let snapshot = applyRealProgressWrapper(
      Nullable.make(fromCompressionProgress(p)),
      phase,
      progress,
      predictable,
      phaseTarget,
      now,
      p.fileSize,
      phasePercent,
    )
    toCompressionProgress(p.itemId, p.fileSize, p.startTime, snapshot)
  | None =>
    let snapshot = applyRealProgressWrapper(
      Nullable.null,
      phase,
      progress,
      predictable,
      phaseTarget,
      now,
      0,
      phasePercent,
    )
    toCompressionProgress("", 0, now, snapshot)
  }
}

let advancePredicted = (p: Types.compressionProgress, now: float): Types.compressionProgress => {
  let snapshot = advancePredictedProgressWrapper(fromCompressionProgress(p), now)
  toCompressionProgress(p.itemId, p.fileSize, p.startTime, snapshot)
}
