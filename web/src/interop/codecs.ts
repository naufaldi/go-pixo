import avifEncodeWasm from '@jsquash/avif/codec/enc/avif_enc.wasm?url';
import avifDecodeWasm from '@jsquash/avif/codec/dec/avif_dec.wasm?url';
import heifModuleUrl from 'libheif-js/libheif-wasm/libheif-bundle.mjs?url';
import type { AVIFModule as AvifEncoder } from '@jsquash/avif/codec/enc/avif_enc.js';
import type { AVIFModule as AvifDecoder } from '@jsquash/avif/codec/dec/avif_dec.js';
import { assertSafeDimensions, inspectImage, sniffInputFormat } from './imageFormats';
import { assertRgbaPixelBuffer } from './pixels';

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

type HeifImage = {handle: number; get_width(): number; get_height(): number; free(): void};
type HeifContext = {decoder: number | null; decode(bytes: Uint8Array): HeifImage[]};
type HeifDecoded = {code?: number; image: number; channels: {id: number; width: number; height: number; stride: number; data: Uint8Array}[]};
type HeifModule = {
  HeifDecoder: new () => HeifContext;
  heif_colorspace_RGB: number; heif_chroma_interleaved_RGBA: number; heif_channel_interleaved: number;
  heif_js_decode_image2(handle: number, colorspace: number, chroma: number): Promise<HeifDecoded>;
  heif_image_release(image: number): void;
  heif_context_free(context: number): void;
  heif_context_get_number_of_top_level_images(context: number): number;
};
let heifModule: Promise<HeifModule> | undefined;

export async function decodeHeic(bytes: Uint8Array): Promise<DecodedPixels> {
  const info=inspectImage(bytes);
  if ((info.bitDepth ?? 8)>8) throw new Error('Only 8-bit HEIC/HEIF images are supported');
  // An emitted, unmodified ESM asset embeds its WASM. No CDN or runtime network
  // dependency; keeping the LGPL module separate also permits replacement.
  heifModule??=import(/* @vite-ignore */ heifModuleUrl).then(module=>module.default());
  const lib=await heifModule!;
  const context=new lib.HeifDecoder();
  let images: HeifImage[]=[];
  let decoded: HeifDecoded | undefined;
  try {
    images=context.decode(bytes);
    const count=context.decoder ? lib.heif_context_get_number_of_top_level_images(context.decoder) : 0;
    if (count!==1 || images.length!==1) throw new Error(count>1 ? 'Multiple-image HEIC is not supported. Export a single still image first.' : 'Invalid or unsupported HEIC/HEIF image');
    const image=images[0], width=image.get_width(), height=image.get_height();
    assertSafeDimensions(width,height); // metadata handles, before HEVC decoding / RGBA allocation
    decoded=await lib.heif_js_decode_image2(image.handle,lib.heif_colorspace_RGB,lib.heif_chroma_interleaved_RGBA);
    if (!decoded || decoded.code) throw new Error('HEIC/HEIF pixel decoding failed');
    const channel=decoded.channels.find(channel=>channel.id===lib.heif_channel_interleaved);
    if (!channel || channel.width!==width || channel.height!==height || channel.stride<width*4 ||
        channel.data.length<channel.stride*(height-1)+width*4) throw new Error('Invalid HEIC/HEIF pixel layout');
    const pixels=new Uint8Array(width*height*4);
    for (let row=0;row<height;row++) pixels.set(channel.data.subarray(row*channel.stride,row*channel.stride+width*4),row*width*4);
    return {pixels,width,height};
  } finally {
    if (decoded?.image) lib.heif_image_release(decoded.image);
    images.forEach(image=>image.free());
    if (context.decoder) { lib.heif_context_free(context.decoder); context.decoder=null; }
  }
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
  // UTIF follows maker notes / EXIF recursively without cycle limits. Pixels do
  // not need those directories; neutralize their entries in our private copy.
  const view=new DataView(buffer), le=bytes[0]===73, p=view.getUint32(4,le), count=view.getUint16(p,le);
  for (let i=0;i<count;i++) {
    const s=p+2+i*12;
    if ([34665,37500,50740].includes(view.getUint16(s,le))) {
      view.setUint16(s,65000,le); view.setUint16(s+2,1,le); view.setUint32(s+4,0,le); view.setUint32(s+8,0,le);
    }
  }
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
  // UTIF allocates a separate tile buffer from these tags, independently of
  // image width/height. Reject even partial tile declarations before decodeImage.
  if ([page.t322,page.t323,page.t324,page.t325].some(value=>value!==undefined)) {
    throw new Error('Tiled TIFF images are not supported. Export a strip-based image first.');
  }
  const samples=page.t277 ?? [1], bits=page.t258 ?? [1], sampleFormat=page.t339 ?? [1];
  if (samples.length!==1 || !Number.isInteger(samples[0]) || samples[0]<1 || samples[0]>4 ||
      bits.length!==samples[0] || bits.some(bit=>![1,2,4,8].includes(bit) || bit!==bits[0]) ||
      (sampleFormat.length!==1 && sampleFormat.length!==samples[0]) || sampleFormat.some(format=>format!==1)) {
    throw new Error('TIFF needs 1–4 unsigned integer channels with matching, uniform 1, 2, 4 or 8-bit samples.');
  }
  // UTIF's strip allocation uses BitsPerSample.length, not SamplesPerPixel.
  // Matching both and limiting channels/bits bounds it to at most RGBA size.
  if ((page.t274?.[0] ?? 1)!==1) throw new Error('Rotated TIFF orientation is not supported. Export an upright image first.');
  UTIF.decodeImage(buffer,page);
  const pixels=UTIF.toRGBA8(page);
  assertRgbaPixelBuffer(pixels,width,height);
  return {pixels,width,height};
}
