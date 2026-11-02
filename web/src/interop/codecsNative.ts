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

export async function decodeNative(bytes: Uint8Array, validateOnly: boolean) {
  const info = inspectImage(bytes);
  if (typeof createImageBitmap !== 'function') throw new Error('This browser cannot validate image bytes in a worker. Use a browser with worker image decoding support.');
  const bitmap = await createImageBitmap(new Blob([new Uint8Array(bytes)], {type: inputMime(info.format)}), {imageOrientation: 'from-image'});
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
