import { afterEach, describe, expect, it, vi } from 'vitest'

afterEach(() => {
  vi.clearAllTimers()
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  vi.unstubAllEnvs()
})

async function environment(online = false) {
  vi.resetModules()
  vi.useFakeTimers()
  vi.stubEnv('PROD', true)
  vi.stubEnv('BASE_URL', '/')
  vi.stubGlobal('isSecureContext', true)
  vi.stubGlobal('caches', {})
  // Keep module-lifetime page listeners from leaking between isolated tests.
  vi.spyOn(window, 'addEventListener').mockImplementation(() => {})
  vi.spyOn(document, 'addEventListener').mockImplementation(() => {})
  class Channel {
    port1 = { close: vi.fn(), onmessage: null as ((event: { data: unknown }) => void) | null }
    port2 = { close: vi.fn(), postMessage: (data: unknown) => this.port1.onmessage?.({ data }) }
  }
  vi.stubGlobal('MessageChannel', Channel)
  let reply: unknown = { type: 'GO_PIXO_OFFLINE_STATUS', ready: true, version: 'test-build', assetCount: 12 }
  const controller = Object.assign(new EventTarget(), {
    scriptURL: new URL('/sw.js', location.origin).href,
    state: 'activated',
    postMessage: vi.fn((_data: unknown, ports: { postMessage: (value: unknown) => void }[]) => {
      if (reply !== undefined) ports[0].postMessage(reply)
    }),
  })
  const registration = Object.assign(new EventTarget(), { installing: null, waiting: null, update: vi.fn(async () => {}) })
  const serviceWorkers = Object.assign(new EventTarget(), {
    controller: controller as typeof controller | null,
    register: vi.fn(async () => registration),
  })
  vi.stubGlobal('navigator', { serviceWorker: serviceWorkers, onLine: online })
  const client = await import('./client')
  return { client, controller, registration, serviceWorkers, setReply: (value: unknown) => { reply = value } }
}

describe('offline readiness', () => {
  it('does not register in development', async () => {
    const { client, serviceWorkers } = await environment()
    vi.stubEnv('PROD', false)
    client.initializeOffline()
    expect(serviceWorkers.register).not.toHaveBeenCalled()
    expect(client.getOfflineStatus().state).toBe('development')
  })

  it('requires active control and complete verified caches, then detects cache loss', async () => {
    const { client, controller, serviceWorkers, setReply } = await environment()
    const listener = vi.fn()
    const unsubscribe = client.subscribeOffline(listener)
    serviceWorkers.controller = null
    client.initializeOffline()
    await vi.advanceTimersByTimeAsync(0)
    expect(client.getOfflineStatus().state).toBe('preparing')
    serviceWorkers.controller = controller
    serviceWorkers.dispatchEvent(new Event('controllerchange'))
    await vi.advanceTimersByTimeAsync(0)
    expect(client.getOfflineStatus().state).toBe('ready')
    expect(serviceWorkers.register).toHaveBeenCalledWith(new URL('/sw.js', location.origin).href,
      { scope: '/', updateViaCache: 'none' })
    setReply({ type: 'GO_PIXO_OFFLINE_STATUS', ready: false, version: 'test-build', assetCount: 12 })
    serviceWorkers.dispatchEvent(new Event('controllerchange'))
    await vi.advanceTimersByTimeAsync(0)
    expect(client.getOfflineStatus().state).toBe('unavailable')
    expect(listener).toHaveBeenCalled()
    unsubscribe()
  })

  it('checks an existing registration for updates online without delaying cache verification', async () => {
    const { client, registration } = await environment(true)
    registration.update.mockImplementation(() => new Promise(() => {}))
    client.initializeOffline()
    await vi.advanceTimersByTimeAsync(0)
    expect(registration.update).toHaveBeenCalledOnce()
    expect(client.getOfflineStatus().state).toBe('ready')
  })

  it('keeps a verified offline build usable when the online update check fails', async () => {
    const { client, registration } = await environment(true)
    registration.update.mockRejectedValue(new Error('Update network unavailable'))
    client.initializeOffline()
    await vi.advanceTimersByTimeAsync(0)
    expect(registration.update).toHaveBeenCalledOnce()
    expect(client.getOfflineStatus().state).toBe('ready')
  })

  it('does not force an update check while offline', async () => {
    const { client, registration } = await environment(false)
    client.initializeOffline()
    await vi.advanceTimersByTimeAsync(0)
    expect(registration.update).not.toHaveBeenCalled()
    expect(client.getOfflineStatus().state).toBe('ready')
  })

  it('rejects wrong worker replies and times out without claiming readiness', async () => {
    const { client, serviceWorkers, setReply } = await environment()
    setReply({ ready: true })
    client.initializeOffline()
    await vi.advanceTimersByTimeAsync(0)
    expect(client.getOfflineStatus().state).toBe('unavailable')
    setReply(undefined)
    serviceWorkers.dispatchEvent(new Event('controllerchange'))
    await vi.advanceTimersByTimeAsync(5000)
    expect(client.getOfflineStatus().state).toBe('unavailable')
    expect(client.getOfflineStatus().message).toContain('could not be verified')
  })

  it('repairs an evicted build online and waits for complete verification before ready', async () => {
    const { client, controller } = await environment(true)
    let complete: (() => void) | undefined
    controller.postMessage.mockImplementation((data, ports) => {
      if ((data as { type: string }).type === 'GO_PIXO_PREPARE_OFFLINE') {
        complete = () => ports[0].postMessage({ type: 'GO_PIXO_OFFLINE_STATUS', ready: true, version: 'same-build', assetCount: 12 })
      } else ports[0].postMessage({ type: 'GO_PIXO_OFFLINE_STATUS', ready: false, version: 'same-build', assetCount: 12 })
    })
    client.initializeOffline()
    await vi.advanceTimersByTimeAsync(0)
    expect(client.getOfflineStatus().state).toBe('preparing')
    expect(complete).toBeDefined()
    complete!()
    await vi.advanceTimersByTimeAsync(0)
    expect(client.getOfflineStatus().state).toBe('ready')
  })
})
