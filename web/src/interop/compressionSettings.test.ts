import { describe, it, expect } from 'vitest';
import {
  buildCompressedFilename,
  compressionCandidates,
  derivePresetSettings,
  mimeForFormat,
  replaceExtension,
  resolveOutputFormat,
  shouldUseLosslessBytePath,
} from './compressionSettings';

describe('compressionSettings', () => {
  it('resolves forced output formats', () => {
    expect(resolveOutputFormat('png', 'jpeg')).toBe('jpeg');
    expect(resolveOutputFormat('jpeg', 'same')).toBe('jpeg');
    expect(resolveOutputFormat('webp', 'same')).toBe('webp');
    expect(resolveOutputFormat('avif', 'same')).toBe('avif');
    expect(resolveOutputFormat('gif', 'same')).toBe('png');
    expect(resolveOutputFormat('tiff', 'same')).toBe('png');
    expect(resolveOutputFormat('png', 'auto')).toBe('png');
  });

  it('keeps HEIC photos as JPEG when lossy and PNG when Perfect Quality is on', () => {
    expect(resolveOutputFormat('heic', 'same')).toBe('jpeg');
    expect(resolveOutputFormat('heic', 'same', true)).toBe('png');
    expect(resolveOutputFormat('heic', 'webp')).toBe('webp');
    expect(compressionCandidates('heic', 'same', false)).toEqual(['jpeg']);
    expect(compressionCandidates('heic', 'same', true)).toEqual(['png']);
    expect(compressionCandidates('heic', 'auto', false)).toEqual(['webp', 'jpeg']);
    expect(compressionCandidates('heic', 'auto', true)).toEqual(['avif', 'png']);
  });

  it('builds compressed filenames with output extension', () => {
    expect(buildCompressedFilename('photo.png', '.jpg')).toBe('compressed_photo.jpg');
    expect(replaceExtension('photo.png', '.webp')).toBe('photo.webp');
  });

  it('maps mime types for output formats', () => {
    expect(mimeForFormat('jpeg')).toBe('image/jpeg');
    expect(mimeForFormat('webp')).toBe('image/webp');
  });

  it('skips lossless byte path when resizing or converting format', () => {
    expect(
      shouldUseLosslessBytePath({
        lossless: true,
        inputFormat: 'png',
        outputFormat: 'png',
        targetWidth: 800,
      }),
    ).toBe(false);

    expect(
      shouldUseLosslessBytePath({
        lossless: true,
        inputFormat: 'png',
        outputFormat: 'jpeg',
      }),
    ).toBe(false);

    expect(
      shouldUseLosslessBytePath({
        lossless: true,
        inputFormat: 'png',
        outputFormat: 'png',
      }),
    ).toBe(true);
  });

  it('derives stronger lossy settings for ultra preset', () => {
    const ultra = derivePresetSettings('ultra', false);
    const faster = derivePresetSettings('faster', false);
    expect(ultra.qualityTarget).toBeLessThan(faster.qualityTarget);
    expect(ultra.maxColors).toBeLessThan(faster.maxColors);
  });
});
