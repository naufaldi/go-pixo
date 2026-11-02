// @vitest-environment node
import {readFile} from 'node:fs/promises';
import {afterEach, expect, it, vi} from 'vitest';
import {decodeNative} from './codecsNative';
import {jpegWithExif} from '../test/jpeg';

const png = new Uint8Array([137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82,0,0,0,1,0,0,0,1,16,6,0,0,0,0,0,0,0]);
afterEach(() => vi.unstubAllGlobals());

it.each([png,new Uint8Array([255,216,255,192,0,7,8,0,1,0,1])])('propagates browser decoder rejection of metadata-only input', async bytes => {
  const decode = vi.fn().mockRejectedValue(new DOMException('Invalid image','InvalidStateError'));
  vi.stubGlobal('createImageBitmap',decode);
  await expect(decodeNative(bytes,true)).rejects.toThrow(/Invalid image/);
  expect(decode).toHaveBeenCalledOnce();
});

it('passes real PNG bytes unchanged to browser validation without canvas readback', async () => {
  const bytes = new Uint8Array(await readFile(new URL('../../e2e/fixtures/sample-32x32.png',import.meta.url)));
  const close = vi.fn();
  const decode = vi.fn(async (blob: Blob) => {
    expect(new Uint8Array(await blob.arrayBuffer())).toEqual(bytes);
    return {width:32,height:32,close};
  });
  const canvas = vi.fn();
  vi.stubGlobal('createImageBitmap',decode); vi.stubGlobal('OffscreenCanvas',canvas);
  await decodeNative(bytes,true);
  expect(close).toHaveBeenCalledOnce();
  expect(decode).toHaveBeenCalledWith(expect.any(Blob),{imageOrientation:'from-image'});
  expect(canvas).not.toHaveBeenCalled();
});

it('validates high-bit-depth bytes without extracting 8-bit samples', async () => {
  const close = vi.fn(), canvas = vi.fn();
  vi.stubGlobal('createImageBitmap',vi.fn(async () => ({width:1,height:1,close})));
  vi.stubGlobal('OffscreenCanvas',canvas);
  expect(await decodeNative(png,true)).toEqual({width:1,height:1});
  expect(canvas).not.toHaveBeenCalled();
  expect(close).toHaveBeenCalledOnce();
});

it.each([{width:3,height:4},{width:24_000_001,height:1}])('rejects native dimensions %j before raster allocation and closes the bitmap', async size => {
  const close = vi.fn(), canvas = vi.fn();
  vi.stubGlobal('createImageBitmap',vi.fn(async () => ({...size,close})));
  vi.stubGlobal('OffscreenCanvas',canvas);
  await expect(decodeNative(jpegWithExif(3,4,6),false)).rejects.toThrow(/dimensions|limit/);
  expect(canvas).not.toHaveBeenCalled();
  expect(close).toHaveBeenCalledOnce();
});

it.each(['success','readback failure'])('rasterizes oriented pixels and releases bitmap/canvas after %s', async outcome => {
  const close = vi.fn();
  const pixels = new Uint8ClampedArray(50).subarray(2);
  let canvas: {width:number;height:number} | undefined;
  vi.stubGlobal('createImageBitmap',vi.fn(async () => ({width:4,height:3,close})));
  const draw = vi.fn();
  vi.stubGlobal('OffscreenCanvas',class {
    width: number; height: number;
    constructor(width:number,height:number) { this.width=width; this.height=height; canvas=this; }
    getContext() { return {
      drawImage:draw,
      getImageData() { if (outcome !== 'success') throw new Error('Readback failed'); return {data:pixels}; },
    }; }
  });
  const pending = decodeNative(jpegWithExif(3,4,6),false);
  if (outcome === 'success') {
    const image = await pending;
    expect(image).toMatchObject({width:4,height:3});
    expect(image.pixels?.buffer).toBe(pixels.buffer);
    expect(image.pixels?.byteOffset).toBe(2);
  } else await expect(pending).rejects.toThrow(/Readback failed/);
  expect(draw).toHaveBeenCalledOnce();
  expect(canvas).toMatchObject({width:0,height:0});
  expect(close).toHaveBeenCalledOnce();
});

it('fails honestly when worker decode validation is unavailable', async () => {
  vi.stubGlobal('createImageBitmap',undefined);
  await expect(decodeNative(png,true)).rejects.toThrow(/cannot validate.*worker/);
});

it('rejects oversized input dimensions before starting browser decoding', async () => {
  const decode = vi.fn();
  vi.stubGlobal('createImageBitmap',decode);
  await expect(decodeNative(jpegWithExif(6000,4001,1),false)).rejects.toThrow(/limit/);
  expect(decode).not.toHaveBeenCalled();
});
