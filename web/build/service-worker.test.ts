// @vitest-environment node
import { readFile } from 'node:fs/promises'
import vm from 'node:vm'
import { describe, expect, it, vi } from 'vitest'

const origin = 'https://pixo.test'
const manifest = {
  version: 'build-one', cachePrefix: 'go-pixo-offline-scope-', cacheName: 'go-pixo-offline-scope-build-one', base: '/',
  assets: ['/index.html', '/main.wasm', '/assets/lazy-codec.wasm'].map((url) => ({ url, integrity: 'sha256-test' })),
}

async function worker() {
  const handlers: Record<string, (event: any) => void> = {}
  const stores = new Map<string, Map<string, Response>>()
  const storedRequests = new Map<string, Request>()
  const addAll = vi.fn(async (requests: Request[]) => {
    for (const request of requests) {
      storedRequests.set(request.url, request)
      stores.get(manifest.cacheName)!.set(request.url, new Response('static asset'))
    }
  })
  const cacheStorage = {
    has: async (name: string) => stores.has(name),
    keys: async () => [...stores.keys()],
    delete: vi.fn(async (name: string) => stores.delete(name)),
    open: async (name: string) => {
      if (!stores.has(name)) stores.set(name, new Map())
      return {
        addAll,
        keys: async () => [...stores.get(name)!.keys()].map((url) => new Request(url)),
        match: async (request: string | Request, options: CacheQueryOptions = {}) => {
          const incoming = typeof request === 'string' ? new Request(request) : request
          const response = stores.get(name)!.get(incoming.url)
          const vary = response?.headers.get('Vary')?.split(',').map((header) => header.trim()) ?? []
          if (!options.ignoreVary && vary.some((header) => header === '*' ||
            incoming.headers.get(header) !== storedRequests.get(incoming.url)?.headers.get(header))) return undefined
          return response
        },
      }
    },
  }
  const claim = vi.fn(async () => {})
  const skipWaiting = vi.fn()
  const network = vi.fn(async () => new Response('network'))
  const source = (await readFile(new URL('./service-worker.js', import.meta.url), 'utf8'))
    .replace('__OFFLINE_MANIFEST__', JSON.stringify(manifest))
  vm.runInNewContext(source, {
    self: { location: { origin }, clients: { claim }, skipWaiting, addEventListener: (name: string, handler: (event: any) => void) => { handlers[name] = handler } },
    caches: cacheStorage, URL, Set, Response, Request: class extends Request {
      constructor(url: string, options: RequestInit) { super(new URL(url, origin), options) }
    }, fetch: network,
  })
  async function dispatch(name: string, fields: Record<string, unknown> = {}) {
    let pending: Promise<unknown> | undefined
    let response: Promise<Response> | undefined
    handlers[name]({ ...fields, waitUntil: (promise: Promise<unknown>) => { pending = promise },
      respondWith: (promise: Promise<Response>) => { response = promise } })
    await pending
    return response
  }
  async function status(pageAssets = [`${origin}/index.html`], repair = false) {
    const reply = vi.fn()
    await dispatch('message', { data: { type: repair ? 'GO_PIXO_PREPARE_OFFLINE' : 'GO_PIXO_VERIFY_OFFLINE', pageAssets }, ports: [{ postMessage: reply }] })
    return reply.mock.calls[0][0]
  }
  return { stores, addAll, cacheStorage, claim, skipWaiting, network, dispatch, status }
}

