// @vitest-environment node
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { compressWithCandidates } from './codecsCompression';
import {jpegWithExif} from '../test/jpeg';
import { imageDisplayDimensions, inspectImage } from './imageFormats';

// Fake decoded pixel value per encoded byte length; equal values mean equal pixels.
let pixelValueFor=(_length: number) => 1;
beforeEach(() => {
  pixelValueFor=() => 1;
  // These tests own candidate selection; the browser decoder is the boundary.
  vi.stubGlobal('createImageBitmap',vi.fn(async (blob: Blob) => ({
    ...imageDisplayDimensions(inspectImage(new Uint8Array(await blob.arrayBuffer()))),close:vi.fn(),value:pixelValueFor(blob.size),
  })));
  vi.stubGlobal('OffscreenCanvas',class {
    width: number; height: number;
    constructor(width: number, height: number) { this.width=width; this.height=height; }
    getContext() {
      let value=0;
      return {clearRect:vi.fn(),drawImage:(bitmap: {value: number}) => { value=bitmap.value; },
        getImageData:(_x: number,_y: number,w: number,h: number) => ({data:new Uint8ClampedArray(w*h*4).fill(value)})};
    }
  });
});
afterEach(() => vi.unstubAllGlobals());

function png(w: number, h: number, length = 45) {
  const b=new Uint8Array(length); b.set([137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82]);
  const v=new DataView(b.buffer); v.setUint32(16,w); v.setUint32(20,h); b.set([8,6],24); return b;
}
function webp() {
  return new Uint8Array([82,73,70,70,18,0,0,0,87,69,66,80,86,80,56,76,5,0,0,0,47,1,0,0,0,0]); // 2x1 VP8L header
}
const base={format:'png' as const,outputFormat:'auto' as const,lossy:true,width:2,height:1,originalFileBytes:png(2,1,81)};

it('auto compares the original PNG with fast WebP without invoking PNG quantization', async () => {
  const result=await compressWithCandidates(base, async format => {
    if (format!=='webp') throw new Error('Unnecessary current-format encode');
    return webp();
  });
  expect(result.outputFormat).toBe('webp');
  expect(result.compressedBytes).toEqual(webp());
});
it('keeps requested resized pixels even when the encoded image is larger', async () => {
  const result=await compressWithCandidates({...base,outputFormat:'png',width:1,height:1,resized:true}, async () => png(1,1,93));
  expect(result.compressedBytes.length).toBe(93);
  expect(new DataView(result.compressedBytes.buffer).getUint32(16)).toBe(1);
});
it('does not label silently returned PNG bytes as AVIF', async () => {
  await expect(compressWithCandidates({...base,outputFormat:'avif'}, async () => png(2,1))).rejects.toThrow(/format|signature/i);
});
it('lossless JPEG keeps original bytes instead of running a lossy recompressor', async () => {
  const jpeg=new Uint8Array([255,216,255,192,0,11,8,0,1,0,2,1,1,17,0,255,217]);
  const result=await compressWithCandidates({...base,format:'jpeg',outputFormat:'same',lossy:false,originalFileBytes:jpeg}, async () => { throw new Error('Lossy encode must not run'); });
  expect(result.compressedBytes).toBe(jpeg);
  expect(result.outputFormat).toBe('jpeg');
});
describe('lossless JPEG entropy optimization', () => {
  // Display 2x1. A COM segment stands in for entropy-coded data so sizes can differ.
  const jpeg=(padding: number, width = 2, height = 1) =>
    new Uint8Array([255,216,255,254,(padding+2)>>8,(padding+2)&255,...new Uint8Array(padding),...jpegWithExif(width,height,1).slice(2)]);
  const original=jpeg(40);
  const request={format:'jpeg' as const,outputFormat:'same' as const,lossy:false,width:2,height:1,originalFileBytes:original};
  const noEncode=async () => { throw new Error('Lossy encode must not run'); };

  it('returns the smaller optimized JPEG in Perfect Quality mode', async () => {
    const optimized=jpeg(10);
    const result=await compressWithCandidates(request,noEncode,undefined,async () => optimized);
    expect(result).toEqual({compressedBytes:optimized,outputFormat:'jpeg'});
  });

  it.each([
    ['fails', async () => { throw new Error('CMYK JPEG is not supported'); }],
    ['is not smaller', async () => jpeg(40)],
    ['changes dimensions', async () => jpeg(1,1,2)],
    ['is not a JPEG', async () => png(2,1,20)],
  ])('keeps the validated original when the optimizer %s', async (_case, optimize) => {
    const result=await compressWithCandidates(request,noEncode,undefined,optimize);
    expect(result.compressedBytes).toBe(original);
  });

  it('keeps the original when the browser decodes the optimized JPEG to different pixels', async () => {
    const optimized=jpeg(5);
    pixelValueFor=length => length===optimized.length ? 2 : 1;
    const result=await compressWithCandidates(request,noEncode,undefined,async () => optimized);
    expect(result.compressedBytes).toBe(original);
  });

  it('keeps the original when pixels cannot be compared', async () => {
    vi.stubGlobal('OffscreenCanvas',undefined);
    const result=await compressWithCandidates(request,noEncode,undefined,async () => jpeg(5));
    expect(result.compressedBytes).toBe(original);
  });

  it('keeps the original when the optimized bytes do not decode', async () => {
    const optimized=jpeg(5);
    vi.stubGlobal('createImageBitmap',vi.fn(async (blob: Blob) => {
      const bytes=new Uint8Array(await blob.arrayBuffer());
      if (bytes.length===optimized.length) throw new DOMException('Invalid image data','InvalidStateError');
      return {...imageDisplayDimensions(inspectImage(bytes)),close:vi.fn()};
    }));
    const result=await compressWithCandidates(request,noEncode,undefined,async () => optimized);
    expect(result.compressedBytes).toBe(original);
  });

  it('competes with lossy re-encoding, so a lossy pass never beats a smaller lossless one', async () => {
    const optimized=jpeg(10);
    const result=await compressWithCandidates({...request,lossy:true},async () => jpeg(30),undefined,async () => optimized);
    expect(result.compressedBytes).toBe(optimized);
    const smallerLossy=jpeg(2);
    expect((await compressWithCandidates({...request,lossy:true},async () => smallerLossy,undefined,async () => optimized)).compressedBytes).toBe(smallerLossy);
  });

  it('skips the optimizer and its pixel check when a lossy result is already far smaller', async () => {
    const optimize=vi.fn(async () => jpeg(10));
    const big=jpeg(400);
    const result=await compressWithCandidates({...request,lossy:true,originalFileBytes:big},async () => jpeg(40),undefined,optimize);
    expect(result.compressedBytes.length).toBe(jpeg(40).length);
    expect(optimize).not.toHaveBeenCalled();
  });

  it('never optimizes non-JPEG originals', async () => {
    const optimize=vi.fn();
    await compressWithCandidates({...base,lossy:false,outputFormat:'png'},async () => png(2,1,90),undefined,optimize);
    expect(optimize).not.toHaveBeenCalled();
  });
});

