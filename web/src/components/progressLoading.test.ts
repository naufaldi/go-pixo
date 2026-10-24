import {afterEach, expect, it, vi} from 'vitest'
import {act, cleanup, render, screen} from '@testing-library/react'
import {createElement} from 'react'
import {make as CompareView} from './CompareView.res.js'
import {make as CompressionProgress} from './CompressionProgress.res.js'
import {make as FileQueue} from './FileQueue.res.js'

const progress = (extra = {}) => ({isActive: true, itemId: 'one', phase: 'quantize', progress: 12,
  startTime: performance.now(), fileSize: 5_000_000, lastRealProgress: 12,
  lastRealUpdateTime: performance.now(), phaseStartTime: performance.now(), phaseTarget: 20,
  predictable: true, phasePercent: 25, ...extra})
const compare = (p: any) => createElement(CompareView, {originalUrl: 'blob:original', originalBytes: 5_000_000,
  compressionProgress: p, onRemove() {}})
const queue = (p: any) => createElement(FileQueue, {items: [{id: 'one', file: new File([''], 'one.png'),
  status: 'Compressing', kind: 'Png', originalBytes: 5_000_000}], selectedId: 'one',
  activeCompressions: [['one', p]], onSelect() {}, onRemove() {}, onClearAll() {}})
afterEach(() => { cleanup(); vi.useRealTimers() })

it.each([['preview', compare], ['queue', queue], ['overlay', (p: any) => createElement(CompressionProgress, {progress: p})]] as const)(
  '%s advances elapsed time without worker messages and never invents progress', (_, view) => {
    vi.useFakeTimers({toFake: ['setInterval', 'clearInterval', 'performance']})
    render(view(progress()))
    expect(screen.getByText(/Time elapsed:/i)).toHaveTextContent('0')
    act(() => vi.advanceTimersByTime(2500))
    expect(screen.getByText(/Time elapsed:/i)).toHaveTextContent('2.5s')
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '25')
    expect(screen.getByText(/25% of stage/)).toBeInTheDocument()
    act(() => vi.advanceTimersByTime(15_000))
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '25')
    expect(screen.getByText(/has not reported progress/i)).toBeInTheDocument()
    cleanup()
    expect(vi.getTimerCount()).toBe(0)
  })

it('shares one clock, removes numeric progress for opaque stages and resets for a new operation', () => {
  vi.useFakeTimers({toFake: ['setInterval', 'clearInterval', 'performance']})
  const p = progress()
  const first = render(compare(p))
  const second = render(queue(p))
  expect(vi.getTimerCount()).toBe(1)
  act(() => vi.advanceTimersByTime(3000))
  first.rerender(compare(progress({phase: 'encoding-jpeg', phasePercent: undefined})))
  const bars = screen.getAllByRole('progressbar')
  expect(bars[0]).not.toHaveAttribute('aria-valuenow')
  expect(screen.getAllByText(/Time elapsed:/i)[0]).toHaveTextContent('0')
  second.unmount()
  expect(vi.getTimerCount()).toBe(1)
  first.rerender(createElement(CompareView, {originalUrl: 'blob:original', compressedUrl: 'blob:output', originalBytes: 100,
    compressedBytes: 50, compressionProgress: undefined, onRemove() {}}))
  expect(vi.getTimerCount()).toBe(0)
})
