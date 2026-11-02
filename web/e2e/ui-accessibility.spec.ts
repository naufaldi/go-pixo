import { test, expect } from '@playwright/test'
import { fileURLToPath } from 'node:url'

const png = fileURLToPath(new URL('./fixtures/sample-32x32.png', import.meta.url))

test('known image geometry stays in place when processing becomes a result', async ({page, context}) => {
  await context.route('**/sw.js', route => route.abort())
  let release!: () => void
  let held = false
  const pending = new Promise<void>(resolve => { release = resolve })
  await context.route('**/main.wasm', async route => {
    held = true
    await pending
    await route.continue()
  })
  try {
    await page.goto('/')
    await page.locator('input[type=file]').setInputFiles(fileURLToPath(new URL('./fixtures/benchmark/codex-still-life.png', import.meta.url)))
    await expect.poll(() => held).toBe(true)
    const frame = page.getByRole('region', {name:'Image preview'}).locator('[style*="aspect-ratio"]')
    const processing = (await frame.boundingBox())!
    release()
    await expect(page.getByTestId('batch-summary')).toContainText('1 completed, 0 failed, 0 remaining')
    const completed = (await frame.boundingBox())!
    for (const axis of ['x','y','width','height'] as const) {
      expect(Math.abs(completed[axis] - processing[axis])).toBeLessThan(1)
    }
  } finally { release() }
})

test('desktop workspace remains usable at doubled CSS zoom', async ({page}) => {
  await page.setViewportSize({width:1280,height:800})
  await page.goto('/')
  await page.locator('input[type=file]').setInputFiles(png)
  await expect(page.getByTestId('download-primary')).toBeEnabled()
  const preview = (await page.getByRole('region',{name:'Image preview'}).boundingBox())!
  const settings = (await page.getByRole('complementary',{name:'Compression settings'}).boundingBox())!
  expect(settings.x).toBeGreaterThanOrEqual(preview.x + preview.width)
  await page.evaluate(() => { document.body.style.zoom = '2' })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  await page.getByLabel('Target width').scrollIntoViewIfNeeded()
  await expect(page.getByLabel('Target width')).toBeVisible()
})

test('download format disclosure supports Escape and outside pointer dismissal with focus intact', async ({ page }) => {
  await page.goto('/')
  await page.locator('input[type=file]').setInputFiles(png)
  const trigger = page.getByRole('button', { name: 'Choose download format' })
  await expect(trigger).toBeEnabled()
  await trigger.click()
  await page.keyboard.press('Tab')
  await expect(page.getByRole('button', { name: 'Download as PNG' })).toBeFocused()
  await page.getByRole('heading', { name: 'Go-Pixo', exact: true }).click()
  await expect(trigger).toHaveAttribute('aria-expanded', 'false')
  await expect(trigger).toBeFocused()
  await trigger.click()
  await page.keyboard.press('Tab')
  await page.keyboard.press('Escape')
  await expect(trigger).toHaveAttribute('aria-expanded', 'false')
  await expect(trigger).toBeFocused()
  await trigger.click()
  await page.getByLabel('Target width').click()
  await expect(trigger).toHaveAttribute('aria-expanded', 'false')
  await expect(page.getByLabel('Target width')).toBeFocused()
  await expect(page.locator('button button, [role="button"] button')).toHaveCount(0)
  await expect(page.locator('input[type=file]')).toHaveCount(1)
})

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
    await page.getByLabel('Add images').setInputFiles(fileURLToPath(new URL('./fixtures/slider-test.png', import.meta.url)))
    const select = page.getByRole('button', { name: /slider-test.png/ })
    await expect(select).toHaveAttribute('aria-current', 'false')
    await select.focus()
    await select.press(key)
    await expect(select).toHaveAttribute('aria-current', 'true')
    const files = page.getByRole('region', { name: 'Files', exact: true })
    const remove = select.locator('..').getByRole('button', { name: 'Remove image' })
    await remove.focus()
    await remove.press(key)
    await expect(select).toHaveCount(0)
    await files.getByRole('button', { name: 'Remove image' }).press(key)
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
    for (const input of [page.getByLabel('Output format'), page.getByLabel('Target width'), page.getByLabel('Target height')]) {
      await expect(input).toHaveCSS('font-size', '16px')
      await expect(input).toHaveCSS('padding-left', '16px')
      expect((await input.boundingBox())!.height).toBeGreaterThanOrEqual(48)
    }
    await expect(page.getByLabel('Output format')).toHaveCSS('appearance', 'none')
    await expect(page.getByLabel('Output format')).toHaveCSS('padding-right', '48px')
    const widthInput = (await page.getByLabel('Target width').boundingBox())!
    const heightInput = (await page.getByLabel('Target height').boundingBox())!
    expect(widthInput.width).toBeGreaterThanOrEqual(120)
    expect(heightInput.width).toBeGreaterThanOrEqual(120)
    expect(heightInput.x - (widthInput.x + widthInput.width)).toBeGreaterThanOrEqual(16)
    await expect(page.getByTestId('download-primary')).toHaveCount(0)
    await expect(page.locator('button button, [role="button"] button')).toHaveCount(0)
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
      expect(bounds.height).toBeGreaterThanOrEqual(48)
      expect(bounds.width).toBeGreaterThanOrEqual(48)
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
