import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'
import { readdir, readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { join } from 'node:path'
import { createServer } from 'node:http'
import { unzipSync } from 'fflate'

async function distFiles(directory: string, prefix = ''): Promise<string[]> {
  const entries = await readdir(directory, { withFileTypes: true })
  const nested = await Promise.all(entries.map((entry) => entry.isDirectory()
    ? distFiles(join(directory, entry.name), `${prefix}${entry.name}/`) : [`${prefix}${entry.name}`]))
  return nested.flat().sort()
}

async function cacheURLs(page: Page): Promise<string[]> {
  return page.evaluate(async () => {
    const names = (await caches.keys()).filter((name) => name.startsWith('go-pixo-offline-'))
    const keys = await Promise.all(names.map(async (name) => (await caches.open(name)).keys()))
    return keys.flat().map((request) => new URL(request.url).pathname).sort()
  })
}

async function uploadFreshImage(page: Page, seed = 0) {
  await page.evaluate(async (variation) => {
    // Created only after the offline reload, so it cannot be a warmed image/result.
    const canvas = document.createElement('canvas')
    canvas.width = 64
    canvas.height = 48
    const context = canvas.getContext('2d')!
    const image = context.createImageData(64, 48)
    for (let i = 0; i < image.data.length; i += 4) {
      image.data[i] = (i * 7 + variation * 53) % 256
      image.data[i + 1] = (i * 3 + variation * 97) % 256
      image.data[i + 2] = (i * 11 + variation * 31) % 256
      image.data[i + 3] = 255
    }
    context.putImageData(image, 0, 0)
    const blob = await new Promise<Blob>((resolve, reject) => canvas.toBlob(
      (value) => value ? resolve(value) : reject(new Error('PNG fixture failed')), 'image/png'))
    const transfer = new DataTransfer()
    transfer.items.add(new File([blob], 'private-fresh-offline.png', { type: 'image/png' }))
    const input = document.querySelector<HTMLInputElement>('input[type="file"]')!
    input.files = transfer.files
    input.dispatchEvent(new Event('change', { bubbles: true }))
  }, seed)
}

async function outputMetadata(page: Page) {
  return page.locator('img[alt="Compressed"]').evaluate(async (image: HTMLImageElement) => {
    const blob = await fetch(image.src).then((response) => response.blob())
    const bytes = new Uint8Array(await blob.arrayBuffer())
    const bitmap = await createImageBitmap(blob)
    const metadata = { type: blob.type, width: bitmap.width, height: bitmap.height,
      size: bytes.length, header: Array.from(bytes.slice(0, 32)) }
    bitmap.close()
    return metadata
  })
}

async function selectOutput(page: Page, format: string, mime: string) {
  await page.getByTestId('download-format-toggle').click()
  await page.getByTestId(`download-as-${format}`).click()
  await expect.poll(async () => (await outputMetadata(page)).type, { timeout: 60000 }).toBe(mime)
  const metadata = await outputMetadata(page)
  expect(metadata.width).toBe(64)
  expect(metadata.height).toBe(48)
  expect(metadata.size).toBeGreaterThan(0)
  return metadata
}

test('complete production cache reloads, compresses fresh images and downloads a complete ZIP offline', async ({ page, context }) => {
  // Prove the service-worker cache alone can boot the app; HTTP cache must not mask misses.
  const network = await context.newCDPSession(page)
  await network.send('Network.enable')
  await network.send('Network.setCacheDisabled', { cacheDisabled: true })
  const errors: string[] = []
  page.on('pageerror', (error) => errors.push(error.message))
  const failedRequests: { url: string; origin: string | undefined; error: string | undefined }[] = []
  context.on('requestfailed', (request) => failedRequests.push({ url: request.url(),
    origin: request.headers().origin, error: request.failure()?.errorText }))
  const nonGetRequests: string[] = []
  context.on('request', (request) => {
    if (!['GET', 'HEAD'].includes(request.method())) nonGetRequests.push(request.url())
  })
  // This URL is outside the precache list and must reach the actual network.
  const networkAvailable = () => page.evaluate(async () => {
    try {
      await fetch(`/offline-network-probe.txt?nonce=${crypto.randomUUID()}`, { cache: 'no-store' })
      return true
    } catch {
      return false
    }
  })
  await page.goto('/')
  await expect(page.getByTestId('offline-status')).toContainText('Ready offline', { timeout: 60000 })
  expect(await page.evaluate(() => navigator.serviceWorker.controller?.state)).toBe('activated')

  const directory = fileURLToPath(new URL('../dist/', import.meta.url))
  const files = (await distFiles(directory)).filter((file) => file !== 'sw.js')
  const cached = await cacheURLs(page)
  expect(cached).toEqual(files.map((file) => '/' + file.split('/').map(encodeURIComponent).join('/')).sort())
  // This is a real codec build, not just an app-shell cache with canvas fallback.
  expect(files.some((file) => /avif.*\.wasm$/i.test(file))).toBe(true)
  expect(await networkAvailable()).toBe(true)

  await context.setOffline(true)
  expect(await networkAvailable()).toBe(false)
  try {
    await page.reload()
    await expect(page.getByTestId('offline-status')).toContainText('Ready offline', { timeout: 30000 })
  } catch (error) {
    console.error('Offline reload diagnostics', JSON.stringify({
      pageErrors: errors, failedRequests, body: await page.locator('body').innerText(),
      cacheReads: await page.evaluate(async () => {
        const name = (await caches.keys()).find((value) => value.startsWith('go-pixo-offline-'))!
        const cache = await caches.open(name)
        const keys = await cache.keys()
        const assets = [...document.querySelectorAll<HTMLScriptElement | HTMLLinkElement>('script[type="module"][src], link[rel="stylesheet"]')]
          .map((element) => element instanceof HTMLScriptElement ? element.src : element.href)
        return Promise.all(assets.map(async (url) => {
          const cached = await cache.match(url, { ignoreVary: true })
          return { url,
            cachedOrigin: keys.find((key) => key.url === url)?.headers.get('Origin'),
            vary: cached?.headers.get('Vary'), ignoreVaryMatch: !!cached }
        }))
      }),
    }))
    throw error
  }
  await uploadFreshImage(page)
  await expect(page.locator('[data-testid="file-item-done"], [data-testid="file-item-optimized"]')).toBeVisible({ timeout: 60000 })

  const png = await selectOutput(page, 'PNG', 'image/png')
  expect(png.header.slice(0, 8)).toEqual([137, 80, 78, 71, 13, 10, 26, 10])
  const jpeg = await selectOutput(page, 'JPEG', 'image/jpeg')
  expect(jpeg.header.slice(0, 2)).toEqual([255, 216])
  const webp = await selectOutput(page, 'WebP', 'image/webp')
  expect(String.fromCharCode(...webp.header.slice(0, 4))).toBe('RIFF')
  expect(String.fromCharCode(...webp.header.slice(8, 12))).toBe('WEBP')
  const avif = await selectOutput(page, 'AVIF', 'image/avif')
  expect(String.fromCharCode(...avif.header.slice(4, 8))).toBe('ftyp')
  expect(String.fromCharCode(...avif.header.slice(8))).toMatch(/avif|avis/)

  // Keep the first AVIF result, then add a second fresh image with the same name
  // and different pixels. Neither input/result existed before going offline.
  await uploadFreshImage(page, 1)
  await expect(page.getByTestId('batch-summary')).toContainText('2 completed, 0 failed, 0 remaining.', { timeout: 60000 })
  await expect(page.locator('[data-testid="file-item-done"], [data-testid="file-item-optimized"]')).toHaveCount(2)
  // Chromium's navigator.onLine hint can stay true after an emulated offline
  // reload. Prove an uncached request is disconnected before accepting the ZIP.
  expect(await networkAvailable()).toBe(false)
  expect(failedRequests.some((request) => request.url.includes('/offline-network-probe.txt') &&
    request.error === 'net::ERR_INTERNET_DISCONNECTED')).toBe(true)
  const [download] = await Promise.all([
    page.waitForEvent('download'),
    page.getByRole('button', { name: 'Download ZIP', exact: true }).click(),
  ])
  expect(download.suggestedFilename()).toBe('go-pixo-compressed.zip')
  const archivePath = await download.path()
  expect(archivePath).not.toBeNull()
  const entries = unzipSync(new Uint8Array(await readFile(archivePath!)))
  const names = Object.keys(entries)
  expect(names).toHaveLength(2)
  expect(names).toEqual(expect.arrayContaining([
    'compressed_private-fresh-offline.avif', 'compressed_private-fresh-offline (2).avif',
  ]))
  const images = Object.values(entries).map((bytes) => Array.from(bytes))
  expect(images[0]).not.toEqual(images[1])
  for (const bytes of images) {
    expect(bytes.length).toBeGreaterThan(32)
    expect(String.fromCharCode(...bytes.slice(4, 8))).toBe('ftyp')
    expect(String.fromCharCode(...bytes.slice(8, 32))).toMatch(/avif|avis/)
  }
  const dimensions = await page.evaluate(async (payloads) => Promise.all(payloads.map(async (bytes) => {
    const bitmap = await createImageBitmap(new Blob([new Uint8Array(bytes)], { type: 'image/avif' }))
    const size = { width: bitmap.width, height: bitmap.height }
    bitmap.close()
    return size
  })), images)
  expect(dimensions).toEqual([{ width: 64, height: 48 }, { width: 64, height: 48 }])

  // Exercise a same-origin SPA route and ensure it reloads from the cached shell.
  await page.goto('/offline-check?source=private')
  await expect(page.getByTestId('offline-status')).toContainText('Ready offline', { timeout: 30000 })
  expect(await cacheURLs(page)).toEqual(cached)
  expect(cached.some((url) => /private|blob:|data:/.test(url))).toBe(false)
  expect(nonGetRequests).toEqual([])
  expect(errors).toEqual([])
})

test('an incomplete cache is not reported as ready after eviction', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByTestId('offline-status')).toContainText('Ready offline', { timeout: 60000 })
  await page.evaluate(async () => {
    for (const name of (await caches.keys()).filter((value) => value.startsWith('go-pixo-offline-'))) {
      await (await caches.open(name)).delete('/main.wasm')
    }
    window.dispatchEvent(new Event('offline'))
  })
  await expect(page.getByTestId('offline-status')).toContainText('Offline files are incomplete', { timeout: 10000 })
  await expect(page.getByTestId('offline-status')).not.toContainText('Ready offline')
  // Reconnecting/reloading the same deployment repairs the list; no SW update is needed.
  await page.reload()
  await expect(page.getByTestId('offline-status')).toContainText('Ready offline', { timeout: 60000 })
})

