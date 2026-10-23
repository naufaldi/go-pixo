import { afterEach, expect, it } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import { createElement } from 'react'
import { make as FileQueue } from './FileQueue.res.js'

afterEach(cleanup)

it('shows increased output bytes as a positive increase instead of negative savings', () => {
  render(createElement(FileQueue, {
    items: [{ id: 'one', generation: 0, file: { name: 'one.png' }, status: 'Done', kind: 'Png', originalBytes: 100, compressedBytes: 110 }],
    activeCompressions: [], onSelect() {}, onRemove() {}, onClearAll() {},
  }))
  expect(screen.getByTestId('file-item-done')).toHaveTextContent('110 bytes (+10%)')
})
