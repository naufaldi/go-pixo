import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { createElement } from 'react'
import { make as CompareView } from './CompareView.res.js'
import { make as BottomBar } from './BottomBar.res.js'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })

it('has one in-image comparison handle with keyboard steps, endpoints, and accessible percentages', () => {
  const { container } = render(createElement(CompareView, {
    originalUrl: 'blob:original', compressedUrl: 'blob:compressed', originalBytes: 100,
    compressedBytes: 50, width: 200, height: 100, onRemove() {},
  }))
  const slider = screen.getByRole('slider', { name: 'Compare original and compressed' })
  expect(screen.getAllByRole('slider')).toEqual([slider])
  expect(screen.getByTestId('comparison-frame')).toContainElement(slider)
  expect(container.querySelector('input[type="range"]')).toBeNull()
  expect(slider).toHaveAttribute('tabindex', '0')
  expect(slider).toHaveAttribute('aria-valuemin', '0')
  expect(slider).toHaveAttribute('aria-valuemax', '100')
  expect(slider).toHaveAttribute('aria-orientation', 'horizontal')
  expect(slider).toHaveAttribute('aria-valuenow', '50')
  const original = screen.getByRole('img', { name: 'Original' }).parentElement!
  for (const [key, value] of [
    ['Home', '0'], ['ArrowLeft', '0'], ['ArrowDown', '0'], ['ArrowRight', '1'],
    ['ArrowUp', '2'], ['End', '100'], ['ArrowRight', '100'], ['ArrowUp', '100'],
    ['ArrowLeft', '99'], ['ArrowDown', '98'],
  ]) {
    expect(fireEvent.keyDown(slider, { key })).toBe(false)
    expect(slider).toHaveAttribute('aria-valuenow', value)
    expect(slider).toHaveAttribute('aria-valuetext', `${value}% original, ${100 - Number(value)}% compressed`)
    expect(original.style.clipPath).toContain(`${value}% 0`)
  }
  expect(fireEvent.keyDown(slider, { key: 'Tab' })).toBe(true)
  expect(slider).toHaveAttribute('aria-valuenow', '98')
  expect(container.querySelector('[style*="aspect-ratio"]')).toHaveStyle({ aspectRatio: '200 / 100' })
})

it('names every preset value and sends the selected preset to the state owner', () => {
  const onPresetChange = vi.fn()
  const props = {
    format: 'PNG', outputFormat: 'SameAsInput', preset: 'Ultra', lossless: false,
    canDownload: false, hasCompletedItems: false, completedCount: 0, failedCount: 0,
    processingAll: false, onPresetChange,
    onOutputFormatChange() {}, onLosslessChange() {}, onDownload() {}, onDownloadFormat() {},
    onDownloadAll() {}, onDownloadZip() {}, onCompressAll() {},
    onTargetWidthChange() {}, onTargetHeightChange() {},
  }
  const view = render(createElement(BottomBar, props))
  const presets = ['Ultra', 'Smaller', 'Balanced', 'Faster']
  const names = ['Ultra (smallest)', 'Smaller', 'Balanced', 'Faster (quickest)']
  presets.forEach((preset, index) => {
    view.rerender(createElement(BottomBar, { ...props, preset }))
    const slider = screen.getByRole('slider', { name: 'Compression preset' })
    expect(slider).toHaveAttribute('aria-valuetext', names[index])
    fireEvent.change(slider, { target: { value: String((index + 1) % 4) } })
    expect(onPresetChange).toHaveBeenLastCalledWith(presets[(index + 1) % 4])
  })
})

const settingsProps = {
  format: 'PNG', outputFormat: 'SameAsInput', preset: 'Balanced', lossless: false,
  canDownload: true, hasCompletedItems: true, completedCount: 1, failedCount: 0,
  processingAll: false, onPresetChange() {}, onOutputFormatChange() {}, onLosslessChange() {},
  onDownload() {}, onDownloadFormat() {}, onDownloadAll() {}, onDownloadZip() {},
  onCompressAll() {}, onTargetWidthChange() {}, onTargetHeightChange() {},
}

