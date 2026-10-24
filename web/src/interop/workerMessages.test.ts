import { describe, expect, it } from 'vitest'
import {
  buildCompressMessage,
  parseWorkerMessage,
  type CompressPayload,
} from './workerMessages'

const payload = {
  id: 'item-1',
  pixels: new Uint8Array([1, 2, 3, 4]),
  width: 1,
  height: 1,
  colorType: 6,
  format: 'png',
  outputFormat: 'same',
  preset: 1,
  lossy: false,
  maxColors: 0,
  dithering: false,
  ditherStrength: 0.5,
  qualityTarget: 85,
  zopfliIterations: 5,
  progressive: true,
  trellis: true,
  subsampling: '420',
  optimizeHuffman: true,
  originalFileBytes: new Uint8Array([9, 8, 7]),
} satisfies CompressPayload

describe('workerMessages', () => {
  it.each(['typed', 'view'])('keeps the received %s output buffer and its subarray boundaries', kind => {
    const buffer = new Uint8Array([9, 1, 2, 3, 8]).buffer
    const bytes = kind === 'typed' ? new Uint8Array(buffer, 1, 3) : new DataView(buffer, 1, 3)
    const parsed = parseWorkerMessage({type: 'compressed', id: 'item', compressedBytes: bytes, outputFormat: 'png'})
    expect(parsed.type).toBe('compressed')
    if (parsed.type !== 'compressed') throw new Error('Expected compressed response')
    expect(parsed.compressedBytes.buffer).toBe(buffer)
    expect(Array.from(parsed.compressedBytes)).toEqual([1, 2, 3])
    expect(parsed.compressedBytes.byteOffset).toBe(1)
    if (kind === 'typed') expect(parsed.compressedBytes).toBe(bytes)
  })
  it('builds a compress message with the expected type and payload fields', () => {
    expect(buildCompressMessage(payload)).toEqual({
      type: 'compress',
      ...payload,
    })
  })

  it('parses ready, progress, compressed, and error worker messages', () => {
    expect(parseWorkerMessage({ type: 'ready' })).toEqual({ type: 'ready' })
    expect(
      parseWorkerMessage({
        type: 'progress',
        id: 'item-1',
        phase: 'deflate',
        progress: 42.4,
        predictable: true,
        phaseTarget: 80.2,
      }),
    ).toEqual({
      type: 'progress',
      id: 'item-1',
      phase: 'deflate',
      progress: 42,
      predictable: true,
      phaseTarget: 80,
    })
    expect(
      parseWorkerMessage({
        type: 'compressed',
        id: 'item-1',
        compressedBytes: new Uint8Array([1, 2]),
        outputFormat: 'jpeg',
      }),
    ).toEqual({
      type: 'compressed',
      id: 'item-1',
      compressedBytes: new Uint8Array([1, 2]),
      outputFormat: 'jpeg',
    })
    expect(parseWorkerMessage({ type: 'error', id: 'item-1', error: 'boom' })).toEqual({
      type: 'error',
      id: 'item-1',
      error: 'boom',
    })
  })

  it('returns unknown for malformed messages', () => {
    expect(parseWorkerMessage(null)).toEqual({ type: 'unknown' })
    expect(parseWorkerMessage({ type: 'progress', id: 123 })).toEqual({ type: 'unknown' })
    expect(parseWorkerMessage({ type: 'compressed' })).toEqual({ type: 'unknown' })
  })
})
