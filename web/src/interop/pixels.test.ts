import { describe, expect, it } from 'vitest';
import {
  assertRgbaPixelBuffer,
  copyImageDataPixels,
  copyTypedArrayView,
  expectedRgbaPixelLength,
} from './pixels';

describe('pixels', () => {
  it('copies only the typed-array view, not the full backing buffer', () => {
    const backing = new ArrayBuffer(64);
    const view = new Uint8ClampedArray(backing, 8, 16);

    const copied = copyTypedArrayView(view);

    expect(copied.length).toBe(16);
    expect(copied.byteLength).toBe(16);
    expect(copied.buffer.byteLength).toBe(16);
    expect(backing.byteLength).toBe(64);
  });

  it('copies ImageData pixels with the same view length', () => {
    const backing = new ArrayBuffer(128);
    const data = new Uint8ClampedArray(backing, 16, 12);

    const copied = copyImageDataPixels(data);

    expect(copied.length).toBe(12);
    expect(copied.buffer.byteLength).toBe(12);
    expect(backing.byteLength).toBe(128);
  });

  it('computes expected RGBA pixel length from dimensions', () => {
    expect(expectedRgbaPixelLength(2, 3)).toBe(24);
  });

  it('asserts matching RGBA buffers without throwing', () => {
    const pixels = new Uint8Array(24);
    expect(() => assertRgbaPixelBuffer(pixels, 2, 3, 6)).not.toThrow();
  });

  it('throws when RGBA buffer length does not match dimensions', () => {
    const pixels = new Uint8Array(32);

    expect(() => assertRgbaPixelBuffer(pixels, 2, 3, 6)).toThrow(
      'Pixel buffer size mismatch: got 32, expected 24 (2x3x4, colorType=6)',
    );
  });
});
