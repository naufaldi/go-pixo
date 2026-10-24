type timing = {elapsed: float, silentFor: float}
@module("../interop/progressClock.ts")
external useProgressTiming: (float, float, bool) => timing = "useProgressTiming"

@react.component
let make = (~progress: Types.compressionProgress, ~compact=false) => {
  let timing = useProgressTiming(progress.startTime, progress.lastRealUpdateTime, progress.isActive)
  let percent = progress.phasePercent
  let label = Progress.phaseLabel(progress.phase)
  <div className={if compact { "flex flex-col gap-1 min-w-40" } else { "flex flex-col items-center gap-3 w-64" }}>
    <span className="text-blue-400 text-sm font-medium" dataTestId={if compact { "file-compression-phase" } else { "compression-phase" }}>
      {React.string(label)}
    </span>
    <span className="text-neutral-300 text-sm" dataTestId={if compact { "file-compression-percent" } else { "compression-percent" }}>
      {React.string(switch percent { | Some(value) => Int.toString(value) ++ "% of stage" | None => "Working…" })}
    </span>
    <div role="progressbar" ariaLabel={label ++ " progress"} ariaValuemin=0.0 ariaValuemax=100.0
      ariaValuenow=?{percent->Option.map(Int.toFloat)}
      className="w-full bg-neutral-800 rounded-full h-2 overflow-hidden">
      <div className={switch percent {
        | Some(_) => "bg-blue-500 h-2 rounded-full transition-all duration-150"
        | None => "bg-blue-500 h-2 rounded-full w-1/3 animate-pulse motion-reduce:animate-none"
      }} style={ReactDOM.Style._dictToStyle(Dict.fromArray(switch percent {
        | Some(value) => [("width", Int.toString(value) ++ "%")]
        | None => []
      }))} />
    </div>
    <span className="text-neutral-500 text-xs" dataTestId={if compact { "file-compression-elapsed" } else { "compression-elapsed" }}>
      {React.string("Time elapsed: " ++ Display.formatTimeFloat(timing.elapsed))}
    </span>
    {if timing.silentFor >= 10000.0 {
      <span className="text-neutral-400 text-xs">{React.string("This stage has not reported progress recently.")}</span>
    } else { React.null }}
  </div>
}
