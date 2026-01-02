open React
open Types

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
let make = (~items, ~selectedId, ~onSelect) => {
  if items->Array.length == 0 {
    React.null
  } else {
    <div className="mt-8 space-y-2">
      <h3 className="text-sm font-medium text-neutral-400 mb-4">
        {React.string("Files")}
      </h3>
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
             "p-3 border border-neutral-600 rounded-lg cursor-pointer bg-neutral-900"
           } else {
             "p-3 border border-neutral-800 rounded-lg cursor-pointer hover:border-neutral-700 hover:bg-neutral-900/50 transition-colors"
           }}>
           <div className="flex items-center justify-between">
             <div className="flex-1 min-w-0">
               <p className="text-sm font-medium text-neutral-200 truncate">
                 {React.string(Types.Web.File.name(item.file))}
               </p>
               <div className="flex items-center gap-3 mt-1">
                 <span className="text-xs text-neutral-500">
                   {React.string(kindText(item.kind))}
                 </span>
                 <span className="text-xs text-neutral-500">
                   {React.string(statusText(item.status))}
                 </span>
                 {switch item.compressedBytes {
                 | Some(bytes) =>
                   let originalSize = item.originalBytes
                   let saved = originalSize - bytes
                   let percent = float_of_int(saved) /. float_of_int(originalSize) *. 100.0
                   let percentStr = {
                    let rounded = Js.Math.round(percent *. 10.0) /. 10.0
                    Js.Float.toString(rounded)
                  }
                   let text = Int.toString(bytes) ++ " bytes (" ++ percentStr ++ "% smaller)"
                   <span className="text-xs text-green-400">
                     {React.string(text)}
                   </span>
                 | None => React.null
                 }}
               </div>
             </div>
           </div>
         </div>
       })
       ->React.array}
    </div>
  }
}