test('Cloudflare-style redirected HTML recovers an existing worker and supports offline navigation', async ({ page: originalPage, context }) => {
  let page = originalPage
  const directory = fileURLToPath(new URL('../dist/', import.meta.url))
  const files = new Set(await distFiles(directory))
  const fixedWorker = await readFile(join(directory, 'sw.js'), 'utf8')
  const normalizationStart = fixedWorker.indexOf('        // Cloudflare redirects')
  const normalizationEnd = fixedWorker.indexOf('        return cached;', normalizationStart)
  expect(normalizationStart).toBeGreaterThan(0)
  expect(normalizationEnd).toBeGreaterThan(normalizationStart)
  const version = JSON.parse(fixedWorker.match(/const manifest = (.*);/)![1]).version as string
  // Reproduce the deployed predecessor without changing final dist or its asset hashes.
  const brokenWorker = (fixedWorker.slice(0, normalizationStart) + fixedWorker.slice(normalizationEnd))
    .replaceAll(version, `${version}-broken`)
  let repaired = false
  const types: Record<string, string> = {
    js: 'text/javascript', css: 'text/css', html: 'text/html', wasm: 'application/wasm',
    webmanifest: 'application/manifest+json', png: 'image/png', svg: 'image/svg+xml',
  }
  let canonicalRedirects = 0
  const server = createServer((request, response) => {
    void (async () => {
      const pathname = new URL(request.url!, 'http://localhost').pathname
      if (pathname === '/sw.js') {
        response.writeHead(200, { 'Content-Type': 'text/javascript', 'Cache-Control': 'no-cache' })
        response.end(repaired ? fixedWorker : brokenWorker)
        return
      }
      if (pathname === '/index.html') {
        canonicalRedirects++
        response.writeHead(307, { Location: '/' })
        response.end()
        return
      }
      const requested = decodeURIComponent(pathname.slice(1)) || 'index.html'
      const file = files.has(requested) ? requested : 'index.html'
      response.setHeader('Content-Type', types[file.split('.').at(-1)!] ?? 'application/octet-stream')
      response.end(await readFile(join(directory, file)))
    })().catch(() => { response.statusCode = 500; response.end() })
  })
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve))
  const address = server.address()
  if (address === null || typeof address === 'string') throw new Error('Canonical-redirect server did not start')
  const url = `http://127.0.0.1:${address.port}`
  try {
    await page.goto(url)
    await expect(page.getByTestId('offline-status')).toContainText('Ready offline', { timeout: 60000 })
    expect(canonicalRedirects).toBeGreaterThan(0)
    expect(await page.evaluate(async () => {
      const name = (await caches.keys()).find(value => value.startsWith('go-pixo-offline-'))!
      return (await (await caches.open(name)).match('/index.html', { ignoreVary: true }))!.redirected
    })).toBe(true)
    await page.evaluate(async () => {
      localStorage.setItem('recovery-sentinel', 'preserve-me')
      await (await caches.open('recovery-sentinel')).put('/recovery-sentinel', new Response('preserve-me'))
    })
    await expect(page.reload()).rejects.toThrow('ERR_FAILED')
    // A hard refresh bypasses the broken worker once. The app must request its own
    // update, since registering an unchanged URL/options can reuse the old registration.
    // Preserve the browser context and registration while avoiding Chromium's
    // asynchronous error-document navigation on the failed page.
    await page.close()
    page = await context.newPage()
    repaired = true
    const cdp = await context.newCDPSession(page)
    await cdp.send('Network.enable')
    await cdp.send('Network.setBypassServiceWorker', { bypass: true })
    await page.goto(url)
    await expect(page.getByTestId('offline-status')).toContainText('Ready offline', { timeout: 60000 })
    expect(await page.evaluate(async () => ({
      stored: localStorage.getItem('recovery-sentinel'),
      cached: await (await (await caches.open('recovery-sentinel')).match('/recovery-sentinel'))!.text(),
      oldCache: (await caches.keys()).some(name => name.endsWith('-broken')),
    }))).toEqual({ stored: 'preserve-me', cached: 'preserve-me', oldCache: false })
    await cdp.send('Network.setBypassServiceWorker', { bypass: false })
    await cdp.detach()
    await page.reload()
    await expect(page.getByTestId('offline-status')).toContainText('Ready offline')
    await context.setOffline(true)
    await page.reload()
    await expect(page.getByTestId('offline-status')).toContainText('Ready offline')
    await page.goto(`${url}/saved/route`)
    await expect(page.getByTestId('offline-status')).toContainText('Ready offline')
  } finally {
    await context.setOffline(false)
    await new Promise<void>((resolve) => { server.close(() => resolve()); server.closeAllConnections() })
  }
})

