type frameRect = {left: float, width: float}
type gesture = {pointerId: Dom.eventPointerId, startPos: float, touch: bool}

@get external pointerFrame: ReactEvent.Pointer.t => Dom.element = "currentTarget"
@send external getBoundingClientRect: Dom.element => frameRect = "getBoundingClientRect"
@send external setPointerCapture: (Dom.element, Dom.eventPointerId) => unit = "setPointerCapture"
@send external releasePointerCapture: (Dom.element, Dom.eventPointerId) => unit = "releasePointerCapture"
@send external hasPointerCapture: (Dom.element, Dom.eventPointerId) => bool = "hasPointerCapture"

@react.component
let make = (
  ~originalUrl,
  ~compressedUrl,
  ~originalBytes,
  ~compressedBytes,
  ~width: option<int>,
  ~height: option<int>,
  ~compressionTime,
  ~compressionProgress: option<Types.compressionProgress>,
  ~onRemove as _: unit => unit,
) => {
  let (sliderPos, setSliderPos) = React.useState(() => 50.0)
  let dragRef: React.ref<option<gesture>> = React.useRef(None)
  let updatePosition = e => {
    let rect = getBoundingClientRect(pointerFrame(e))
    if rect.width > 0.0 {
      let position = Math.round((Int.toFloat(ReactEvent.Pointer.clientX(e)) -. rect.left) /. rect.width *. 1000.0) /. 10.0
      let position = Math.max(0.0, Math.min(100.0, position))
      setSliderPos(previous => previous == position ? previous : position)
    }
  }
  let release = e => {
    let frame = pointerFrame(e)
    let pointerId = ReactEvent.Pointer.pointerId(e)
    dragRef.current = None
    if hasPointerCapture(frame, pointerId) {
      releasePointerCapture(frame, pointerId)
    }
  }

  let knownDimensions = switch (width, height) {
  | (Some(w), Some(h)) => w > 0 && h > 0
  | _ => false
  }

  let compareFrameStyle = switch (width, height) {
  | (Some(imageWidth), Some(imageHeight)) if imageWidth > 0 && imageHeight > 0 =>
    let aspect = Int.toFloat(imageWidth) /. Int.toFloat(imageHeight)
    let maxWidth = Int.fromFloat(Math.max(1.0, Math.round(500.0 *. aspect)))
    ReactDOM.Style._dictToStyle(
      Dict.fromArray([
        ("aspectRatio", Int.toString(imageWidth) ++ " / " ++ Int.toString(imageHeight)),
        ("width", "min(100%, " ++ Int.toString(maxWidth) ++ "px)"),
      ]),
    )
  | _ => ReactDOM.Style._dictToStyle(Dict.fromArray([]))
  }

  let to1dp = (value: float): string => {
    let rounded = Math.round(value *. 10.0) /. 10.0
    Float.toString(rounded)
  }

  switch (originalUrl, compressedUrl) {
  | (None, None) | (Some(_), None) =>
    <div className="flex flex-col gap-4">
      <div className="flex min-h-16 items-center justify-between gap-4 text-sm tabular-nums text-neutral-400 sm:min-h-6">
        <p>{React.string(Display.formatSize(originalBytes))}</p>
        <p>{React.string("Processing image…")}</p>
      </div>
      <div className="rounded-lg border border-neutral-800 bg-neutral-900/50">
      <div
        style=compareFrameStyle
        className={"mx-auto flex flex-col items-center justify-center " ++ if knownDimensions {
          ""
        } else {
          "h-72 sm:h-96"
        }}
      >
        {switch compressionProgress {
        | Some(progress) => <ProgressStatus progress />
        | None =>
          <div className="flex flex-col items-center gap-4">
            <div
              className="w-10 h-10 border-4 border-neutral-700 border-t-blue-500 rounded-full animate-spin motion-reduce:animate-none"
            />
            <p className="text-neutral-400"> {React.string("Waiting to process image")} </p>
          </div>
        }}
      </div>
      </div>
      <div ariaHidden=true className="h-16" />
    </div>
  | (Some(orig), Some(comp)) =>
    let compressedSize = compressedBytes->Option.getOr(0)
    let savings = Display.calculateSavings(originalBytes, compressedSize)
    let optimized = Display.isAlreadyOptimized(originalBytes, compressedSize)

    <div className="flex flex-col gap-4">
      <div
        className="flex min-h-16 flex-wrap items-center justify-between gap-x-4 gap-y-2 text-sm tabular-nums text-neutral-400 sm:min-h-6"
      >
        <p>
          {React.string(
            Display.formatSize(originalBytes) ++ " → " ++ Display.formatSize(compressedSize),
          )}
        </p>
        {if optimized {
          <div className="flex items-center">
            <div className="flex items-center gap-2 text-blue-400">
              <svg
                className="w-4 h-4 flex-shrink-0"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="2"
                  d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              <span className="text-sm font-medium">
                {"No smaller output with current settings"->React.string}
              </span>
            </div>
          </div>
        } else {
          switch savings {
          | Some((percent, saved)) =>
            <div className="flex items-center">
              <div className={"text-base font-medium " ++ Display.savingsColor(percent)}>
                {React.string(
                  "Saved " ++
                  saved ++
                  " (" ++
                  to1dp(percent) ++
                  "%)" ++
                  switch compressionTime {
                  | Some(time) => " in " ++ Display.formatTime(time)
                  | None => ""
                  },
                )}
              </div>
            </div>
          | None => React.null
          }
        }}
      </div>
      <div className="relative rounded-lg overflow-hidden select-none border border-neutral-800">
        <div
          dataTestId="comparison-frame"
          onPointerDown={e => {
            if dragRef.current == None && ReactEvent.Pointer.button(e) == 0 {
              let touch = ReactEvent.Pointer.pointerType(e) == "touch"
              let pointerId = ReactEvent.Pointer.pointerId(e)
              dragRef.current = Some({pointerId, startPos: sliderPos, touch})
              setPointerCapture(pointerFrame(e), pointerId)
              if !touch {
                ReactEvent.Pointer.preventDefault(e)
                updatePosition(e)
              }
            }
          }}
          onPointerMove={e => {
            switch dragRef.current {
            | Some(drag) if drag.pointerId == ReactEvent.Pointer.pointerId(e) => updatePosition(e)
            | _ => ()
            }
          }}
          onPointerUp={e => {
            switch dragRef.current {
            | Some(drag) if drag.pointerId == ReactEvent.Pointer.pointerId(e) =>
              updatePosition(e)
              release(e)
            | _ => ()
            }
          }}
          onPointerCancel={e => {
            switch dragRef.current {
            | Some(drag) if drag.pointerId == ReactEvent.Pointer.pointerId(e) =>
              // Native vertical scrolling cancels touch; undo its provisional split.
              if drag.touch { setSliderPos(_ => drag.startPos) }
              release(e)
            | _ => ()
            }
          }}
          onLostPointerCapture={e => {
            switch dragRef.current {
            | Some(drag) if drag.pointerId == ReactEvent.Pointer.pointerId(e) => dragRef.current = None
            | _ => ()
            }
          }}
          className={"relative w-full mx-auto bg-neutral-900 touch-pan-y cursor-ew-resize select-none [-webkit-touch-callout:none] " ++ if knownDimensions {
            ""
          } else {
            "h-72 sm:h-96"
          }}
          style={compareFrameStyle}
        >
          /* Background: Compressed Image (Right Side) */
          <img
            draggable=false
            src=comp alt="Compressed" className="absolute inset-0 w-full h-full object-contain"
          />

          /* Overlay: Original Image (Left Side, Clipped) */
          {
            let clipPath =
              "polygon(0 0, " ++
              to1dp(sliderPos) ++
              "% 0, " ++
              to1dp(sliderPos) ++ "% 100%, 0 100%)"
            <div
              className="absolute inset-0 overflow-hidden"
              style={ReactDOM.Style._dictToStyle(Dict.fromArray([("clipPath", clipPath)]))}
            >
              <img draggable=false src=orig alt="Original" className="w-full h-full object-contain" />
            </div>
          }

          /* The frame and native keyboard range share the same split. */
          <div
            className="pointer-events-none absolute top-0 bottom-0 w-0.5 bg-white z-10 shadow-xl"
            style={ReactDOM.Style._dictToStyle(Dict.fromArray([("left", to1dp(sliderPos) ++ "%")]))}
            ariaHidden=true
          >
            <div className="absolute top-1/2 left-1/2 size-8 -translate-x-1/2 -translate-y-1/2 rounded-full bg-white text-neutral-900 shadow-md flex items-center justify-center">
              <svg className="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path d="M8 8l-4 4 4 4m8-8l4 4-4 4M4 12h16" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </div>
          </div>

          <div
            className="pointer-events-none absolute bottom-4 left-4 bg-black/50 text-white px-4 py-2 rounded text-xs font-medium"
          >
            {React.string("Original")}
          </div>
          <div
            className="pointer-events-none absolute bottom-4 right-4 bg-black/50 text-white px-4 py-2 rounded text-xs font-medium"
          >
            {React.string("Compressed")}
          </div>
        </div>
      </div>
      <label className="flex h-16 flex-col text-base leading-4 text-neutral-300">
        <span> {React.string("Compare original and compressed")} </span>
        <input
          type_="range"
          min="0"
          max="100"
          step=1.0
          value={Float.toString(Math.round(sliderPos))}
          ariaValuetext={Float.toString(Math.round(sliderPos)) ++
          "% original, " ++
          Float.toString(100.0 -. Math.round(sliderPos)) ++ "% compressed"}
          onChange={e => {
            let value =
              ReactEvent.Form.target(e)["value"]
              ->Option.flatMap(value => Int.fromString(value))
              ->Option.getOr(50)
              ->Int.toFloat
            setSliderPos(_ => Math.max(0.0, Math.min(100.0, value)))
          }}
          className="w-full min-h-12 cursor-ew-resize accent-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-300"
        />
      </label>
    </div>
  | _ => React.null
  }
}
