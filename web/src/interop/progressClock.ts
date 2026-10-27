import {useSyncExternalStore} from 'react';

// All visible progress views share one clock. It stops as soon as the last
// operation/view unsubscribes; worker messages never drive elapsed time.
const listeners = new Set<() => void>();
let clockNow = performance.now();
let timer: ReturnType<typeof setInterval> | undefined;
const snapshot = () => clockNow;
function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  if (timer === undefined) {
    clockNow = performance.now();
    timer = setInterval(() => {
      clockNow = performance.now();
      listeners.forEach(notify => notify());
    }, 250);
  }
  return () => {
    listeners.delete(listener);
    if (!listeners.size) { clearInterval(timer); timer = undefined; }
  };
}
const inactive = () => () => {};
export function useProgressTiming(startTime: number, lastUpdate: number, active: boolean) {
  const now = useSyncExternalStore(active ? subscribe : inactive, snapshot, snapshot);
  return {elapsed: Math.max(0, now - startTime), silentFor: Math.max(0, now - lastUpdate)};
}
