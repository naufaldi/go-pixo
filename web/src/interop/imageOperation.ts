import { resolveOutputFormat, resolveResizeDimensions, type OutputFormatChoice } from './compressionSettings';
import { imageDisplayDimensions, MAX_IMAGE_PIXELS, type ImageInfo } from './imageFormats';

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
  const display = imageDisplayDimensions(info);
  const size = resolveResizeDimensions(display.width, display.height, settings?.targetWidth, settings?.targetHeight);
  const same = settings?.outputFormat !== 'auto' && resolveOutputFormat(info.format, settings?.outputFormat ?? 'same') === info.format;
  const bytesOnly = settings?.lossy === false && !size.resized && same;
  const kind = bytesOnly && (info.format === 'jpeg' || info.format === 'webp') ? 'original'
    : bytesOnly && info.format === 'png' ? 'png-bytes' : 'pixels';
  const sourcePixels = info.width! * info.height!; // Storage remains charged when downsizing.
  const outputPixels = size.width * size.height;
  return {
    kind, ...size,
    workspaceBytes: kind === 'original' ? 0 : WORKSPACE_BYTES_PER_PIXEL * Math.max(sourcePixels, outputPixels),
  } as const;
}
