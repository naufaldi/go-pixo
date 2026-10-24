import {afterEach, expect, it, vi} from 'vitest';
const decoder = vi.hoisted(() => ({decodeTiff: vi.fn()}));
vi.mock('./codecs', () => ({...decoder, decodeAvif: vi.fn(), decodeHeic: vi.fn()}));
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks() });
it.each(['success','failure'])('bundled preview borrows its owned pixel view and releases canvas after %s', async outcome => {
  vi.resetModules();
  const pixels = new Uint8Array([9,8,1,2,3,4,7,6]).subarray(2,6);
  decoder.decodeTiff.mockResolvedValue({pixels,width:1,height:1});
  let canvas: any, view: Uint8ClampedArray | undefined;
  vi.stubGlobal('ImageData', class { constructor(data: Uint8ClampedArray) { view = data } });
  vi.stubGlobal('OffscreenCanvas', class {
    width: number; height: number;
    constructor(width: number, height: number) { this.width = width; this.height = height; canvas = this }
    getContext() { return {putImageData() {}} }
    async convertToBlob() {
      if (outcome === 'failure') throw new Error('Preview failed');
      return {arrayBuffer: async () => new ArrayBuffer(1)};
    }
  });
  const messages: any[] = [];
  const scope: any = {postMessage: (message: unknown) => messages.push(message)};
  vi.stubGlobal('self', scope);
  await import('./codecsDecode.worker');
  const bytes = new Uint8Array([73,73,42,0,8,0,0,0,2,0,0,1,3,0,1,0,0,0,1,0,0,0,1,1,3,0,1,0,0,0,1,0,0,0,0,0,0,0]);
  await scope.onmessage({data: {bytes: bytes.buffer}});
  expect(view!.buffer).toBe(pixels.buffer);
  expect(view!.byteOffset).toBe(2);
  expect(Array.from(view!)).toEqual([1,2,3,4]);
  expect([canvas.width,canvas.height]).toEqual([0,0]);
  expect(outcome === 'success' ? messages[0].pixels : messages[0].error).toEqual(outcome === 'success' ? pixels : 'Preview failed');
});
