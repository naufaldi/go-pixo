open Types

let formatFileSize = (bytes: int): string => {
  if bytes >= 1048576 {
    let mb = Float.fromInt(bytes) /. 1048576.0
    `${Float.toFixed(~digits=1, mb)} MB`
  } else if bytes >= 1024 {
    let kb = Float.fromInt(bytes) /. 1024.0
    `${Float.toFixed(~digits=1, kb)} KB`
  } else {
    `${Int.toString(bytes)} B`
  }
}

let formatTime = (milliseconds: float): string => {
  let seconds = milliseconds /. 1000.0
  if seconds >= 60.0 {
    let mins = Int.toFloat(Float.toInt(seconds /. 60.0))
    let secs = Float.toFixed(~digits=1, seconds -. mins *. 60.0)
    `${Float.toFixed(~digits=0, mins)}m ${secs}s`
  } else {
    `${Float.toFixed(~digits=1, seconds)}s`
  }
}

@react.component
let make = (~progress: Types.compressionProgress) => {
  let elapsed = %raw("performance.now()") -. progress.startTime

  <div
    className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50"
    ariaLabel="Compression progress">
    <div
      className="bg-neutral-900 border border-neutral-800 rounded-2xl px-8 py-10 flex flex-col items-center shadow-2xl min-w-[280px]">
      <div className="relative w-16 h-16 mb-6">
        <svg className="w-16 h-16 transform -rotate-90" viewBox="0 0 36 36">
          <path
            className="text-neutral-700"
            d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
            fill="none"
            stroke="currentColor"
            strokeWidth="3"
          />
          <path
            className="text-blue-500 transition-all duration-300"
            strokeDasharray={`${Float.toString(Float.fromInt(progress.progress))}, 100`}
            d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
            fill="none"
            stroke="currentColor"
            strokeWidth="3"
            strokeLinecap="round"
          />
        </svg>
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-sm font-semibold text-white">
            {React.string(`${Int.toString(progress.progress)}%`)}
          </span>
        </div>
      </div>

      <span className="text-neutral-300 text-sm font-medium mb-1 capitalize">
        {React.string(Progress.phaseLabel(progress.phase))}
      </span>

      <span className="text-neutral-400 text-xs mb-6">
        {React.string(formatFileSize(progress.fileSize))}
      </span>

      <div className="w-full bg-neutral-800 rounded-full h-1.5 mb-4 overflow-hidden">
        <div
          className="bg-blue-500 h-1.5 rounded-full transition-all duration-300 ease-out"
          style={{ width: `${Int.toString(progress.progress)}%` }}
        />
      </div>

      <div className="flex items-center justify-center w-full text-xs mt-2">
        <span className="text-neutral-300 font-medium">
          {React.string("Time Elapsed: " ++ formatTime(elapsed))}
        </span>
      </div>
    </div>
  </div>
}
