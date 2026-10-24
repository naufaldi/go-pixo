open Types

type decodeResult = {
  pixels: option<array<int>>,
  width: int,
  height: int,
  colorType: int,
  inputFormat: string,
  previewUrl: string,
  originalFileBytes: array<int>,
}

type decodeOptions = {
  lossy: bool,
  outputFormat: string,
  targetWidth: option<int>,
  targetHeight: option<int>,
  admit: int => Promise.t<unit>,
}

@module("../interop/imageDecode")
external decodeFile: (Web.File.t, Download.abortSignal, decodeOptions) => Promise.t<decodeResult> = "decodeFile"

