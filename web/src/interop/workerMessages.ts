export type CompressPayload = {
  id: string
  pixels: Uint8Array | undefined
  width: number
  height: number
  colorType: number
  format: string
  outputFormat: string
  preset: number
  lossy: boolean
  maxColors: number
  pngAllowRGBAFallback?: boolean
  dithering: boolean
  ditherStrength: number
  qualityTarget: number
  zopfliIterations: number
  progressive: boolean
  trellis: boolean
  subsampling: string
  optimizeHuffman: boolean
  originalFileBytes: Uint8Array
  targetWidth?: number
  targetHeight?: number
}

export type CompressMessage = CompressPayload & {
  type: 'compress'
}

export type ParsedWorkerMessage =
  | { type: 'ready' }
  | {
      type: 'progress'
      id: string
      phase: string
      progress: number
      predictable: boolean
      phaseTarget?: number
    }
  | {
      type: 'compressed'
      id: string
      compressedBytes: Uint8Array
      outputFormat: string
    }
  | { type: 'error'; id?: string; error: string }
  | { type: 'unknown' }

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

// Worker messages already own their received storage. Preserve view offsets
// instead of constructing a second full output allocation.
export function receivedBytes(value: unknown): Uint8Array {
  if (value instanceof Uint8Array) return value;
  if (ArrayBuffer.isView(value)) return new Uint8Array(value.buffer, value.byteOffset, value.byteLength);
  if (value instanceof ArrayBuffer) return new Uint8Array(value);
  throw new Error('Invalid compression worker bytes');
}

export function buildCompressMessage(payload: CompressPayload): CompressMessage {
  return { type: 'compress', ...payload }
}

export function buildCompressMessageFromFields(
  id: string,
  pixels: Uint8Array | undefined,
  width: number,
  height: number,
  colorType: number,
  format: string,
  outputFormat: string,
  preset: number,
  lossy: boolean,
  maxColors: number,
  dithering: boolean,
  ditherStrength: number,
  qualityTarget: number,
  zopfliIterations: number,
  progressive: boolean,
  trellis: boolean,
  subsampling: string,
  optimizeHuffman: boolean,
  originalFileBytes: Uint8Array,
  targetWidth?: number,
  targetHeight?: number,
  pngAllowRGBAFallback = false,
): CompressMessage {
  return buildCompressMessage({
    id,
    pixels,
    width,
    height,
    colorType,
    format,
    outputFormat,
    preset,
    lossy,
    maxColors,
    dithering,
    ditherStrength,
    qualityTarget,
    zopfliIterations,
    progressive,
    trellis,
    subsampling,
    optimizeHuffman,
    originalFileBytes,
    targetWidth,
    targetHeight,
    pngAllowRGBAFallback,
  })
}

export function parseWorkerMessage(value: unknown): ParsedWorkerMessage {
  if (!isRecord(value) || typeof value.type !== 'string') {
    return { type: 'unknown' }
  }

  switch (value.type) {
    case 'ready':
      return { type: 'ready' }

    case 'progress':
      if (
        typeof value.id !== 'string' ||
        typeof value.phase !== 'string' ||
        typeof value.progress !== 'number'
      ) {
        return { type: 'unknown' }
      }

      return {
        type: 'progress',
        id: value.id,
        phase: value.phase,
        progress: Math.round(value.progress),
        predictable: value.predictable === true,
        phaseTarget:
          typeof value.phaseTarget === 'number'
            ? Math.round(value.phaseTarget)
            : undefined,
      }

    case 'compressed':
      if (typeof value.id !== 'string' || value.compressedBytes == null) {
        return { type: 'unknown' }
      }

      try { return {
        type: 'compressed',
        id: value.id,
        compressedBytes: receivedBytes(value.compressedBytes),
        outputFormat: typeof value.outputFormat === 'string' ? value.outputFormat : 'png',
      } } catch { return {type: 'unknown'} }

    case 'error':
      return {
        type: 'error',
        id: typeof value.id === 'string' ? value.id : undefined,
        error: typeof value.error === 'string' ? value.error : String(value.error),
      }

    default:
      return { type: 'unknown' }
  }
}
