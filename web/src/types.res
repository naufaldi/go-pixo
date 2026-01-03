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

type quantizationLevel =
  | Lossless
  | Colors256
  | Colors128
  | Colors64
  | Colors32
  | Colors16
  | Colors8

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
  quantization: quantizationLevel,
  dithering: bool,
}

let presetToInt = (preset: preset): int => {
  switch preset {
  | Smaller => 0
  | Balanced => 1
  | Faster => 2
  }
}

let quantizationToInt = (quantization: quantizationLevel): int => {
  switch quantization {
  | Lossless => 0
  | Colors256 => 256
  | Colors128 => 128
  | Colors64 => 64
  | Colors32 => 32
  | Colors16 => 16
  | Colors8 => 8
  }
}

let intToQuantization = (value: int): quantizationLevel => {
  switch value {
  | 0 => Lossless
  | 256 => Colors256
  | 128 => Colors128
  | 64 => Colors64
  | 32 => Colors32
  | 16 => Colors16
  | 8 => Colors8
  | _ => Lossless
  }
}

let isLossless = (quantization: quantizationLevel): bool => {
  switch quantization {
  | Lossless => true
  | _ => false
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
