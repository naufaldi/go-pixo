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
  <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" ariaLabel="Compression progress">
    <div className="bg-neutral-900 border border-neutral-800 rounded-2xl px-8 py-10 flex flex-col items-center shadow-2xl min-w-[280px]">
      <ProgressStatus progress />
      <span className="text-neutral-400 text-xs mt-4">{React.string(formatFileSize(progress.fileSize))}</span>
    </div>
  </div>
}
