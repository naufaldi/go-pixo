open Types

type decodeResult = {
  pixels: array<int>,
  width: int,
  height: int,
  colorType: int,
  previewUrl: string,
  originalFileBytes: array<int>,
}

@module("../interop/imageDecode")
external decodeFile: Web.File.t => Promise.t<decodeResult> = "decodeFile"

