open React
open Types

let formatSize = (bytes: int): string => {
  if bytes >= 1_000_000 {
    let mb = Js.Math.round(float_of_int(bytes) /. 1000000.0 *. 10.0) /. 10.0
    Float.toString(mb) ++ " MB"
  } else if bytes >= 1000 {
    let kb = Js.Math.round(float_of_int(bytes) /. 1000.0 *. 10.0) /. 10.0
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
    let percent = float_of_int(saved) /. float_of_int(original) *. 100.0
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
let make = (~mode, ~originalUrl, ~compressedUrl, ~originalBytes, ~compressedBytes) => {
  let (sliderPos, setSliderPos) = React.useState(() => 50.0)
  let sliderRef = React.useRef(Nullable.null)
  let containerRef = React.useRef(Nullable.null)
  let isDragging = %raw("{ current: false }")

  let styleHeight500 =
    ReactDOM.Style._dictToStyle(Js.Dict.fromArray([("height", "500px")]))

  let to1dp = (value: float): string => {
    let rounded = Js.Math.round(value *. 10.0) /. 10.0
    Float.toString(rounded)
  }

  let handleMouseDown = _ => {
    %raw("isDragging.current = true")
  }

  let handleMouseMove = (e: ReactEvent.Mouse.t) => {
    if %raw("isDragging.current") {
      switch containerRef.current->Nullable.toOption {
      | Some(_container) =>
        let rect = %raw("_container.getBoundingClientRect()")
        let x = Int.toFloat(ReactEvent.Mouse.clientX(e))
        let left = %raw("rect.left")
        let width = %raw("rect.width")
        let percent = ((x -. left) /. width) *. 100.0
        let clamped = max(0.0, min(100.0, percent))
        setSliderPos(_ => clamped)
      | None => ()
      }
    }
  }

  let handleMouseUp = _ => {
    %raw("isDragging.current = false")
  }

  let handleTouchMove = (e: ReactEvent.Touch.t) => {
    if %raw("isDragging.current") {
      switch containerRef.current->Nullable.toOption {
      | Some(_container) =>
        let rect = %raw("_container.getBoundingClientRect()")
        let touch = %raw("ReactEvent.Touch.touches(e)[0]")
        let x = %raw("touch.clientX")
        let left = %raw("rect.left")
        let width = %raw("rect.width")
        let percent = ((x -. left) /. width) *. 100.0
        let clamped = max(0.0, min(100.0, percent))
        setSliderPos(_ => clamped)
      | None => ()
      }
    }
  }

  let handleTouchStart = _ => {
    %raw("isDragging.current = true")
  }

  let handleTouchEnd = _ => {
    %raw("isDragging.current = false")
  }

  switch (originalUrl, compressedUrl) {
  | (None, None) =>
    <div className="mt-8 flex flex-col items-center justify-center h-96 bg-neutral-900/50 rounded-lg border border-neutral-800">
      <div className="w-10 h-10 border-4 border-neutral-700 border-t-blue-500 rounded-full animate-spin mb-4"></div>
      <p className="text-neutral-400">{React.string("processing image")}</p>
    </div>
  | (Some(orig), None) =>
    <div className="mt-8 flex flex-col items-center justify-center h-96 bg-neutral-900/50 rounded-lg border border-neutral-800">
      <div className="w-10 h-10 border-4 border-neutral-700 border-t-blue-500 rounded-full animate-spin mb-4"></div>
      <p className="text-neutral-400">{React.string("compressing")}</p>
      <p className="text-neutral-500 text-sm mt-2">{React.string(formatSize(originalBytes))}</p>
    </div>
  | (Some(orig), Some(comp)) =>
    let savings = calculateSavings(originalBytes, compressedBytes->Option.getWithDefault(0))
    
    switch mode {
    | SideBySide =>
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
        
        <div className="grid grid-cols-2 gap-4">
          <div className="relative rounded-lg overflow-hidden bg-neutral-900">
            <img src=orig alt="Original" className="w-full h-auto" />
            <div className="absolute top-3 left-3 bg-black/70 text-white px-3 py-1.5 rounded-md text-sm font-medium">
              {React.string("Original: " ++ formatSize(originalBytes))}
            </div>
          </div>
          <div className="relative rounded-lg overflow-hidden bg-neutral-900">
            <img src=comp alt="Compressed" className="w-full h-auto" />
            {switch compressedBytes {
            | Some(bytes) =>
              <div className="absolute top-3 left-3 bg-black/70 text-white px-3 py-1.5 rounded-md text-sm font-medium">
                {React.string("Compressed: " ++ formatSize(bytes))}
              </div>
            | None => React.null
            }}
          </div>
        </div>
      </div>
    | Slider =>
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
          className="relative rounded-lg overflow-hidden cursor-ew-resize select-none"
          onMouseMove=handleMouseMove
          onMouseUp=handleMouseUp
          onMouseLeave=handleMouseUp
          onTouchMove=handleTouchMove
          onTouchStart=handleTouchStart
          onTouchEnd=handleTouchEnd>
          <div className="relative w-full" style={styleHeight500}>
            <img src=orig alt="Original" className="absolute inset-0 w-full h-full object-contain" />
            {switch compressedBytes {
            | Some(_) =>
              let clipPath =
                "polygon(0 0, "
                ++ to1dp(sliderPos)
                ++ "% 0, "
                ++ to1dp(sliderPos)
                ++ "% 100%, 0 100%)"
              <div
                className="absolute inset-0 overflow-hidden"
                style={ReactDOM.Style._dictToStyle(Js.Dict.fromArray([("clipPath", clipPath)]))}>
                <img src=comp alt="Compressed" className="w-full h-full object-contain" />
              </div>
            | None => React.null
            }}
            
            <div
              ref={ReactDOM.Ref.domRef(sliderRef)}
              className="absolute top-0 bottom-0 w-0.5 bg-white z-10 shadow-xl"
              style={
                ReactDOM.Style._dictToStyle(
                  Js.Dict.fromArray([("left", to1dp(sliderPos) ++ "%")]),
                )
              }
              onMouseDown=handleMouseDown>
              <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-10 h-10 bg-white rounded-full shadow-lg flex items-center justify-center">
                <svg className="w-5 h-5 text-neutral-700" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 19l-7-7 7-7" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 5l7 7-7 7" />
                </svg>
              </div>
            </div>
            
            <div className="absolute top-3 left-3 bg-black/70 text-white px-3 py-1.5 rounded-md text-sm font-medium">
              {React.string("Original: " ++ formatSize(originalBytes))}
            </div>
            {switch compressedBytes {
            | Some(bytes) =>
              <div className="absolute top-3 right-3 bg-black/70 text-white px-3 py-1.5 rounded-md text-sm font-medium">
                {React.string("Compressed: " ++ formatSize(bytes))}
              </div>
            | None => React.null
            }}
            
            <div className="absolute bottom-3 left-3 bg-black/70 text-white px-3 py-1 rounded text-xs">
              {React.string("Original")}
            </div>
            <div className="absolute bottom-3 right-3 bg-black/70 text-white px-3 py-1 rounded text-xs">
              {React.string("Compressed")}
            </div>
          </div>
        </div>
      </div>
    }
  | _ => React.null
  }
}
