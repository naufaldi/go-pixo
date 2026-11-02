@react.component
let make = (~originalUrl, ~compressedUrl, ~originalBytes, ~compressedBytes, ~width: option<int>, ~height: option<int>, ~compressionTime, ~compressionProgress: option<Types.compressionProgress>, ~onRemove) => {
  let (sliderPos, setSliderPos) = React.useState(() => 50.0)

  let styleHeight500 =
    ReactDOM.Style._dictToStyle(Dict.fromArray([("height", "500px")]))

  let compareFrameStyle = switch (width, height) {
  | (Some(imageWidth), Some(imageHeight)) when imageWidth > 0 && imageHeight > 0 =>
    let aspect = Int.toFloat(imageWidth) /. Int.toFloat(imageHeight)
    let maxWidth = Int.fromFloat(Math.max(1.0, Math.round(500.0 *. aspect)))
    ReactDOM.Style._dictToStyle(Dict.fromArray([
      ("aspectRatio", Int.toString(imageWidth) ++ " / " ++ Int.toString(imageHeight)),
      ("width", "min(100%, " ++ Int.toString(maxWidth) ++ "px)"),
    ]))
  | _ => styleHeight500
  }

  let to1dp = (value: float): string => {
    let rounded = Math.round(value *. 10.0) /. 10.0
    Float.toString(rounded)
  }

  switch (originalUrl, compressedUrl) {
  | (None, None) | (Some(_), None) =>
    <div className="mt-8 flex flex-col items-center justify-center h-96 bg-neutral-900/50 rounded-lg border border-neutral-800">
      {switch compressionProgress {
      | Some(progress) => <ProgressStatus progress />
      | None =>
        <div className="flex flex-col items-center gap-4">
          <div className="w-10 h-10 border-4 border-neutral-700 border-t-blue-500 rounded-full animate-spin motion-reduce:animate-none" />
          <p className="text-neutral-400">{React.string("Waiting to process image")}</p>
        </div>
      }}
      <p className="text-neutral-400 text-sm mt-4">{React.string(Display.formatSize(originalBytes))}</p>
    </div>
  | (Some(orig), Some(comp)) =>
    let compressedSize = compressedBytes->Option.getOr(0)
    let savings = Display.calculateSavings(originalBytes, compressedSize)
    let optimized = Display.isAlreadyOptimized(originalBytes, compressedSize)
    
    <div className="mt-8">
      {if optimized {
        <div className="flex justify-center mb-4">
          <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-500/10 border border-blue-500/20 text-blue-400">
            <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span className="text-sm font-medium">{"No smaller output with current settings"->React.string}</span>
          </div>
        </div>
      } else {
        switch savings {
        | Some((percent, saved)) =>
          <div className="flex justify-center mb-4">
            <div className={"text-lg font-medium " ++ Display.savingsColor(percent)}>
              {React.string(
                "Saved " ++ saved ++ " (" ++ to1dp(percent) ++ "%)" ++
                switch compressionTime {
                | Some(time) => " in " ++ Display.formatTime(time)
                | None => ""
                }
              )}
            </div>
          </div>
        | None => React.null
        }
      }}
      
      <div className="relative rounded-lg overflow-hidden select-none border border-neutral-800">
        <div className="relative w-full mx-auto bg-neutral-900" style={compareFrameStyle}>
          /* Background: Compressed Image (Right Side) */
          <img src=comp alt="Compressed" className="absolute inset-0 w-full h-full object-contain" />
          
          /* Overlay: Original Image (Left Side, Clipped) */
          {
            let clipPath =
              "polygon(0 0, "
              ++ to1dp(sliderPos)
              ++ "% 0, "
              ++ to1dp(sliderPos)
              ++ "% 100%, 0 100%)"
            <div
              className="absolute inset-0 overflow-hidden"
              style={ReactDOM.Style._dictToStyle(Dict.fromArray([("clipPath", clipPath)]))}>
              <img src=orig alt="Original" className="w-full h-full object-contain" />
            </div>
          }
          
          /* The range below controls this visual divider. */
          <div
            className="pointer-events-none absolute top-0 bottom-0 w-0.5 bg-white z-10 shadow-xl"
            style={
              ReactDOM.Style._dictToStyle(
                Dict.fromArray([("left", to1dp(sliderPos) ++ "%")]),
              )
            }
            ariaHidden=true />
          
          <div className="absolute top-3 inset-x-3 flex flex-wrap items-center gap-2 z-20">
          <div className="bg-black/60 backdrop-blur-md text-white px-3 py-1.5 rounded-md text-sm font-medium">
            {React.string("Original: " ++ Display.formatSize(originalBytes))}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {switch compressedBytes {
            | Some(bytes) =>
              <div className="bg-black/60 backdrop-blur-md text-white px-3 py-1.5 rounded-md text-sm font-medium">
                {React.string("Compressed: " ++ Display.formatSize(bytes))}
              </div>
            | None => React.null
            }}
            <button
              onClick={_ => onRemove()}
              className="min-h-11 min-w-11 p-2 bg-black/60 backdrop-blur-md text-white/70 hover:text-white hover:bg-red-500 rounded-md transition-colors motion-reduce:transition-none shadow-lg border border-white/10"
              ariaLabel="Remove image">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          </div>
          
          <div className="absolute bottom-4 left-4 bg-black/50 backdrop-blur-sm text-white px-3 py-1 rounded text-xs font-semibold z-20">
            {React.string("Original")}
          </div>
          <div className="absolute bottom-4 right-4 bg-black/50 backdrop-blur-sm text-white px-3 py-1 rounded text-xs font-semibold z-20">
            {React.string("Compressed")}
          </div>
        </div>
      </div>
      <label className="mt-3 flex flex-col text-sm text-neutral-300">
        <span>{React.string("Compare original and compressed")}</span>
        <input type_="range" min="0" max="100" step=1.0
          value={to1dp(sliderPos)}
          ariaValuetext={to1dp(sliderPos) ++ "% original, " ++ to1dp(100.0 -. sliderPos) ++ "% compressed"}
          onChange={e => {
            let value = ReactEvent.Form.target(e)["value"]->Option.flatMap(value => Int.fromString(value))->Option.getOr(50)->Int.toFloat
            setSliderPos(_ => Math.max(0.0, Math.min(100.0, value)))
          }}
          className="w-full min-h-11 cursor-ew-resize accent-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-300" />
      </label>
    </div>
  | _ => React.null
  }
}
