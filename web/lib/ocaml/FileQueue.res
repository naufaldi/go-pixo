open React
open Types

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

let savingsColor = (percent: float): string => {
  if percent >= 30.0 {
    "text-green-400"
  } else if percent >= 10.0 {
    "text-yellow-400"
  } else {
    "text-gray-400"
  }
}

let statusText = (status: fileStatus): string => {
  switch status {
  | Pending => "Pending"
  | Decoding => "Decoding..."
  | Compressing => "Compressing..."
  | Done => "Done"
  | Error(msg) => "Error: " ++ msg
  }
}

let kindText = (kind: fileKind): string => {
  switch kind {
  | Png => "PNG"
  | Jpeg => "JPEG"
  | Unknown => "Unknown"
  }
}

@react.component
let make = (~items, ~selectedId, ~onSelect, ~onRemove, ~onClearAll) => {
  if items->Array.length == 0 {
    React.null
  } else {
    <div className="mt-8 space-y-2">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-medium text-neutral-400">
          {React.string("Files (" ++ Int.toString(items->Array.length) ++ ")")}
        </h3>
        <button
          onClick={_ => onClearAll()}
          className="text-xs text-neutral-500 hover:text-neutral-300 transition-colors">
          {React.string("Clear all")}
        </button>
      </div>
      {items
       ->Array.map(item => {
         let isSelected = switch selectedId {
         | Some(id) => id == item.id
         | None => false
         }
         <div
           key=item.id
           onClick={_ => onSelect(item.id)}
           className={if isSelected {
             "p-4 border border-neutral-600 rounded-lg cursor-pointer bg-neutral-900 transition-colors group relative"
           } else {
             "p-4 border border-neutral-800 rounded-lg cursor-pointer hover:border-neutral-700 hover:bg-neutral-900/50 transition-colors group relative"
           }}
           role="button"
           tabIndex=0
           onKeyDown={e => {
             let key = ReactEvent.Keyboard.key(e)
             if key == "Enter" || key == " " {
               ReactEvent.Keyboard.preventDefault(e)
               onSelect(item.id)
             }
           }}>
           <div className="flex items-center justify-between">
             <div className="flex-1 min-w-0">
               <p className="text-sm font-medium text-neutral-200 truncate pr-8">
                 {React.string(Types.Web.File.name(item.file))}
               </p>
               <div className="flex items-center gap-3 mt-2">
                 <span className="text-xs text-neutral-500">
                   {React.string(kindText(item.kind))}
                 </span>
                 <span className="text-xs text-neutral-400">
                   {React.string(formatSize(item.originalBytes))}
                 </span>
                 {switch item.status {
                 | Compressing =>
                   <span className="text-xs text-blue-400 animate-pulse">
                     {React.string("Compressing...")}
                   </span>
                 | Done =>
                   switch item.compressedBytes {
                   | Some(bytes) =>
                     let originalSize = item.originalBytes
                     let saved = originalSize - bytes
                    let percent = Int.toFloat(saved) /. Int.toFloat(originalSize) *. 100.0
                    <span className={"text-xs font-medium " ++ savingsColor(percent)}>
                      {React.string(
                        formatSize(bytes) ++ " (" ++ savingsColor(percent) ++ " -" ++
                        Float.toString(Math.round(percent *. 10.0) /. 10.0) ++ "%)",
                      )}
                    </span>
                   | None => React.null
                   }
                 | Types.Error(msg) =>
                   <span className="text-xs text-red-400">
                     {React.string(msg)}
                   </span>
                 | _ =>
                   <span className="text-xs text-neutral-500">
                     {React.string(statusText(item.status))}
                   </span>
                 }}
               </div>
             </div>
             <div className="ml-4 flex items-center gap-2">
               {switch item.status {
               | Done =>
                 <div className="w-6 h-6 rounded-full bg-green-500/20 flex items-center justify-center">
                   <svg className="w-4 h-4 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                     <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M5 13l4 4L19 7" />
                   </svg>
                 </div>
               | Compressing =>
                 <div className="w-6 h-6 rounded-full border-2 border-blue-400 border-t-transparent animate-spin"></div>
               | Types.Error(_) =>
                 <div className="w-6 h-6 rounded-full bg-red-500/20 flex items-center justify-center">
                   <svg className="w-4 h-4 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                     <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                   </svg>
                 </div>
               | _ => React.null
               }}
               <button
                 onClick={e => {
                   ReactEvent.Mouse.stopPropagation(e)
                   onRemove(item.id)
                 }}
                 className="p-2 text-neutral-400 hover:text-red-400 transition-all rounded-full hover:bg-red-500/10 border border-neutral-800 hover:border-red-500/20"
                 title="Remove image"
                 ariaLabel="Remove image">
                 <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                   <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                 </svg>
               </button>
             </div>
           </div>
         </div>
       })
       ->React.array}
    </div>
  }
}
