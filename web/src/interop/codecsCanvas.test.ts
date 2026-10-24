import {afterEach, expect, it, vi} from 'vitest';
import {encodeWebp} from './codecs';
const webp = new Uint8Array([82,73,70,70,18,0,0,0,87,69,66,80,86,80,56,76,5,0,0,0,47,0,0,0,0,0]);
afterEach(() => vi.unstubAllGlobals());
it.each(['success', 'encode failure', 'context failure'])('WebP borrows the exact pixel view and releases its canvas after %s', async outcome => {
  const storage = new Uint8Array([9,8,1,2,3,4,7,6]);
  const pixels = storage.subarray(2,6);
  let imagePixels: Uint8ClampedArray | undefined;
  let canvas: any;
  vi.stubGlobal('ImageData', class { constructor(data: Uint8ClampedArray) { imagePixels = data } });
  vi.stubGlobal('OffscreenCanvas', class {
    width: number; height: number;
    constructor(width: number, height: number) { this.width = width; this.height = height; canvas = this }
    getContext() { return outcome === 'context failure' ? null : {putImageData() {}} }
    async convertToBlob() {
      if (outcome === 'encode failure') throw new Error('Failed to encode');
      return {type: 'image/webp', arrayBuffer: async () => webp.buffer};
    }
  });
  const result = encodeWebp(pixels,1,1,85);
  if (outcome === 'success') expect(Array.from(await result)).toEqual(Array.from(webp));
  else await expect(result).rejects.toThrow();
  if (outcome !== 'context failure') {
    expect(imagePixels!.buffer).toBe(storage.buffer);
    expect(imagePixels!.byteOffset).toBe(2);
    expect(Array.from(imagePixels!)).toEqual([1,2,3,4]);
  }
  expect([canvas.width,canvas.height]).toEqual([0,0]);
  expect(Array.from(pixels)).toEqual([1,2,3,4]); // Borrowing never detaches request storage.
});
