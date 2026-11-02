import { describe, expect, it } from 'vitest'
import { resolveJpegEncodeOptions, type JpegEncodeOptionInput } from './jpegOptions'

const input = (overrides: Partial<JpegEncodeOptionInput>): JpegEncodeOptionInput => ({
  inputFormat: 'jpeg', outputFormat: 'jpeg', width: 64, height: 64, preset: 1,
  trellis: true, optimizeHuffman: true, progressive: true, ...overrides,
})

describe('jpegOptions', () => {
  it('normalizes app ultra preset to the strongest JPEG preset', () => {
    expect(resolveJpegEncodeOptions(input({ preset: 4 })).preset).toBe(2)
  })

  it('disables only trellis for cross-format conversion', () => {
    expect(resolveJpegEncodeOptions(input({ inputFormat: 'png' }))).toMatchObject({
      trellis: false, progressive: true, optimizeHuffman: true,
    })
  })

  it('passes entropy options through, alone or with trellis', () => {
    expect(resolveJpegEncodeOptions(input({}))).toMatchObject({ trellis: true, progressive: true, optimizeHuffman: true })
    expect(resolveJpegEncodeOptions(input({ trellis: false, progressive: false, optimizeHuffman: false })))
      .toMatchObject({ trellis: false, progressive: false, optimizeHuffman: false })
  })
})
