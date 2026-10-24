import { MAX_SINGLE_WORKSPACE, WORKSPACE_BUDGET } from './imageOperation';

type Operation = {id: string; controller: AbortController; bytes?: number; cancelAdmission?: () => void};
type Waiting = {operation: Operation; bytes: number; grant: () => void};
const cancelled = () => new DOMException('Image admission cancelled', 'AbortError');

// Owns admission from decode start until the entire operation settles.
export class TaskRegistry {
  private active = new Map<string, Operation>();
  private removed = new Set<string>();
  private committed = new Set<string>();
  private waiting: Waiting[] = [];
  private workspaceBytes = 0;

  get size(): number { return this.active.size; }
  has(key: string): boolean { return this.active.has(key); }
  isRemoved(id: string): boolean { return this.removed.has(id); }

  reserve(key: string, id: string): AbortSignal {
    if (this.active.has(key)) throw new Error('Image task already reserved');
    const controller = new AbortController();
    this.active.set(key, {id, controller});
    return controller.signal;
  }

  // Reserve before rasterization. Cancellation of admitted work deliberately
  // keeps its bytes charged until decode AND compression resources settle.
  admit(key: string, bytes: number): Promise<void> {
    const operation = this.active.get(key);
    if (!operation || operation.controller.signal.aborted) return Promise.reject(cancelled());
    if (!Number.isSafeInteger(bytes) || bytes < 0 || bytes > MAX_SINGLE_WORKSPACE) {
      return Promise.reject(new Error('Image workspace estimate exceeds the memory admission budget'));
    }
    if (operation.bytes !== undefined || operation.cancelAdmission) return Promise.reject(new Error('Image workspace already reserved'));
    return new Promise((resolve, reject) => {
      const signal = operation.controller.signal;
      const cleanup = () => { signal.removeEventListener('abort', cancel); operation.cancelAdmission = undefined };
      const cancel = () => {
        this.waiting = this.waiting.filter(entry => entry.operation !== operation);
        cleanup(); reject(cancelled()); this.admitWaiting();
      };
      operation.cancelAdmission = cancel;
      signal.addEventListener('abort', cancel, {once: true});
      this.waiting.push({operation, bytes, grant: () => {
        cleanup(); operation.bytes = bytes; this.workspaceBytes += bytes; resolve();
      }});
      this.admitWaiting();
    });
  }

  private admitWaiting(): void {
    while (this.waiting.length) {
      const next = this.waiting[0];
      // FIFO prevents starvation. A single valid job above the normal budget
      // may run alone, bounded by 24 MP x 32 bytes, so it cannot deadlock.
      if (this.workspaceBytes + next.bytes > WORKSPACE_BUDGET && this.workspaceBytes !== 0) return;
      this.waiting.shift(); next.grant();
    }
  }

  remove(id: string): void {
    // Protect synchronously, including the interval before React commits removal.
    this.removed.add(id);
    for (const operation of this.active.values()) {
      if (operation.id === id) operation.controller.abort();
    }
  }

  abortAll(): void {
    for (const operation of this.active.values()) operation.controller.abort();
  }

  settle(key: string): void {
    const operation = this.active.get(key);
    if (!operation) return;
    operation.cancelAdmission?.();
    this.workspaceBytes -= operation.bytes ?? 0;
    this.active.delete(key);
    this.admitWaiting();
    this.prune();
  }

  commit(ids: string[]): void {
    this.committed = new Set(ids);
    this.prune();
  }

  private prune(): void {
    const outstanding = new Set([...this.active.values()].map(operation => operation.id));
    for (const id of this.removed) {
      if (!this.committed.has(id) && !outstanding.has(id)) this.removed.delete(id);
    }
  }
}
