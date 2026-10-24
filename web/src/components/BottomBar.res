open Types

let presetLabel = (preset: preset): string => {
  switch preset {
  | Ultra => "Ultra (smallest)"
  | Smaller => "Smaller"
  | Balanced => "Balanced"
  | Faster => "Faster (quickest)"
  }
}

@react.component
let make = (
  ~format: string,
  ~outputFormat: outputFormat,
  ~onOutputFormatChange: outputFormat => unit,
  ~preset,
  ~lossless: bool,
  ~onPresetChange,
  ~onLosslessChange,
  ~onDownload,
  ~onDownloadFormat: outputFormat => unit,
  ~onDownloadAll,
  ~onDownloadZip: unit => unit,
  ~exportingZip: bool=false,
  ~hasCompletedItems: bool,
  ~completedCount: int,
  ~failedCount: int,
  ~appliedOptimizations: option<array<string>>,
  ~processingAll: bool,
  ~onCompressAll: unit => unit,
  ~targetWidth: option<int>,
  ~targetHeight: option<int>,
  ~onTargetWidthChange: option<int> => unit,
  ~onTargetHeightChange: option<int> => unit,
) => {
  let (downloadMenuOpen, setDownloadMenuOpen) = React.useState(_ => false)

  let handleSliderChange = (e: ReactEvent.Form.t) => {
    let raw = ReactEvent.Form.target(e)["value"]
    let value = switch raw {
    | Some(s) => Int.fromString(s)->Option.getOr(1)
    | None => 1
    }
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

  let downloadFormats = [
    ("PNG", ForcePng),
    ("JPEG", ForceJpeg),
    ("WebP", ForceWebp),
    ("AVIF", ForceAvif),
  ]

  <div
    dataTestId="bottom-bar"
    className="fixed bottom-0 left-0 right-0 bg-neutral-900 border-t border-neutral-800 px-4 py-3 z-50">
    <div className="max-w-6xl mx-auto flex flex-col gap-3">
      <div className="flex min-w-0 flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 min-w-0 flex-1">
          <label className="flex items-center gap-2 text-sm text-neutral-400">
            <span>{React.string("Convert:")}</span>
            <select ariaLabel="Output format" value={Types.outputFormatToChoice(outputFormat)} onChange={e => {
              let value = ReactEvent.Form.target(e)["value"]->Option.getOr("same")
              onOutputFormatChange(switch value {
                | "auto" => AutoSmallest | "png" => ForcePng | "jpeg" => ForceJpeg
                | "webp" => ForceWebp | "avif" => ForceAvif | _ => SameAsInput
              })
            }} className="max-w-44 rounded border border-neutral-600 bg-neutral-800 px-2 py-1 text-neutral-200">
              <option value="same">{React.string("Same as input")}</option>
              <option value="auto">{React.string("Auto (smallest tested)")}</option>
              <option value="png">{React.string("PNG")}</option>
              <option value="jpeg">{React.string("JPEG")}</option>
              <option value="webp">{React.string("WebP")}</option>
              <option value="avif">{React.string("AVIF")}</option>
            </select>
          </label>
          <span className="text-xs text-neutral-500">{React.string("Output: " ++ format)}</span>
          <div className="flex items-center gap-2 shrink-0" title="Each image keeps its own aspect ratio">
            <span className="text-xs text-neutral-500">{React.string("Resize:")}</span>
            <input
              type_="number"
              placeholder="W"
              ariaLabel="Target width"
              value={switch targetWidth { | Some(w) => Int.toString(w) | None => "" }}
              min="1"
              max="8000"
              onChange={e => {
                let raw = ReactEvent.Form.target(e)["value"]
                let v = switch raw {
                | Some(s) => Int.fromString(s)
                | None => None
                }
                switch v {
                | Some(n) when n > 0 => onTargetWidthChange(Some(n))
                | _ => onTargetWidthChange(None)
                }
              }}
              className="w-16 text-xs bg-neutral-800 text-neutral-300 border border-neutral-600 rounded px-2 py-0.5"
            />
            <span className="text-xs text-neutral-600">{React.string("×")}</span>
            <input
              type_="number"
              placeholder="H"
              ariaLabel="Target height"
              value={switch targetHeight { | Some(h) => Int.toString(h) | None => "" }}
              min="1"
              max="8000"
              onChange={e => {
                let raw = ReactEvent.Form.target(e)["value"]
                let v = switch raw {
                | Some(s) => Int.fromString(s)
                | None => None
                }
                switch v {
                | Some(n) when n > 0 => onTargetHeightChange(Some(n))
                | _ => onTargetHeightChange(None)
                }
              }}
              className="w-16 text-xs bg-neutral-800 text-neutral-300 border border-neutral-600 rounded px-2 py-0.5"
            />
            <span className="text-xs text-neutral-600">{React.string("px")}</span>
          </div>
          {switch appliedOptimizations {
          | Some(optimizations) =>
            <div className="basis-full min-w-0 break-words text-xs text-neutral-500">
              <span className="text-neutral-400">{React.string("What we did: ")}</span>
              {React.string(optimizations->Array.join(", "))}
            </div>
          | None => React.null
          }}
        </div>
        <div className="flex w-full min-w-0 max-w-full flex-wrap items-center gap-2 lg:w-auto lg:flex-1">
          <label className="flex items-center gap-2 cursor-pointer" title="Lossless encoding; resize changes image dimensions">
            <input
              type_="checkbox"
              checked=lossless
              onChange={e => {
                let checked = ReactEvent.Form.target(e)["checked"]
                onLosslessChange(checked == Some(true))
              }}
              className="w-4 h-4 rounded border-neutral-600 bg-neutral-800 text-white focus:ring-2 focus:ring-neutral-500"
            />
            <span className="text-sm text-neutral-300">{React.string("Perfect Quality")}</span>
          </label>
          <div className="relative flex">
            <button
              type_="button"
              dataTestId="download-primary"
              onClick={_ => onDownload()}
              className="text-sm bg-white text-neutral-900 px-3 py-1 rounded-l font-medium hover:bg-neutral-100 transition-colors border-r border-neutral-200">
              {React.string("Download")}
            </button>
            <button
              type_="button"
              dataTestId="download-format-toggle"
              ariaExpanded={downloadMenuOpen}
              ariaLabel="Choose download format"
              onClick={_ => setDownloadMenuOpen(prev => !prev)}
              className="text-sm bg-white text-neutral-900 px-2 py-1 rounded-r font-medium hover:bg-neutral-100 transition-colors">
              {React.string("▾")}
            </button>
            {downloadMenuOpen
              ? <div
                  className="absolute bottom-full right-0 mb-1 min-w-36 rounded border border-neutral-700 bg-neutral-800 py-1 shadow-lg">
                  {downloadFormats->Array.map(((label, fmt)) =>
                    <button
                      key=label
                      type_="button"
                      dataTestId={"download-as-" ++ label}
                      onClick={_ => {
                        setDownloadMenuOpen(_ => false)
                        onDownloadFormat(fmt)
                      }}
                      className="block w-full text-left text-xs px-3 py-1.5 text-neutral-200 hover:bg-neutral-700 transition-colors">
                      {React.string("Download as " ++ label)}
                    </button>
                  )->React.array}
                </div>
              : React.null}
          </div>
          <button
            type_="button"
            disabled=processingAll
            onClick={_ => onCompressAll()}
            className={
              "text-sm px-3 py-1 rounded font-medium transition-colors " ++
              if processingAll {
                "bg-neutral-700 text-neutral-500 cursor-not-allowed"
              } else {
                "bg-blue-600 text-white hover:bg-blue-500"
              }
            }>
            {React.string(if processingAll {"Processing..."} else {"Compress All"})}
          </button>
          {hasCompletedItems
            ? <button
                type_="button"
                disabled=processingAll
                onClick={_ => onDownloadAll()}
                className="text-sm bg-neutral-800 text-neutral-200 px-3 py-1 rounded font-medium hover:bg-neutral-700 transition-colors disabled:cursor-wait disabled:opacity-50">
                {React.string(if failedCount > 0 { "Download successful files (" ++ Int.toString(completedCount) ++ ")" } else { "Download All" })}
              </button>
            : React.null}
          {completedCount + failedCount >= 2 && hasCompletedItems
            ? <button
                type_="button"
                disabled={processingAll || exportingZip}
                ariaBusy=exportingZip
                onClick={_ => onDownloadZip()}
                className="text-sm bg-neutral-700 text-neutral-200 px-3 py-1 rounded font-medium hover:bg-neutral-600 transition-colors disabled:cursor-wait disabled:opacity-50">
                {React.string(if exportingZip { "Exporting ZIP..." } else if failedCount > 0 { "Download ZIP (" ++ Int.toString(completedCount) ++ " successful)" } else { "Download ZIP" })}
              </button>
            : React.null}
        </div>
      </div>
      {if processingAll && hasCompletedItems {
        <p className="text-xs text-neutral-400">{React.string("Batch downloads are available when every file finishes.")}</p>
      } else if failedCount > 0 {
        <p className="text-xs text-amber-400">{React.string(Int.toString(failedCount) ++ (if failedCount == 1 { " failed file excluded from downloads." } else { " failed files excluded from downloads." }))}</p>
      } else { React.null }}
      <div className="w-full max-w-lg mx-auto px-2">
        <div className="text-center mb-1">
          <span className="text-xs text-neutral-300 font-medium">
            {React.string(presetLabel(preset))}
          </span>
        </div>
        <input
          type_="range"
          min="0"
          max="3"
          step=1.0
          value={Int.toString(sliderValue)}
          onChange=handleSliderChange
          className="w-full min-w-48 h-2 bg-neutral-700 rounded-lg appearance-none cursor-pointer accent-white"
        />
        <div className="flex justify-between mt-0.5">
          <span className="text-xs text-neutral-500">{React.string("Smaller file")}</span>
          <span className="text-xs text-neutral-500">{React.string("Faster")}</span>
        </div>
      </div>
    </div>
  </div>
}
