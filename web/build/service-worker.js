/* Build replaces this value after all Vite and public assets have been emitted. */
const manifest = __OFFLINE_MANIFEST__;
const assetURLs = new Set(manifest.assets.map((asset) => new URL(asset.url, self.location.origin).href));
const shellURL = new URL(`${manifest.base}index.html`, self.location.origin).href;
let preparation;

function prepareCache() {
  // Deduplicate install/repair requests. Both may write only this exact build list.
  preparation ||= (async () => {
    const cache = await caches.open(manifest.cacheName);
    await cache.addAll(manifest.assets.map((asset) => new Request(asset.url, {
      cache: 'reload',
      credentials: 'same-origin',
      integrity: asset.integrity,
    })));
  })().finally(() => { preparation = undefined; });
  return preparation;
}

self.addEventListener('install', (event) => {
  // addAll is atomic. Integrity rejects missing assets served as an SPA fallback,
  // stale CDN responses and mixed deployments, including lazy codec WASM.
  event.waitUntil(prepareCache());
  // No skipWaiting: existing tabs keep their matching JS, workers and codecs.
});

self.addEventListener('activate', (event) => {
  event.waitUntil((async () => {
    const names = await caches.keys();
    await Promise.all(names.filter((name) => name.startsWith(manifest.cachePrefix) && name !== manifest.cacheName)
      .map((name) => caches.delete(name)));
    await self.clients.claim();
  })());
});

self.addEventListener('fetch', (event) => {
  const request = event.request;
  const url = new URL(request.url);
  if (request.method !== 'GET' || url.origin !== self.location.origin) return;
  const isNavigation = request.mode === 'navigate' && url.pathname.startsWith(manifest.base);
  if (!isNavigation && !assetURLs.has(url.href)) return;
  event.respondWith((async () => {
    try {
      const cache = await caches.open(manifest.cacheName);
      // Every entry has this build's verified bytes. Header variants (notably
      // module requests' Origin) cannot select a different representation here.
      const cached = await cache.match(isNavigation ? shellURL : request, { ignoreVary: true });
      if (cached) {
        // Cloudflare redirects /index.html to /. Navigations use redirect mode
        // "manual", which cannot accept that cached redirect history. Keep the
        // integrity-verified body/headers while returning a fresh response.
        if (isNavigation && cached.redirected) {
          return new Response(cached.body, {
            status: cached.status,
            statusText: cached.statusText,
            headers: cached.headers,
          });
        }
        return cached;
      }
    } catch {
      // Storage may be unavailable even while online. Keep navigation/assets usable.
    }
    // Never write runtime responses. Images, uploads, blobs and arbitrary URLs
    // cannot enter the app cache, even when they are fetched by a controlled tab.
    return fetch(request);
  })());
});

self.addEventListener('message', (event) => {
  if (!['GO_PIXO_VERIFY_OFFLINE', 'GO_PIXO_PREPARE_OFFLINE'].includes(event.data?.type) || !event.ports[0]) return;
  event.waitUntil((async () => {
    let ready = false;
    try {
      if (event.data.type === 'GO_PIXO_PREPARE_OFFLINE') await prepareCache();
      if (await caches.has(manifest.cacheName)) {
        const cache = await caches.open(manifest.cacheName);
        const cachedURLs = new Set((await cache.keys()).map((request) => request.url));
        const pageAssets = event.data.pageAssets;
        ready = Array.isArray(pageAssets) && pageAssets.every((url) => assetURLs.has(url)) &&
          [...assetURLs].every((url) => cachedURLs.has(url));
      }
    } catch {
      // Eviction/storage access failures must remove the ready claim.
    }
    event.ports[0].postMessage({
      type: 'GO_PIXO_OFFLINE_STATUS',
      ready,
      version: manifest.version,
      assetCount: manifest.assets.length,
    });
  })());
});
