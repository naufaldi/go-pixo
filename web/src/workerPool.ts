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
type Slot = { worker: Worker; ready: boolean; task: Task | null; timer?: ReturnType<typeof setTimeout> };

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

  constructor(size = recommendedConcurrency()) {
    this.capacity = Math.max(1, Math.min(2, Math.floor(size) || 1));
  }

  private startWorkers(): void {
    while (!this.stopped && this.workers.length < this.capacity) {
      const worker = new Worker(new URL('./worker.ts', import.meta.url), { type: 'module' });
      const slot: Slot = { worker, ready: false, task: null };
      this.workers.push(slot);
      slot.timer = setTimeout(() => this.fail(new Error('Worker initialization timed out')), 30_000);
      worker.onmessage = event => this.receive(slot, event.data);
      worker.onerror = event => {
        event.preventDefault();
        this.fail(new Error(event.message || 'Compression worker crashed'));
      };
      worker.onmessageerror = () => this.fail(new Error('Failed to read worker response'));
      worker.postMessage({ type: 'init' });
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
      this.fail(new Error(data.error || 'Worker initialization failed'));
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
    if (data.type === 'error') task.reject(new Error(data.error || 'Compression failed'));
    else {
      try {
        if (!['png', 'jpeg', 'webp', 'avif'].includes(data.outputFormat) || data.compressedBytes == null) {
          throw new Error('Invalid compression worker result');
        }
        const compressedBytes = new Uint8Array(data.compressedBytes);
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
      slot.timer = setTimeout(() => this.fail(new Error('Compression worker timed out')), 300_000);
      try {
        const buffers = [task.data.pixels, task.data.originalFileBytes]
          .filter((value): value is Uint8Array => value instanceof Uint8Array)
          .map(value => value.buffer)
          .filter((buffer): buffer is ArrayBuffer => buffer instanceof ArrayBuffer);
        slot.worker.postMessage({ ...task.data, type: 'compress' }, [...new Set(buffers)]);
      } catch (error) {
        this.fail(error instanceof Error ? error : new Error('Failed to start compression'));
        return;
      }
    }
  }

  submit(data: CompressionRequest, onProgress: ProgressCallback): Promise<CompressionResult> {
    if (this.stopped) return Promise.reject(this.stopped);
    return new Promise((resolve, reject) => {
      this.queue.push({ data, onProgress, resolve, reject });
      try { this.startWorkers(); this.dispatch(); }
      catch (error) { this.fail(error instanceof Error ? error : new Error('Worker initialization failed')); }
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
      clearTimeout(slot.timer);
      slot.worker.terminate();
      slot.task!.reject(cancelled);
      slot.task = null;
      this.workers = this.workers.filter(candidate => candidate !== slot);
      try { this.startWorkers(); this.dispatch(); }
      catch (error) { this.fail(error instanceof Error ? error : new Error('Worker initialization failed')); }
    }
  }

  private fail(error: Error): void {
    if (this.stopped) return;
    this.stopped = error;
    const queued = this.queue.splice(0);
    for (const slot of this.workers) {
      clearTimeout(slot.timer);
      slot.worker.terminate();
      if (slot.task) queued.push(slot.task);
      slot.task = null;
    }
    this.workers = [];
    queued.forEach(task => task.reject(error));
  }

  terminate(): void { this.fail(new Error('Worker pool terminated')); }
}

export const globalPool = new WorkerPool();
