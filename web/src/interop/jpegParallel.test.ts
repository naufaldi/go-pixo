import { expect, it, vi } from 'vitest';
import { encodeJpegRgba, jpegBandCount, type JpegEncodeRuntime } from './jpegParallel';

it('uses up to four bands on a desktop and stays serial for small images, phones and low memory', () => {
  const desktop = { hardwareConcurrency: 16, deviceMemory: 8, mobile: false, nestedWorkers: true };
  const photo = 4032 * 2688;
  expect(jpegBandCount(photo, desktop)).toBe(4);
  expect(jpegBandCount(photo, { ...desktop, hardwareConcurrency: 4 })).toBe(3);
  expect(jpegBandCount(photo, { ...desktop, deviceMemory: undefined })).toBe(1); // Safari (incl. iPadOS) and Firefox hide it.
  expect(jpegBandCount(1536 * 1024, desktop)).toBe(1);
  expect(jpegBandCount(photo, { ...desktop, hardwareConcurrency: 2 })).toBe(1);
  expect(jpegBandCount(photo, { ...desktop, deviceMemory: 4 })).toBe(1);
  expect(jpegBandCount(photo, { ...desktop, mobile: true })).toBe(1);
  expect(jpegBandCount(photo, { ...desktop, nestedWorkers: false })).toBe(1);
});

it('falls back to the serial encoder when a band worker fails', async () => {
  const serial = new Uint8Array([0xff, 0xd8]);
  const runtime: JpegEncodeRuntime = {
    bands: 3,
    serial: vi.fn(() => serial),
    band: vi.fn(() => new Uint8Array(2)),
    remote: vi.fn(async () => { throw new Error('band worker crashed'); }),
    assemble: vi.fn(() => new Uint8Array(1)),
    release: vi.fn(),
    onFallback: vi.fn(),
  };
  const width = 8, height = 48;
  const rgba = new Uint8Array(width * height * 4).fill(255);
  await expect(encodeJpegRgba(rgba, width, height, 0, runtime)).resolves.toBe(serial);
  expect(runtime.remote).toHaveBeenCalledTimes(2);
  expect(runtime.assemble).not.toHaveBeenCalled();
  expect(runtime.release).toHaveBeenCalledWith(true);
  expect(runtime.onFallback).toHaveBeenCalledWith(new Error('band worker crashed'));
  // The serial encode receives the whole image composited to RGB.
  expect(vi.mocked(runtime.serial).mock.calls[0][0]).toHaveLength(width * height * 3);
});

it('posts remote bands before its own band and falls back when a band worker cannot start', async () => {
  const order: string[] = [];
  let rejectFirst: (error: Error) => void = () => {};
  const serial = new Uint8Array([0xff, 0xd8]);
  const runtime: JpegEncodeRuntime = {
    bands: 3,
    serial: vi.fn(() => serial),
    band: vi.fn(() => { order.push('local'); return new Uint8Array(2); }),
    remote: vi.fn((index: number) => {
      order.push(`remote${index}`);
      if (index === 1) throw new Error('Worker constructor failed');
      return new Promise<Uint8Array>((_, reject) => { rejectFirst = reject; });
    }),
    assemble: vi.fn(() => new Uint8Array(1)),
    // Stopping band workers rejects the first band's pending promise.
    release: vi.fn(() => rejectFirst(new Error('Band workers stopped'))),
  };
  const width = 8, height = 48;
  await expect(encodeJpegRgba(new Uint8Array(width * height * 4), width, height, 0, runtime)).resolves.toBe(serial);
  expect(order).toEqual(['remote0', 'remote1', 'local']);
  expect(runtime.release).toHaveBeenCalledWith(true);
});
