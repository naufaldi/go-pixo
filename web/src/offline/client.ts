export type OfflineSnapshot = {
  state: 'preparing' | 'ready' | 'unavailable' | 'unsupported' | 'development'
  message: string
  updateAvailable: boolean
}

let snapshot: OfflineSnapshot = {
  state: 'preparing', message: 'Preparing offline files…', updateAvailable: false,
}
const listeners = new Set<(status: OfflineSnapshot) => void>()
let started = false
let verification = 0

export function getOfflineStatus(): OfflineSnapshot { return snapshot }

export function subscribeOffline(listener: (status: OfflineSnapshot) => void): () => void {
  listeners.add(listener)
  // The component can mount after registration has already finished.
  listener(snapshot)
  return () => { listeners.delete(listener) }
}

function publish(state: OfflineSnapshot['state'], message: string, updateAvailable = snapshot.updateAvailable) {
  snapshot = { state, message, updateAvailable }
  for (const listener of listeners) listener(snapshot)
}

export function isVerifiedStatus(value: unknown): value is { ready: true; version: string; assetCount: number } {
  if (typeof value !== 'object' || value === null) return false
  const status = value as Record<string, unknown>
  return status.type === 'GO_PIXO_OFFLINE_STATUS' && status.ready === true &&
    typeof status.version === 'string' && status.version.length > 0 &&
    typeof status.assetCount === 'number' && Number.isInteger(status.assetCount) && status.assetCount > 0
}

function askController(controller: ServiceWorker, repair = false): Promise<unknown> {
  return new Promise((resolve, reject) => {
    const channel = new MessageChannel()
    const finish = () => {
      clearTimeout(timeout)
      channel.port1.close()
      channel.port2.close()
    }
    const timeout = setTimeout(() => { finish(); reject(new Error('Offline verification timed out')) }, repair ? 120000 : 5000)
    channel.port1.onmessage = (event) => { finish(); resolve(event.data) }
    const pageAssets = [...document.querySelectorAll<HTMLScriptElement | HTMLLinkElement>(
      'script[src], link[rel="stylesheet"]',
    )].map((element) => element instanceof HTMLScriptElement ? element.src : element.href)
      .filter((url) => new URL(url).origin === location.origin)
    try {
      controller.postMessage({ type: repair ? 'GO_PIXO_PREPARE_OFFLINE' : 'GO_PIXO_VERIFY_OFFLINE', pageAssets }, [channel.port2])
    } catch (error) {
      finish()
      reject(error)
    }
  })
}

export function initializeOffline(): void {
  if (started) return
  started = true
  if (!import.meta.env.PROD) {
    publish('development', 'Offline preparation is available in the production build.')
    return
  }
  if (!window.isSecureContext || !('serviceWorker' in navigator) || !('caches' in window)) {
    publish('unsupported', 'Offline mode needs a browser with service workers on HTTPS or localhost.')
    return
  }

  const serviceWorkers = navigator.serviceWorker
  const baseURL = new URL(import.meta.env.BASE_URL, location.origin)
  const workerURL = new URL('sw.js', baseURL).href
  let registration: ServiceWorkerRegistration | undefined

  const verify = async (repair = false) => {
    const attempt = ++verification
    const controller = serviceWorkers.controller
    if (!controller || controller.scriptURL !== workerURL || controller.state !== 'activated') {
      if (snapshot.state === 'ready') publish('unavailable', 'Offline files are not verified. Reconnect and reload to prepare them.')
      return
    }
    try {
      let response = await askController(controller)
      if (attempt !== verification || controller !== serviceWorkers.controller) return
      if (!isVerifiedStatus(response) && repair && navigator.onLine) {
        publish('preparing', 'Preparing offline files…')
        response = await askController(controller, true)
      }
      if (attempt !== verification || controller !== serviceWorkers.controller) return
      if (isVerifiedStatus(response)) publish('ready', 'Ready offline')
      else publish('unavailable', 'Offline files are incomplete. Reconnect and reload to prepare them.')
    } catch {
      if (attempt === verification && controller === serviceWorkers.controller) {
        publish('unavailable', 'Offline files could not be verified. Reconnect and reload to try again.')
      }
    }
  }

  const watched = new WeakSet<ServiceWorker>()
  const watch = (worker: ServiceWorker | null) => {
    if (!worker || watched.has(worker)) return
    watched.add(worker)
    worker.addEventListener('statechange', () => {
      if (registration?.waiting && serviceWorkers.controller) {
        publish(snapshot.state, snapshot.message, true)
      }
      if (worker.state === 'redundant' && !serviceWorkers.controller) {
        publish('unavailable', 'Offline setup failed. Reconnect and reload to try again.')
      }
      void verify()
    })
  }

  serviceWorkers.addEventListener('controllerchange', () => {
    watch(serviceWorkers.controller)
    void verify()
  })
  window.addEventListener('pageshow', () => { void verify() })
  window.addEventListener('offline', () => { void verify() })
  window.addEventListener('online', () => {
    void verify(true)
    void registration?.update().catch(() => { /* Existing verified offline build remains usable. */ })
  })
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') void verify()
  })
  // Recheck after storage eviction as well as lifecycle/network changes.
  window.setInterval(() => {
    if (document.visibilityState === 'visible') void verify()
  }, 30000)

  void serviceWorkers.register(workerURL, { scope: baseURL.pathname, updateViaCache: 'none' }).then((value) => {
    registration = value
    if (value.waiting && serviceWorkers.controller) publish(snapshot.state, snapshot.message, true)
    value.addEventListener('updatefound', () => { watch(value.installing) })
    watch(value.installing)
    watch(serviceWorkers.controller)
    // Identical registration options can reuse a broken worker without checking
    // its script. A network-bypassing refresh must request the repaired version.
    if (navigator.onLine) {
      void value.update().catch(() => { /* Keep the verified offline build usable. */ })
    }
    void verify(true)
  }).catch(() => {
    publish('unavailable', 'Offline setup failed. Reconnect and reload to try again.')
    // An update failure does not invalidate a complete, previously installed build.
    void verify()
  })
}
