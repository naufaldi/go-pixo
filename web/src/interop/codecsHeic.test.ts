// @vitest-environment node
import { readFile } from 'node:fs/promises';
import { afterEach, expect, it, vi } from 'vitest';
import { decodeHeic } from './codecs';

afterEach(() => vi.unstubAllGlobals());

it.each(['rainbow-451x461.heic','multi-image-header.heic'])('disables %s before loading any bundled decoder', async fixture => {
  const fetch = vi.fn(() => { throw new Error('Codec must not load'); });
  vi.stubGlobal('fetch',fetch);
  const bytes = await readFile(new URL(`./codecs-fixtures/${fixture}`,import.meta.url));
  await expect(decodeHeic(bytes)).rejects.toThrow(/temporarily unavailable.*security.*PNG or JPEG.*retry/i);
  await expect(decodeHeic(bytes)).rejects.toThrow(/temporarily unavailable/i);
  expect(fetch).not.toHaveBeenCalled();
});

it('also rejects malformed HEIC without entering the vulnerable decoder', async () => {
  await expect(decodeHeic(new Uint8Array())).rejects.toThrow(/temporarily unavailable/i);
});
