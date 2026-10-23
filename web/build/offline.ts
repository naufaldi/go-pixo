import { createHash } from 'node:crypto'
import { readdir, readFile, writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import type { Plugin, ResolvedConfig } from 'vite'

export type OfflineManifest = {
  version: string
  cachePrefix: string
  cacheName: string
  base: string
  assets: { url: string; integrity: string }[]
}

const hash = (value: string | Uint8Array) => createHash('sha256').update(value)

async function emittedFiles(directory: string, prefix = ''): Promise<string[]> {
  const entries = await readdir(directory, { withFileTypes: true })
  const nested = await Promise.all(entries.map(async (entry) => {
    const relative = `${prefix}${entry.name}`
    if (entry.isDirectory()) return emittedFiles(join(directory, entry.name), `${relative}/`)
    if (!entry.isFile()) throw new Error(`Offline build cannot precache non-file: ${relative}`)
    return [relative]
  }))
  return nested.flat().sort()
}

export async function generateOfflineWorker(outDir: string, base: string): Promise<OfflineManifest> {
  if (!base.startsWith('/') || !base.endsWith('/') || base.startsWith('//') || /[?#]/.test(base)) {
    throw new Error('Offline builds require an absolute same-origin Vite base path, such as "/".')
  }
  const template = await readFile(new URL('./service-worker.js', import.meta.url), 'utf8')
  const files = (await emittedFiles(outDir)).filter((file) => file !== 'sw.js')
  for (const required of ['index.html', 'main.wasm', 'wasm_exec.js', 'site.webmanifest']) {
    if (!files.includes(required)) throw new Error(`Offline build is missing ${required}. Run bun run build:wasm first.`)
  }
  const assets = await Promise.all(files.map(async (file) => ({
    url: base + file.split('/').map(encodeURIComponent).join('/'),
    integrity: `sha256-${hash(await readFile(join(outDir, file))).digest('base64')}`,
  })))
  // Include the worker implementation, public file bytes and scope, not only hashed JS filenames.
  const version = hash(JSON.stringify({ base, assets, template })).digest('hex').slice(0, 24)
  const cachePrefix = `go-pixo-offline-${hash(base).digest('hex').slice(0, 12)}-`
  const manifest = { version, cachePrefix, cacheName: cachePrefix + version, base, assets }
  await writeFile(join(outDir, 'sw.js'), template.replace('__OFFLINE_MANIFEST__', JSON.stringify(manifest)))
  return manifest
}

export function offlinePlugin(): Plugin {
  let config: ResolvedConfig
  return {
    name: 'go-pixo-complete-offline-build',
    apply: 'build',
    configResolved(resolved) { config = resolved },
    // Workers, their lazy chunks/WASM and publicDir have all been written at this point.
    writeBundle: {
      order: 'post',
      sequential: true,
      async handler() {
        const manifest = await generateOfflineWorker(resolve(config.root, config.build.outDir), config.base)
        config.logger.info(`Offline: ${manifest.assets.length} verified assets (${manifest.version})`)
      },
    },
  }
}