describe('offline service worker boundaries', () => {
  it('installs the complete integrity-checked list and rejects incomplete installation', async () => {
    const sw = await worker()
    await sw.dispatch('install')
    const requests = sw.addAll.mock.calls[0][0]
    expect(requests.map((request) => request.url)).toEqual(manifest.assets.map((asset) => origin + asset.url))
    expect(requests.every((request) => request.integrity === 'sha256-test' && request.cache === 'reload')).toBe(true)
    expect(sw.skipWaiting).not.toHaveBeenCalled()
    expect(await sw.status()).toMatchObject({ ready: true, assetCount: 3 })
    sw.stores.get(manifest.cacheName)!.delete(`${origin}/main.wasm`)
    expect(await sw.status()).toMatchObject({ ready: false })
    expect(await sw.status(undefined, true)).toMatchObject({ ready: true })
    expect(sw.addAll).toHaveBeenCalledTimes(2)
    expect(await sw.status([`${origin}/assets/different-build.js`])).toMatchObject({ ready: false })

    const broken = await worker()
    broken.addAll.mockRejectedValueOnce(new Error('missing or integrity mismatch'))
    await expect(broken.dispatch('install')).rejects.toThrow('integrity mismatch')
    expect(await broken.status()).toMatchObject({ ready: false })
  })

  it('only serves static assets and navigation; never writes runtime/user requests', async () => {
    const sw = await worker()
    await sw.dispatch('install')
    const originalKeys = [...sw.stores.get(manifest.cacheName)!.keys()]
    const cached = await sw.dispatch('fetch', { request: new Request(`${origin}/main.wasm`) })
    expect(await cached!.text()).toBe('static asset')
    const navigation = await sw.dispatch('fetch', { request: { url: `${origin}/saved/route?private=value`, method: 'GET', mode: 'navigate' } })
    expect(await navigation!.text()).toBe('static asset')
    for (const request of [
      new Request(`${origin}/private-photo.png`), new Request(`${origin}/main.wasm?private=value`),
      new Request('https://another.test/main.wasm'), new Request(`blob:${origin}/user-image`),
      new Request(`${origin}/main.wasm`, { method: 'POST', body: 'user bytes' }),
    ]) expect(await sw.dispatch('fetch', { request })).toBeUndefined()
    expect([...sw.stores.get(manifest.cacheName)!.keys()]).toEqual(originalKeys)
    expect(sw.addAll).toHaveBeenCalledTimes(1)
    expect(sw.network).not.toHaveBeenCalled()
  })

  it('cleans only this application scope when the waiting build activates', async () => {
    const sw = await worker()
    sw.stores.set(`${manifest.cachePrefix}old`, new Map())
    sw.stores.set('another-app-cache', new Map())
    sw.stores.set('go-pixo-offline-other-scope-old', new Map())
    await sw.dispatch('install')
    expect(sw.cacheStorage.delete).not.toHaveBeenCalled()
    await sw.dispatch('activate')
    expect(sw.cacheStorage.delete.mock.calls).toEqual([[`${manifest.cachePrefix}old`]])
    expect(sw.stores.has('another-app-cache')).toBe(true)
    expect(sw.stores.has('go-pixo-offline-other-scope-old')).toBe(true)
    expect(sw.claim).toHaveBeenCalledOnce()
  })

  it('serves redirected cached HTML as a navigation-safe response with the same bytes and headers', async () => {
    const sw = await worker()
    await sw.dispatch('install')
    const shell = new Response('<!doctype html><title>Verified app</title>', {
      status: 200, statusText: 'OK', headers: { 'Content-Type': 'text/html', 'X-Build': 'build-one' },
    })
    // Cloudflare canonicalizes /index.html to /. Cache.addAll follows that redirect.
    Object.defineProperty(shell, 'redirected', { value: true })
    sw.stores.get(manifest.cacheName)!.set(`${origin}/index.html`, shell)
    const response = await sw.dispatch('fetch', {
      request: { url: `${origin}/`, method: 'GET', mode: 'navigate', redirect: 'manual' },
    })
    expect(response!.redirected).toBe(false)
    expect(response!.status).toBe(200)
    expect(response!.statusText).toBe('OK')
    expect(response!.headers.get('Content-Type')).toBe('text/html')
    expect(response!.headers.get('X-Build')).toBe('build-one')
    expect(await response!.text()).toBe('<!doctype html><title>Verified app</title>')
    expect(sw.network).not.toHaveBeenCalled()
    expect(sw.addAll).toHaveBeenCalledTimes(1)
  })

  it.each(['open', 'match'] as const)('falls back to the original network request when cache %s rejects', async (failure) => {
    const sw = await worker()
    await sw.dispatch('install')
    const cachedURLs = [...sw.stores.get(manifest.cacheName)!.keys()]
    const storageError = new DOMException('Cache storage is unavailable', 'UnknownError')
    const cache = await sw.cacheStorage.open(manifest.cacheName)
    const open = vi.spyOn(sw.cacheStorage, 'open')
    if (failure === 'open') open.mockRejectedValue(storageError)
    else open.mockResolvedValue({ ...cache, match: vi.fn().mockRejectedValue(storageError) })

    const requests = [new Request(`${origin}/main.wasm`),
      { url: `${origin}/saved/route?private=value`, method: 'GET', mode: 'navigate' }]
    for (const request of requests) {
      const response = await sw.dispatch('fetch', { request })
      expect(await response!.text()).toBe('network')
      expect(sw.network).toHaveBeenLastCalledWith(request)
    }
    expect(sw.network).toHaveBeenCalledTimes(2)
    expect(sw.addAll).toHaveBeenCalledTimes(1)
    expect([...sw.stores.get(manifest.cacheName)!.keys()]).toEqual(cachedURLs)
  })

  it('serves a verified static asset when module Origin headers differ from the precache request', async () => {
    const sw = await worker()
    await sw.dispatch('install')
    const url = `${origin}/main.wasm`
    sw.stores.get(manifest.cacheName)!.set(url, new Response('verified static payload', { headers: { Vary: 'Origin' } }))
    const moduleRequest = new Request(url, { headers: { Origin: origin } })
    const cache = await sw.cacheStorage.open(manifest.cacheName)
    expect(await cache.match(moduleRequest)).toBeUndefined()
    expect(await cache.match(moduleRequest, { ignoreVary: true })).toBeDefined()
    const response = await sw.dispatch('fetch', { request: moduleRequest })
    expect(await response!.text()).toBe('verified static payload')
    expect(sw.network).not.toHaveBeenCalled()
  })
})
