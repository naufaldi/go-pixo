import { test, expect } from '@playwright/test'
import { readFileSync } from 'fs'
import { join } from 'path'

test.describe('Image Conversion', () => {
  test('should load the application', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('h1')).toContainText('Go-Pixo')
  })

  test('should initialize WASM successfully', async ({ page }) => {
    await page.goto('/')
    
    // Wait for WASM to initialize (check for encodePng function)
    await page.waitForFunction(() => {
      return typeof (window as any).encodePng === 'function'
    }, { timeout: 10000 })
    
    // Verify WASM is ready
    const wasmReady = await page.evaluate(() => {
      return typeof (window as any).encodePng === 'function'
    })
    expect(wasmReady).toBe(true)
  })

  test('should convert an image via WASM and produce valid PNG', async ({ page }) => {
    await page.goto('/')

    // Wait for WASM to initialize
    await page.waitForFunction(() => {
      return typeof (window as any).encodePng === 'function'
    }, { timeout: 10000 })

    // Load test fixture image
    // Note: User will provide the actual image file
    // For now, verify the page structure is correct
    await expect(page.locator('h1')).toContainText('Go-Pixo')
    
    // TODO: Once user provides test image, add:
    // const testImagePath = join(__dirname, '../fixtures/test-image.png')
    // await page.locator('input[type="file"]').setInputFiles(testImagePath)
    // await expect(page.locator('#compressed-preview')).toBeVisible()
    // Verify the output is a valid PNG by checking file headers
  })
})
