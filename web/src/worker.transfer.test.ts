import { afterEach, expect, it, vi } from 'vitest'

const { compress } = vi.hoisted(() => ({compress: vi.fn()}))
vi.mock('./interop/codecsCompression', () => ({compressWithCandidates: compress}))
vi.mock('./interop/codecs', () => ({encodeAvif: vi.fn(), encodeWebp: vi.fn()}))

afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks() })

it.each(['encoded', 'original'])('transfers exclusively owned %s output, preserving the view boundaries', async selection => {
  vi.resetModules()
  const received: any[] = []
  const scope: any = {
    onmessage: null,
    postMessage: (message: unknown, transfer: Transferable[] = []) => received.push(structuredClone(message, {transfer})),
    Go: class { importObject = {}; run() { scope.goWasmInit() } },
  }
  vi.stubGlobal('self', scope)
  vi.stubGlobal('fetch', vi.fn(async () => ({arrayBuffer: async () => new ArrayBuffer(0)})))
  vi.spyOn(WebAssembly, 'instantiate').mockResolvedValue({instance: {}} as any)
  const storage = new Uint8Array([9, 1, 2, 3, 8])
  const output = storage.subarray(1, 4)
  compress.mockImplementation(async request => ({compressedBytes: selection === 'original' ? request.originalFileBytes : output, outputFormat: 'png'}))
  await import('./worker')
  await scope.onmessage({data: {type: 'init'}})
  await scope.onmessage({data: {
    type: 'compress', id: 'transfer', pixels: new Uint8Array(4), width: 1, height: 1, colorType: 6,
    format: 'png', preset: 2, lossy: true, originalFileBytes: output,
  }})
  const result = received.find(message => message.type === 'compressed')
  expect(result).toBeDefined()
  expect(Array.from(result.compressedBytes)).toEqual([1, 2, 3])
  expect(result.compressedBytes.byteOffset).toBe(1)
  expect(result.compressedBytesCount).toBe(3)
  expect(result.outputFormat).toBe('png')
  expect(storage.byteLength).toBe(0) // Transfer detaches the sender's buffer.
})
