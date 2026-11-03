import type { JpegBand, JpegEncodeOptions } from './interop/jpegParallel';

// Idle band workers keep their Go heaps, so stop them soon after an encode.
const IDLE_TERMINATE_MS = 3_000;
// A band worker that cannot start in time sends the encode to the serial path.
const START_TIMEOUT_MS = 10_000;

type Pending = { resolve(coefficients: Uint8Array): void; reject(error: Error): void };
type Helper = { worker: Worker; pending: Map<number, Pending>; startTimer?: ReturnType<typeof setTimeout> };
export type GoModuleSource = { module: WebAssembly.Module; wasmExec: string };

const createBandWorker = () => new Worker(new URL('./jpegBand.worker.ts', import.meta.url), { type: 'module' });

// Band workers owned by one compression worker. They end with it when the
// pool terminates it on cancel or failure.
export class JpegBandWorkers {
  private helpers: Helper[] = [];
  private nextId = 0;
  private idle?: ReturnType<typeof setTimeout>;
  private readonly source: GoModuleSource;
  private readonly create: () => Worker;

  constructor(source: GoModuleSource, create: () => Worker = createBandWorker) {
    this.source = source;
    this.create = create;
  }

  // Transfers rgba (an owned band of RGBA rows) to band worker index.
  encode(index: number, rgba: Uint8Array, width: number, height: number, band: JpegBand, options: JpegEncodeOptions): Promise<Uint8Array> {
    clearTimeout(this.idle);
    const helper = this.helpers[index] ??= this.start();
    const id = ++this.nextId;
    return new Promise((resolve, reject) => {
      helper.pending.set(id, { resolve, reject });
      helper.worker.postMessage({ type: 'band', id, rgba, width, height, band, options }, [rgba.buffer]);
    });
  }

  // A failed encode never reuses band workers; a successful one keeps them
  // briefly for the next image.
  release(failed: boolean): void {
    clearTimeout(this.idle);
    if (failed) this.terminate();
    else this.idle = setTimeout(() => this.terminate(), IDLE_TERMINATE_MS);
  }

  terminate(error = new Error('Band workers stopped')): void {
    clearTimeout(this.idle);
    const helpers = this.helpers;
    this.helpers = [];
    for (const helper of helpers) {
      if (!helper) continue;
      clearTimeout(helper.startTimer);
      helper.worker.terminate();
      for (const pending of helper.pending.values()) pending.reject(error);
      helper.pending.clear();
    }
  }

  private start(): Helper {
    const worker = this.create();
    const helper: Helper = { worker, pending: new Map() };
    // Ignore late events from band workers already stopped.
    const fail = (message: string) => { if (this.helpers.includes(helper)) this.terminate(new Error(message)); };
    helper.startTimer = setTimeout(() => fail('Band worker start timed out'), START_TIMEOUT_MS);
    worker.onmessage = (event: MessageEvent) => {
      const message = event.data;
      if (message?.type === 'ready') clearTimeout(helper.startTimer);
      else if (message?.type === 'band' && message.coefficients instanceof Uint8Array) {
        helper.pending.get(message.id)?.resolve(message.coefficients);
        helper.pending.delete(message.id);
      } else if (message?.type === 'error') {
        clearTimeout(helper.startTimer);
        fail(typeof message.error === 'string' ? message.error : 'Band worker failed');
      }
    };
    worker.onerror = (event: ErrorEvent) => {
      event.preventDefault();
      clearTimeout(helper.startTimer);
      fail(event.message || 'Band worker failed');
    };
    worker.onmessageerror = () => { clearTimeout(helper.startTimer); fail('Band worker message failed'); };
    worker.postMessage({ type: 'init', ...this.source });
    return helper;
  }
}
