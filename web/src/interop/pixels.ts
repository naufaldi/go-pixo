export const RGBA_BYTES_PER_PIXEL = 4;

export function copyTypedArrayView(view: ArrayBufferView): Uint8Array {
  return new Uint8Array(new Uint8Array(view.buffer, view.byteOffset, view.byteLength));
}

export function copyImageDataPixels(data: Uint8ClampedArray): Uint8Array {
  return copyTypedArrayView(data);
}

// Only for ImageData returned exclusively to this operation, never codec/Go
// heap views or storage shared with another request. Preserve subarray offsets.
export function ownedImageDataPixels(data: Uint8ClampedArray): Uint8Array {
  return new Uint8Array(data.buffer, data.byteOffset, data.byteLength);
}

export function expectedRgbaPixelLength(width: number, height: number): number {
  return width * height * RGBA_BYTES_PER_PIXEL;
}

export function assertRgbaPixelBuffer(
  pixels: Uint8Array,
  width: number,
  height: number,
  colorType: number = 6,
): void {
  const expected = expectedRgbaPixelLength(width, height);
  if (pixels.length !== expected) {
    throw new Error(
      `Pixel buffer size mismatch: got ${pixels.length}, expected ${expected} ` +
        `(${width}x${height}x${RGBA_BYTES_PER_PIXEL}, colorType=${colorType})`,
    );
  }
}

// JPEG has no alpha: composite RGBA onto white. Parallel JPEG band workers
// run this on their rows, so it must stay a pure per-pixel function.
export function compositeRgbaToRgbWhite(
  pixels: Uint8Array,
  width: number,
  height: number,
): Uint8Array {
  const rgb = new Uint8Array(width * height * 3);
  for (let i = 0; i < width * height; i++) {
    const src = i * 4;
    const dst = i * 3;
    const alpha = pixels[src + 3];

    if (alpha === 255) {
      rgb[dst] = pixels[src];
      rgb[dst + 1] = pixels[src + 1];
      rgb[dst + 2] = pixels[src + 2];
    } else if (alpha === 0) {
      rgb[dst] = 255;
      rgb[dst + 1] = 255;
      rgb[dst + 2] = 255;
    } else {
      const opacity = alpha / 255;
      rgb[dst] = Math.round(pixels[src] * opacity + 255 * (1 - opacity));
      rgb[dst + 1] = Math.round(pixels[src + 1] * opacity + 255 * (1 - opacity));
      rgb[dst + 2] = Math.round(pixels[src + 2] * opacity + 255 * (1 - opacity));
    }
  }
  return rgb;
}
