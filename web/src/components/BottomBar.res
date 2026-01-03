open React
open Types

let presetLabel = (preset: preset): string => {
  switch preset {
  | Smaller => "Smaller"
  | Balanced => "Balanced"
  | Faster => "Faster"
  }
}

let quantizationLabel = (quantization: quantizationLevel): string => {
  switch quantization {
  | Lossless => "Lossless"
  | Colors256 => "256 colors"
  | Colors128 => "128 colors"
  | Colors64 => "64 colors"
  | Colors32 => "32 colors"
  | Colors16 => "16 colors"
  | Colors8 => "8 colors"
  }
}

@react.component
let make = (
  ~format,
  ~preset,
  ~lossless,
  ~quantization,
  ~dithering,
  ~onPresetChange,
  ~onLosslessChange,
  ~onQuantizationChange,
  ~onDitheringChange,
  ~onDownload,
  ~onDownloadAll,
  ~hasCompletedItems,
) => {
  let handleSliderChange = (e: ReactEvent.Form.t) => {
    let value = %raw("parseInt(ReactEvent.Form.target(e).value, 10)")
    switch value {
    | 0 => onPresetChange(Smaller)
    | 1 => onPresetChange(Balanced)
    | 2 => onPresetChange(Faster)
    | _ => ()
    }
  }

  let sliderValue = switch preset {
  | Smaller => 0
  | Balanced => 1
  | Faster => 2
  }

  let isLosslessMode = lossless || isLossless(quantization)

  <div className="fixed bottom-0 left-0 right-0 bg-neutral-900 border-t border-neutral-800 px-6 py-3 flex items-center justify-between z-50">
    <div className="text-sm text-neutral-400">
      {React.string("Format " ++ format)}
    </div>

    <div className="flex-1 max-w-md mx-8">
      <div className="flex items-center gap-4">
        <span className="text-xs text-neutral-500">{React.string("Smaller")}</span>
        <input
          type_="range"
          min="0"
          max="2"
          step=1.0
          value={Int.toString(sliderValue)}
          onChange=handleSliderChange
          className="flex-1 h-2 bg-neutral-700 rounded-lg appearance-none cursor-pointer accent-white"
        />
        <span className="text-xs text-neutral-500">{React.string("Faster")}</span>
      </div>
    </div>

    <div className="flex items-center gap-4">
      {!isLosslessMode
        ? <select
            value={quantization->quantizationToInt->Int.toString}
            onChange={e => {
              let value = %raw("parseInt(ReactEvent.Form.target(e).value, 10)")
              onQuantizationChange(intToQuantization(value))
            }}
            className="bg-neutral-800 text-neutral-300 text-sm px-3 py-1.5 rounded border border-neutral-700 focus:outline-none focus:ring-2 focus:ring-neutral-500"
          >
            <option value="256">256 colors</option>
            <option value="128">128 colors</option>
            <option value="64">64 colors</option>
            <option value="32">32 colors</option>
            <option value="16">16 colors</option>
            <option value="8">8 colors</option>
          </select>
        : React.null}

      {!isLosslessMode
        ? <label className="flex items-center gap-2 cursor-pointer">
            <input
              type_="checkbox"
              checked=dithering
              onChange={e => {
                let checked = %raw("ReactEvent.Form.target(e).checked")
                onDitheringChange(checked)
              }}
              className="w-4 h-4 rounded border-neutral-600 bg-neutral-800 text-white focus:ring-2 focus:ring-neutral-500"
            />
            <span className="text-sm text-neutral-300">{React.string("Dithering")}</span>
          </label>
        : React.null}

      <label className="flex items-center gap-2 cursor-pointer">
        <input
          type_="checkbox"
          checked=lossless
          onChange={e => {
            let checked = %raw("ReactEvent.Form.target(e).checked")
            onLosslessChange(checked)
          }}
          className="w-4 h-4 rounded border-neutral-600 bg-neutral-800 text-white focus:ring-2 focus:ring-neutral-500"
        />
        <span className="text-sm text-neutral-300">{React.string("Lossless")}</span>
      </label>

      <button
        type_="button"
        onClick={_ => onDownload()}
        className="text-sm bg-white text-neutral-900 px-4 py-1.5 rounded font-medium hover:bg-neutral-100 transition-colors">
        {React.string("Download")}
      </button>

      {hasCompletedItems
        ? <button
            type_="button"
            onClick={_ => onDownloadAll()}
            className="text-sm bg-neutral-800 text-neutral-200 px-4 py-1.5 rounded font-medium hover:bg-neutral-700 transition-colors">
            {React.string("Download All")}
          </button>
        : React.null}
    </div>
  </div>
}
