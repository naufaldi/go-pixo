import { assertSafeDimensions, imageDisplayDimensions, inspectImage, inputMime, type ImageInfo } from './imageFormats';
import { ownedImageDataPixels } from './pixels';

export function assertDecodedDimensions(info: ImageInfo, width: number, height: number): void {
  assertSafeDimensions(width, height);
  const display = imageDisplayDimensions(info);
  const fits = (maxWidth: number, maxHeight: number) => width <= maxWidth && height <= maxHeight;
  // HEIC 'ispe' stores the coded size. Decoders then apply 'clap' crops and
  // 'irot' quarter turns (portrait phone photos), so the output may be smaller
  // or transposed, but never exceeds the pixel budget admitted from 'ispe'.
  const matches = info.format === 'heic'
    ? fits(display.width, display.height) || fits(display.height, display.width)
    : width === display.width && height === display.height;
  if (!matches) throw new Error('Native image dimensions disagree with display/EXIF preflight');
}

const decodeBitmap = (bytes: Uint8Array, info: ImageInfo) =>
  createImageBitmap(new Blob([new Uint8Array(bytes)], {type: inputMime(info.format)}), {imageOrientation: 'from-image'});

export async function decodeNative(bytes: Uint8Array, validateOnly: boolean) {
  const info = inspectImage(bytes);
  if (typeof createImageBitmap !== 'function') throw new Error('This browser cannot validate image bytes in a worker. Use a browser with worker image decoding support.');
  const bitmap = await decodeBitmap(bytes, info);
  let canvas: OffscreenCanvas | undefined;
  try {
    const {width, height} = bitmap;
    assertDecodedDimensions(info, width, height);
    // Validation must not send high-bit-depth samples through an 8-bit canvas.
    if (validateOnly) return {width, height};
    canvas = new OffscreenCanvas(width, height);
    const context = canvas.getContext('2d');
    if (!context) throw new Error('Worker canvas is unavailable');
    context.drawImage(bitmap, 0, 0);
    const pixels = ownedImageDataPixels(context.getImageData(0, 0, width, height).data);
    return {width, height, pixels};
  } finally {
    bitmap.close();
    if (canvas) { canvas.width = 0; canvas.height = 0; }
  }
}

const COMPARE_BAND_BYTES = 16 * 1024 * 1024;

// Proves that two encodings of one image decode to identical pixels with the
// browser's own decoder, which is independent of the Go coefficient reader.
// The images are decoded one after the other and compared through SHA-256
// digests of row bands, so only one decoded bitmap and one band of readback
// are alive at a time. A 24 MP photo is ~96 MB decoded; holding two at once
// can exhaust a phone's worker memory and kill the job.
export async function assertSameDecodedPixels(expected: Uint8Array, actual: Uint8Array): Promise<void> {
  if (typeof createImageBitmap !== 'function' || typeof OffscreenCanvas !== 'function') {
    throw new Error('Pixel comparison needs worker image decoding support');
  }
  const reference = await bandDigests(expected);
  const candidate = await bandDigests(actual, reference);
  if (candidate.digests.some((digest, band) => !sameBytes(digest, reference.digests[band]))) throw new Error('Decoded pixels differ');
}

type BandDigests = {width: number; height: number; digests: Uint8Array[]};

async function bandDigests(bytes: Uint8Array, expectedSize?: {width: number; height: number}): Promise<BandDigests> {
  const bitmap = await decodeBitmap(bytes, inspectImage(bytes));
  let canvas: OffscreenCanvas | undefined;
  try {
    const {width, height} = bitmap;
    if (expectedSize && (width !== expectedSize.width || height !== expectedSize.height)) throw new Error('Decoded dimensions differ');
    const band = Math.max(1, Math.min(height, Math.floor(COMPARE_BAND_BYTES / (width * 4))));
    canvas = new OffscreenCanvas(width, band);
    const context = canvas.getContext('2d', {willReadFrequently: true});
    if (!context) throw new Error('Worker canvas is unavailable');
    const digests: Uint8Array[] = [];
    for (let y = 0; y < height; y += band) {
      const h = Math.min(band, height - y);
      context.clearRect(0, 0, width, band);
      context.drawImage(bitmap, 0, y, width, h, 0, 0, width, h);
      digests.push(new Uint8Array(await crypto.subtle.digest('SHA-256', context.getImageData(0, 0, width, h).data)));
    }
    return {width, height, digests};
  } finally {
    bitmap.close();
    if (canvas) { canvas.width = 0; canvas.height = 0; }
  }
}

const sameBytes = (a: Uint8Array, b: Uint8Array | undefined) => b !== undefined && a.length === b.length && a.every((value, i) => value === b[i]);
