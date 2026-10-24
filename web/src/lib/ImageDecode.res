open Types

type decodeResult = {
  pixels: array<int>,
  width: int,
  height: int,
  colorType: int,
  inputFormat: string,
  previewUrl: string,
  originalFileBytes: array<int>,
}

@module("../interop/imageDecode")
external decodeFile: (Web.File.t, Download.abortSignal) => Promise.t<decodeResult> = "decodeFile"