test('a missing Go asset served as a successful SPA fallback fails offline installation', async ({ page }) => {
  const directory = fileURLToPath(new URL('../dist/', import.meta.url))
  const files = new Set(await distFiles(directory))
  const types: Record<string, string> = {
    js: 'text/javascript', css: 'text/css', html: 'text/html', wasm: 'application/wasm',
    webmanifest: 'application/manifest+json', png: 'image/png', svg: 'image/svg+xml',
  }
  // A separate origin isolates this failed install without mutating final dist.
  const server = createServer((request, response) => {
    void (async () => {
      const requested = new URL(request.url!, 'http://localhost').pathname.slice(1) || 'index.html'
      const file = requested !== 'main.wasm' && files.has(requested) ? requested : 'index.html'
      response.setHeader('Content-Type', types[file.split('.').at(-1)!] ?? 'application/octet-stream')
      response.end(await readFile(join(directory, file)))
    })().catch(() => { response.statusCode = 500; response.end() })
  })
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve))
  const address = server.address()
  if (address === null || typeof address === 'string') throw new Error('Fault-injection server did not start')
  try {
    await page.goto(`http://127.0.0.1:${address.port}/`)
    await expect(page.getByTestId('offline-status')).toContainText('Offline setup failed', { timeout: 60000 })
    await expect(page.getByTestId('offline-status')).not.toContainText('Ready offline')
    expect(await page.evaluate(() => navigator.serviceWorker.controller)).toBeNull()
  } finally {
    await new Promise<void>((resolve) => { server.close(() => resolve()); server.closeAllConnections() })
  }
})


