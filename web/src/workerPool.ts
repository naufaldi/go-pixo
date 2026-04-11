// Worker pool for parallel image compression
// Each pooled worker owns its own WASM instance

type CompressTask = {
  data: any; // CompressionRequest fields merged with { type: 'compress' }
  resolve: (bytes: Uint8Array) => void;
  reject: (err: Error) => void;
  onProgress: (phase: string, progress: number) => void;
};

class PooledWorker {
  private worker: Worker;
  private currentTask: CompressTask | null = null;
  private ready = false;
  private readyResolve?: () => void;
  private readyPromise: Promise<void>;

  constructor() {
    this.worker = new Worker(new URL('./worker.ts', import.meta.url), { type: 'module' });
    this.readyPromise = new Promise(r => { this.readyResolve = r; });

    this.worker.onmessage = (event) => {
      const data = event.data as any;
      if (data.type === 'ready') {
        this.ready = true;
        this.readyResolve?.();
      } else if (data.type === 'progress' && this.currentTask) {
        this.currentTask.onProgress(data.phase, data.progress);
      } else if (data.type === 'compressed' && this.currentTask) {
        const task = this.currentTask;
        this.currentTask = null;
        task.resolve(new Uint8Array(data.compressedBytes));
      } else if (data.type === 'error' && this.currentTask) {
        const task = this.currentTask;
        this.currentTask = null;
        task.reject(new Error(data.error || 'Compression failed'));
      }
    };

    this.worker.postMessage({ type: 'init' });
  }

  async waitReady(): Promise<void> {
    return this.readyPromise;
  }

  get isIdle(): boolean {
    return this.ready && this.currentTask === null;
  }

  run(task: CompressTask): void {
    this.currentTask = task;
    this.worker.postMessage({ type: 'compress', ...task.data });
  }

  terminate(): void {
    this.worker.terminate();
  }
}

export class WorkerPool {
  private workers: PooledWorker[];
  private queue: CompressTask[] = [];
  private poolSize: number;

  constructor(size?: number) {
    this.poolSize = size ?? Math.max(2, Math.min(4, (navigator.hardwareConcurrency ?? 4) - 1));
    // Initialize workers lazily — create them on first use
    this.workers = [];
  }

  private ensureWorkers(): void {
    while (this.workers.length < this.poolSize) {
      this.workers.push(new PooledWorker());
    }
  }

  private dispatch(): void {
    if (this.queue.length === 0) return;
    const idle = this.workers.find(w => w.isIdle);
    if (!idle) return;
    const task = this.queue.shift()!;
    idle.run(task);
  }

  async submit(data: any, onProgress: (phase: string, progress: number) => void): Promise<Uint8Array> {
    this.ensureWorkers();
    // Wait for at least one worker to be ready
    await Promise.race(this.workers.map(w => w.waitReady()));

    return new Promise((resolve, reject) => {
      const task: CompressTask = {
        data,
        resolve: (bytes) => {
          resolve(bytes);
          this.dispatch(); // process next in queue
        },
        reject: (err) => {
          reject(err);
          this.dispatch();
        },
        onProgress,
      };
      this.queue.push(task);
      this.dispatch();
    });
  }

  terminate(): void {
    this.workers.forEach(w => w.terminate());
    this.workers = [];
  }
}

export const globalPool = new WorkerPool();
