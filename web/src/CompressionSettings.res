type derivedSettings = {
  presetInt: int,
  qualityTarget: int,
  maxColors: int,
  zopfliIterations: int,
  progressive: bool,
  trellis: bool,
  optimizeHuffman: bool,
  subsampling: string,
  dithering: bool,
  ditherStrength: float,
}

@module("./interop/compressionSettings.ts")
external resolveOutputFormat: (string, string) => string = "resolveOutputFormat"

@module("./interop/compressionSettings.ts")
external mimeForFormat: string => string = "mimeForFormat"

@module("./interop/compressionSettings.ts")
external extensionForFormat: string => string = "extensionForFormat"

@module("./interop/compressionSettings.ts")
external buildCompressedFilename: (string, string) => string = "buildCompressedFilename"

@module("./interop/compressionSettings.ts")
external derivePresetSettings: (string, bool) => derivedSettings = "derivePresetSettings"

let resolveForItem = (kind: Types.fileKind, outputFormat: Types.outputFormat): string => {
  resolveOutputFormat(Types.kindToInputFormat(kind), Types.outputFormatToChoice(outputFormat))
}

let settingsForState = (state: Types.appState): derivedSettings => {
  derivePresetSettings(Types.presetToKey(state.preset), state.lossless)
}
