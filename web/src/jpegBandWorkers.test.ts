// @vitest-environment node
import {afterEach, expect, it, vi} from 'vitest';
import {JpegBandWorkers} from './jpegBandWorkers';
import type {JpegBand, JpegEncodeOptions} from './interop/jpegParallel';

afterEach(() => vi.useRealTimers());

it('terminates a band worker whose init message cannot be posted', () => {
  vi.useFakeTimers();
  const worker = {
    postMessage: vi.fn(() => { throw new DOMException('Module could not be cloned', 'DataCloneError'); }),
    terminate: vi.fn(),
  } as unknown as Worker;
  const workers = new JpegBandWorkers({module: {} as WebAssembly.Module, wasmExec: ''}, () => worker);

  expect(() => workers.encode(0, new Uint8Array(4), 1, 1, {} as JpegBand, {} as JpegEncodeOptions)).toThrow(/cloned/);
  expect(worker.terminate).toHaveBeenCalledOnce();
  expect(vi.getTimerCount()).toBe(0);
});
