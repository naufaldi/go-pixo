import { test, expect, type Locator } from '@playwright/test'
import { fileURLToPath } from 'node:url'

const png = fileURLToPath(new URL('./fixtures/sample-32x32.png', import.meta.url))

async function expectInImageControl(frame: Locator, slider: Locator) {
  await expect(frame.getByRole('slider')).toHaveCount(1)
  await expect(frame.locator('..').locator('..').getByRole('slider')).toHaveCount(1)
  await expect(frame.locator('..').locator('..').locator('input[type=range]')).toHaveCount(0)
  const bounds = (await frame.boundingBox())!
  const viewport = (await frame.locator('..').boundingBox())!
  const handle = (await slider.boundingBox())!
  expect(handle.width).toBe(48)
  expect(handle.height).toBe(48)
  expect(handle.x).toBeGreaterThanOrEqual(viewport.x + 0.5)
  expect(handle.x + handle.width).toBeLessThanOrEqual(viewport.x + viewport.width - 0.5)
  expect(handle.y).toBeGreaterThanOrEqual(viewport.y + 0.5)
  expect(handle.y + handle.height).toBeLessThanOrEqual(viewport.y + viewport.height - 0.5)
  if (bounds.width < 48) {
    expect(Math.abs(handle.x + handle.width / 2 - (bounds.x + bounds.width / 2))).toBeLessThan(1)
  }
  const circle = slider.locator(':scope > div')
  await expect(circle).toHaveCSS('width', '32px')
  await expect(circle).toHaveCSS('height', '32px')
  const preview = (await frame.locator('..').locator('..').boundingBox())!
  expect(Math.abs(preview.y + preview.height - (viewport.y + viewport.height))).toBeLessThan(2)
}

async function expectRemoveCentered(remove: Locator) {
  // Measure the drawn path after SVG transforms, including CSS zoom and active scale.
  const geometry = await remove.evaluate(button => {
    const svg = button.querySelector('svg')!
    const path = svg.querySelector('path')!
    const ink = path.getBBox()
    const center = new DOMPoint(ink.x + ink.width / 2, ink.y + ink.height / 2)
      .matrixTransform(path.getScreenCTM()!)
    const bounds = button.getBoundingClientRect()
    return { dx: center.x - (bounds.x + bounds.width / 2),
      dy: center.y - (bounds.y + bounds.height / 2),
      glyph: getComputedStyle(svg).width, border: getComputedStyle(button).borderTopWidth }
  })
  expect(Math.abs(geometry.dx)).toBeLessThan(0.5)
  expect(Math.abs(geometry.dy)).toBeLessThan(0.5)
  expect(geometry.glyph).toBe('20px')
  expect(geometry.border).toBe('0px')
}

for (const [width, height] of [[400, 200], [200, 400], [390, 8000], [8000, 20]]) {
  test(`direct image drag uses ${width}x${height} image bounds and resets on selection`, async ({ page }) => {
    await page.goto('/')
    const bytes = await page.evaluate(([width, height]) => {
      const canvas = document.createElement('canvas')
      canvas.width = width
      canvas.height = height
      canvas.getContext('2d')!.fillRect(0, 0, width, height)
      return Array.from(atob(canvas.toDataURL('image/png').split(',')[1]), c => c.charCodeAt(0))
    }, [width, height])
    await page.locator('input[type=file]').setInputFiles({ name: 'geometry.png', mimeType: 'image/png', buffer: Buffer.from(bytes) })
    const frame = page.getByTestId('comparison-frame')
    const slider = page.getByRole('slider', { name: 'Compare original and compressed' })
    await expect(frame).toBeVisible()
    await expectInImageControl(frame, slider)
    const box = (await frame.boundingBox())!
    await page.mouse.move(box.x + box.width / 4, box.y + box.height / 2)
    await page.mouse.down()
    await expect(slider).toHaveAttribute('aria-valuenow', '25')
    await page.mouse.move(box.x + box.width + 30, box.y + box.height / 2)
    await expect(slider).toHaveAttribute('aria-valuenow', '100')
    await expectInImageControl(frame, slider)
    await page.mouse.move(box.x - 20, box.y + box.height / 2)
    await page.mouse.up()
    await expect(slider).toHaveAttribute('aria-valuenow', '0')
    await expectInImageControl(frame, slider)
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
    await expect(slider).toHaveAttribute('aria-valuenow', '0')
    await expect(page.getByRole('img', { name: 'Original', exact: true })).toHaveAttribute('draggable', 'false')
    await page.getByLabel('Add images').setInputFiles(png)
    await page.getByRole('button', { name: /sample-32x32.png/ }).click()
    await expect(slider).toHaveAttribute('aria-valuenow', '50')
    await page.getByRole('button', { name: /geometry.png/ }).click()
    await expect(slider).toHaveAttribute('aria-valuenow', '50')
  })
}

