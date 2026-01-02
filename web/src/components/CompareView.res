open React
open Types

@send external getBoundingClientRect: Dom.element => {..} = "getBoundingClientRect"

let formatSize = (bytes: int): string => {
  if bytes >= 1_000_000 {
    let mb = Math.round(Int.toFloat(bytes) /. 1000000.0 *. 10.0) /. 10.0
    Float.toString(mb) ++ " MB"
  } else if bytes >= 1000 {
    let kb = Math.round(Int.toFloat(bytes) /. 1000.0 *. 10.0) /. 10.0
    Float.toString(kb) ++ " KB"
  } else {
    Int.toString(bytes) ++ " bytes"
  }
}

let calculateSavings = (original: int, compressed: int): option<(float, string)> => {
  if original <= 0 {
    None
  } else {
    let saved = original - compressed
    let percent = Int.toFloat(saved) /. Int.toFloat(original) *. 100.0
    if percent > 0.0 {
      Some((percent, formatSize(saved)))
    } else {
      None
    }
  }
}

let savingsColor = (percent: float): string => {
  if percent >= 30.0 {
    "text-green-400"
  } else if percent >= 10.0 {
    "text-yellow-400"
  } else {
    "text-gray-400"
  }
}

@react.component
let make = (~originalUrl, ~compressedUrl, ~originalBytes, ~compressedBytes, ~onRemove) => {
  let (sliderPos, setSliderPos) = React.useState(() => 50.0)
  let sliderRef = React.useRef(Nullable.null)
  let containerRef = React.useRef(Nullable.null)
  let isDragging = React.useRef(false)

  let styleHeight500 =
    ReactDOM.Style._dictToStyle(Dict.fromArray([("height", "500px")]))

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
    <div className="mt-8 flex flex-col items-center justify-center h-96 bg-neutral-900/50 rounded-lg border border-neutral-800">
      <div className="w-10 h-10 border-4 border-neutral-700 border-t-blue-500 rounded-full animate-spin mb-4"></div>
      <p className="text-neutral-400">{React.string("compressing")}</p>
      <p className="text-neutral-500 text-sm mt-2">{React.string(formatSize(originalBytes))}</p>
    </div>
  | (Some(orig), Some(comp)) =>
    let savings = calculateSavings(originalBytes, compressedBytes->Option.getOr(0))
    
    <div className="mt-8">
      {switch savings {
      | Some((percent, saved)) =>
        <div className="flex justify-center mb-4">
          <div className={"text-lg font-medium " ++ savingsColor(percent)}>
            {React.string("Saved " ++ saved ++ " (" ++ to1dp(percent) ++ "%)")}
          </div>
        </div>
      | None => React.null
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
        <div className="relative w-full bg-neutral-900" style={styleHeight500}>
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
            {React.string("Original: " ++ formatSize(originalBytes))}
          </div>
          <div className="absolute top-3 right-3 flex items-center gap-2 z-20">
            {switch compressedBytes {
            | Some(bytes) =>
              <div className="bg-black/60 backdrop-blur-md text-white px-3 py-1.5 rounded-md text-sm font-medium">
                {React.string("Compressed: " ++ formatSize(bytes))}
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
