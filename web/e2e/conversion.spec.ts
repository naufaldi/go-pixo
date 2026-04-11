import { test, expect } from '@playwright/test'
import { writeFileSync } from 'fs'
import { join } from 'path'

// Minimal 1x1 red PNG (67 bytes) — valid PNG header + IHDR + IDAT + IEND
const TINY_PNG_BASE64 = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwADhQGAWjR9awAAAABJRU5ErkJggg=='

function writeTinyPng(path: string): void {
  writeFileSync(path, Buffer.from(TINY_PNG_BASE64, 'base64'))
}

test.describe('go-pixo', () => {
  test('app loads with correct title', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('h1')).toContainText('Go-Pixo')
  })

  test('WASM initializes successfully', async ({ page }) => {
    await page.goto('/')
    await page.waitForFunction(
      () => typeof (window as any).encodePng === 'function',
      { timeout: 15000 }
    )
    const ready = await page.evaluate(() => typeof (window as any).encodePng === 'function')
    expect(ready).toBe(true)
  })

  test('PNG file is accepted and compresses to Done status', async ({ page }) => {
    const fixturePath = join(__dirname, 'fixtures', 'test.png')
    writeTinyPng(fixturePath)

    await page.goto('/')
    await page.waitForFunction(
      () => typeof (window as any).encodePng === 'function',
      { timeout: 15000 }
    )

    // Upload via file input (find the hidden file input in the dropzone)
    const fileInput = page.locator('input[type="file"]')
    await fileInput.setInputFiles(fixturePath)

    // Wait for compression to complete (status changes to Done)
    // FileQueue items should show compression savings
    await page.waitForSelector('[data-testid="file-item-done"], .text-green-400, .text-emerald-400', {
      timeout: 30000,
    })
  })

  test('recompressJpeg WASM function is registered', async ({ page }) => {
    await page.goto('/')
    await page.waitForFunction(
      () => typeof (window as any).encodePng === 'function',
      { timeout: 15000 }
    )
    const hasRecompressJpeg = await page.evaluate(
      () => typeof (window as any).recompressJpeg === 'function'
    )
    expect(hasRecompressJpeg).toBe(true)
  })
})
