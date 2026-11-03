import {
  resolveResizeDimensions,
  type InputFormat,
  type OutputFormatChoice,
} from './interop/compressionSettings';
import { compressWithCandidates } from './interop/codecsCompression';
import { encodeAvif, encodeWebp } from './interop/codecs';
import { INITIAL_PROGRESS, STAGE, createProgressThrottle, mapPngPhaseToGlobal } from './interop/progress';
import { unwrapWasmResult } from './interop/wasmResult';
import { assertRgbaPixelBuffer, ownedImageDataPixels } from './interop/pixels';
import { imageDisplayDimensions, inspectImage } from './interop/imageFormats';
import { imageOperation } from './interop/imageOperation';
import { resolveJpegEncodeOptions } from './interop/jpegOptions';
import { browserParallelEnvironment, encodeJpegRgba, jpegBandCount, type JpegEncodeOptions, type JpegEncodeRuntime } from './interop/jpegParallel';
import { JpegBandWorkers, type GoModuleSource } from './jpegBandWorkers';

interface CompressionRequest {
  id: string;
  pixels?: Uint8Array;
  width: number;
  height: number;
  colorType: number;
  format: InputFormat;
  outputFormat?: OutputFormatChoice;
  preset: number;
  lossy: boolean;
  maxColors: number;
  // Preset intent is explicit; custom quantizer requests remain strict.
  pngAllowRGBAFallback?: boolean;
  dithering: boolean;
  ditherStrength?: number;
  qualityTarget?: number;
  zopfliIterations?: number;
  // JPEG-specific options
  progressive?: boolean;
  subsampling?: '420' | '444';
  trellis?: boolean;
  optimizeHuffman?: boolean;
  originalFileBytes?: Uint8Array; // Original PNG file bytes for "never larger" guard
  targetWidth?: number;
  targetHeight?: number;
}

interface CompressionResponse {
  id: string;
  compressedBytes: Uint8Array;
  originalBytes: number;
  compressedBytesCount: number;
  ratio: number;
  outputFormat: 'png' | 'jpeg' | 'webp' | 'avif';
  error?: string;
}

interface ProgressMessage {
  type: "progress";
  id: string;
  phase: string;
  progress: number;
  predictable?: boolean;
  phaseTarget?: number;
  phasePercent?: number;
}

// Go loads only for PNG/JPEG operations. Reuse in-flight and settled startup.
let wasmInitialization: Promise<void> | undefined;
class GoInitializationError extends Error {}
function initWasm(): Promise<void> {
  return wasmInitialization ??= new Promise<void>((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('Go WASM initialization timed out')), 30_000);
    initializeGo().then(resolve, reject).finally(() => clearTimeout(timer));
  }).catch(error => {
    throw new GoInitializationError(error instanceof Error ? error.message : 'Failed to initialize Go WASM');
  });
}

// The compiled module and wasm_exec.js let JPEG band workers start their own
// Go instances without fetching or compiling main.wasm again.
let goModuleSource: GoModuleSource | undefined;
let wasmExecSource: string | undefined;
let jpegBandWorkers: JpegBandWorkers | undefined;

// Initialize WASM in worker
async function initializeGo(): Promise<void> {

  // This worker is created as an ES module worker.
  // Module workers do NOT support importScripts(), so we load wasm_exec.js via dynamic import.
  // wasm_exec.js attaches Go to globalThis as a side effect.
  // @ts-ignore
  if (typeof (self as any).Go === "undefined") {
    console.debug("[worker] Loading /wasm_exec.js");
    // wasm_exec.js is shipped as a classic script (not an ES module).
    // In a module worker we can't use importScripts(), and Vite tries to bundle `import('/wasm_exec.js')`.
    // Fetch + indirect eval loads it into the worker global scope.
    const wasmExecJs = await fetch("/wasm_exec.js").then((r) => r.text());
    // eslint-disable-next-line no-eval
    (0, eval)(wasmExecJs);
    wasmExecSource = wasmExecJs;
  }

  // @ts-ignore
  const go = new (self as any).Go();

  return new Promise((resolve, reject) => {
    // Set up initialization callback called from Go's main()
    (self as any).goWasmInit = () => {
      console.debug("[worker] Go WASM initialized");
      resolve();
    };

    console.debug("[worker] Fetching /main.wasm");
    fetch("/main.wasm")
      .then((response) => {
        if (!response.ok) throw new Error(`Failed to load Go WASM (${response.status})`);
        return response.arrayBuffer();
      })
      .then((buffer) => WebAssembly.instantiate(buffer, go.importObject))
      .then((result) => {
        console.debug("[worker] Running WASM instance");
        if (result.module && wasmExecSource) goModuleSource = {module: result.module, wasmExec: wasmExecSource};
        go.run(result.instance);
      })
      .catch((err) => {
        console.error("[worker] Failed to load WASM:", err);
        reject(err);
      });
  });
}

