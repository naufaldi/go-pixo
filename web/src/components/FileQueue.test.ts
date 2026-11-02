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
  fireEvent.keyDown(remove.closest('[role="button"]')!, { key })
  expect(onSelect).toHaveBeenCalledWith('one')
})

it('shows increased output bytes as a positive increase instead of negative savings', () => {
  render(createElement(FileQueue, {
    items: [{ id: 'one', generation: 0, file: { name: 'one.png' }, status: 'Done', kind: 'Png', originalBytes: 100, compressedBytes: 110 }],
    activeCompressions: [], onSelect() {}, onRemove() {}, onClearAll() {},
  }))
  expect(screen.getByTestId('file-item-done')).toHaveTextContent('110 bytes (+10%)')
})
