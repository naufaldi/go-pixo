import { compressionCandidates, resolveOutputFormat, selectCompressionResult, type CompressionResult, type InputFormat, type OutputFormatChoice, type ResolvedFormat } from './compressionSettings';
import { assertSafeDimensions, imageDisplayDimensions, inspectImage } from './imageFormats';

export type CandidateRequest = {
  format: InputFormat; outputFormat?: OutputFormatChoice; lossy: boolean;
  width: number; height: number; resized?: boolean; originalFileBytes?: Uint8Array;
};

export async function compressWithCandidates(
  request: CandidateRequest,
  encode: (format: ResolvedFormat, fast: boolean) => Promise<Uint8Array>,
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

  if (!request.lossy && (source?.bitDepth ?? 8)>8) {
    if (original) return original;
    throw new Error('Lossless high-bit-depth conversion or resizing is unavailable in the 8-bit pixel pipeline. Keep the original format or enable lossy compression.');
  }

  // The Go JPEG byte path re-encodes pixels. Native WebP has no lossless switch.
  // Keeping the original is the only honest same-format lossless option for these codecs.
  if (!request.lossy && original && choice!=='auto' && (input==='jpeg' || input==='webp')) return original;

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
  if (!candidates.length && !original) throw lastError ?? new Error('No supported image encoder');
  return selectCompressionResult(candidates,original,request.resized);
}
