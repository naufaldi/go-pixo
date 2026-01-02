open React
open Types

@react.component
let make = (~mode, ~originalUrl, ~compressedUrl, ~originalBytes, ~compressedBytes) => {
  let (sliderPos, setSliderPos) = React.useState(() => 50.0)
  let sliderRef = React.useRef(Nullable.null)
  let containerRef = React.useRef(Nullable.null)
  let isDragging = %raw("{ current: false }")

  let styleHeight600 =
    ReactDOM.Style._dictToStyle(Js.Dict.fromArray([("height", "600px")]))

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
  
  switch (originalUrl, compressedUrl) {
  | (Some(orig), Some(comp)) =>
    switch mode {
    | SideBySide =>
      <div className="grid grid-cols-2 gap-4 mt-8">
        <div className="relative">
          <img src=orig alt="Original" className="w-full h-auto rounded-lg" />
          <div className="absolute top-2 left-2 bg-black/70 text-white px-2 py-1 rounded text-xs">
            {React.string("Original (" ++ Int.toString(originalBytes) ++ " bytes)")}
          </div>
        </div>
        <div className="relative">
        {switch compressedBytes {
        | Some(bytes) =>
          <>
            <img src=comp alt="Compressed" className="w-full h-auto rounded-lg" />
            <div className="absolute top-2 left-2 bg-black/70 text-white px-2 py-1 rounded text-xs">
              {React.string("Compressed (" ++ Int.toString(bytes) ++ " bytes)")}
            </div>
          </>
        | None => React.null
        }}
        </div>
      </div>
    | Slider =>
      <div
        ref={ReactDOM.Ref.domRef(containerRef)}
        className="relative mt-8 rounded-lg overflow-hidden"
        onMouseMove=handleMouseMove
        onMouseUp=handleMouseUp>
        <div className="relative w-full" style={styleHeight600}>
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
            className="absolute top-0 bottom-0 w-1 bg-white cursor-ew-resize z-10"
            style={
              ReactDOM.Style._dictToStyle(
                Js.Dict.fromArray([("left", to1dp(sliderPos) ++ "%")]),
              )
            }
            onMouseDown=handleMouseDown>
            <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-8 h-8 bg-white rounded-full shadow-lg flex items-center justify-center">
              <div className="w-1 h-4 bg-neutral-900"></div>
            </div>
          </div>
          <div className="absolute top-2 left-2 bg-black/70 text-white px-2 py-1 rounded text-xs">
            {React.string("Original (" ++ Int.toString(originalBytes) ++ " bytes)")}
          </div>
          {switch compressedBytes {
          | Some(bytes) =>
            <div className="absolute top-2 right-2 bg-black/70 text-white px-2 py-1 rounded text-xs">
              {React.string("Compressed (" ++ Int.toString(bytes) ++ " bytes)")}
            </div>
          | None => React.null
          }}
        </div>
      </div>
    }
  | _ => React.null
  }
}
