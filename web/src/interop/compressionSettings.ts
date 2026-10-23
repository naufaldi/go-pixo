import { assertSafeDimensions } from './imageFormats';

export type ResolvedFormat = 'png' | 'jpeg' | 'webp' | 'avif';

export type OutputFormatChoice =
  | 'same'
  | 'auto'
  | 'png'
  | 'jpeg'
  | 'webp'
  | 'avif';

export type InputFormat = 'png' | 'jpeg' | 'webp' | 'avif' | 'gif' | 'bmp' | 'svg' | 'heic' | 'tiff' | 'unknown';

export function resolveOutputFormat(
  inputFormat: InputFormat,
  outputFormat: OutputFormatChoice,
): ResolvedFormat {
  if (outputFormat === 'same' || outputFormat === 'auto') {
    if (inputFormat === 'jpeg') return 'jpeg';
    if (inputFormat === 'webp') return 'webp';
    if (inputFormat === 'avif') return 'avif';
    return 'png';
  }
  return outputFormat;
}

export function compressionCandidates(input: InputFormat, output: OutputFormatChoice, lossless: boolean): ResolvedFormat[] {
  const same=resolveOutputFormat(input,'same');
  if (output==='auto') {
    const modern=lossless ? 'avif' : 'webp';
    const current=lossless && (same==='jpeg' || same==='webp') ? 'png' : same;
    return modern===current ? [modern] : [modern,current];
  }
  const resolved=resolveOutputFormat(input,output);
  if (lossless && (resolved==='jpeg' || resolved==='webp')) {
    throw new Error(`${resolved.toUpperCase()} encoding is lossy. Choose PNG or AVIF, or enable lossy compression.`);
  }
  return [resolved];
}

export function resolveResizeDimensions(width: number, height: number, targetWidth?: number, targetHeight?: number) {
  assertSafeDimensions(width,height);
  if (targetWidth!==undefined && (!Number.isSafeInteger(targetWidth) || targetWidth<=0)) throw new Error('Invalid resize width');
  if (targetHeight!==undefined && (!Number.isSafeInteger(targetHeight) || targetHeight<=0)) throw new Error('Invalid resize height');
  const w=targetWidth ?? (targetHeight===undefined ? width : Math.max(1,Math.round(targetHeight*width/height)));
  const h=targetHeight ?? (targetWidth===undefined ? height : Math.max(1,Math.round(targetWidth*height/width)));
  assertSafeDimensions(w,h);
  return {width:w,height:h,resized:w!==width || h!==height};
}

export type CompressionResult = {compressedBytes: Uint8Array; outputFormat: ResolvedFormat};

export function selectCompressionResult(candidates: CompressionResult[], original?: CompressionResult, resized = false): CompressionResult {
  const results=original && !resized ? [original,...candidates] : candidates;
  const best=results.reduce<CompressionResult | undefined>((best,result) =>
    result.compressedBytes.length>0 && (!best || result.compressedBytes.length<best.compressedBytes.length) ? result : best, undefined);
  if (!best) throw new Error('No valid compression result');
  return best;
}

export function mimeForFormat(format: ResolvedFormat): string {
  switch (format) {
    case 'jpeg':
      return 'image/jpeg';
    case 'webp':
      return 'image/webp';
    case 'avif':
      return 'image/avif';
    default:
      return 'image/png';
  }
}

export function extensionForFormat(format: ResolvedFormat): string {
  switch (format) {
    case 'jpeg':
      return '.jpg';
    case 'webp':
      return '.webp';
    case 'avif':
      return '.avif';
    default:
      return '.png';
  }
}

export function replaceExtension(filename: string, extension: string): string {
  const dot = filename.lastIndexOf('.');
  const base = dot > 0 ? filename.slice(0, dot) : filename;
  return `${base}${extension}`;
}

export function buildCompressedFilename(
  originalName: string,
  extension: string,
): string {
  const dot = originalName.lastIndexOf('.');
  const base = dot > 0 ? originalName.slice(0, dot) : originalName;
  return `compressed_${base}${extension}`;
}

export function shouldUseLosslessBytePath(options: {
  lossless: boolean;
  inputFormat: InputFormat;
  outputFormat: ResolvedFormat;
  targetWidth?: number;
  targetHeight?: number;
}): boolean {
  if (!options.lossless) return false;
  if (options.targetWidth != null || options.targetHeight != null) return false;
  if (options.outputFormat !== options.inputFormat) return false;
  return options.inputFormat === 'png' || options.inputFormat === 'jpeg';
}

export type PresetKey = 'ultra' | 'smaller' | 'balanced' | 'faster';

export type DerivedPresetSettings = {
  presetInt: number;
  qualityTarget: number;
  maxColors: number;
  zopfliIterations: number;
  progressive: boolean;
  trellis: boolean;
  optimizeHuffman: boolean;
  subsampling: '420' | '444';
  dithering: boolean;
  ditherStrength: number;
};

export function derivePresetSettings(
  preset: PresetKey,
  lossless: boolean,
): DerivedPresetSettings {
  if (lossless) {
    return {
      presetInt: preset === 'ultra' ? 4 : preset === 'smaller' ? 0 : preset === 'faster' ? 2 : 1,
      qualityTarget: 100,
      maxColors: 0,
      zopfliIterations: preset === 'ultra' ? 20 : preset === 'smaller' ? 10 : 0,
      progressive: false,
      trellis: false,
      optimizeHuffman: preset === 'smaller' || preset === 'ultra',
      subsampling: '444',
      dithering: false,
      ditherStrength: 0,
    };
  }

  switch (preset) {
    case 'ultra':
      return {
        presetInt: 4,
        qualityTarget: 72,
        maxColors: 128,
        zopfliIterations: 15,
        progressive: true,
        trellis: true,
        optimizeHuffman: true,
        subsampling: '420',
        dithering: true,
        ditherStrength: 0.45,
      };
    case 'smaller':
      return {
        presetInt: 0,
        qualityTarget: 80,
        maxColors: 256,
        zopfliIterations: 10,
        progressive: true,
        trellis: true,
        optimizeHuffman: true,
        subsampling: '420',
        dithering: true,
        ditherStrength: 0.5,
      };
    case 'faster':
      return {
        presetInt: 2,
        qualityTarget: 88,
        maxColors: 256,
        zopfliIterations: 0,
        progressive: false,
        trellis: false,
        optimizeHuffman: false,
        subsampling: '420',
        dithering: false,
        ditherStrength: 0,
      };
    case 'balanced':
    default:
      return {
        presetInt: 1,
        qualityTarget: 85,
        maxColors: 256,
        zopfliIterations: 5,
        progressive: true,
        trellis: true,
        optimizeHuffman: true,
        subsampling: '420',
        dithering: true,
        ditherStrength: 0.4,
      };
  }
}

export function presetKeyFromInt(value: number): PresetKey {
  switch (value) {
    case 0:
      return 'smaller';
    case 2:
      return 'faster';
    case 4:
      return 'ultra';
    default:
      return 'balanced';
  }
}
