open React

@send external getBoundingClientRect: Dom.element => {..} = "getBoundingClientRect"

@react.component
let make = (~originalUrl, ~compressedUrl, ~originalBytes, ~compressedBytes, ~width: option<int>, ~height: option<int>, ~compressionTime, ~compressionProgress: option<Types.compressionProgress>, ~onRemove) => {
  let (sliderPos, setSliderPos) = React.useState(() => 50.0)
  let sliderRef = React.useRef(Nullable.null)
  let containerRef = React.useRef(Nullable.null)
  let isDragging = React.useRef(false)

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

  let handleMouseDown = _ => {
    isDragging.current = true
  }

  let handleMouseMove = (e: ReactEvent.Mouse.t) => {
    if isDragging.current {
      switch containerRef.current->Nullable.toOption {
      | Some(container) =>
        let rect = container->getBoundingClientRect
        let x = Int.toFloat(ReactEvent.Mouse.clientX(e))
        let left: float = rect["left"]
        let width: float = rect["width"]
        let percent = ((x -. left) /. width) *. 100.0
        let clamped = Math.max(0.0, Math.min(100.0, percent))
        setSliderPos(_ => clamped)
      | None => ()
      }
    }
  }

  let handleMouseUp = _ => {
    isDragging.current = false
  }

  let handleTouchMove = (_e: ReactEvent.Touch.t) => {
    if isDragging.current {
      switch containerRef.current->Nullable.toOption {
      | Some(container) =>
        let rect = container->getBoundingClientRect
        let touchX = %raw("e.touches[0].clientX")
        let left: float = rect["left"]
        let width: float = rect["width"]
        let percent = ((touchX -. left) /. width) *. 100.0
        let clamped = Math.max(0.0, Math.min(100.0, percent))
        setSliderPos(_ => clamped)
      | None => ()
      }
    }
  }

  let handleTouchStart = _ => {
    isDragging.current = true
  }

  let handleTouchEnd = _ => {
    isDragging.current = false
  }

  switch (originalUrl, compressedUrl) {
  | (None, None) =>
    <div className="mt-8 flex flex-col items-center justify-center h-96 bg-neutral-900/50 rounded-lg border border-neutral-800">
      <div className="w-10 h-10 border-4 border-neutral-700 border-t-blue-500 rounded-full animate-spin mb-4"></div>
      <p className="text-neutral-400">{React.string("processing image")}</p>
    </div>
  | (Some(_), None) =>
    switch compressionProgress {
    | Some(progress) =>
      let elapsed = %raw("performance.now()") -. progress.startTime
      <div className="mt-8 flex flex-col items-center justify-center h-96 bg-neutral-900/50 rounded-lg border border-neutral-800">
        <div className="w-20 h-20 mb-6 relative">
          <svg className="w-20 h-20 transform -rotate-90" viewBox="0 0 36 36">
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
            <span className="text-lg font-semibold text-white" dataTestId="compression-percent">
              {React.string(`${Int.toString(progress.progress)}%`)}
            </span>
          </div>
        </div>
        <p className="text-neutral-300 text-base font-medium mb-2 capitalize" dataTestId="compression-phase">
          {React.string(Progress.phaseLabel(progress.phase))}
        </p>
        <p className="text-neutral-400 text-sm mb-4">
          {React.string(Display.formatSize(originalBytes))}
        </p>
        <div className="w-64 bg-neutral-800 rounded-full h-2 mb-4 overflow-hidden">
          <div
            className="bg-blue-500 h-2 rounded-full transition-all duration-300 ease-out"
            style={ReactDOM.Style._dictToStyle(Dict.fromArray([("width", Int.toString(progress.progress) ++ "%")]))}
          />
        </div>
        <p className="text-neutral-500 text-xs">
          {React.string("Time elapsed: " ++ Display.formatTimeFloat(elapsed))}
        </p>
      </div>
    | None =>
      <div className="mt-8 flex flex-col items-center justify-center h-96 bg-neutral-900/50 rounded-lg border border-neutral-800">
        <div className="w-10 h-10 border-4 border-neutral-700 border-t-blue-500 rounded-full animate-spin mb-4"></div>
        <p className="text-neutral-400">{React.string("compressing")}</p>
        <p className="text-neutral-500 text-sm mt-2">{React.string(Display.formatSize(originalBytes))}</p>
      </div>
    }
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
      
      <div
        ref={ReactDOM.Ref.domRef(containerRef)}
        className="relative rounded-lg overflow-hidden cursor-ew-resize select-none border border-neutral-800"
        onMouseMove=handleMouseMove
        onMouseUp=handleMouseUp
        onMouseLeave=handleMouseUp
        onTouchMove=handleTouchMove
        onTouchStart=handleTouchStart
        onTouchEnd=handleTouchEnd>
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
          
          /* Slider Handle */
          <div
            ref={ReactDOM.Ref.domRef(sliderRef)}
            className="absolute top-0 bottom-0 w-0.5 bg-white z-10 shadow-xl"
            style={
              ReactDOM.Style._dictToStyle(
                Dict.fromArray([("left", to1dp(sliderPos) ++ "%")]),
              )
            }
            onMouseDown=handleMouseDown>
            <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-10 h-10 bg-white rounded-full shadow-lg flex items-center justify-center">
              <svg className="w-6 h-6 text-neutral-800" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M15 19l-7-7 7-7" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M9 5l7 7-7 7" />
              </svg>
              <div className="absolute -top-8 bg-white text-neutral-900 px-2 py-0.5 rounded text-[10px] font-bold shadow-sm">
                {React.string(to1dp(sliderPos) ++ "%")}
              </div>
            </div>
          </div>
          
          <div className="absolute top-3 left-3 bg-black/60 backdrop-blur-md text-white px-3 py-1.5 rounded-md text-sm font-medium z-20">
            {React.string("Original: " ++ Display.formatSize(originalBytes))}
          </div>
          <div className="absolute top-3 right-3 flex items-center gap-2 z-20">
            {switch compressedBytes {
            | Some(bytes) =>
              <div className="bg-black/60 backdrop-blur-md text-white px-3 py-1.5 rounded-md text-sm font-medium">
                {React.string("Compressed: " ++ Display.formatSize(bytes))}
              </div>
            | None => React.null
            }}
            <button
              onClick={_ => onRemove()}
              className="p-2 bg-black/60 backdrop-blur-md text-white/70 hover:text-white hover:bg-red-500 rounded-md transition-all shadow-lg border border-white/10"
              ariaLabel="Remove image">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          
          <div className="absolute bottom-4 left-4 bg-black/50 backdrop-blur-sm text-white px-3 py-1 rounded text-xs font-semibold z-20">
            {React.string("Original")}
          </div>
          <div className="absolute bottom-4 right-4 bg-black/50 backdrop-blur-sm text-white px-3 py-1 rounded text-xs font-semibold z-20">
            {React.string("Compressed")}
          </div>
        </div>
      </div>
    </div>
  | _ => React.null
  }
}
