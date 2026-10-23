// @vitest-environment node
import { mkdtemp, mkdir, readFile, realpath, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterEach, describe, expect, it } from 'vitest'
import { build } from 'vite'
import { generateOfflineWorker, offlinePlugin } from './offline'

const directories: string[] = []
afterEach(async () => {
  await Promise.all(directories.splice(0).map((directory) => rm(directory, { recursive: true, force: true })))
})

async function fixture() {
  const directory = await mkdtemp(join(tmpdir(), 'go-pixo-offline-'))
  directories.push(directory)
  await mkdir(join(directory, 'assets', 'codecs'), { recursive: true })
  for (const name of ['index.html', 'main.wasm', 'wasm_exec.js', 'site.webmanifest',
    'assets/app.js', 'assets/worker.js', 'assets/codecs/avif.js', 'assets/codecs/avif.wasm']) {
    await writeFile(join(directory, name), `fixture:${name}`)
  }
  return directory
}

describe('production offline asset generation', () => {
  it('includes every emitted file, including public Go assets and lazy worker codecs', async () => {
    const directory = await fixture()
    const manifest = await generateOfflineWorker(directory, '/')
    expect(manifest.assets.map((asset) => asset.url)).toEqual([
      '/assets/app.js', '/assets/codecs/avif.js', '/assets/codecs/avif.wasm',
      '/assets/worker.js', '/index.html', '/main.wasm', '/site.webmanifest', '/wasm_exec.js',
    ])
    expect(manifest.assets.every((asset) => /^sha256-[A-Za-z0-9+/]{43}=$/.test(asset.integrity))).toBe(true)
    const worker = await readFile(join(directory, 'sw.js'), 'utf8')
    expect(worker).toContain(manifest.version)
    expect(worker).toContain('/assets/codecs/avif.wasm')
    // Repeated builds never precache the generated worker itself.
    expect(await generateOfflineWorker(directory, '/')).toEqual(manifest)
  })

  it('changes cache identity for any changed asset and isolates different app scopes', async () => {
    const directory = await fixture()
    const original = await generateOfflineWorker(directory, '/')
    await writeFile(join(directory, 'main.wasm'), 'new Go build')
    const changed = await generateOfflineWorker(directory, '/')
    expect(changed.cacheName).not.toBe(original.cacheName)
    expect(changed.cachePrefix).toBe(original.cachePrefix)
    const nested = await generateOfflineWorker(directory, '/tools/pixo/')
    expect(nested.cachePrefix).not.toBe(changed.cachePrefix)
    expect(nested.assets.every((asset) => asset.url.startsWith('/tools/pixo/'))).toBe(true)
  })

  it('fails the build when a required Go runtime asset is absent', async () => {
    const directory = await fixture()
    await rm(join(directory, 'main.wasm'))
    await expect(generateOfflineWorker(directory, '/')).rejects.toThrow('main.wasm')
  })

  it('runs after Vite writes lazy worker JS/WASM and copies public assets', async () => {
    const directory = await realpath(await mkdtemp(join(tmpdir(), 'go-pixo-vite-offline-')))
    directories.push(directory)
    await mkdir(join(directory, 'public'))
    for (const name of ['main.wasm', 'wasm_exec.js', 'site.webmanifest']) {
      await writeFile(join(directory, 'public', name), `public:${name}`)
    }
    await writeFile(join(directory, 'index.html'), '<script type="module" src="/main.js"></script>')
    await writeFile(join(directory, 'main.js'), 'new Worker(new URL("./worker.js", import.meta.url), {type:"module"})')
    await writeFile(join(directory, 'worker.js'), 'self.onmessage=async()=>self.postMessage((await import("./codec.js")).url)')
    await writeFile(join(directory, 'codec.js'), 'import url from "./codec.wasm?url";export {url}')
    await writeFile(join(directory, 'codec.wasm'), new Uint8Array(8192))
    await build({
      root: directory, configFile: false, logLevel: 'silent', worker: { format: 'es' },
      build: { outDir: 'dist', assetsInlineLimit: 0 }, plugins: [offlinePlugin()],
    })
    const source = await readFile(join(directory, 'dist', 'sw.js'), 'utf8')
    const manifest = JSON.parse(source.match(/const manifest = ([^\n]+);/)![1])
    const urls = manifest.assets.map((asset: { url: string }) => asset.url)
    expect(urls).toContain('/main.wasm')
    expect(urls.some((url: string) => /\/worker-[^/]+\.js$/.test(url))).toBe(true)
    expect(urls.some((url: string) => /\/codec-[^/]+\.js$/.test(url))).toBe(true)
    expect(urls.some((url: string) => /\/codec-[^/]+\.wasm$/.test(url))).toBe(true)
  })
})