it('dismisses the download disclosure with Escape and returns focus to its trigger', () => {
  render(createElement(BottomBar, settingsProps))
  const trigger = screen.getByRole('button', { name: 'Choose download format' })
  fireEvent.click(trigger)
  const option = screen.getByRole('button', { name: 'Download as JPEG' })
  option.focus()
  fireEvent.keyDown(option, { key: 'Escape' })
  expect(trigger).toHaveAttribute('aria-expanded', 'false')
  expect(trigger).toHaveFocus()
  expect(screen.queryByRole('button', { name: 'Download as JPEG' })).not.toBeInTheDocument()
})

it('dismisses outside pointers without stealing focus from another control', () => {
  const { unmount } = render(createElement(BottomBar, settingsProps))
  const trigger = screen.getByRole('button', { name: 'Choose download format' })
  const outside = screen.getByRole('combobox', { name: 'Output format' })
  fireEvent.click(trigger)
  screen.getByRole('button', { name: 'Download as PNG' }).focus()
  expect(fireEvent.pointerDown(document.body)).toBe(false)
  expect(trigger).toHaveAttribute('aria-expanded', 'false')
  expect(trigger).toHaveFocus()
  fireEvent.click(trigger)
  outside.focus()
  fireEvent.pointerDown(outside)
  expect(trigger).toHaveAttribute('aria-expanded', 'false')
  expect(outside).toHaveFocus()
  fireEvent.click(trigger)
  unmount()
  outside.focus()
  fireEvent.keyDown(document, { key: 'Escape' })
  expect(trigger).not.toHaveFocus()
})

it('keeps the disclosure open for inside pointers and restores focus after choosing a format', () => {
  const onDownloadFormat = vi.fn()
  render(createElement(BottomBar, { ...settingsProps, onDownloadFormat }))
  const trigger = screen.getByRole('button', { name: 'Choose download format' })
  fireEvent.click(trigger)
  const option = screen.getByRole('button', { name: 'Download as WebP' })
  fireEvent.pointerDown(option)
  expect(trigger).toHaveAttribute('aria-expanded', 'true')
  option.focus()
  fireEvent.click(option)
  expect(onDownloadFormat).toHaveBeenCalledExactlyOnceWith('ForceWebp')
  expect(trigger).toHaveFocus()
  expect(trigger).toHaveAttribute('aria-expanded', 'false')
})

it('does not reopen an old disclosure after processing temporarily disables downloads', () => {
  const view = render(createElement(BottomBar, settingsProps))
  const trigger = screen.getByRole('button', { name: 'Choose download format' })
  fireEvent.click(trigger)
  view.rerender(createElement(BottomBar, { ...settingsProps, canDownload: false }))
  expect(trigger).toHaveAttribute('aria-expanded', 'false')
  view.rerender(createElement(BottomBar, settingsProps))
  expect(screen.queryByRole('button', { name: 'Download as PNG' })).not.toBeInTheDocument()
  expect(trigger).toHaveAttribute('aria-expanded', 'false')
})

it('keeps settings usable without files and hides item-only actions', () => {
  const onOutputFormatChange = vi.fn()
  const onTargetWidthChange = vi.fn()
  render(createElement(BottomBar, { ...settingsProps, hasItems: false, onOutputFormatChange, onTargetWidthChange }))
  expect(screen.queryByRole('button')).not.toBeInTheDocument()
  fireEvent.change(screen.getByRole('combobox', { name: 'Output format' }), { target: { value: 'jpeg' } })
  expect(onOutputFormatChange).toHaveBeenCalledWith('ForceJpeg')
  fireEvent.change(screen.getByRole('spinbutton', { name: 'Target width' }), { target: { value: '200' } })
  expect(onTargetWidthChange).toHaveBeenCalledWith(200)
  expect(screen.getByText(/Each image keeps its own aspect ratio/)).toBeVisible()
  expect(screen.getByRole('checkbox', { name: /^Lossless$/ })).toHaveAccessibleDescription(/preserves every pixel unless resized.*File size may stay the same/)
})

it.each([
  [1_000_000, 'Same file size with these settings'],
  [999_999, 'Saved 1 byte (<0.1%)'],
  [999_500, 'Saved 500 bytes (<0.1%)'],
  [999_000, 'Saved 1 KB (0.1%)'],
])('reports the actual size outcome in the comparison (%i bytes)', (compressedBytes, message) => {
  render(createElement(CompareView, {
    originalUrl: 'blob:original', compressedUrl: 'blob:compressed', originalBytes: 1_000_000,
    compressedBytes, width: 200, height: 100, onRemove() {},
  }))
  expect(screen.getByText(message)).toBeVisible()
  if (compressedBytes < 1_000_000) expect(screen.queryByText('Same file size with these settings')).not.toBeInTheDocument()
})

