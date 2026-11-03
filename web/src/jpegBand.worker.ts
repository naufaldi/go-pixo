// A JPEG band worker (#128): its own Go WASM instance, started from the
// compression worker's compiled module, encodes bands of MCU rows into
// quantized coefficients for that worker to assemble.
import { compositeRgbaToRgbWhite } from './interop/pixels';
import { unwrapWasmResult } from './interop/wasmResult';
import type { JpegBand, JpegEncodeOptions } from './interop/jpegParallel';

type InitMessage = { type: 'init'; module: WebAssembly.Module; wasmExec: string };
type BandMessage = { type: 'band'; id: number; rgba: Uint8Array; width: number; height: number; band: JpegBand; options: JpegEncodeOptions };
type BandScope = {
  Go?: new () => { importObject: WebAssembly.Imports; run(instance: WebAssembly.Instance): Promise<void> };
  goWasmInit?: () => void;
  encodeJpegBand?: (...args: unknown[]) => Uint8Array | string;
  postMessage(message: unknown, transfer?: Transferable[]): void;
  onmessage: ((event: MessageEvent) => void) | null;
};

const scope = self as unknown as BandScope;
let ready: Promise<void> | undefined;

async function start({ module, wasmExec }: InitMessage): Promise<void> {
  // wasm_exec.js is a classic script; indirect eval defines Go globally.
  // eslint-disable-next-line no-eval
  if (!scope.Go) (0, eval)(wasmExec);
  const go = new scope.Go!();
  const initialized = new Promise<void>(resolve => { scope.goWasmInit = resolve; });
  const instance = await WebAssembly.instantiate(module, go.importObject);
  void go.run(instance);
  await initialized;
}

function encode({ rgba, width, height, band, options }: BandMessage): Uint8Array {
  if (!scope.encodeJpegBand) throw new Error('encodeJpegBand not available');
  const rgb = compositeRgbaToRgbWhite(rgba, width, band.height);
  return unwrapWasmResult(scope.encodeJpegBand(rgb, width, height, 3, options.quality, options.subsampling,
    options.progressive, options.trellis, options.optimizeHuffman, options.preset, band.firstRow, band.rows));
}

scope.onmessage = (event: MessageEvent) => {
  const message = event.data as InitMessage | BandMessage;
  if (message?.type === 'init') {
    ready = start(message);
    ready.then(() => scope.postMessage({ type: 'ready' }),
      error => scope.postMessage({ type: 'error', error: error instanceof Error ? error.message : 'Band worker failed to start' }));
    return;
  }
  if (message?.type !== 'band') return;
  (ready ?? Promise.reject(new Error('Band worker is not initialized'))).then(() => {
    const coefficients = encode(message);
    scope.postMessage({ type: 'band', id: message.id, coefficients }, [coefficients.buffer]);
  }).catch(error => {
    scope.postMessage({ type: 'error', id: message.id, error: error instanceof Error ? error.message : 'Band encode failed' });
  });
};
