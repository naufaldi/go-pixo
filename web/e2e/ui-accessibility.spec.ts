import { test, expect } from '@playwright/test'
import { fileURLToPath } from 'node:url'

const png = fileURLToPath(new URL('./fixtures/sample-32x32.png', import.meta.url))

test('comparison supports keyboard endpoints and queue Remove supports Enter and Space', async ({ page }) => {
  await page.goto('/')
  await page.locator('input[type=file]').setInputFiles(png)
  const compare = page.getByRole('slider', { name: 'Compare original and compressed' })
  await expect(compare).toBeVisible()
  await compare.focus()
  await compare.press('Home')
  await expect(compare).toHaveValue('0')
  await compare.press('ArrowRight')
  await expect(compare).toHaveValue('1')
  await compare.press('End')
  await expect(compare).toHaveValue('100')

  const add = page.getByLabel('Add images')
  await add.focus()
  await expect(add.locator('..')).toHaveCSS('outline-style', 'solid')

  for (const key of ['Enter', 'Space']) {
    const remove = page.locator('[role=button]').filter({ has: page.getByRole('button', { name: 'Remove image' }) })
      .getByRole('button', { name: 'Remove image' })
    await remove.focus()
    await remove.press(key)
    await expect(page.getByTestId('batch-summary')).toHaveCount(0)
    if (key === 'Enter') {
      await page.locator('input[type=file]').setInputFiles(png)
      await expect(compare).toBeVisible()
    }
  }
})

test.describe('mobile touch and reduced motion', () => {
  test.use({ viewport: { width: 390, height: 844 }, hasTouch: true, reducedMotion: 'reduce' })

  test('native comparison accepts touch, controls have 44px targets, and long filenames stay bounded', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', error => errors.push(error.message))
    await page.goto('/')
    await page.locator('input[type=file]').setInputFiles(png)
    const compare = page.getByRole('slider', { name: 'Compare original and compressed' })
    await expect(compare).toBeVisible()
    await compare.scrollIntoViewIfNeeded()
    const box = (await compare.boundingBox())!
    await page.touchscreen.tap(box.x + box.width * 0.75, box.y + box.height / 2)
    await expect.poll(async () => Number(await compare.inputValue())).toBeGreaterThan(50)
    for (const control of [compare, page.getByRole('slider', { name: 'Compression preset' }),
      page.getByTestId('download-primary'), page.getByRole('button', { name: 'Choose download format' }),
      page.getByRole('button', { name: 'Clear all' }), page.getByLabel('Add images').locator('..')]) {
      const bounds = (await control.boundingBox())!
      expect(bounds.height).toBeGreaterThanOrEqual(44)
      expect(bounds.width).toBeGreaterThanOrEqual(44)
    }
    await page.getByLabel('Add images').setInputFiles({
      name: 'unbroken'.repeat(60) + '.png', mimeType: 'image/png', buffer: Buffer.from('invalid image'),
    })
    await expect(page.getByTestId('batch-summary')).toContainText('1 failed')
    await page.getByText('unbroken'.repeat(60) + '.png', { exact: true }).click()
    await expect(page.getByRole('alert')).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    for (const element of await page.locator('[class*="animate-"]').all()) {
      await expect(element).toHaveCSS('animation-name', 'none')
    }
    expect(errors).toEqual([])
  })
})
