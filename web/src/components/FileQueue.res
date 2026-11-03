let statusText = (status: Types.fileStatus): string => {
  switch status {
  | Pending => "Pending"
  | Decoding => "Decoding..."
  | Compressing => "Compressing..."
  | Done => "Done"
  | Error(msg) => "Error: " ++ msg
  }
}

let kindText = Types.kindLabel

let extensionText = (extension: string): string => {
  switch extension {
  | ".jpg" | ".jpeg" => "JPEG"
  | ".webp" => "WebP"
  | ".avif" => "AVIF"
  | ".png" => "PNG"
  | _ => "Unknown"
  }
}

@react.component
let make = (
  ~headerAction: React.element=React.null,
  ~items: array<Types.queueItem>,
  ~selectedId: option<string>,
  ~compressionProgress: option<Types.compressionProgress>,
  ~activeCompressions: array<(string, Types.compressionProgress)>,
  ~onSelect: string => unit,
  ~onRemove: string => unit,
  ~onClearAll: unit => unit,
) => {
  if items->Array.length == 0 {
    React.null
  } else {
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-4 mb-6">
        <h2 className="text-base font-medium text-neutral-400">
          {React.string("Files (" ++ Int.toString(items->Array.length) ++ ")")}
        </h2>
        <div className="flex items-center gap-4">
          {headerAction}
          <button
            type_="button"
            onClick={_ => onClearAll()}
            className="min-h-12 min-w-12 text-sm text-neutral-400 hover:text-neutral-300 transition-colors motion-reduce:transition-none"
          >
            {React.string("Clear all")}
          </button>
        </div>
      </div>
      {items
      ->Array.map((item: Types.queueItem) => {
        let isSelected = switch selectedId {
        | Some(id) => id == item.Types.id
        | None => false
        }
        <div
          key=item.Types.id
          className={"flex min-w-0 items-center gap-2 rounded-lg border p-2 transition-colors motion-reduce:transition-none " ++ if (
            isSelected
          ) {
            "border-neutral-600 bg-neutral-900"
          } else {
            "border-neutral-800 hover:border-neutral-700"
          }}
        >
          <button
            type_="button"
            ariaCurrent={isSelected ? #"true" : #"false"}
            onClick={_ => onSelect(item.id)}
            className="min-h-12 min-w-0 flex-1 rounded p-2 text-left focus-visible:outline focus-visible:outline-2 focus-visible:outline-neutral-300"
          >
            <p className="text-base font-medium text-neutral-200 truncate">
              {React.string(Types.Web.File.name(item.file))}
            </p>
            <div className="flex flex-wrap items-center gap-x-2 gap-y-2 mt-2 tabular-nums">
              <span className="text-sm text-neutral-400">
                {React.string(kindText(item.kind))}
              </span>
              {switch item.compressedExtension {
              | Some(extension) =>
                <span className="text-sm text-neutral-400">
                  {React.string("→ " ++ extensionText(extension))}
                </span>
              | None => React.null
              }}
              <span className="text-sm text-neutral-400">
                {React.string("· " ++ Display.formatSize(item.originalBytes))}
              </span>
              {switch item.status {
              | Decoding | Compressing =>
                let activeProgress = switch compressionProgress {
                | Some(progress) if progress.itemId == item.id => Some(progress)
                | _ =>
                  activeCompressions
                  ->Array.find(((pid, _)) => pid == item.Types.id)
                  ->Option.map(((_, p)) => p)
                }
                switch activeProgress {
                | Some(progress) => <ProgressStatus progress compact=true />
                | None =>
                  <span className="text-sm text-blue-400 animate-pulse motion-reduce:animate-none">
                    {React.string(statusText(item.status))}
                  </span>
                }
              | Done =>
                switch item.compressedBytes {
                | Some(bytes) =>
                  let originalSize = item.originalBytes
                  if Display.hasSameSize(originalSize, bytes) {
                    <span
                      className="text-sm font-medium text-blue-400" dataTestId="file-item-optimized"
                    >
                      {React.string("Same file size with these settings")}
                    </span>
                  } else {
                    let saved = originalSize - bytes
                    let percent = Int.toFloat(saved) /. Int.toFloat(originalSize) *. 100.0
                    <span
                      className={"text-sm font-medium " ++ Display.savingsColor(percent)}
                      dataTestId="file-item-done"
                    >
                      {React.string(
                        "→ " ++
                        Display.formatSize(bytes) ++
                        " (" ++
                        if Math.abs(percent) > 0.0 && Math.abs(percent) < 0.1 {
                          Display.formatPercent(Math.abs(percent)) ++ "% " ++ if saved > 0 { "smaller" } else { "larger" }
                        } else {
                          (if saved >= 0 { "-" } else { "+" }) ++ Display.formatPercent(Math.abs(percent)) ++ "%"
                        } ++ ")",
                      )}
                    </span>
                  }
                | None => React.null
                }
              | Error(msg) =>
                <span className="text-sm text-red-400 [overflow-wrap:anywhere]">
                  {React.string(msg)}
                </span>
              | Pending =>
                <span className="text-sm text-neutral-400">
                  {React.string(statusText(item.status))}
                </span>
              }}
            </div>
          </button>
          <div className="shrink-0 flex items-center gap-2">
            <span ariaHidden=true className="pointer-events-none flex size-5 shrink-0 items-center justify-center">
              {switch item.status {
              | Done =>
                <svg className="size-5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M5 13l4 4L19 7" />
                </svg>
              | Decoding | Compressing =>
                <span className="size-5 rounded-full border-2 border-blue-400 border-t-transparent animate-spin motion-reduce:animate-none" />
              | Error(_) =>
                <svg className="size-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              | Pending => React.null
              }}
            </span>
            <button
              type_="button"
              onClick={e => {
                ReactEvent.Mouse.stopPropagation(e)
                onRemove(item.id)
              }}
              className="inline-flex size-12 shrink-0 items-center justify-center rounded-full text-neutral-400 hover:bg-red-500/10 hover:text-red-400 active:scale-[0.96] transition-[color,background-color,transform] duration-150 ease-out motion-reduce:transition-none focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-300"
              title="Remove image"
              ariaLabel="Remove image"
            >
              <svg ariaHidden=true className="pointer-events-none size-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="2"
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        </div>
      })
      ->React.array}
    </div>
  }
}