test('production PNG reports measured stage work, advances its clock and allows cancellation', async ({page}) => {
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await page.addInitScript(() => {
    const state = {updates: [] as {phase: string; phasePercent?: number; at: number}[], elapsed: [] as string[]}
    ;(window as any).__progressEvidence = state
    const NativeWorker = window.Worker
    window.Worker = class extends NativeWorker {
      constructor(url: string | URL, options?: WorkerOptions) {
        super(url, options)
        this.addEventListener('message', event => {
          if (event.data?.type === 'progress') state.updates.push({...event.data, at: performance.now()})
        })
      }
    }
    new MutationObserver(() => {
      const elapsed = document.querySelector('[data-testid="compression-elapsed"]')?.textContent
      if (elapsed && state.elapsed.at(-1) !== elapsed) state.elapsed.push(elapsed)
    }).observe(document, {subtree: true, childList: true, characterData: true})
  })
  await page.goto('/')
  await expect(page.getByTestId('offline-status')).toContainText('Ready offline', {timeout: 60000})
  const fixture = fileURLToPath(new URL('./fixtures/benchmark/codex-still-life.png', import.meta.url))
  const started = Date.now()
  await page.locator('input[type="file"]').setInputFiles(fixture)
  await expect(page.locator('img[alt="Compressed"]')).toBeVisible({timeout: 30000})
  const wallMs = Date.now() - started
  const evidence = await page.evaluate(() => (window as any).__progressEvidence)
  // Interior updates are throttled, so a fast phase may legitimately report only
  // its 0%/100% boundaries. Every phase must still report measured work, and
  // the whole encode must show at least one interior update.
  const measuredPhases = ['histogram', 'palette', 'quantize']
  for (const phase of measuredPhases) {
    const values = evidence.updates.filter((update: any) => update.phase === phase && typeof update.phasePercent === 'number').map((update: any) => update.phasePercent)
    expect(values.length, `${phase} reported measured work`).toBeGreaterThan(0)
  }
  expect(evidence.updates.some((update: any) => measuredPhases.includes(update.phase) && update.phasePercent > 0 && update.phasePercent < 100)).toBe(true)
  expect(new Set(evidence.elapsed).size).toBeGreaterThan(2)
  const metadata = await outputMetadata(page)
  expect(metadata.type).toBe('image/png')
  expect([metadata.width, metadata.height]).toEqual([1536, 1024])
  expect(metadata.header.slice(0,8)).toEqual([137,80,78,71,13,10,26,10])
  console.info(JSON.stringify({wallMs, ...metadata, updates: evidence.updates.length, elapsed: evidence.elapsed,
    measured: ['histogram','palette','quantize','filtering'].map(phase => ({phase, values: evidence.updates.filter((update: any) => update.phase === phase).map((update: any) => update.phasePercent)}))}))

  await page.getByRole('button', {name: 'Clear all'}).click()
  await page.locator('input[type="file"]').setInputFiles(fixture)
  await expect(page.getByTestId('compression-phase')).toHaveText('Building palette', {timeout: 15000})
  await page.getByRole('button', {name: 'Remove image', exact: true}).click()
  await expect(page.getByText('codex-still-life.png', {exact: true})).not.toBeVisible()
  await expect(page.getByRole('progressbar')).not.toBeVisible()
  // A new job can claim the cancelled slot; late updates cannot revive its UI.
  await page.locator('input[type="file"]').setInputFiles(fileURLToPath(new URL('./fixtures/sample-32x32.png', import.meta.url)))
  await expect(page.locator('img[alt="Compressed"]')).toBeVisible({timeout: 15000})
  expect(errors).toEqual([])
})
