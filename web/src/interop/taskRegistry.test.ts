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

const tick = async () => { for (let i = 0; i < 8; i++) await Promise.resolve() }

it('serializes large workspace estimates but admits two small jobs together', async () => {
  const tasks = new TaskRegistry()
  tasks.reserve('a', 'a'); tasks.reserve('b', 'b')
  await tasks.admit('a', 384_000_000) // 12 MP x eight RGBA surfaces
  let second = false
  const waiting = tasks.admit('b', 384_000_000).then(() => { second = true })
  await tick()
  expect(second).toBe(false)
  tasks.settle('a')
  await waiting
  tasks.settle('b')
  tasks.reserve('c', 'c'); tasks.reserve('d', 'd')
  await Promise.all([tasks.admit('c', 32_000_000), tasks.admit('d', 32_000_000)])
  expect(tasks.size).toBe(2)
})

it.each(['remove', 'reset'])('%s cancels waiting admission, holds admitted bytes until settlement, and releases once', async action => {
  const tasks = new TaskRegistry()
  tasks.reserve('a', 'a'); tasks.reserve('b', 'b'); tasks.reserve('c', 'c')
  await tasks.admit('a', 384_000_000)
  const cancelled = tasks.admit('b', 384_000_000).catch(error => error.name)
  let next = false
  const waiting = tasks.admit('c', 384_000_000).then(() => { next = true })
  if (action === 'remove') { tasks.remove('a'); tasks.remove('b') }
  else {
    tasks.abortAll()
    // A new generation can wait while old admitted resources unwind.
    tasks.settle('c'); tasks.reserve('c:1', 'c')
  }
  expect(await cancelled).toBe('AbortError')
  if (action === 'reset') await expect(waiting).rejects.toMatchObject({name: 'AbortError'})
  const fresh = action === 'reset' ? tasks.admit('c:1', 384_000_000).then(() => { next = true }) : waiting
  await tick()
  expect(next).toBe(false)
  tasks.settle('a')
  await fresh
  // Duplicate settlement must not release the still-active next job's budget.
  tasks.settle('a'); tasks.settle('b')
  tasks.reserve('d', 'd')
  let last = false
  const lastAdmission = tasks.admit('d', 384_000_000).then(() => { last = true })
  await tick()
  expect(last).toBe(false)
  tasks.settle(action === 'reset' ? 'c:1' : 'c')
  await lastAdmission
})

it('allows one explicitly bounded 24 MP job alone and rejects an over-limit estimate', async () => {
  const tasks = new TaskRegistry()
  tasks.reserve('large', 'large'); tasks.reserve('small', 'small'); tasks.reserve('unsafe', 'unsafe')
  await tasks.admit('large', 768_000_000)
  let small = false
  const next = tasks.admit('small', 32).then(() => { small = true })
  await tick()
  expect(small).toBe(false)
  await expect(tasks.admit('unsafe', 768_000_001)).rejects.toThrow(/budget|workspace/i)
  tasks.settle('large')
  await next
})