// Encode PNG using WASM with basic parameters
function encodePng(
  pixels: Uint8Array,
  width: number,
  height: number,
  colorType: number = 6,
  preset: number = 1,
  lossy: boolean = false,
  maxColors: number = 0,
  _dithering: boolean = false,
  allowRGBAFallback: boolean = true,
): Uint8Array {
  // @ts-ignore - encodePng is exposed by Go WASM on the worker global
  if (!(self as any).encodePng) {
    throw new Error("WASM not initialized");
  }

  // @ts-ignore - encodePng is exposed by Go WASM on the worker global
  const result = (self as any).encodePng(
    pixels,
    width,
    height,
    colorType,
    preset,
    lossy,
    maxColors,
    allowRGBAFallback,
  );

  return unwrapWasmResult(result);
}

// Encode PNG using WASM with advanced parameters
function encodePngAdvanced(
  pixels: Uint8Array,
  width: number,
  height: number,
  colorType: number = 6,
  preset: number = 1,
  lossy: boolean = false,
  maxColors: number = 0,
  dithering: boolean = false,
  ditherStrength: number = 0.5,
  qualityTarget: number = 75,
  zopfliIterations: number = 0,
  filterStrategy: number = 0,
  usePerceptual: boolean = false,
  distanceMetric: number = 0,
  optimalDeflate: boolean = false,
  filterEarlyTerm: boolean = false,
  onProgress?: (phase: string, progress: number) => void,
  allowRGBAFallback: boolean = false,
): Uint8Array {
  // @ts-ignore - encodePngAdvanced is exposed by Go WASM on the worker global
  if (!(self as any).encodePngAdvanced) {
    // Fallback to basic encodePng if advanced not available
    console.debug(
      "[worker] encodePngAdvanced not available, falling back to encodePng",
    );
    return encodePng(
      pixels,
      width,
      height,
      colorType,
      preset,
      lossy,
      maxColors,
      dithering,
      allowRGBAFallback,
    );
  }

  // @ts-ignore - encodePngAdvanced is exposed by Go WASM on the worker global
  const result = (self as any).encodePngAdvanced(
    pixels,
    width,
    height,
    colorType,
    preset,
    lossy,
    maxColors,
    dithering,
    ditherStrength,
    qualityTarget,
    zopfliIterations,
    filterStrategy,
    usePerceptual,
    distanceMetric,
    optimalDeflate,
    filterEarlyTerm,
    onProgress,
    allowRGBAFallback,
  );

  return unwrapWasmResult(result);
}

// Encode JPEG using WASM with advanced parameters
function encodeJpegAdvanced(
  pixels: Uint8Array,
  width: number,
  height: number,
  colorType: number = 3,
  quality: number = 85,
  subsampling: number = 0, // 0 = 420, 1 = 444
  progressive: boolean = false,
  trellis: boolean = false,
  optimizeHuffman: boolean = false,
  preset: number = 1,
): Uint8Array {
  // @ts-ignore - encodeJpegAdvanced is exposed by Go WASM on the worker global
  if (!(self as any).encodeJpegAdvanced) {
    throw new Error("encodeJpegAdvanced not available");
  }

  // @ts-ignore - encodeJpegAdvanced is exposed by Go WASM on the worker global
  const result = (self as any).encodeJpegAdvanced(
    pixels,
    width,
    height,
    colorType,
    quality,
    subsampling,
    progressive,
    trellis,
    optimizeHuffman,
    preset,
  );

  return unwrapWasmResult(result);
}

function recompressPngLossless(
  pngBytes: Uint8Array,
  preset: number,
  zopfliIterations: number = 0,
  onProgress?: (phase: string, progress: number) => void,
): Uint8Array {
  // @ts-ignore - recompressPngLossless is exposed by Go WASM on the worker global
  if (!(self as any).recompressPngLossless) {
    throw new Error("recompressPngLossless not available");
  }

  // @ts-ignore - recompressPngLossless is exposed by Go WASM on the worker global
  const result = (self as any).recompressPngLossless(
    pngBytes,
    preset,
    zopfliIterations,
    onProgress,
  );

  return unwrapWasmResult(result);
}

type JpegBandExports = {
  encodeJpegBand?: (...args: unknown[]) => Uint8Array | string;
  encodeJpegBands?: (...args: unknown[]) => Uint8Array | string;
};

