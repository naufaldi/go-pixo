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
  | Webp
  | Unknown

type preset =
  | Ultra
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

type outputFormat =
  | SameAsInput
  | ForcePng
  | ForceJpeg
  | ForceWebp

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
  compressionTime: option<int>,
}

type compressionProgress = {
  isActive: bool,
  itemId: string,
  phase: string,
  progress: int,
  startTime: float,
  fileSize: int,
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
  ditherStrength: float,
  qualityTarget: int,
  zopfliIterations: int,
  progressive: bool,
  subsampling: string,
  trellis: bool,
  optimizeHuffman: bool,
  compressionProgress: option<compressionProgress>,
  compressionTime: option<int>,
  outputFormat: outputFormat,
  activeCompressions: array<(string, compressionProgress)>,
  processingAll: bool,
}

let presetToInt = (preset: preset): int => {
  switch preset {
  | Ultra => 4
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
  quantization == Lossless
}

let fileKindFromMime = (mime: string, name: string): fileKind => {
  if mime->String.includes("png") || name->String.endsWith(".png") {
    Png
  } else if mime->String.includes("jpeg") || mime->String.includes("jpg") || name->String.endsWith(".jpg") || name->String.endsWith(".jpeg") {
    Jpeg
  } else if mime->String.includes("webp") || name->String.endsWith(".webp") {
    Webp
  } else {
    Unknown
  }
}
