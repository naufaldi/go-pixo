import { test, expect } from '@playwright/test'
import { dirname, join } from 'path'
import { fileURLToPath } from 'url'

const e2eDir = dirname(fileURLToPath(import.meta.url))
const indomaretScreenshot = join(
  e2eDir,
  'fixtures',
  'browser-verify',
  'indomaret-screenshot.png',
)

test.describe('pixel mismatch regression', () => {
  test('720x900 Indomaret screenshot encodes successfully', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('h1')).toContainText('Go-Pixo')

    await page.locator('input[type="file"]').setInputFiles(indomaretScreenshot)

    await expect(page.getByText(/pixel count mismatch/i)).not.toBeVisible()
    await expect(page.getByText(/Pixel buffer size mismatch/i)).not.toBeVisible()

    await page.waitForSelector(
      '[data-testid="file-item-done"], [data-testid="file-item-optimized"]',
      { timeout: 120000 },
    )

    const bodyText = await page.locator('body').innerText()
    expect(bodyText).not.toMatch(/pixel count mismatch/i)
    expect(bodyText).not.toMatch(/Pixel buffer size mismatch/i)
  })
})
