import { describe, it, expect } from 'vitest';
import { buildCompressedFilename } from './download';

describe('download helpers', () => {
  it('builds compressed filenames with selected extension', () => {
    expect(buildCompressedFilename('photo.png', '.jpg')).toBe('compressed_photo.jpg');
    expect(buildCompressedFilename('image.jpeg', '.webp')).toBe('compressed_image.webp');
  });
});
