import { receivedBytes } from './interop/workerMessages';

// A worker owns one WASM instance. The app also bounds decoding to capacity.
export type CompressionResult = { compressedBytes: Uint8Array; outputFormat: string };
type CompressionRequest = { id: string; [field: string]: unknown };
type ProgressCallback = (phase: string, progress: number, predictable?: boolean, phaseTarget?: number) => void;
type Task = {
  data: CompressionRequest;
  resolve: (result: CompressionResult) => void;
  reject: (error: Error) => void;
  onProgress: ProgressCallback;
};
type Slot = { worker: Worker | null; ready: boolean; task: Task | null; initializationFailures: number; timer?: ReturnType<typeof setTimeout> };

export function recommendedConcurrency(): number {
  const hardware = typeof navigator === 'undefined' ? undefined : navigator.hardwareConcurrency;
  const memory = typeof navigator === 'undefined' ? undefined : (navigator as Navigator & { deviceMemory?: number }).deviceMemory;
  return (hardware != null && hardware <= 2) || (memory != null && memory <= 2) ? 1 : 2;
}

export class WorkerPool {
  readonly capacity: number;
  private workers: Slot[] = [];
  private queue: Task[] = [];
  private stopped: Error | null = null;
  private terminated = false;

  constructor(size = recommendedConcurrency()) {
    this.capacity = Math.max(1, Math.min(2, Math.floor(size) || 1));
  }

  private startWorkers(): void {
    while (!this.stopped && this.workers.length < this.capacity) {
      this.workers.push({worker: null, ready: false, task: null, initializationFailures: 0});
    }
    for (const slot of this.workers) {
      if (!this.stopped && !slot.worker && slot.initializationFailures < 2) this.startSlot(slot);
    }
  }

  private startSlot(slot: Slot): void {
    try {
      const worker = new Worker(new URL('./worker.ts', import.meta.url), { type: 'module' });
      slot.worker = worker;
      slot.timer = setTimeout(() => this.failSlot(slot, new Error('Worker initialization timed out')), 30_000);
      worker.onmessage = event => {
        if (slot.worker === worker) this.receive(slot, event.data);
      };
      worker.onerror = event => {
        event.preventDefault();
        if (slot.worker === worker) this.failSlot(slot, new Error(event.message || 'Compression worker crashed'));
      };
      worker.onmessageerror = () => {
        if (slot.worker === worker) this.failSlot(slot, new Error('Failed to read worker response'));
      };
      worker.postMessage({ type: 'init' });
    } catch (error) {
      this.failSlot(slot, error instanceof Error ? error : new Error('Worker initialization failed'));
    }
  }

  private receive(slot: Slot, data: any): void {
    if (this.stopped || !data || typeof data !== 'object') return;
    if (data.type === 'ready' && !slot.ready) {
      clearTimeout(slot.timer);
      slot.ready = true;
      this.dispatch();
      return;
    }
    if (data.type === 'error' && data.id == null) {
      this.failSlot(slot, new Error(data.error || 'Worker initialization failed'));
      return;
    }
    const task = slot.task;
    // Late messages must never complete a different operation in the same slot.
    if (!task || data.id !== task.data.id) return;
    if (data.type === 'progress') {
      if (typeof data.phase === 'string' && Number.isFinite(data.progress)) {
        task.onProgress(data.phase, Math.max(0, Math.min(99, Math.round(data.progress))), data.predictable === true, data.phaseTarget);
      }
      return;
    }
    if (data.type !== 'compressed' && data.type !== 'error') return;
    clearTimeout(slot.timer);
    slot.task = null;
    slot.initializationFailures = 0;
    if (data.type === 'error') task.reject(new Error(data.error || 'Compression failed'));
    else {
      try {
        if (!['png', 'jpeg', 'webp', 'avif'].includes(data.outputFormat) || data.compressedBytes == null) {
          throw new Error('Invalid compression worker result');
        }
        const compressedBytes = receivedBytes(data.compressedBytes);
        if (compressedBytes.byteLength === 0) throw new Error('Compression returned an empty image');
        task.resolve({ compressedBytes, outputFormat: data.outputFormat });
      } catch (error) {
        task.reject(error instanceof Error ? error : new Error('Invalid compression worker result'));
      }
    }
    this.dispatch();
  }

