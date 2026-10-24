// Owns admission from decode start until the entire operation settles.
export class TaskRegistry {
  private active = new Map<string, {id: string; controller: AbortController}>();
  private removed = new Set<string>();
  private committed = new Set<string>();

  get size(): number { return this.active.size; }
  has(key: string): boolean { return this.active.has(key); }
  isRemoved(id: string): boolean { return this.removed.has(id); }

  reserve(key: string, id: string): AbortSignal {
    const controller = new AbortController();
    this.active.set(key, {id, controller});
    return controller.signal;
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
    this.active.delete(key);
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
