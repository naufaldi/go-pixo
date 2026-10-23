import { afterEach, describe, it, expect, vi } from 'vitest';
import { unzipSync } from 'fflate';
import { buildCompressedFilename, downloadAllAsZip } from './download';

afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); });

describe('download helpers', () => {
  it('builds compressed filenames with selected extension', () => {
    expect(buildCompressedFilename('photo.png', '.jpg')).toBe('compressed_photo.jpg');
    expect(buildCompressedFilename('image.jpeg', '.webp')).toBe('compressed_image.webp');
  });

  it('ZIP retains every duplicate and colliding suffixed filename with its original bytes', async () => {
    let archive: Blob | undefined;
    vi.stubGlobal('fetch', async (url: string) => ({ blob: async () => ({ arrayBuffer: async () => new Uint8Array([Number(url.slice(-1))]).buffer }) }));
    vi.stubGlobal('URL', class extends URL {
      static createObjectURL(blob: Blob) { archive = blob; return 'blob:zip'; }
      static revokeObjectURL() {}
    });
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
    const items = ['photo.png', 'photo.png', 'photo (2).png'].map((name, index) => ({
      file: { name } as File, status: 'Done', compressedUrl: `blob:${index + 1}`, compressedExtension: '.png',
    }));
    await downloadAllAsZip(items);
    const buffer = await new Promise<ArrayBuffer>((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result as ArrayBuffer);
      reader.onerror = reject;
      reader.readAsArrayBuffer(archive!);
    });
    const files = unzipSync(new Uint8Array(buffer));
    expect(Object.keys(files)).toHaveLength(3);
    expect(Object.values(files).map(bytes => [...bytes])).toEqual([[1], [2], [3]]);
    expect(Object.keys(files)).toEqual(['compressed_photo.png', 'compressed_photo (2).png', 'compressed_photo (2) (2).png']);
  });
});
