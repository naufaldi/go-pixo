import { afterEach, expect, it, vi } from 'vitest';
import { decodeFile } from './imageDecode';

afterEach(() => vi.unstubAllGlobals());

it('rejects animated files before a browser can flatten them', async () => {
  // Browser decode must never run for a known animation, even when MIME lies.
  vi.stubGlobal('Image', class {
    onerror?: () => void;
    set src(_: string) { queueMicrotask(() => this.onerror?.()); }
  });
  vi.stubGlobal('URL', {createObjectURL: () => 'blob:test', revokeObjectURL: () => {}});
  const frame = [44,0,0,0,0,1,0,1,0,0,2,2,68,1,0];
  const bytes = new Uint8Array([71,73,70,56,57,97,1,0,1,0,0,0,0,...frame,...frame,59]);
  await expect(decodeFile(new File([bytes], 'fake.png', {type:'IMAGE/PNG'}))).rejects.toThrow(/animat/i);
});
