export const RGBA_BYTES_PER_PIXEL = 4;

export function copyTypedArrayView(view: ArrayBufferView): Uint8Array {
  return new Uint8Array(view);
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