it('uses the same known aspect frame for loading and finished previews', () => {
  const props = { originalUrl: 'blob:original', originalBytes: 100, width: 200, height: 100, onRemove() {} }
  const view = render(createElement(CompareView, props))
  const loadingFrame = view.container.querySelector('[style*="aspect-ratio"]') as HTMLElement
  const loadingStyle = loadingFrame.style.cssText
  expect(view.container.firstElementChild?.children).toHaveLength(2)
  expect(screen.queryByRole('slider')).not.toBeInTheDocument()
  view.rerender(createElement(CompareView, { ...props, compressedUrl: 'blob:output', compressedBytes: 50 }))
  const finishedFrame = view.container.querySelector('[style*="aspect-ratio"]') as HTMLElement
  expect(finishedFrame.style.cssText).toBe(loadingStyle)
  expect(view.container.firstElementChild?.children).toHaveLength(2)
  expect(screen.queryByRole('button', { name: 'Remove image' })).not.toBeInTheDocument()
})

const comparisonProps = {
  originalUrl: 'blob:original', compressedUrl: 'blob:compressed', originalBytes: 100,
  compressedBytes: 50, width: 100, height: 200, onRemove() {},
}

function renderPointerComparison() {
  // jsdom cannot perform browser pointer capture or native touch arbitration.
  class TestPointerEvent extends MouseEvent {
    pointerId: number
    pointerType: string
    isPrimary: boolean
    constructor(type: string, init: PointerEventInit = {}) {
      super(type, init)
      this.pointerId = init.pointerId ?? 1
      this.pointerType = init.pointerType ?? 'mouse'
      this.isPrimary = init.isPrimary ?? true
    }
  }
  vi.stubGlobal('PointerEvent', TestPointerEvent)
  const view = render(createElement(CompareView, comparisonProps))
  const frame = screen.getByTestId('comparison-frame')
  vi.spyOn(frame, 'getBoundingClientRect').mockReturnValue({ left: 100, width: 200 } as DOMRect)
  const captures = new Set<number>()
  frame.setPointerCapture = vi.fn(id => { captures.add(id) })
  frame.hasPointerCapture = vi.fn(id => captures.has(id))
  frame.releasePointerCapture = vi.fn(id => { captures.delete(id) })
  const slider = screen.getByRole('slider', { name: 'Compare original and compressed' })
  const clip = screen.getByRole('img', { name: 'Original' }).parentElement!
  const assertSplit = (value: number) => {
    expect(slider).toHaveAttribute('aria-valuenow', String(value))
    expect(slider).toHaveAttribute('aria-valuetext', `${value}% original, ${100 - value}% compressed`)
    expect(clip.style.clipPath).toContain(`${value}% 0`)
    expect(frame.querySelector('[style*="left:"]')).toHaveStyle({ left: `${value}%` })
  }
  return { ...view, frame, slider, assertSplit }
}

it('keeps one pointer owner from down through outside release and clamps to image bounds', () => {
  const { frame, assertSplit } = renderPointerComparison()
  fireEvent.pointerDown(frame, { pointerId: 1, clientX: 150, button: 2 })
  fireEvent.pointerDown(frame, { pointerId: 2, clientX: 150, button: 0, isPrimary: false })
  expect(frame.setPointerCapture).not.toHaveBeenCalled()
  assertSplit(50)
  fireEvent.pointerDown(frame, { pointerId: 1, clientX: 150, button: 0 })
  assertSplit(25)
  expect(screen.getByRole('slider')).toHaveFocus()
  fireEvent.pointerDown(frame, { pointerId: 2, clientX: 290, button: 0 })
  fireEvent.pointerMove(frame, { pointerId: 2, clientX: 290 })
  fireEvent.pointerUp(frame, { pointerId: 2, clientX: 290 })
  expect(frame.setPointerCapture).toHaveBeenCalledExactlyOnceWith(1)
  assertSplit(25)
  fireEvent.pointerMove(frame, { pointerId: 1, clientX: 400 })
  assertSplit(100)
  fireEvent.pointerUp(frame, { pointerId: 1, clientX: 0 })
  assertSplit(0)
  expect(frame.releasePointerCapture).toHaveBeenCalledExactlyOnceWith(1)
  fireEvent.pointerMove(frame, { pointerId: 1, clientX: 200 })
  assertSplit(0)
})

