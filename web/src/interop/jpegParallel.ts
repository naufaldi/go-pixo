import { compositeRgbaToRgbWhite } from './pixels';

// Parallel JPEG encode (#128): band workers each run their own Go WASM
// instance on a band of MCU rows (colour conversion, DCT, quantization,
// trellis). The compression worker encodes the first band itself, then writes
// the file once from all coefficients, so the output is byte-identical to
// encodeJpegAdvanced.

// Below this many output pixels the worker startup outweighs the saving.
export const PARALLEL_JPEG_MIN_PIXELS = 4_000_000;
// Entropy coding stays serial, so more bands add memory for little gain.
export const MAX_JPEG_BANDS = 4;

// Admission charge on top of the serial workspace, per output pixel, for
// 4:4:4 (the 4:2:0 default needs 6 bytes less): RGBA band copies (4), RGB
// composite (3) and its Go copy (3) in the band workers, and the band's int16
// coefficients in the band worker's Go heap and in transit (6 + 6).
export const JPEG_BAND_BYTES_PER_PIXEL = 22;
// Each band worker's Go runtime and WASM linear memory before any band.
export const JPEG_BAND_WORKER_BYTES = 16 * 1024 * 1024;

export type ParallelEnvironment = {
  hardwareConcurrency?: number;
  deviceMemory?: number;
  mobile?: boolean;
  // Workers can start nested workers (Safari 15.5+, Chromium, Firefox).
  nestedWorkers?: boolean;
};

type BrowserNavigator = {
  hardwareConcurrency?: number;
  deviceMemory?: number;
  userAgent?: string;
  userAgentData?: { mobile?: boolean };
};

// The same signals exist on Navigator and WorkerNavigator, so the main
// thread's admission charge and the worker's choice agree.
export function browserParallelEnvironment(nav: BrowserNavigator | undefined = globalThis.navigator): ParallelEnvironment {
  return {
    hardwareConcurrency: nav?.hardwareConcurrency,
    deviceMemory: nav?.deviceMemory,
    mobile: nav?.userAgentData?.mobile ?? /Mobi|Android|iPhone|iPad|iPod/i.test(nav?.userAgent ?? ''),
    nestedWorkers: typeof Worker === 'function',
  };
}

// Number of bands to encode for an image, counting the compression worker's
// own band; 1 means the serial encoder. Phones and low-memory devices stay
// serial: every band worker holds a second Go heap (#126).
export function jpegBandCount(pixels: number, env: ParallelEnvironment): number {
  if (env.nestedWorkers === false || pixels < PARALLEL_JPEG_MIN_PIXELS) return 1;
  if (env.mobile || (env.deviceMemory != null && env.deviceMemory <= 4)) return 1;
  const cores = env.hardwareConcurrency ?? 0;
  // Leave one core for the page. Two bands barely beat the serial entropy pass.
  const bands = Math.min(cores - 1, MAX_JPEG_BANDS);
  return bands >= 3 ? bands : 1;
}

// Extra admission bytes for a JPEG encode of this many pixels.
export function jpegParallelWorkspaceBytes(pixels: number, env: ParallelEnvironment): number {
  const bands = jpegBandCount(pixels, env);
  return bands > 1 ? pixels * JPEG_BAND_BYTES_PER_PIXEL + (bands - 1) * JPEG_BAND_WORKER_BYTES : 0;
}

export type JpegEncodeOptions = {
  quality: number;
  subsampling: number; // 0 = 4:2:0, 1 = 4:4:4
  progressive: boolean;
  trellis: boolean;
  optimizeHuffman: boolean;
  preset: number;
};

export type JpegBand = {
  firstRow: number; // first MCU row
  rows: number; // MCU rows
  y: number; // first pixel row
  height: number; // pixel rows
};

// Splits RGB MCU rows evenly into at most count bands, matching the Go
// encoder's MCU height (16 rows for 4:2:0, 8 for 4:4:4).
export function planJpegBands(width: number, height: number, subsampling: number, count: number): JpegBand[] {
  if (width < 1 || height < 1) return [];
  const mcuHeight = subsampling === 0 ? 16 : 8;
  const mcuRows = Math.ceil(height / mcuHeight);
  const bands = Math.max(1, Math.min(count, mcuRows));
  const plan: JpegBand[] = [];
  for (let i = 0, firstRow = 0; i < bands; i++) {
    const rows = Math.floor(mcuRows / bands) + (i < mcuRows % bands ? 1 : 0);
    const y = firstRow * mcuHeight;
    plan.push({ firstRow, rows, y, height: Math.min(height, (firstRow + rows) * mcuHeight) - y });
    firstRow += rows;
  }
  return plan;
}

export type JpegEncodeRuntime = {
  bands: number;
  // encodeJpegAdvanced on the composited RGB image.
  serial(rgb: Uint8Array): Uint8Array;
  // encodeJpegBand in this worker's Go instance.
  band(rgb: Uint8Array, band: JpegBand): Uint8Array;
  // Sends an owned RGBA band to band worker `index` and resolves with its
  // coefficients. Called for every remote band before this worker's band.
  remote(index: number, rgba: Uint8Array, band: JpegBand): Promise<Uint8Array>;
  // encodeJpegBands in this worker's Go instance.
  assemble(bands: Uint8Array[]): Uint8Array;
  // After the encode; failed means band workers must not be reused.
  release(failed: boolean): void;
  onFallback?(error: unknown): void;
};

// Encodes RGBA pixels as JPEG across band workers, falling back to the serial
// encoder on any band failure. Both paths produce identical bytes.
export async function encodeJpegRgba(
  rgba: Uint8Array, width: number, height: number, subsampling: number, runtime: JpegEncodeRuntime,
): Promise<Uint8Array> {
  const plan = runtime.bands > 1 ? planJpegBands(width, height, subsampling, runtime.bands) : [];
  if (plan.length < 2) return runtime.serial(compositeRgbaToRgbWhite(rgba, width, height));
  const rowBytes = width * 4;
  let failed = false;
  try {
    // Post every remote band before this worker blocks on its own band.
    const remote = Promise.all(plan.slice(1).map((band, i) =>
      runtime.remote(i, rgba.slice(band.y * rowBytes, (band.y + band.height) * rowBytes), band)));
    remote.catch(() => {}); // Observed below; avoid an unhandled rejection while this band runs.
    const first = plan[0];
    const local = runtime.band(compositeRgbaToRgbWhite(rgba.subarray(0, first.height * rowBytes), width, first.height), first);
    return runtime.assemble([local, ...await remote]);
  } catch (error) {
    failed = true;
    runtime.onFallback?.(error);
  } finally {
    runtime.release(failed);
  }
  return runtime.serial(compositeRgbaToRgbWhite(rgba, width, height));
}
