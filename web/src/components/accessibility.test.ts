import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { createElement } from 'react'
import { make as CompareView } from './CompareView.res.js'
import { make as BottomBar } from './BottomBar.res.js'

afterEach(cleanup)

it('uses a named native comparison range and updates the original image clip at both endpoints', () => {
  const { container } = render(createElement(CompareView, {
    originalUrl: 'blob:original', compressedUrl: 'blob:compressed', originalBytes: 100,
    compressedBytes: 50, width: 200, height: 100, onRemove() {},
  }))
  const slider = screen.getByRole('slider', { name: 'Compare original and compressed' })
  expect(slider).toHaveAttribute('type', 'range')
  expect(slider).toHaveValue('50')
  const original = screen.getByRole('img', { name: 'Original' }).parentElement!
  for (const value of ['0', '25', '100']) {
    fireEvent.change(slider, { target: { value } })
    expect(slider).toHaveAttribute('aria-valuetext', `${value}% original, ${100 - Number(value)}% compressed`)
    expect(original.style.clipPath).toContain(`${value}% 0`)
  }
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
  expect(screen.getByText(/Lossless encoding/)).toBeVisible()
})

it('uses the same known aspect frame for loading and finished previews', () => {
  const props = { originalUrl: 'blob:original', originalBytes: 100, width: 200, height: 100, onRemove() {} }
  const view = render(createElement(CompareView, props))
  const loadingFrame = view.container.querySelector('[style*="aspect-ratio"]') as HTMLElement
  const loadingStyle = loadingFrame.style.cssText
  view.rerender(createElement(CompareView, { ...props, compressedUrl: 'blob:output', compressedBytes: 50 }))
  const finishedFrame = view.container.querySelector('[style*="aspect-ratio"]') as HTMLElement
  expect(finishedFrame.style.cssText).toBe(loadingStyle)
  expect(screen.queryByRole('button', { name: 'Remove image' })).not.toBeInTheDocument()
})
