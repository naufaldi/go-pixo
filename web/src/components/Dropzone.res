open React

@react.component
let make = (~dragActive, ~onDragEnter, ~onDragOver, ~onDragLeave, ~onDrop, ~onFileSelect) => {
  let fileInputRef = React.useRef(Nullable.null)

  let handleClick = _ => {
    switch fileInputRef.current->Nullable.toOption {
    | Some(input) => {
        let _ = input
        %raw("input.click()")
      }
    | None => ()
    }
  }

  let handleKeyDown = (e: ReactEvent.Keyboard.t) => {
    let key = ReactEvent.Keyboard.key(e)
    if key == "Enter" || key == " " {
      ReactEvent.Keyboard.preventDefault(e)
      handleClick(e)
    }
  }

  let handleFileChange = (e: ReactEvent.Form.t) => {
    let files = ReactEvent.Form.target(e)["files"]
    if files->Nullable.isNullable == false {
      let fileArray = %raw("Array.from(files)")
      onFileSelect(fileArray)
      let _ = %raw("e.target.value = ''")
    }
  }

  <div
    className={if dragActive {
      "border-2 border-dashed border-neutral-400 rounded-lg px-4 py-8 sm:py-16 text-center transition-colors motion-reduce:transition-none bg-neutral-900"
    } else {
      "border-2 border-dashed border-neutral-700 rounded-lg px-4 py-8 sm:py-16 text-center transition-colors motion-reduce:transition-none hover:border-neutral-500 cursor-pointer"
    }}
    onDragEnter=onDragEnter
    onDragOver=onDragOver
    onDragLeave=onDragLeave
    onDrop=onDrop
    onClick=handleClick
    onKeyDown=handleKeyDown
    tabIndex=0
    role="button"
    ariaLabel="Drop images here, or click to select files"
  >
    <input
      ref={ReactDOM.Ref.domRef(fileInputRef)}
      type_="file"
      multiple=true
      accept=Types.acceptedImageTypes
      onChange=handleFileChange
      className="hidden"
    />
    <div className="flex flex-col items-center gap-4">
      <div className="w-16 h-16 flex items-center justify-center">
        <svg
          className="w-full h-full text-neutral-400"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth="2"
            d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
          />
        </svg>
      </div>
      <p className="text-lg font-medium text-neutral-200"> {React.string("Drop images here")} </p>
      <p className="text-sm text-neutral-400"> {React.string("or paste from clipboard")} </p>
      <span
        className="min-h-12 min-w-12 inline-flex items-center justify-center mt-0 px-6 py-2 bg-white text-neutral-900 rounded-md font-medium hover:bg-neutral-100 transition-colors motion-reduce:transition-none"
      >
        {React.string("Select Files")}
      </span>
      <p className="text-sm text-neutral-400"> {React.string("Ctrl+V or ⌘V to paste")} </p>
      <p className="max-w-md text-sm text-neutral-400">
        {React.string(
          "PNG, JPEG, WebP, AVIF, TIFF, GIF, BMP and SVG. Still images only; up to 24 MP and 64 MiB.",
        )}
      </p>
      <p className="text-sm text-neutral-400">
        {React.string("HEIC/HEIF is temporarily unavailable.")}
      </p>
    </div>
  </div>
}