it('restores the starting touch split on cancel and permits a later touch tap', () => {
  const { frame, slider, assertSplit } = renderPointerComparison()
  for (let i = 0; i < 10; i++) fireEvent.keyDown(slider, { key: 'ArrowLeft' })
  fireEvent.pointerDown(frame, { pointerId: 1, pointerType: 'touch', clientX: 250 })
  assertSplit(40)
  fireEvent.pointerMove(frame, { pointerId: 1, pointerType: 'touch', clientX: 250 })
  assertSplit(75)
  fireEvent.pointerDown(frame, { pointerId: 2, pointerType: 'touch', clientX: 120 })
  fireEvent.pointerCancel(frame, { pointerId: 2, pointerType: 'touch' })
  assertSplit(75)
  fireEvent.pointerCancel(frame, { pointerId: 1, pointerType: 'touch' })
  assertSplit(40)
  expect(frame.releasePointerCapture).toHaveBeenCalledWith(1)
  fireEvent.pointerMove(frame, { pointerId: 1, clientX: 200 })
  assertSplit(40)
  fireEvent.pointerDown(frame, { pointerId: 3, pointerType: 'touch', clientX: 250 })
  fireEvent.pointerUp(frame, { pointerId: 3, pointerType: 'touch', clientX: 250 })
  assertSplit(75)
})

it('ends ownership on capture loss, retains fractional clip precision, and resets on a new image key', () => {
  const { frame, slider, assertSplit, rerender, unmount } = renderPointerComparison()
  fireEvent.pointerDown(frame, { pointerId: 1, pointerType: 'pen', clientX: 151 })
  assertSplit(25.5)
  fireEvent.lostPointerCapture(frame, { pointerId: 2 })
  fireEvent.pointerMove(frame, { pointerId: 1, clientX: 160 })
  assertSplit(30)
  fireEvent.lostPointerCapture(frame, { pointerId: 1 })
  fireEvent.pointerMove(frame, { pointerId: 1, clientX: 290 })
  assertSplit(30)
  fireEvent.keyDown(slider, { key: 'End' })
  assertSplit(100)
  fireEvent.pointerDown(frame, { pointerId: 3, clientX: 150 })
  rerender(createElement(CompareView, { ...comparisonProps, key: 'another-image' }))
  expect(screen.getByRole('slider', { name: 'Compare original and compressed' })).toHaveAttribute('aria-valuenow', '50')
  fireEvent.pointerMove(screen.getByTestId('comparison-frame'), { pointerId: 3, clientX: 290 })
  expect(screen.getByRole('slider', { name: 'Compare original and compressed' })).toHaveAttribute('aria-valuenow', '50')
  expect(() => unmount()).not.toThrow()
})

it('ignores zero-width geometry and disables native image dragging', () => {
  const { frame, assertSplit } = renderPointerComparison()
  vi.mocked(frame.getBoundingClientRect).mockReturnValue({ left: 100, width: 0 } as DOMRect)
  fireEvent.pointerDown(frame, { pointerId: 1, clientX: 150 })
  assertSplit(50)
  for (const image of screen.getAllByRole('img')) expect(image).toHaveAttribute('draggable', 'false')
})

it.each(['width', 'height'])('keeps %s numeric bounds and forwards typing and clearing', axis => {
  const onChange = vi.fn()
  const view = render(createElement(BottomBar, { ...settingsProps,
    onTargetWidthChange: onChange, onTargetHeightChange: onChange,
  }))
  const input = screen.getByRole('spinbutton', { name: `Target ${axis}` })
  expect(input).toHaveAttribute('type', 'number')
  expect(input).toHaveAttribute('min', '1')
  expect(input).toHaveAttribute('max', '8000')
  fireEvent.change(input, { target: { value: '256' } })
  expect(onChange).toHaveBeenLastCalledWith(256)
  view.rerender(createElement(BottomBar, { ...settingsProps,
    targetWidth: 256, targetHeight: 256,
    onTargetWidthChange: onChange, onTargetHeightChange: onChange,
  }))
  fireEvent.change(input, { target: { value: '' } })
  expect(onChange).toHaveBeenLastCalledWith(undefined)
})
