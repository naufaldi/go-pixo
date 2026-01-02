type file

module Web = {
  module File = {
    type t = file
    @get external name: t => string = "name"
    @get external size: t => int = "size"
    @get external type_: t => string = "type"
  }
}

type fileStatus =
  | Pending
  | Decoding
  | Compressing
  | Done
  | Error(string)

type fileKind =
  | Png
  | Jpeg
  | Unknown

type preset =
  | Smaller
  | Balanced
  | Faster

type queueItem = {
  id: string,
  file: Web.File.t,
  kind: fileKind,
  status: fileStatus,
  originalUrl: option<string>,
  compressedUrl: option<string>,
  originalBytes: int,
  compressedBytes: option<int>,
  width: option<int>,
  height: option<int>,
}

type appState = {
  wasmReady: bool,
  dragActive: bool,
  items: array<queueItem>,
  selectedId: option<string>,
  preset: preset,
  lossless: bool,
}

let presetToInt = (preset: preset): int => {
  switch preset {
  | Smaller => 0
  | Balanced => 1
  | Faster => 2
  }
}

let fileKindFromMime = (mime: string, name: string): fileKind => {
  if mime->String.includes("png") || name->String.endsWith(".png") {
    Png
  } else if mime->String.includes("jpeg") || mime->String.includes("jpg") || name->String.endsWith(".jpg") || name->String.endsWith(".jpeg") {
    Jpeg
  } else {
    Unknown
  }
}
