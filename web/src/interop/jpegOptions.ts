import type { InputFormat, ResolvedFormat } from './compressionSettings'

export type JpegEncodeOptionInput = {
  inputFormat: InputFormat
  outputFormat: ResolvedFormat
  width: number
  height: number
  preset: number
  progressive: boolean
  trellis: boolean
  optimizeHuffman: boolean
}

export type JpegEncodeOptions = {
  preset: number
  progressive: boolean
  trellis: boolean
  optimizeHuffman: boolean
}

export function resolveJpegEncodeOptions(input: JpegEncodeOptionInput): JpegEncodeOptions {
  const crossFormat = input.inputFormat !== 'jpeg' || input.outputFormat !== 'jpeg'
  const preset = input.preset === 4 ? 2 : input.preset
  const trellis = crossFormat ? false : input.trellis

  return {
    preset,
    progressive: crossFormat ? false : input.progressive,
    trellis,
    optimizeHuffman: false,
  }
}
