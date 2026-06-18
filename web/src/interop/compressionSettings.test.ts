import { describe, it, expect } from 'vitest';
import {
  buildCompressedFilename,
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
