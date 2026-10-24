import { resolveOutputFormat, resolveResizeDimensions, type OutputFormatChoice } from './compressionSettings';
import { assertSafeDimensions, MAX_IMAGE_PIXELS, type ImageInfo } from './imageFormats';

export type OperationSettings = {
  lossy: boolean;
  outputFormat?: OutputFormatChoice;
  targetWidth?: number;
  targetHeight?: number;
};

// Admission estimate: eight RGBA-sized surfaces cover decode/canvas, resize,
// worker input, codec/Go copies and workspace at the larger source/output size.
// This is not a bound on browser peak memory or retained WASM heap capacity.
const WORKSPACE_BYTES_PER_PIXEL = 32;
export const WORKSPACE_BUDGET = 384 * 1024 * 1024;
export const MAX_SINGLE_WORKSPACE = MAX_IMAGE_PIXELS * WORKSPACE_BYTES_PER_PIXEL;

export function imageOperation(info: ImageInfo, settings?: OperationSettings) {
  const {width, height} = info;
  if (width === undefined || height === undefined) throw new Error('Image needs bounded dimensions before admission');
  assertSafeDimensions(width, height);
  const size = resolveResizeDimensions(width, height, settings?.targetWidth, settings?.targetHeight);
  const same = settings?.outputFormat !== 'auto' && resolveOutputFormat(info.format, settings?.outputFormat ?? 'same') === info.format;
  const bytesOnly = settings?.lossy === false && !size.resized && same;
  const kind = bytesOnly && (info.format === 'jpeg' || info.format === 'webp') ? 'original'
    : bytesOnly && info.format === 'png' ? 'png-bytes' : 'pixels';
  let outputPixels = size.width * size.height;
  // Native JPEG decode may apply EXIF orientation. A single target axis can
  // then produce a different output area. Reserve both possible orientations;
  // outputs beyond 24 MP will be rejected by the worker before resizing.
  if (kind === 'pixels' && info.format === 'jpeg' && (settings?.targetWidth !== undefined || settings?.targetHeight !== undefined)) {
    try {
      const rotated = resolveResizeDimensions(height, width, settings?.targetWidth, settings?.targetHeight);
      outputPixels = Math.max(outputPixels, rotated.width * rotated.height);
    } catch { outputPixels = MAX_IMAGE_PIXELS; }
  }
  return {
    kind, ...size,
    workspaceBytes: kind === 'original' ? 0 : WORKSPACE_BYTES_PER_PIXEL * Math.max(width * height, outputPixels),
  } as const;
}
