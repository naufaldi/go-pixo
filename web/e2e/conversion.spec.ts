import { test, expect } from '@playwright/test'
import { dirname, join } from 'path'
import { fileURLToPath } from 'url'

const e2eDir = dirname(fileURLToPath(import.meta.url))
const samplePng = join(e2eDir, 'fixtures', 'sample-32x32.png')

async function openApp(page: import('@playwright/test').Page): Promise<void> {
  await page.goto('/')
  await expect(page.locator('h1')).toContainText('Go-Pixo')
}

async function uploadSamplePng(page: import('@playwright/test').Page): Promise<void> {
  await openApp(page)
  await page.locator('input[type="file"]').setInputFiles(samplePng)
}

async function waitForCompression(page: import('@playwright/test').Page): Promise<void> {
  await page.waitForSelector('[data-testid="file-item-done"], [data-testid="file-item-optimized"]', {
    timeout: 90000,
  })
}

test.describe('go-pixo', () => {
  test('app loads with correct title', async ({ page }) => {
    await openApp(page)
  })

  test('PNG file is accepted and compresses to Done status', async ({ page }) => {
    await uploadSamplePng(page)
    await expect(page.getByTestId('compression-percent')).toBeVisible({ timeout: 15000 })
    await waitForCompression(page)
  })

  test('compression slider updates preset label state', async ({ page }) => {
    await uploadSamplePng(page)
    await waitForCompression(page)

    const slider = page.locator('input[type="range"]')
    await expect(slider).toHaveValue('2')
    await slider.fill('0')
    await expect(slider).toHaveValue('0')
  })

  test('download format menu offers JPEG and triggers recompress', async ({ page }) => {
    await uploadSamplePng(page)
    await waitForCompression(page)

    await page.getByTestId('download-format-toggle').click()
    await expect(page.getByTestId('download-as-JPEG')).toBeVisible()
    await page.getByTestId('download-as-JPEG').click()
    await waitForCompression(page)
  })

  test('perfect quality checkbox toggles', async ({ page }) => {
    await uploadSamplePng(page)
    await waitForCompression(page)

    const checkbox = page.getByRole('checkbox', { name: 'Perfect Quality' })
    await expect(checkbox).not.toBeChecked()
    await checkbox.click()
    await expect(checkbox).toBeChecked()
    await waitForCompression(page)
  })

  test('resize inputs accept dimensions', async ({ page }) => {
    await uploadSamplePng(page)
    await waitForCompression(page)

    const widthInput = page.getByPlaceholder('W')
    await widthInput.fill('16')
    await expect(widthInput).toHaveValue('16')
  })
})
