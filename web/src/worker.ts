// Web Worker for off-main-thread image compression

interface CompressionRequest {
  id: string;
  pixels: Uint8Array;
  width: number;
  height: number;
  colorType: number;
  preset: number;
  lossy: boolean;
  maxColors: number;
  dithering: boolean;
}

interface CompressionResponse {
  id: string;
  compressedBytes: Uint8Array;
  error?: string;
}

// Store pending requests
const pendingRequests = new Map<string, {
  resolve: (bytes: Uint8Array) => void;
  reject: (error: Error) => void;
}>();

// WASM state
let wasmReady = false;

// Initialize WASM in worker
async function initWasm(): Promise<void> {
  if (wasmReady) return;

  // This worker is created as an ES module worker.
  // Module workers do NOT support importScripts(), so we load wasm_exec.js via dynamic import.
  // wasm_exec.js attaches Go to globalThis as a side effect.
  // @ts-ignore
  if (typeof (self as any).Go === 'undefined') {
    console.debug('[worker] Loading /wasm_exec.js');
    // wasm_exec.js is shipped as a classic script (not an ES module).
    // In a module worker we can't use importScripts(), and Vite tries to bundle `import('/wasm_exec.js')`.
    // Fetch + indirect eval loads it into the worker global scope.
    const wasmExecJs = await fetch('/wasm_exec.js').then(r => r.text());
    // eslint-disable-next-line no-eval
    (0, eval)(wasmExecJs);
  }

  // @ts-ignore
  const go = new (self as any).Go();

  return new Promise((resolve, reject) => {
    // Set up initialization callback called from Go's main()
    (self as any).goWasmInit = () => {
      wasmReady = true;
      console.debug('[worker] Go WASM initialized');
      resolve();
    };

    console.debug('[worker] Fetching /main.wasm');
    fetch('/main.wasm')
      .then(response => response.arrayBuffer())
      .then(buffer => WebAssembly.instantiate(buffer, go.importObject))
      .then(result => {
        console.debug('[worker] Running WASM instance');
        go.run(result.instance);
      })
      .catch(err => {
        console.error('[worker] Failed to load WASM:', err);
        reject(err);
      });
  });
}

// Encode PNG using WASM
function encodePng(
  pixels: Uint8Array,
  width: number,
  height: number,
  colorType: number = 6,
  preset: number = 1,
  lossy: boolean = false,
  maxColors: number = 0,
  dithering: boolean = false
): Uint8Array {
  // @ts-ignore - encodePng is exposed by Go WASM on the worker global
  if (!(self as any).encodePng) {
    throw new Error('WASM not initialized');
  }

  // @ts-ignore - encodePng is exposed by Go WASM on the worker global
  const result = (self as any).encodePng(pixels, width, height, colorType, preset, lossy, maxColors);

  if (typeof result === 'string' && result.startsWith('error:')) {
    throw new Error(result);
  }

  return result as Uint8Array;
}

// Handle messages from main thread
self.onmessage = async (event: MessageEvent<{
  type: 'compress';
  data?: CompressionRequest;
} | ({
  type: 'compress';
} & CompressionRequest) | {
  type: 'init';
}>) => {
  const msg: any = event.data as any;
  const type: string = msg?.type;
  console.debug('[worker] message', type, msg?.id ?? msg?.data?.id ?? null);

  switch (type) {
    case 'init':
      try {
        console.debug('[worker] init requested');
        await initWasm();
        self.postMessage({ type: 'ready' });
      } catch (err) {
        self.postMessage({ 
          type: 'error', 
          error: err instanceof Error ? err.message : 'Failed to initialize WASM' 
        });
      }
      break;

    case 'compress':
      // Support both `{type:'compress', data:{...}}` and `{type:'compress', ...fields}`
      const req: CompressionRequest | undefined = msg?.data ?? msg;

      if (!req || typeof req.id !== 'string') {
        self.postMessage({
          type: 'error',
          error: 'Invalid compress message'
        });
        return;
      }

      if (!wasmReady) {
        self.postMessage({
          type: 'error',
          id: req.id,
          error: 'WASM not initialized'
        });
        return;
      }

      try {
        console.debug('[worker] compress start', req.id);
        const startTime = Date.now();
        const compressedBytes = encodePng(
          req.pixels,
          req.width,
          req.height,
          req.colorType,
          req.preset,
          req.lossy,
          req.maxColors,
          req.dithering
        );
        const duration = Date.now() - startTime;

        self.postMessage({
          type: 'compressed',
          id: req.id,
          compressedBytes: compressedBytes
        });
        console.debug('[worker] compress done', req.id, compressedBytes?.byteLength ?? null);
      } catch (err) {
        console.error('[worker] compress failed', req.id, err);
        self.postMessage({
          type: 'error',
          id: req.id,
          error: err instanceof Error ? err.message : 'Compression failed'
        });
      }
      break;
    default:
      console.warn('[worker] unknown message type', type, msg);
  }
};

console.log('Compression worker initialized');