// Encodes RGBA pixels as JPEG, across band workers for large images on
// capable devices (#128). Any band failure falls back to the serial encoder.
function encodeJpeg(pixels: Uint8Array, width: number, height: number, options: JpegEncodeOptions): Promise<Uint8Array> {
  const go = self as unknown as JpegBandExports;
  const source = goModuleSource;
  const bands = source && go.encodeJpegBand && go.encodeJpegBands
    ? jpegBandCount(width * height, browserParallelEnvironment()) : 1;
  const workers = bands > 1 && source ? (jpegBandWorkers ??= new JpegBandWorkers(source)) : undefined;
  const args = [width, height, 3, options.quality, options.subsampling, options.progressive, options.trellis,
    options.optimizeHuffman, options.preset] as const;
  const runtime: JpegEncodeRuntime = {
    bands,
    serial: rgb => encodeJpegAdvanced(rgb, ...args),
    band: (rgb, band) => unwrapWasmResult(go.encodeJpegBand!(rgb, ...args, band.firstRow, band.rows)),
    remote: (index, rgba, band) => workers!.encode(index, rgba, width, height, band, options),
    assemble: coefficients => unwrapWasmResult(go.encodeJpegBands!(coefficients, ...args)),
    release: () => workers?.release(),
    onFallback: error => console.warn('[worker] Parallel JPEG encode failed; encoding serially', error),
  };
  return encodeJpegRgba(pixels, width, height, options.subsampling, runtime);
}

function optimizeJpegLossless(jpegBytes: Uint8Array): Uint8Array {
  const optimize = (self as unknown as {optimizeJpegLossless?: (bytes: Uint8Array) => Uint8Array | string}).optimizeJpegLossless;
  if (!optimize) throw new Error("optimizeJpegLossless not available");
  return unwrapWasmResult(optimize(jpegBytes));
}

function resizePixels(
  pixels: Uint8Array,
  srcW: number,
  srcH: number,
  dstW: number,
  dstH: number
): { pixels: Uint8Array; width: number; height: number } {
  const src = new OffscreenCanvas(srcW, srcH);
  let dst: OffscreenCanvas | undefined;
  try {
    const srcCtx = src.getContext('2d');
    if (!srcCtx) throw new Error('Canvas is unavailable');
    const imageData = new ImageData(
      new Uint8ClampedArray(pixels.buffer as ArrayBuffer, pixels.byteOffset, pixels.byteLength), srcW, srcH,
    );
    srcCtx.putImageData(imageData, 0, 0);
    dst = new OffscreenCanvas(dstW, dstH);
    const dstCtx = dst.getContext('2d');
    if (!dstCtx) throw new Error('Canvas is unavailable');
    dstCtx.drawImage(src, 0, 0, dstW, dstH);
    const resized = dstCtx.getImageData(0, 0, dstW, dstH);
    return {pixels: ownedImageDataPixels(resized.data), width: dstW, height: dstH};
  } finally {
    src.width = 0; src.height = 0;
    if (dst) { dst.width = 0; dst.height = 0; }
  }
}

