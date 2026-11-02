import avifEncodeWasm from '@jsquash/avif/codec/enc/avif_enc.wasm?url';
import avifDecodeWasm from '@jsquash/avif/codec/dec/avif_dec.wasm?url';
import type { AVIFModule as AvifEncoder } from '@jsquash/avif/codec/enc/avif_enc.js';
import type { AVIFModule as AvifDecoder } from '@jsquash/avif/codec/dec/avif_dec.js';
import { assertSafeDimensions, HEIC_UNAVAILABLE_MESSAGE, inspectImage, sniffInputFormat, TIFF_TAG_VALUE_LIMITS } from './imageFormats';
import { assertRgbaPixelBuffer } from './pixels';
import { decodeTiffStrips } from './tiffStrips';

let encoder: Promise<AvifEncoder> | undefined;
let decoder: Promise<AvifDecoder> | undefined;
async function wasmBinary(url: string): Promise<Uint8Array> {
  const response=await fetch(url);
  if (!response.ok) throw new Error(`Failed to load local image codec (${response.status})`);
  return new Uint8Array(await response.arrayBuffer());
}

export async function encodeAvif(pixels: Uint8Array, width: number, height: number, quality: number, lossless: boolean, speed = 6): Promise<Uint8Array> {
  assertSafeDimensions(width,height); assertRgbaPixelBuffer(pixels,width,height);
  // Explicit single-thread factory needs no SharedArrayBuffer, isolation headers, or subworkers.
  encoder??=Promise.all([import('@jsquash/avif/codec/enc/avif_enc.js'),wasmBinary(avifEncodeWasm)])
    .then(([factory,wasm])=>factory.default({wasmBinary:wasm}));
  const {defaultOptions}=await import('@jsquash/avif/meta.js');
  const module=await encoder;
  const output=module.encode(new Uint8Array(pixels),width,height,{
    ...defaultOptions, quality:lossless?100:quality, qualityAlpha:-1,
    subsample:lossless?3:1, speed,
  });
  if (!output) throw new Error('AVIF encoding failed');
  const bytes=new Uint8Array(output);
  if (sniffInputFormat(bytes)!=='avif') throw new Error('AVIF encoder returned an invalid signature');
  return bytes;
}

export type DecodedPixels = {pixels: Uint8Array; width: number; height: number};

export async function decodeHeic(_bytes: Uint8Array): Promise<DecodedPixels> {
  // Keep the vulnerable libheif bundle unreachable until a patched wrapper exists.
  throw new Error(HEIC_UNAVAILABLE_MESSAGE);
}

export async function decodeAvif(bytes: Uint8Array): Promise<DecodedPixels> {
  inspectImage(bytes);
  decoder??=Promise.all([import('@jsquash/avif/codec/dec/avif_dec.js'),wasmBinary(avifDecodeWasm)])
    .then(([factory,wasm])=>factory.default({wasmBinary:wasm}));
  const image=(await decoder).decode(new Uint8Array(bytes),8);
  if (!image) throw new Error('Invalid or unsupported AVIF image');
  assertSafeDimensions(image.width,image.height);
  return {pixels:new Uint8Array(image.data),width:image.width,height:image.height};
}

export async function encodeWebp(pixels: Uint8Array, width: number, height: number, quality: number): Promise<Uint8Array> {
  assertSafeDimensions(width,height); assertRgbaPixelBuffer(pixels,width,height);
  const canvas=new OffscreenCanvas(width,height);
  try {
    const ctx=canvas.getContext('2d'); if (!ctx) throw new Error('Canvas is unavailable');
    // putImageData consumes the view synchronously; request storage stays owned
    // by this worker and is never detached while the encoder uses the canvas.
    ctx.putImageData(new ImageData(new Uint8ClampedArray(pixels.buffer as ArrayBuffer,pixels.byteOffset,pixels.byteLength),width,height),0,0);
    const blob=await canvas.convertToBlob({type:'image/webp',quality:quality/100});
    const bytes=new Uint8Array(await blob.arrayBuffer());
    if (blob.type!=='image/webp' || sniffInputFormat(bytes)!=='webp') {
      throw new Error('This browser cannot encode WebP. Choose PNG, JPEG or AVIF.');
    }
    return bytes;
  } finally {
    canvas.width=0; canvas.height=0;
  }
}

export async function decodeTiff(bytes: Uint8Array): Promise<DecodedPixels> {
  inspectImage(bytes);
  const UTIF=await import('utif');
  const buffer=new Uint8Array(bytes).buffer;
  // Only pixel-decoding tags reach UTIF. In particular, do not materialize
  // arbitrary metadata arrays or follow recursive EXIF / maker-note pointers.
  const view=new DataView(buffer), le=bytes[0]===73, p=view.getUint32(4,le), count=view.getUint16(p,le);
  let retained=0;
  for (let i=0;i<count;i++) {
    const s=p+2+i*12;
    const tag=view.getUint16(s,le);
    if ([322,323,324,325].includes(tag)) {
      throw new Error('Tiled TIFF images are not supported. Export a strip-based image first.');
    }
    if (TIFF_TAG_VALUE_LIMITS[tag]!==undefined) {
      new Uint8Array(buffer).copyWithin(p+2+retained*12,s,s+12);
      retained++;
    }
  }
  view.setUint16(p,retained,le); view.setUint32(p+2+retained*12,0,le);
  const pages=UTIF.decode(buffer);
  if (pages.length!==1) throw new Error('Multiple-page TIFF images are not supported. Export a single page first.');
  const page=pages[0];
  if (page.t50706 || page.t33421 || page.t262?.[0]===32803 || page.t330) throw new Error('RAW / DNG TIFF images are not supported');
  const width=page.t256?.[0], height=page.t257?.[0];
  assertSafeDimensions(width,height);
  const planar=page.t284 ?? [1];
  if (planar.length!==1 || planar[0]!==1) {
    throw new Error('Planar TIFF images are not supported. Export an interleaved image first.');
  }
  const samples=page.t277 ?? [1], bits=page.t258 ?? [1], sampleFormat=page.t339 ?? [1];
  if (samples.length!==1 || !Number.isInteger(samples[0]) || samples[0]<1 || samples[0]>4 ||
      bits.length!==samples[0] || bits.some(bit=>![1,2,4,8].includes(bit) || bit!==bits[0]) ||
      (sampleFormat.length!==1 && sampleFormat.length!==samples[0]) || sampleFormat.some(format=>format!==1)) {
    throw new Error('TIFF needs 1–4 unsigned integer channels with matching, uniform 1, 2, 4 or 8-bit samples.');
  }
  if ((page.t274?.[0] ?? 1)!==1) throw new Error('Rotated TIFF orientation is not supported. Export an upright image first.');
  const photometric=page.t262?.[0] ?? 2;
  if (photometric===3 && (samples[0]!==1 || bits[0]!==8 || page.t320?.length!==768)) {
    throw new Error('Palette TIFF needs an 8-bit index and a 256-entry RGB color map');
  }
  page.width=width; page.height=height;
  page.data=decodeTiffStrips(bytes,page,width,height);
  const pixels=UTIF.toRGBA8(page);
  assertRgbaPixelBuffer(pixels,width,height);
  return {pixels,width,height};
}