it('auto retains a valid original when the optional modern codec fails', async () => {
  const result=await compressWithCandidates(base, async () => { throw new Error('Codec unavailable'); });
  expect(result.compressedBytes).toBe(base.originalFileBytes);
  expect(result.outputFormat).toBe('png');
});
it('preserves high-bit-depth input and rejects lossless conversion through 8-bit pixels', async () => {
  const bytes=png(2,1); bytes[24]=16;
  const request={...base,lossy:false,originalFileBytes:bytes};
  const result=await compressWithCandidates(request,async () => png(2,1));
  expect(result.compressedBytes).toBe(bytes);
  await expect(compressWithCandidates({...request,outputFormat:'avif'},async () => png(2,1))).rejects.toThrow(/bit.depth|precision/i);
});

it('checks encoded JPEG display dimensions rather than its storage axes', async () => {
  const bytes=jpegWithExif(3,4,6);
  const result=await compressWithCandidates({format:'png',outputFormat:'jpeg',lossy:true,width:4,height:3},async () => bytes);
  expect(result.compressedBytes).toBe(bytes);
  await expect(compressWithCandidates({format:'png',outputFormat:'jpeg',lossy:true,width:3,height:4},async () => bytes))
    .rejects.toThrow(/dimension/i);
});
it('never selects a rotated original with different displayed dimensions even without a resized flag', async () => {
  await expect(compressWithCandidates({format:'jpeg',outputFormat:'same',lossy:false,width:3,height:4,
    originalFileBytes:jpegWithExif(3,4,6)},async () => { throw new Error('Unexpected encode'); }))
    .rejects.toThrow(/lossy/i);
});

it.each(['jpeg','png'] as const)('rejects undecodable metadata-only %s before selecting original bytes or invoking an encoder', async format => {
  const bytes = format === 'jpeg' ? new Uint8Array([255,216,255,192,0,7,8,0,1,0,1]) : png(1,1,33);
  if (format === 'png') bytes[24]=16;
  const decode = vi.fn().mockRejectedValue(new DOMException('Invalid image data','InvalidStateError'));
  vi.stubGlobal('createImageBitmap',decode);
  const encode = vi.fn();
  await expect(compressWithCandidates({format,lossy:false,width:1,height:1,originalFileBytes:bytes},encode)).rejects.toThrow(/Invalid image data/);
  expect(decode).toHaveBeenCalledTimes(1);
  expect(encode).not.toHaveBeenCalled();
});

it('releases validation bitmap while retaining exact high-bit-depth original bytes', async () => {
  const bytes=png(2,1); bytes[24]=16;
  const close=vi.fn();
  vi.stubGlobal('createImageBitmap',vi.fn(async () => ({width:2,height:1,close})));
  const canvas=vi.fn(); vi.stubGlobal('OffscreenCanvas',canvas);
  const encode=vi.fn();
  const result=await compressWithCandidates({...base,lossy:false,originalFileBytes:bytes},encode);
  expect(result.compressedBytes).toBe(bytes);
  expect(close).toHaveBeenCalledOnce();
  expect(canvas).not.toHaveBeenCalled();
  expect(encode).not.toHaveBeenCalled();
});