// Every codec result is selected together with its actual format.
self.onmessage = async (event: MessageEvent) => {
  const msg=event.data;
  if (msg?.type==='init') {
    // Readiness covers the worker transport. Required codec startup is covered
    // by the pool's per-task deadline and isolated worker failure recovery.
    self.postMessage({type:'ready'});
    return;
  }
  if (msg?.type!=='compress') return;
  const req: CompressionRequest=msg.data ?? msg;
  if (!req || typeof req.id!=='string') {
    self.postMessage({type:'error',error:'Invalid compress message'}); return;
  }
  const postProgress=(phase: string,progress: number,options?: {predictable?: boolean; phaseTarget?: number; phasePercent?: number})=>{
    self.postMessage({type:'progress',id:req.id,phase,progress,...options} as ProgressMessage);
  };
  try {
    console.debug('[worker] compress start', req.id);
    const size=resolveResizeDimensions(req.width,req.height,req.targetWidth,req.targetHeight);
    const source = !req.pixels && req.originalFileBytes?.length ? inspectImage(req.originalFileBytes) : undefined;
    const operation = source ? imageOperation(source, req) : undefined;
    const sourceDisplay = source ? imageDisplayDimensions(source) : undefined;
    if (req.pixels) assertRgbaPixelBuffer(req.pixels,req.width,req.height,req.colorType);
    else if (!operation || operation.kind === 'pixels' || sourceDisplay?.width !== req.width || sourceDisplay?.height !== req.height) {
      throw new Error('Pixel data is required for this image operation');
    }
    const quality=req.qualityTarget ?? 85;
    if (!Number.isFinite(quality) || quality<0 || quality>100) throw new Error('Quality must be between 0 and 100');
    postProgress(STAGE.PREPARING.label,INITIAL_PROGRESS);
    let work: {pixels?: Uint8Array; width: number; height: number} = {pixels:req.pixels,width:req.width,height:req.height};
    const requirePixels = (): Uint8Array => {
      if (!work.pixels) throw new Error('Pixel data is required for this image operation');
      return work.pixels;
    };
    if (size.resized) {
      postProgress(STAGE.RESIZING.label,STAGE.RESIZING.start);
      work=resizePixels(requirePixels(),req.width,req.height,size.width,size.height);
    }
    const shouldPostPngProgress = createProgressThrottle(() => performance.now());
    const pngProgress=(phase: string,progress: number)=>{
      if (!shouldPostPngProgress(phase,progress)) return;
      postProgress(phase,mapPngPhaseToGlobal(phase,progress),{
      // Only these phases expose completed units. Boundary-only and legacy
      // deflate callbacks stay indeterminate until a real pass finishes.
      phasePercent: ['histogram','palette','quantize','filtering'].includes(phase) || phase === 'deflate' && progress > 0 ? progress : undefined,
      phaseTarget: mapPngPhaseToGlobal(phase,100),
      });
    };
    // The Faster preset (2) favors speed, so lossy mode skips the extra lossless
    // pass and its pixel check. Perfect Quality always tries it.
    const optimizeOriginalJpeg=req.lossy && req.preset===2 ? undefined : async (bytes: Uint8Array) => {
      await initWasm();
      postProgress('optimizing-jpeg',STAGE.ENCODING.start);
      return optimizeJpegLossless(bytes);
    };
    const result=await compressWithCandidates({
      ...req,width:work.width,height:work.height,resized:size.resized,
    },async (format,fast)=>{
      postProgress('initializing',STAGE.ENCODING.start);
      if (format==='webp' || format==='avif') postProgress(`encoding-${format}`,STAGE.ENCODING.start);
      if (format==='webp') return encodeWebp(requirePixels(),work.width,work.height,quality);
      if (format==='avif') return encodeAvif(requirePixels(),work.width,work.height,quality,!req.lossy,fast?8:req.preset===2?8:req.preset===4?5:6);
      await initWasm();
      postProgress(`encoding-${format}`,STAGE.ENCODING.start);
      if (format==='jpeg') {
        const options=resolveJpegEncodeOptions({
          inputFormat:req.format,outputFormat:format,width:work.width,height:work.height,preset:req.preset,
          progressive:req.progressive ?? false,trellis:req.trellis ?? false,optimizeHuffman:req.optimizeHuffman ?? false,
        });
        return encodeJpeg(requirePixels(),work.width,work.height,{quality,subsampling:req.subsampling==='444'?1:0,
          progressive:options.progressive,trellis:options.trellis,optimizeHuffman:options.optimizeHuffman,preset:options.preset});
      }
      if (!req.lossy && !size.resized && req.format==='png' && req.originalFileBytes?.length) {
        return recompressPngLossless(req.originalFileBytes,req.preset,req.zopfliIterations ?? 0,pngProgress);
      }
      // Auto's PNG fallback is a quick lossless encode, with no palette search.
      if (fast) return encodePng(requirePixels(),work.width,work.height,6,2,false,0,false);
      return encodePngAdvanced(requirePixels(),work.width,work.height,6,req.preset,req.lossy,req.maxColors,req.dithering,
        req.ditherStrength ?? 0.5,quality,req.zopfliIterations ?? 0,0,false,0,false,false,pngProgress,req.pngAllowRGBAFallback === true);
    }, async bytes => {
      await initWasm();
      const validator = (self as unknown as {validateImageBytes(bytes: Uint8Array): boolean | string}).validateImageBytes;
      const result = validator(bytes);
      if (result !== true) throw new Error(typeof result === 'string' ? result.replace(/^error:\s*/, '') : 'Invalid image data');
    }, optimizeOriginalJpeg);
    postProgress(STAGE.FINALIZING.label,STAGE.FINALIZING.end);
    const originalBytes=req.originalFileBytes?.length || req.pixels?.length || 0;
    // Codec/Go boundaries have already copied borrowed memory. The selected
    // output (including an unchanged input candidate) belongs to this worker.
    const outputBuffer=result.compressedBytes.buffer;
    if (!(outputBuffer instanceof ArrayBuffer)) throw new Error('Compression output must own a transferable buffer');
    self.postMessage({type:'compressed',id:req.id,...result,originalBytes,compressedBytesCount:result.compressedBytes.length,
      ratio:originalBytes ? result.compressedBytes.length/originalBytes*100 : 0} as CompressionResponse,
      {transfer: [outputBuffer]});
  } catch(error) {
    self.postMessage({type:'error',id:error instanceof GoInitializationError ? undefined : req.id,
      error:error instanceof Error ? error.message : 'Compression failed'});
  }
};
