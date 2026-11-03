import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { createElement } from 'react'
import { make as FileQueue } from './FileQueue.res.js'

afterEach(cleanup)

it.each([
  [1_000_000, 'Same file size with these settings'],
  [999_999, '<0.1% smaller'],
  [999_500, '<0.1% smaller'],
  [999_000, '-0.1%'],
  [1_000_001, '<0.1% larger'],
])('reports the actual size outcome in the file queue (%i bytes)', (compressedBytes, message) => {
  render(createElement(FileQueue, {
    items: [{ id: 'one', file: { name: 'one.png' }, status: 'Done', kind: 'Png', originalBytes: 1_000_000, compressedBytes }],
    activeCompressions: [], onSelect() {}, onRemove() {}, onClearAll() {},
  }))
  expect(screen.getByRole('button', { name: /one.png/ })).toHaveTextContent(message)
  if (compressedBytes < 1_000_000) expect(screen.queryByTestId('file-item-optimized')).not.toBeInTheDocument()
})

it.each([
  ['Pending', 'Pending'], ['Decoding', 'Decoding...'], ['Compressing', 'Compressing...'],
  ['Done', '50 bytes (-50%)'], [{ TAG: 'Error', _0: 'Cannot decode image' }, 'Cannot decode image'],
])('preserves meaningful status text alongside decorative glyphs (%j)', (status, text) => {
  render(createElement(FileQueue, {
    items: [{ id: 'one', file: { name: 'one.png' }, status, kind: 'Png', originalBytes: 100, compressedBytes: 50 }],
    activeCompressions: [], onSelect() {}, onRemove() {}, onClearAll() {},
  }))
  expect(screen.getByRole('button', { name: /one.png/ })).toHaveTextContent(text)
  const remove = screen.getByRole('button', { name: 'Remove image' })
  expect(remove.querySelector('svg')).toHaveAttribute('aria-hidden', 'true')
  const decoration = remove.previousElementSibling!
  expect(decoration).toHaveAttribute('aria-hidden', 'true')
  if (typeof status === 'object') {
    expect(decoration.querySelector('path')!.getAttribute('d')).not.toBe(remove.querySelector('path')!.getAttribute('d'))
  }
})

it.each(['Enter', ' '])('leaves Remove %j activation uncancelled without selecting its row', key => {
  const onSelect = vi.fn()
  const onRemove = vi.fn()
  render(createElement(FileQueue, {
    items: [{ id: 'one', generation: 0, file: { name: 'one.png' }, status: 'Pending', kind: 'Png', originalBytes: 100 }],
    activeCompressions: [], onSelect, onRemove, onClearAll() {},
  }))
  const remove = screen.getByRole('button', { name: 'Remove image' })
  expect(fireEvent.keyDown(remove, { key })).toBe(true)
  expect(onSelect).not.toHaveBeenCalled()
  // jsdom does not synthesize the native button click from a keyboard event.
  fireEvent.click(remove)
  expect(onRemove).toHaveBeenCalledWith('one')
  expect(onSelect).not.toHaveBeenCalled()
  const select = screen.getByRole('button', { name: /one.png/ })
  expect(select.tagName).toBe('BUTTON')
  expect(select).toHaveAttribute('type', 'button')
  expect(select.contains(remove)).toBe(false)
  expect(fireEvent.keyDown(select, { key })).toBe(true)
  fireEvent.click(select)
  expect(onSelect).toHaveBeenCalledWith('one')
})

it('exposes the selected file independently from its remove action', () => {
  const { container } = render(createElement(FileQueue, {
    items: [{ id: 'one', file: { name: 'one.png' }, status: 'Pending', kind: 'Png', originalBytes: 100 }],
    selectedId: 'one', activeCompressions: [], onSelect() {}, onRemove() {}, onClearAll() {},
  }))
  expect(screen.getByRole('button', { name: /one.png/ })).toHaveAttribute('aria-current', 'true')
  expect(container.querySelector('button button, [role="button"] button')).toBeNull()
  expect(screen.getByRole('heading', { name: 'Files (1)' })).toHaveProperty('tagName', 'H2')
})

it('shows increased output bytes as a positive increase instead of negative savings', () => {
  render(createElement(FileQueue, {
    items: [{ id: 'one', generation: 0, file: { name: 'one.png' }, status: 'Done', kind: 'Png', originalBytes: 100, compressedBytes: 110 }],
    activeCompressions: [], onSelect() {}, onRemove() {}, onClearAll() {},
  }))
  expect(screen.getByTestId('file-item-done')).toHaveTextContent('110 bytes (+10%)')
})
