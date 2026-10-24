import { expect, it } from 'vitest'
import { TaskRegistry } from './taskRegistry'

it.each(['settle-first', 'commit-first'])('keeps synchronous removal protection until both commit and settlement (%s)', order => {
  const tasks = new TaskRegistry()
  tasks.commit(['removed', 'peer'])
  const removed = tasks.reserve('removed:0', 'removed')
  const peer = tasks.reserve('peer:0', 'peer')
  tasks.remove('removed')
  expect(removed.aborted).toBe(true)
  expect(peer.aborted).toBe(false)
  expect(tasks.size).toBe(2)
  if (order === 'settle-first') tasks.settle('removed:0')
  else tasks.commit(['peer'])
  expect(tasks.isRemoved('removed')).toBe(true)
  if (order === 'settle-first') tasks.commit(['peer'])
  else tasks.settle('removed:0')
  expect(tasks.isRemoved('removed')).toBe(false)
  expect(tasks.size).toBe(1)
})

it('bounds removed IDs over repeated removal, including multiple outstanding generations', () => {
  const tasks = new TaskRegistry()
  for (let i = 0; i < 100; i++) {
    const id = `file-${i}`
    tasks.commit([id])
    tasks.reserve(`${id}:0`, id)
    tasks.reserve(`${id}:1`, id)
    tasks.remove(id)
    tasks.commit([])
    tasks.settle(`${id}:0`)
    expect(tasks.isRemoved(id)).toBe(true)
    tasks.settle(`${id}:1`)
    expect(tasks.isRemoved(id)).toBe(false)
    expect(tasks.size).toBe(0)
  }
})

it('abortAll retains each reservation until its operation settles', () => {
  const tasks = new TaskRegistry()
  const first = tasks.reserve('a:0', 'a'), second = tasks.reserve('b:0', 'b')
  tasks.abortAll()
  expect(first.aborted && second.aborted).toBe(true)
  expect(tasks.size).toBe(2)
  tasks.settle('a:0')
  expect(tasks.has('b:0')).toBe(true)
  expect(tasks.size).toBe(1)
})
