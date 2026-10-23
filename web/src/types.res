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
  | Avif
  | Gif
  | Bmp
  | Svg
  | Heic
  | Tiff
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
  | AutoSmallest
  | ForcePng
  | ForceJpeg
  | ForceWebp
  | ForceAvif

type queueItem = {
  id: string,
  generation: int,
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
  compressedMime: option<string>,
  compressedExtension: option<string>,
}

type compressionProgress = {
  isActive: bool,
  itemId: string,
  phase: string,
  progress: int,
  startTime: float,
  fileSize: int,
  lastRealProgress: int,
  lastRealUpdateTime: float,
  phaseStartTime: float,
  phaseTarget: int,
  predictable: bool,
}

type appState = {
  downloadError: option<string>,
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
  targetWidth: option<int>,
  targetHeight: option<int>,
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
  let mime = mime->String.toLowerCase
  let name = name->String.toLowerCase
  if mime == "image/png" || name->String.endsWith(".png") { Png }
  else if mime == "image/jpeg" || mime == "image/jpg" || name->String.endsWith(".jpg") || name->String.endsWith(".jpeg") { Jpeg }
  else if mime == "image/webp" || name->String.endsWith(".webp") { Webp }
  else if mime == "image/avif" || name->String.endsWith(".avif") { Avif }
  else if mime == "image/gif" || name->String.endsWith(".gif") { Gif }
  else if mime == "image/bmp" || name->String.endsWith(".bmp") { Bmp }
  else if mime == "image/svg+xml" || name->String.endsWith(".svg") { Svg }
  else if mime == "image/heic" || mime == "image/heif" || name->String.endsWith(".heic") || name->String.endsWith(".heif") { Heic }
  else if mime == "image/tiff" || name->String.endsWith(".tif") || name->String.endsWith(".tiff") { Tiff }
  else { Unknown }
}

let kindToInputFormat = (kind: fileKind): string => {
  switch kind {
  | Png => "png"
  | Jpeg => "jpeg"
  | Webp => "webp"
  | Avif => "avif"
  | Gif => "gif"
  | Bmp => "bmp"
  | Svg => "svg"
  | Heic => "heic"
  | Tiff => "tiff"
  | Unknown => "unknown"
  }
}

let kindLabel = (kind: fileKind): string => {
  switch kind {
  | Png => "PNG" | Jpeg => "JPEG" | Webp => "WebP" | Avif => "AVIF"
  | Gif => "GIF" | Bmp => "BMP" | Svg => "SVG" | Heic => "HEIC" | Tiff => "TIFF" | Unknown => "Unknown"
  }
}

let outputFormatToChoice = (fmt: outputFormat): string => {
  switch fmt {
  | SameAsInput => "same"
  | AutoSmallest => "auto"
  | ForcePng => "png"
  | ForceJpeg => "jpeg"
  | ForceWebp => "webp"
  | ForceAvif => "avif"
  }
}

let presetToKey = (preset: preset): string => {
  switch preset {
  | Ultra => "ultra"
  | Smaller => "smaller"
  | Balanced => "balanced"
  | Faster => "faster"
  }
}

let mimeForResolvedFormat = (format: string): string => {
  switch format {
  | "jpeg" => "image/jpeg"
  | "webp" => "image/webp"
  | "avif" => "image/avif"
  | _ => "image/png"
  }
}

let extensionForResolvedFormat = (format: string): string => {
  switch format {
  | "jpeg" => ".jpg"
  | "webp" => ".webp"
  | "avif" => ".avif"
  | _ => ".png"
  }
}

let acceptedImageTypes = "image/png,image/jpeg,image/webp,image/avif,image/gif,image/bmp,image/svg+xml,image/heic,image/heif,image/tiff,.png,.jpg,.jpeg,.webp,.avif,.gif,.bmp,.svg,.heic,.heif,.tif,.tiff"
