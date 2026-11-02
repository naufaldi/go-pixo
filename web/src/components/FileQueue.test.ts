import { afterEach, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { createElement } from 'react'
import { make as FileQueue } from './FileQueue.res.js'

afterEach(cleanup)

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
