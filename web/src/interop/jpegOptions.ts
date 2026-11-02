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

// Progressive scans and optimized Huffman tables are a lossless entropy pass
// over the encoder's baseline output, so they are safe for every decoder and
// every source format. Trellis stays limited to JPEG-to-JPEG work.
export function resolveJpegEncodeOptions(input: JpegEncodeOptionInput): JpegEncodeOptions {
  const crossFormat = input.inputFormat !== 'jpeg' || input.outputFormat !== 'jpeg'
  return {
    preset: input.preset === 4 ? 2 : input.preset,
    progressive: input.progressive,
    trellis: crossFormat ? false : input.trellis,
    optimizeHuffman: input.optimizeHuffman,
  }
}
