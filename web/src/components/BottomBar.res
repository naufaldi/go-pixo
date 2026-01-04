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
  ~format: string,
  ~preset,
  ~lossless: bool,
  ~quantization,
  ~dithering: bool,
  ~ditherStrength: float,
  ~qualityTarget: int,
  ~zopfliIterations: int,
  ~onPresetChange,
  ~onLosslessChange,
  ~onQuantizationChange,
  ~onDitheringChange,
  ~onDitheringStrengthChange,
  ~onQualityTargetChange,
  ~onZopfliIterationsChange,
  ~onDownload,
  ~onDownloadAll,
  ~hasCompletedItems: bool,
) => {
  // #endregion
  let handleSliderChange = (_e: ReactEvent.Form.t) => {
    let value = %raw("parseInt(ReactEvent.Form.target(_e).value, 10)")
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
            onChange={_e => {
              let value = %raw("parseInt(ReactEvent.Form.target(_e).value, 10)")
              onQuantizationChange(intToQuantization(value))
            }}
            className="bg-neutral-800 text-neutral-300 text-sm px-3 py-1.5 rounded border border-neutral-700 focus:outline-none focus:ring-2 focus:ring-neutral-500"
          >
            <option value="256">{React.string("256 colors")}</option>
            <option value="128">{React.string("128 colors")}</option>
            <option value="64">{React.string("64 colors")}</option>
            <option value="32">{React.string("32 colors")}</option>
            <option value="16">{React.string("16 colors")}</option>
            <option value="8">{React.string("8 colors")}</option>
          </select>
        : React.null}

      {!isLosslessMode
        ? <label className="flex items-center gap-2 cursor-pointer">
            <input
              type_="checkbox"
              checked=dithering
              onChange={_e => {
                let checked = %raw("ReactEvent.Form.target(_e).checked")
                onDitheringChange(checked)
              }}
              className="w-4 h-4 rounded border-neutral-600 bg-neutral-800 text-white focus:ring-2 focus:ring-neutral-500"
            />
            <span className="text-sm text-neutral-300">{React.string("Dithering")}</span>
          </label>
        : React.null}

      {!isLosslessMode && dithering
        ? <div className="flex items-center gap-2">
            <span className="text-xs text-neutral-500">{React.string("0%")}</span>
            <input
              type_="range"
              min="0"
              max="100"
              step=1.0
              value={Int.toString(Float.toInt(ditherStrength *. 100.0))}
              onChange={_e => {
                let value = %raw("parseInt(ReactEvent.Form.target(_e).value, 10)")
                onDitheringStrengthChange(Float.fromInt(value) /. 100.0)
              }}
              className="w-24 h-2 bg-neutral-700 rounded-lg appearance-none cursor-pointer accent-white"
            />
            <span className="text-xs text-neutral-500">{React.string("100%")}</span>
          </div>
        : React.null}

      {!isLosslessMode
        ? <div className="flex items-center gap-2">
            <span className="text-xs text-neutral-500">{React.string("Q:")}</span>
            <input
              type_="range"
              min="0"
              max="100"
              step=1.0
              value={Int.toString(qualityTarget)}
              onChange={_e => {
                let value = %raw("parseInt(ReactEvent.Form.target(_e).value, 10)")
                onQualityTargetChange(value)
              }}
              className="w-24 h-2 bg-neutral-700 rounded-lg appearance-none cursor-pointer accent-white"
            />
            <span className="text-xs text-neutral-400">{React.string(Int.toString(qualityTarget))}</span>
          </div>
        : React.null}

      <div className="flex items-center gap-2">
        <span className="text-xs text-neutral-500">{React.string("Z:")}</span>
        <input
          type_="number"
          min="0"
          max="50"
          value={Int.toString(zopfliIterations)}
          onChange={_e => {
            let value = %raw("parseInt(ReactEvent.Form.target(_e).value, 10)")
            if value >= 0 && value <= 50 {
              onZopfliIterationsChange(value)
            }
          }}
          className="w-16 bg-neutral-800 text-neutral-300 text-sm px-2 py-1 rounded border border-neutral-700 focus:outline-none focus:ring-2 focus:ring-neutral-500"
        />
      </div>

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