  private dispatch(): void {
    if (this.stopped) return;
    for (const slot of this.workers) {
      if (!slot.ready || slot.task || !this.queue.length) continue;
      const task = this.queue.shift()!;
      slot.task = task;
      // A fixed deadline also catches workers that stop emitting progress.
      slot.timer = setTimeout(() => this.failSlot(slot, new Error('Compression worker timed out')), 300_000);
      try {
        const buffers = [task.data.pixels, task.data.originalFileBytes]
          .filter((value): value is Uint8Array => value instanceof Uint8Array)
          .map(value => value.buffer)
          .filter((buffer): buffer is ArrayBuffer => buffer instanceof ArrayBuffer);
        slot.worker!.postMessage({ ...task.data, type: 'compress' }, [...new Set(buffers)]);
      } catch (error) {
        this.failSlot(slot, error instanceof Error ? error : new Error('Failed to start compression'));
      }
    }
  }

  submit(data: CompressionRequest, onProgress: ProgressCallback): Promise<CompressionResult> {
    if (this.stopped) return Promise.reject(this.stopped);
    return new Promise((resolve, reject) => {
      this.queue.push({ data, onProgress, resolve, reject });
      this.startWorkers(); this.dispatch();
    });
  }

  cancel(id: string): void {
    const cancelled = new Error('Compression cancelled');
    this.queue = this.queue.filter(task => {
      if (task.data.id !== id) return true;
      task.reject(cancelled);
      return false;
    });
    const slot = this.workers.find(candidate => candidate.task?.data.id === id);
    if (slot) {
      this.failSlot(slot, cancelled);
    }
  }

  private disposeSlot(slot: Slot): void {
    clearTimeout(slot.timer);
    if (slot.worker) {
      slot.worker.onmessage = null; slot.worker.onerror = null; slot.worker.onmessageerror = null;
      slot.worker.terminate();
    }
    slot.worker = null;
    slot.ready = false;
  }

  private failSlot(slot: Slot, error: Error): void {
    if (this.stopped) return;
    const task = slot.task;
    // One automatic replacement per failed initialization sequence. A failed
    // input is rejected once and never requeued, even after transferring its bytes.
    slot.initializationFailures = slot.ready && task ? 0 : slot.initializationFailures + 1;
    slot.task = null;
    this.disposeSlot(slot);
    task?.reject(error);
    if (slot.initializationFailures < 2) this.startSlot(slot);
    const available = this.workers.some(candidate => candidate.worker || candidate.initializationFailures < 2);
    if (!available) this.stop(error);
    else this.dispatch();
  }

  private stop(error: Error): void {
    if (this.stopped) return;
    this.stopped = error;
    const queued = this.queue.splice(0);
    for (const slot of this.workers) {
      this.disposeSlot(slot);
      if (slot.task) queued.push(slot.task);
      slot.task = null;
    }
    queued.forEach(task => task.reject(error));
  }

  // Explicit recovery from exhausted initialization attempts; never revives an
  // intentionally terminated pool. Clear/settings also create a fresh pool.
  recover(): boolean {
    if (this.terminated) return false;
    this.stopped = null;
    for (const slot of this.workers) {
      if (!slot.worker) slot.initializationFailures = 0;
    }
    if (this.queue.length) { this.startWorkers(); this.dispatch(); }
    return true;
  }

  terminate(): void {
    this.terminated = true;
    this.stop(new Error('Worker pool terminated'));
  }
}

export const globalPool = new WorkerPool();
