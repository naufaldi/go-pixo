import { compressionCandidates, resolveOutputFormat, selectCompressionResult, type CompressionResult, type InputFormat, type OutputFormatChoice, type ResolvedFormat } from './compressionSettings';
import { assertSafeDimensions, imageDisplayDimensions, inspectImage } from './imageFormats';
import { assertDecodedDimensions, assertSameDecodedPixels, decodeNative } from './codecsNative';
import {decodeAvif} from './codecs';

// Heuristic: entropy recoding rarely shrinks a JPEG's non-metadata bytes by
// more than 20%, so lossy candidates below 80% of them skip the lossless pass.
const LOSSLESS_JPEG_WORTH_TRYING = 0.8;

// Bytes in APPn and COM segments before the first scan. The optimizer drops
// most of them, so they do not bound its output size.
function jpegMetadataBytes(bytes: Uint8Array): number {
  let total=0;
  for (let p=2; p+4<=bytes.length && bytes[p]===0xff;) {
    const marker=bytes[p+1], length=(bytes[p+2]<<8)|bytes[p+3];
    if (marker===0xda || marker===0xd9) break;
    if ((marker>=0xe0 && marker<=0xef) || marker===0xfe) total+=2+length;
    p+=2+length;
  }
  return total;
}

export type CandidateRequest = {
  format: InputFormat; outputFormat?: OutputFormatChoice; lossy: boolean;
  width: number; height: number; resized?: boolean; originalFileBytes?: Uint8Array;
};

export async function compressWithCandidates(
  request: CandidateRequest,
  encode: (format: ResolvedFormat, fast: boolean) => Promise<Uint8Array>,
  validateFallback?: (bytes: Uint8Array) => Promise<void>,
  optimizeJpegLossless?: (bytes: Uint8Array) => Promise<Uint8Array>,
): Promise<CompressionResult> {
  assertSafeDimensions(request.width,request.height);
  const source=request.originalFileBytes?.length ? inspectImage(request.originalFileBytes) : undefined;
  const sourceDisplay=source ? imageDisplayDimensions(source) : undefined;
  const input=source?.format ?? request.format;
  const choice=request.outputFormat ?? 'same';
  const encodable=['png','jpeg','webp','avif'].includes(input);
  const original: CompressionResult | undefined=!request.resized && sourceDisplay?.width===request.width &&
    sourceDisplay.height===request.height && encodable && request.originalFileBytes?.length &&
    (choice==='auto' || resolveOutputFormat(input,choice)===input)
      ? {compressedBytes:request.originalFileBytes,outputFormat:input as ResolvedFormat} : undefined;

  const validate=async (bytes: Uint8Array) => {
    if (input === 'avif') {
      const decoded = await decodeAvif(bytes);
      assertDecodedDimensions(source!, decoded.width, decoded.height);
    } else if (typeof createImageBitmap === 'function') {
      await decodeNative(bytes, true);
    } else if (validateFallback) {
      await validateFallback(bytes);
    } else {
      throw new Error('Image validation requires a supported decoder');
    }
  };
  // This worker boundary also receives byte-only requests; headers alone do not
  // establish that an original candidate contains a decodable image.
  if (original) await validate(original.compressedBytes);

  // Lossless JPEG optimization rewrites only the entropy coding, so decoded
  // pixels stay identical. It is the only real saving for a JPEG in Perfect
  // Quality mode, and a candidate in lossy mode when re-encoding saves little.
  const optimizeOriginal=async (): Promise<CompressionResult | undefined> => {
    if (!original || input!=='jpeg' || !optimizeJpegLossless) return undefined;
    try {
      const bytes=await optimizeJpegLossless(original.compressedBytes);
      if (bytes.length>=original.compressedBytes.length) return undefined;
      const info=inspectImage(bytes), display=imageDisplayDimensions(info);
      if (info.format!=='jpeg' || display.width!==request.width || display.height!==request.height) throw new Error('Optimized JPEG changed format or dimensions');
      // Decodable is not enough: decoders can disagree on unusual progressions.
      await assertSameDecodedPixels(original.compressedBytes,bytes);
      return {compressedBytes:bytes,outputFormat:'jpeg'};
    } catch (error) {
      // The validated original remains a correct result. Unsupported inputs
      // (CMYK, arithmetic coding) are expected to land here.
      console.debug('[compress] lossless JPEG optimization skipped:', error instanceof Error ? error.message : error);
      return undefined;
    }
  };

  if (!request.lossy && (source?.bitDepth ?? 8)>8) {
    if (original) return original;
    throw new Error('Lossless high-bit-depth conversion or resizing is unavailable in the 8-bit pixel pipeline. Keep the original format or enable lossy compression.');
  }

  // Pixel re-encoding is lossy for JPEG and native WebP, and AVIF bytes are kept
  // as-is. Same-format lossless output is the original or its optimized entropy coding.
  if (!request.lossy && original && choice!=='auto' && (input==='jpeg' || input==='webp' || input==='avif')) return await optimizeOriginal() ?? original;

  const formats=compressionCandidates(input,choice,!request.lossy);
  const candidates: CompressionResult[]=[];
  let lastError: unknown;
  for (const [index,format] of formats.entries()) {
    // An unchanged original is already a valid current-format candidate. Avoid slow
    // PNG quantization / AVIF re-encoding just to discover it loses to fast WebP.
    if (choice==='auto' && original && index>0) continue;
    try {
      const compressedBytes=await encode(format,choice==='auto');
      const actual=inspectImage(compressedBytes);
      if (actual.format!==format) throw new Error(`Encoder format/signature mismatch: expected ${format}, got ${actual.format}`);
      const display=imageDisplayDimensions(actual);
      if (display.width!==request.width || display.height!==request.height) throw new Error('Encoder returned incorrect image dimensions');
      candidates.push({compressedBytes,outputFormat:format});
    } catch (error) {
      if (choice!=='auto') throw error;
      lastError=error;
    }
  }
  // Skip the optimizer and its pixel check when a lossy candidate is already
  // far below what recoding the original's non-metadata bytes could reach.
  const smallest=Math.min(...candidates.map(candidate => candidate.compressedBytes.length));
  if (original && input==='jpeg' && smallest>(original.compressedBytes.length-jpegMetadataBytes(original.compressedBytes))*LOSSLESS_JPEG_WORTH_TRYING) {
    const optimized=await optimizeOriginal();
    if (optimized) candidates.push(optimized);
  }
  if (!candidates.length && !original) throw lastError ?? new Error('No supported image encoder');
  return selectCompressionResult(candidates,original,request.resized);
}
