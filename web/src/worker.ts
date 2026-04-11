// Web Worker for off-main-thread image compression

interface CompressionRequest {
  id: string;
  pixels: Uint8Array;
  width: number;
  height: number;
  colorType: number;
  format: 'png' | 'jpeg';
  outputFormat?: 'same' | 'png' | 'jpeg' | 'webp';
  preset: number;
  lossy: boolean;
  maxColors: number;
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
}

interface CompressionResponse {
  id: string;
  compressedBytes: Uint8Array;
  originalBytes: number;
  compressedBytesCount: number;
  ratio: number;
  error?: string;
}

interface ProgressMessage {
  type: "progress";
  id: string;
  phase: string;
  progress: number;
}

// WASM state
let wasmReady = false;

// Initialize WASM in worker
async function initWasm(): Promise<void> {
  if (wasmReady) return;

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
  }

  // @ts-ignore
  const go = new (self as any).Go();

  return new Promise((resolve, reject) => {
    // Set up initialization callback called from Go's main()
    (self as any).goWasmInit = () => {
      wasmReady = true;
      console.debug("[worker] Go WASM initialized");
      resolve();
    };

    console.debug("[worker] Fetching /main.wasm");
    fetch("/main.wasm")
      .then((response) => response.arrayBuffer())
      .then((buffer) => WebAssembly.instantiate(buffer, go.importObject))
      .then((result) => {
        console.debug("[worker] Running WASM instance");
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
  );

  if (typeof result === "string" && result.startsWith("error:")) {
    throw new Error(result);
  }

  return result as Uint8Array;
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
  onProgress?: (phase: string, progress: number) => void,
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
    onProgress,
  );

  if (typeof result === "string" && result.startsWith("error:")) {
    throw new Error(result);
  }

  return result as Uint8Array;
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

  if (typeof result === "string" && result.startsWith("error:")) {
    throw new Error(result);
  }

  return result as Uint8Array;
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

  if (typeof result === "string" && result.startsWith("error:")) {
    throw new Error(result);
  }

  return result as Uint8Array;
}

function recompressJpeg(
  jpegBytes: Uint8Array,
  preset: number,
  quality: number = 85,
  progressive: boolean = false,
  trellis: boolean = false,
  optimizeHuffman: boolean = false,
): Uint8Array {
  // @ts-ignore - recompressJpeg is exposed by Go WASM on the worker global
  if (!(self as any).recompressJpeg) {
    throw new Error("recompressJpeg not available");
  }

  // @ts-ignore
  const result = (self as any).recompressJpeg(
    jpegBytes,
    preset,
    quality,
    progressive,
    trellis,
    optimizeHuffman,
  );

  if (typeof result === "string" && result.startsWith("error:")) {
    throw new Error(result);
  }

  return result as Uint8Array;
}

// Encode image as WebP using OffscreenCanvas
async function encodeWebp(pixels: Uint8Array, width: number, height: number, quality: number): Promise<Uint8Array> {
  const canvas = new OffscreenCanvas(width, height);
  const ctx = canvas.getContext('2d')!;
  // pixels is RGBA from the decoder
  const imageData = new ImageData(new Uint8ClampedArray(pixels.buffer as ArrayBuffer, pixels.byteOffset, pixels.byteLength), width, height);
  ctx.putImageData(imageData, 0, 0);
  const blob = await canvas.convertToBlob({ type: 'image/webp', quality: quality / 100 });
  const arrayBuffer = await blob.arrayBuffer();
  return new Uint8Array(arrayBuffer);
}

// Handle messages from main thread
self.onmessage = async (
  event: MessageEvent<
    | {
        type: "compress";
        data?: CompressionRequest;
      }
    | ({
        type: "compress";
      } & CompressionRequest)
    | {
        type: "init";
      }
  >,
) => {
  const msg: any = event.data as any;
  const type: string = msg?.type;
  console.debug("[worker] message", type, msg?.id ?? msg?.data?.id ?? null);

  switch (type) {
    case "init":
      try {
        console.debug("[worker] init requested");
        await initWasm();
        self.postMessage({ type: "ready" });
      } catch (err) {
        self.postMessage({
          type: "error",
          error:
            err instanceof Error ? err.message : "Failed to initialize WASM",
        });
      }
      break;

    case "compress":
      // Support both `{type:'compress', data:{...}}` and `{type:'compress', ...fields}`
      const req: CompressionRequest | undefined = msg?.data ?? msg;

      if (!req || typeof req.id !== "string") {
        self.postMessage({
          type: "error",
          error: "Invalid compress message",
        });
        return;
      }

      if (!wasmReady) {
        self.postMessage({
          type: "error",
          id: req.id,
          error: "WASM not initialized",
        });
        return;
      }

      // Wrap in async IIFE to support await (needed for WebP encoding)
      (async () => {
        console.debug("[worker] compress start", req.id);
        const startTime = Date.now();
        const originalFileBytesLen = req.originalFileBytes?.length ?? 0;
        const originalBytes =
          originalFileBytesLen > 0 ? originalFileBytesLen : req.pixels.length;

        // Resolve the effective output format
        const resolvedFormat = (() => {
          if (!req.outputFormat || req.outputFormat === 'same') return req.format;
          return req.outputFormat;
        })();

        let compressedBytes: Uint8Array;

        // Handle WebP encoding via OffscreenCanvas
        if (resolvedFormat === 'webp') {
          compressedBytes = await encodeWebp(req.pixels, req.width, req.height, req.qualityTarget ?? 85);
        } else if (resolvedFormat === 'jpeg') {
          // For lossless JPEG re-encode from original bytes (no pixel round-trip quality loss)
          const useLosslessJpegBytes =
            req.lossy === false &&
            req.format === 'jpeg' &&  // only when input is natively JPEG
            req.originalFileBytes !== undefined &&
            req.originalFileBytes.length > 0;

          if (useLosslessJpegBytes) {
            compressedBytes = recompressJpeg(
              req.originalFileBytes!,
              req.preset,
              req.qualityTarget ?? 85,
              req.progressive ?? false,
              req.trellis ?? false,
              req.optimizeHuffman ?? false,
            );
          } else {
            // Pixel-based JPEG encode (format conversion or lossy path)
            // Convert RGBA to RGB if needed
            let jpegPixels = req.pixels;
            let jpegColorType = req.colorType;

            if (req.colorType === 6) {
              jpegPixels = new Uint8Array(req.width * req.height * 3);
              for (let i = 0; i < req.width * req.height; i++) {
                jpegPixels[i * 3] = req.pixels[i * 4];
                jpegPixels[i * 3 + 1] = req.pixels[i * 4 + 1];
                jpegPixels[i * 3 + 2] = req.pixels[i * 4 + 2];
              }
              jpegColorType = 3; // RGB
            }

            compressedBytes = encodeJpegAdvanced(
              jpegPixels,
              req.width,
              req.height,
              jpegColorType,
              req.qualityTarget ?? 85,
              req.subsampling === '444' ? 1 : 0,
              req.progressive ?? false,
              req.trellis ?? false,
              req.optimizeHuffman ?? false,
              req.preset,
            );
          }
        } else {
          // Handle PNG encoding (existing logic)
          const useLosslessPngBytes =
            req.lossy === false &&
            req.originalFileBytes !== undefined &&
            req.originalFileBytes.length > 0;

          if (useLosslessPngBytes) {
            const sendProgress = (phase: string, progress: number) => {
              self.postMessage({
                type: "progress",
                id: req.id,
                phase,
                progress,
              } as ProgressMessage);
            };

            compressedBytes = recompressPngLossless(
              req.originalFileBytes!,
              req.preset,
              req.zopfliIterations ?? 0,
              sendProgress,
            );
          } else {
            // Determine if we should use advanced encoding
            const useAdvanced =
              (req.zopfliIterations !== undefined && req.zopfliIterations > 0) ||
              req.ditherStrength !== undefined ||
              req.qualityTarget !== undefined;

            if (useAdvanced) {
              // Use advanced encoding with progress reporting
              const sendProgress = (phase: string, progress: number) => {
                self.postMessage({
                  type: "progress",
                  id: req.id,
                  phase,
                  progress,
                } as ProgressMessage);
              };

              compressedBytes = encodePngAdvanced(
                req.pixels,
                req.width,
                req.height,
                req.colorType,
                req.preset,
                req.lossy,
                req.maxColors,
                req.dithering,
                req.ditherStrength ?? 0.5,
                req.qualityTarget ?? 75,
                req.zopfliIterations ?? 0,
                sendProgress,
              );
            } else {
              // Use basic encoding
              compressedBytes = encodePng(
                req.pixels,
                req.width,
                req.height,
                req.colorType,
                req.preset,
                req.lossy,
                req.maxColors,
                req.dithering,
              );
            }
          }
        }

        const duration = Date.now() - startTime;

        // Size guard: if compressed is larger than original file, return original bytes
        // Note: skipped for cross-format conversions (e.g. PNG→JPEG) since sizes differ
        if (
          resolvedFormat === req.format &&
          req.originalFileBytes &&
          req.originalFileBytes.length > 0 &&
          compressedBytes.length > req.originalFileBytes.length
        ) {
          console.debug(
            "[worker] compression resulted in larger file, returning original bytes",
            req.id,
            compressedBytes.length,
            ">",
            req.originalFileBytes.length,
          );
          compressedBytes = req.originalFileBytes;
        }

        const ratio =
          originalBytes > 0
            ? (compressedBytes.length / originalBytes) * 100
            : 0;

        console.debug(
          "[worker] compress done",
          req.id,
          compressedBytes?.length ?? null,
          duration + "ms",
        );

        self.postMessage({
          type: "compressed",
          id: req.id,
          compressedBytes: compressedBytes,
          originalBytes,
          compressedBytesCount: compressedBytes.length,
          ratio,
        } as CompressionResponse);
      })().catch((err) => {
        console.error("[worker] compress failed", req.id, err);
        self.postMessage({
          type: "error",
          id: req.id,
          error: err instanceof Error ? err.message : "Compression failed",
        });
      });
      break;
    default:
      console.warn("[worker] unknown message type", type, msg);
  }
};

console.log("Compression worker initialized");
