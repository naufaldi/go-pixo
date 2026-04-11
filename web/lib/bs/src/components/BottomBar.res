open Types

@react.component
let make = (
  ~format: string,
  ~preset,
  ~lossless: bool,
  ~onPresetChange,
  ~onLosslessChange,
  ~onDownload,
  ~onDownloadAll,
  ~hasCompletedItems: bool,
  ~appliedOptimizations: option<array<string>>,
) => {
  let handleSliderChange = (_e: ReactEvent.Form.t) => {
    let value = %raw("parseInt(ReactEvent.Form.target(_e).value, 10)")
    switch value {
    | 0 => onPresetChange(Ultra)
    | 1 => onPresetChange(Smaller)
    | 2 => onPresetChange(Balanced)
    | 3 => onPresetChange(Faster)
    | _ => ()
    }
  }

  let sliderValue = switch preset {
  | Ultra => 0
  | Smaller => 1
  | Balanced => 2
  | Faster => 3
  }

  <div className="fixed bottom-0 left-0 right-0 bg-neutral-900 border-t border-neutral-800 px-4 py-2 z-50">
    <div className="flex items-center justify-between max-w-6xl mx-auto">
      <div className="flex items-center gap-4">
        <div className="text-sm text-neutral-400">
          {React.string("Format: " ++ format)}
        </div>
        {switch appliedOptimizations {
        | Some(optimizations) => 
          <div className="text-xs text-neutral-500">
            <span className="text-neutral-400">{React.string("What we did: ")}</span>
            {React.string(optimizations->Array.join(", "))}
          </div>
        | None => React.null
        }}
      </div>
      <div className="flex-1 max-w-md mx-6">
        <div className="flex items-center gap-3">
          <div className="text-center">
            <div className="text-xs text-neutral-500 font-medium">{React.string("Ultra")}</div>
            <div className="text-xs text-neutral-600 max-w-20 leading-tight">
              {React.string("Max size")}
            </div>
          </div>
          <div className="flex-1">
            <input
              type_="range"
              min="0"
              max="3"
              step=1.0
              value={Int.toString(sliderValue)}
              onChange=handleSliderChange
              className="w-full h-2 bg-neutral-700 rounded-lg appearance-none cursor-pointer accent-white"
            />
            <div className="flex justify-between mt-0.5">
              <span className="text-xs text-neutral-500">{React.string("Ultra")}</span>
              <span className="text-xs text-neutral-500">{React.string("Fast")}</span>
            </div>
          </div>
          <div className="text-center">
            <div className="text-xs text-neutral-500 font-medium">{React.string("Fast")}</div>
            <div className="text-xs text-neutral-600 max-w-20 leading-tight">
              {React.string("Quick")}
            </div>
          </div>
        </div>
      </div>
      <div className="flex items-center gap-3">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type_="checkbox"
            checked=lossless
            onChange={_e => {
              let checked = %raw("ReactEvent.Form.target(_e).checked")
              onLosslessChange(checked)
            }}
            className="w-4 h-4 rounded border-neutral-600 bg-neutral-800 text-white focus:ring-2 focus:ring-neutral-500"
          />
          <span className="text-sm text-neutral-300">{React.string("Perfect Quality")}</span>
        </label>
        <button
          type_="button"
          onClick={_ => onDownload()}
          className="text-sm bg-white text-neutral-900 px-3 py-1 rounded font-medium hover:bg-neutral-100 transition-colors">
          {React.string("Download")}
        </button>
        {hasCompletedItems
          ? <button
              type_="button"
              onClick={_ => onDownloadAll()}
              className="text-sm bg-neutral-800 text-neutral-200 px-3 py-1 rounded font-medium hover:bg-neutral-700 transition-colors">
              {React.string("Download All")}
            </button>
          : React.null}
      </div>
    </div>
  </div>
}
