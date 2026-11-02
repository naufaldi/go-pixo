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