test('resize fields retain numeric keyboard stepping without native spinner chrome', async ({ page }) => {
  await page.goto('/')
  for (const axis of ['width', 'height']) {
    const input = page.getByRole('spinbutton', { name: `Target ${axis}` })
    await expect(input).toHaveCSS('appearance', 'textfield')
    await input.fill('200')
    const box = (await input.boundingBox())!
    await input.click({ position: { x: box.width - 6, y: 10 } })
    await expect(input).toHaveValue('200')
    await input.click({ position: { x: box.width - 6, y: box.height - 10 } })
    await expect(input).toHaveValue('200')
    await input.press('ArrowUp')
    await expect(input).toHaveValue('201')
    await input.fill('')
    await expect(input).toHaveValue('')
  }
})

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
    const preview = page.getByRole('region', { name: 'Image preview' })
    const loadingPreview = (await preview.boundingBox())!
    expect(Math.abs(loadingPreview.y + loadingPreview.height - (processing.y + processing.height))).toBeLessThan(2)
    await expect(preview.getByRole('slider')).toHaveCount(0)
    release()
    await expect(page.getByTestId('batch-summary')).toContainText('1 completed, 0 failed, 0 remaining')
    const completed = (await frame.boundingBox())!
    await expectInImageControl(frame, preview.getByRole('slider'))
    const completedPreview = (await preview.boundingBox())!
    expect(Math.abs(completedPreview.height - loadingPreview.height)).toBeLessThan(1)
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
  await expectRemoveCentered(page.getByRole('button', { name: 'Remove image', exact: true }))
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
  await expect(compare).toBeFocused()
  await expect(compare).toHaveCSS('outline-style', 'solid')
  await expect.poll(() => compare.evaluate(handle => {
    const expected = document.createElement('span')
    expected.style.color = 'var(--color-neutral-300)'
    handle.append(expected)
    const matches = getComputedStyle(handle).outlineColor === getComputedStyle(expected).color
    expected.remove()
    return matches
  })).toBe(true)
  await compare.press('Home')
  await expect(compare).toHaveAttribute('aria-valuenow', '0')
  await compare.press('ArrowRight')
  await expect(compare).toHaveAttribute('aria-valuenow', '1')
  await compare.press('End')
  await expect(compare).toHaveAttribute('aria-valuenow', '100')
  await expect(compare).toHaveAttribute('aria-valuetext', '100% original, 0% compressed')
  await compare.press('ArrowDown')
  await expect(compare).toHaveAttribute('aria-valuenow', '99')
  await compare.press('ArrowUp')
  await expect(compare).toHaveAttribute('aria-valuenow', '100')
  await page.keyboard.press('Tab')
  await expect(compare).not.toBeFocused()

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
    await page.getByRole('heading', { name: 'Go-Pixo', exact: true }).hover()
    await expectRemoveCentered(remove)
    await remove.hover()
    await expectRemoveCentered(remove)
    await page.mouse.down()
    try {
      await expect.poll(() => remove.evaluate(button => parseFloat(getComputedStyle(button).scale))).toBe(0.96)
      await expectRemoveCentered(remove)
    } finally {
      // Release away from the button so this state check does not remove the file.
      await page.mouse.move(0, 0)
      await page.mouse.up()
    }
    await page.keyboard.press('Tab')
    await remove.focus()
    await expect(remove).toHaveCSS('outline-style', 'solid')
    await expectRemoveCentered(remove)
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

  test('trusted Chromium emulated touch drags horizontally and yields vertical scrolling', async ({ page, context, browserName }) => {
    test.skip(browserName !== 'chromium', 'CDP touch input is Chromium-specific')
    await page.goto('/')
    await page.locator('input[type=file]').setInputFiles(png)
    const frame = page.getByTestId('comparison-frame')
    const slider = page.getByRole('slider', { name: 'Compare original and compressed' })
    await expect(frame).toBeVisible()
    await frame.scrollIntoViewIfNeeded()
    const session = await context.newCDPSession(page)
    let touching = false
    const swipe = async (x: number, y: number, dx: number, dy: number) => {
      await session.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y, id: 1 }] })
      touching = true
      for (let step = 1; step <= 8; step++) {
        await session.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: x + dx * step / 8, y: y + dy * step / 8, id: 1 }] })
        await page.waitForTimeout(16)
      }
      await session.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] })
      touching = false
    }
    try {
      let box = (await frame.boundingBox())!
      await swipe(box.x + box.width * 0.5, box.y + box.height / 2, box.width * 0.3, 0)
      await expect.poll(async () => Number(await slider.getAttribute('aria-valuenow'))).toBeGreaterThan(70)
      const split = (await slider.getAttribute('aria-valuenow'))!
      box = (await frame.boundingBox())!
      const before = await page.evaluate(() => scrollY)
      await swipe(box.x + box.width / 2, box.y + box.height * 0.7, 0, -120)
      await expect.poll(() => page.evaluate(() => scrollY)).toBeGreaterThan(before)
      await expect(slider).toHaveAttribute('aria-valuenow', split)
    } finally {
      if (touching) await session.send('Input.dispatchTouchEvent', { type: 'touchCancel', touchPoints: [] })
      await session.detach()
    }
  })

  test('in-image comparison accepts touch, controls have 48px targets, and long filenames stay bounded', async ({ page }) => {
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
    const box = (await page.getByTestId('comparison-frame').boundingBox())!
    await page.touchscreen.tap(box.x + box.width * 0.75, box.y + box.height / 2)
    await expect.poll(async () => Number(await compare.getAttribute('aria-valuenow'))).toBeGreaterThan(50)
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
