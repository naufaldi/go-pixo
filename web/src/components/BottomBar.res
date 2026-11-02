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
  ~hasItems: bool=true,
  ~format: string,
  ~outputFormat: outputFormat,
  ~onOutputFormatChange: outputFormat => unit,
  ~preset,
  ~lossless: bool,
  ~onPresetChange,
  ~onLosslessChange,
  ~onDownload,
  ~canDownload: bool,
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

  let menuRef = React.useRef(Nullable.null)
  let triggerRef = React.useRef(Nullable.null)
  let restoreTriggerFocus: unit => unit = () => {
    let trigger = triggerRef.current
    let focus: Nullable.t<Dom.element> => unit = %raw("trigger => trigger?.focus()")
    focus(trigger)
  }
  React.useEffect2(() => {
    if downloadMenuOpen && canDownload && hasItems {
      let dismiss: unit => unit = () => setDownloadMenuOpen(_ => false)
      let root = menuRef.current
      let trigger = triggerRef.current
      let listen: (Nullable.t<Dom.element>, Nullable.t<Dom.element>, unit => unit) => (unit => unit) = %raw(
        `(root, trigger, dismiss) => {
          const keydown = event => {
            if (event.key === 'Escape') {
              event.preventDefault(); dismiss(); trigger?.focus();
            }
          };
          const pointerdown = event => {
            if (!root?.contains(event.target)) {
              dismiss();
              if (root?.contains(document.activeElement)) {
                const focusTarget = event.target instanceof Element && event.target.closest(
                  'button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), a[href], [tabindex], [contenteditable="true"]'
                );
                // Let an outside control receive focus normally; blank space returns to the trigger.
                if (!focusTarget) {
                  event.preventDefault(); trigger?.focus();
                }
              }
            }
          };
          document.addEventListener('keydown', keydown);
          document.addEventListener('pointerdown', pointerdown);
          return () => {
            document.removeEventListener('keydown', keydown);
            document.removeEventListener('pointerdown', pointerdown);
          };
        }`
      )
      Some(listen(root, trigger, dismiss))
    } else {
      if downloadMenuOpen { setDownloadMenuOpen(_ => false) }
      None
    }
  }, (downloadMenuOpen, canDownload && hasItems))

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

  <div dataTestId="bottom-bar" className="rounded-lg border border-neutral-800 bg-neutral-900 p-6">
    <div className="flex min-w-0 flex-col gap-6">
      {hasItems
        ? <div className="flex flex-col gap-2">
            <div ref={ReactDOM.Ref.domRef(menuRef)} className="relative flex w-full">
              <button
                type_="button"
                dataTestId="download-primary"
                disabled={!canDownload}
                onClick={_ => onDownload()}
                className="min-h-12 min-w-12 flex-1 disabled:opacity-50 disabled:cursor-not-allowed text-base bg-white text-neutral-900 px-4 py-2 rounded-l font-medium hover:bg-neutral-100 transition-colors motion-reduce:transition-none border-r border-neutral-200"
              >
                {React.string("Download")}
              </button>
              <button
                type_="button"
                dataTestId="download-format-toggle"
                ref={ReactDOM.Ref.domRef(triggerRef)}
                disabled={!canDownload}
                ariaExpanded={downloadMenuOpen && canDownload}
                ariaLabel="Choose download format"
                onClick={_ => setDownloadMenuOpen(prev => !prev)}
                className="min-h-12 min-w-12 inline-flex items-center justify-center disabled:opacity-50 disabled:cursor-not-allowed text-base bg-white text-neutral-900 px-4 py-2 rounded-r font-medium hover:bg-neutral-100 transition-colors motion-reduce:transition-none"
              >
                <svg className="pointer-events-none size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" ariaHidden=true>
                  <path d="M6 9l6 6 6-6" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                </svg>
              </button>
              {downloadMenuOpen && canDownload
                ? <div
                    className="absolute top-full right-0 mt-2 z-20 min-w-48 rounded border border-neutral-700 bg-neutral-800 py-2 shadow-lg"
                  >
                    {downloadFormats
                    ->Array.map(((label, fmt)) =>
                      <button
                        key=label
                        type_="button"
                        dataTestId={"download-as-" ++ label}
                        onClick={_ => {
                          setDownloadMenuOpen(_ => false)
                          restoreTriggerFocus()
                          onDownloadFormat(fmt)
                        }}
                        className="min-h-12 block w-full text-left text-base px-4 py-2 text-neutral-200 hover:bg-neutral-700 transition-colors motion-reduce:transition-none"
                      >
                        {React.string("Download as " ++ label)}
                      </button>
                    )
                    ->React.array}
                  </div>
                : React.null}
            </div>
            {hasCompletedItems
              ? <button
                  type_="button"
                  disabled=processingAll
                  onClick={_ => onDownloadAll()}
                  className="w-full min-h-12 text-base border border-neutral-700 bg-neutral-800 text-neutral-200 px-4 py-2 rounded font-medium hover:bg-neutral-700 transition-colors motion-reduce:transition-none disabled:cursor-wait disabled:opacity-50"
                >
                  {React.string(
                    if failedCount > 0 {
                      "Download successful files (" ++ Int.toString(completedCount) ++ ")"
                    } else {
                      "Download All"
                    },
                  )}
                </button>
              : React.null}
            {completedCount + failedCount >= 2 && hasCompletedItems
              ? <button
                  type_="button"
                  disabled={processingAll || exportingZip}
                  ariaBusy=exportingZip
                  onClick={_ => onDownloadZip()}
                  className="w-full min-h-12 text-base border border-neutral-700 bg-neutral-700 text-neutral-200 px-4 py-2 rounded font-medium hover:bg-neutral-600 transition-colors motion-reduce:transition-none disabled:cursor-wait disabled:opacity-50"
                >
                  {React.string(
                    if exportingZip {
                      "Exporting ZIP..."
                    } else if failedCount > 0 {
                      "Download ZIP (" ++ Int.toString(completedCount) ++ " successful)"
                    } else {
                      "Download ZIP"
                    },
                  )}
                </button>
              : React.null}
            {if processingAll && hasCompletedItems {
              <p className="text-sm text-neutral-400">
                {React.string("Batch downloads are available when every file finishes.")}
              </p>
            } else if failedCount > 0 {
              <p className="text-sm text-amber-400">
                {React.string(
                  Int.toString(failedCount) ++ if failedCount == 1 {
                    " failed file excluded from downloads."
                  } else {
                    " failed files excluded from downloads."
                  },
                )}
              </p>
            } else {
              React.null
            }}
          </div>
        : React.null}
      <div className="space-y-2">
        <label className="flex flex-col gap-2 text-base text-neutral-400">
          <span> {React.string("Convert to")} </span>
          <div className="relative">
          <select
            ariaLabel="Output format"
            value={Types.outputFormatToChoice(outputFormat)}
            onChange={e => {
              let value = ReactEvent.Form.target(e)["value"]->Option.getOr("same")
              onOutputFormatChange(
                switch value {
                | "auto" => AutoSmallest
                | "png" => ForcePng
                | "jpeg" => ForceJpeg
                | "webp" => ForceWebp
                | "avif" => ForceAvif
                | _ => SameAsInput
                },
              )
            }}
            className="min-h-12 w-full appearance-none text-base focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-300 rounded border border-neutral-600 bg-neutral-800 pl-4 pr-12 py-2 text-neutral-200"
          >
            <option value="same"> {React.string("Same as input")} </option>
            <option value="auto"> {React.string("Auto (smallest tested)")} </option>
            <option value="png"> {React.string("PNG")} </option>
            <option value="jpeg"> {React.string("JPEG")} </option>
            <option value="webp"> {React.string("WebP")} </option>
            <option value="avif"> {React.string("AVIF")} </option>
          </select>
          <svg className="pointer-events-none absolute right-4 top-1/2 size-4 -translate-y-1/2 text-neutral-300" viewBox="0 0 24 24" fill="none" stroke="currentColor" ariaHidden=true>
                  <path d="M6 9l6 6 6-6" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                </svg>
          </div>
        </label>
        <span className="text-sm text-neutral-400"> {React.string("Output: " ++ format)} </span>
      </div>
      <div className="space-y-2">
        <div className="w-full min-w-0 space-y-2">
          <div className="flex items-center justify-between gap-2 text-base">
            <h2 className="font-medium"> {React.string("Compression")} </h2>
            <span className="text-base text-neutral-300 font-medium">
              {React.string(presetLabel(preset))}
            </span>
          </div>
          <input
            type_="range"
            ariaLabel="Compression preset"
            ariaValuetext={presetLabel(preset)}
            min="0"
            max="3"
            step=1.0
            value={Int.toString(sliderValue)}
            onChange=handleSliderChange
            className="w-full min-w-0 min-h-12 rounded-lg cursor-pointer accent-white"
          />
          <div className="flex justify-between">
            <span className="text-sm text-neutral-400"> {React.string("Smaller file")} </span>
            <span className="text-sm text-neutral-400"> {React.string("Faster")} </span>
          </div>
        </div>
        <label
          className="min-h-12 flex items-center gap-2 cursor-pointer"
          title="Lossless encoding; resize changes image dimensions"
        >
          <input
            type_="checkbox"
            checked=lossless
            onChange={e => {
              let checked = ReactEvent.Form.target(e)["checked"]
              onLosslessChange(checked == Some(true))
            }}
            className="w-4 h-4 rounded border-neutral-600 bg-neutral-800 text-white focus:ring-2 focus:ring-neutral-500"
          />
          <span className="text-base text-neutral-300"> {React.string("Perfect Quality")} </span>
        </label>
        <p className="text-sm text-neutral-400">
          {React.string("Lossless encoding. Resizing still changes image dimensions.")}
        </p>
      </div>
      <div className="grid grid-cols-2 items-center gap-x-4 gap-y-2">
        <h2 className="col-span-2 text-base font-medium"> {React.string("Resize")} </h2>
        <label className="flex min-w-0 flex-col gap-2 text-base text-neutral-300">
          {React.string("Width (px)")}
          <input
            type_="number"
            placeholder="W"
            ariaLabel="Target width"
            value={switch targetWidth {
            | Some(w) => Int.toString(w)
            | None => ""
            }}
            min="1"
            max="8000"
            onChange={e => {
              let raw = ReactEvent.Form.target(e)["value"]
              let v = switch raw {
              | Some(s) => Int.fromString(s)
              | None => None
              }
              switch v {
              | Some(n) if n > 0 => onTargetWidthChange(Some(n))
              | _ => onTargetWidthChange(None)
              }
            }}
            className="min-h-12 w-full min-w-0 text-base tabular-nums [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:m-0 [&::-webkit-outer-spin-button]:m-0 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-300 bg-neutral-800 text-neutral-300 border border-neutral-600 rounded px-4 py-2"
          />
        </label>
        <label className="flex min-w-0 flex-col gap-2 text-base text-neutral-300">
          {React.string("Height (px)")}
          <input
            type_="number"
            placeholder="H"
            ariaLabel="Target height"
            value={switch targetHeight {
            | Some(h) => Int.toString(h)
            | None => ""
            }}
            min="1"
            max="8000"
            onChange={e => {
              let raw = ReactEvent.Form.target(e)["value"]
              let v = switch raw {
              | Some(s) => Int.fromString(s)
              | None => None
              }
              switch v {
              | Some(n) if n > 0 => onTargetHeightChange(Some(n))
              | _ => onTargetHeightChange(None)
              }
            }}
            className="min-h-12 w-full min-w-0 text-base tabular-nums [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:m-0 [&::-webkit-outer-spin-button]:m-0 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-300 bg-neutral-800 text-neutral-300 border border-neutral-600 rounded px-4 py-2"
          />
        </label>
        <p className="col-span-2 text-sm text-neutral-400">
          {React.string("Each image keeps its own aspect ratio. Up to 8000 px.")}
        </p>
      </div>
      {hasItems
        ? <div className="space-y-6">
            <button
              type_="button"
              disabled=processingAll
              onClick={_ => onCompressAll()}
              className={"w-full min-h-12 text-base px-4 py-2 rounded font-medium transition-colors motion-reduce:transition-none " ++ if (
                processingAll
              ) {
                "border border-neutral-700 bg-neutral-800 text-neutral-400 cursor-not-allowed"
              } else {
                "border border-neutral-700 bg-neutral-800 text-neutral-200 hover:bg-neutral-700"
              }}
            >
              {React.string(
                if processingAll {
                  "Processing…"
                } else {
                  "Compress All"
                },
              )}
            </button>
            {switch appliedOptimizations {
            | Some(optimizations) =>
              <div className="basis-full min-w-0 break-words text-sm text-neutral-400">
                <span className="text-neutral-400"> {React.string("What we did: ")} </span>
                {React.string(optimizations->Array.join(", "))}
              </div>
            | None => React.null
            }}
          </div>
        : React.null}
    </div>
  </div>
}
