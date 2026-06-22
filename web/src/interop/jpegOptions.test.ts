import { describe, expect, it } from 'vitest'
import { resolveJpegEncodeOptions } from './jpegOptions'

describe('jpegOptions', () => {
  it('normalizes app ultra preset to the strongest JPEG preset', () => {
    const options = resolveJpegEncodeOptions({
      inputFormat: 'png',
      outputFormat: 'jpeg',
      width: 64,
      height: 64,
      preset: 4,
      trellis: false,
      optimizeHuffman: false,
      progressive: true,
    })

    expect(options.preset).toBe(2)
  })

  it('disables trellis for PNG to JPEG conversion', () => {
    const options = resolveJpegEncodeOptions({
      inputFormat: 'png',
      outputFormat: 'jpeg',
      width: 800,
      height: 600,
      preset: 4,
      trellis: true,
      optimizeHuffman: true,
      progressive: true,
    })

    expect(options.trellis).toBe(false)
    expect(options.optimizeHuffman).toBe(false)
    expect(options.progressive).toBe(false)
  })

  it('disables optimized Huffman for browser-safe JPEG output', () => {
    const options = resolveJpegEncodeOptions({
      inputFormat: 'jpeg',
      outputFormat: 'jpeg',
      width: 32,
      height: 32,
      preset: 2,
      trellis: false,
      optimizeHuffman: true,
      progressive: true,
    })

    expect(options.trellis).toBe(false)
    expect(options.optimizeHuffman).toBe(false)
  })

  it('never combines trellis with optimized Huffman tables', () => {
    const options = resolveJpegEncodeOptions({
      inputFormat: 'jpeg',
      outputFormat: 'jpeg',
      width: 32,
      height: 32,
      preset: 2,
      trellis: true,
      optimizeHuffman: true,
      progressive: true,
    })

    expect(options.trellis).toBe(true)
    expect(options.optimizeHuffman).toBe(false)
  })
})
