import { describe, expect, it } from 'vitest'
import { unwrapWasmResult } from './wasmResult'

describe('unwrapWasmResult', () => {
  it('returns Uint8Array results unchanged', () => {
    const bytes = new Uint8Array([1, 2, 3])
    expect(unwrapWasmResult(bytes)).toBe(bytes)
  })

  it('throws for prefixed and bare Go bridge error strings', () => {
    expect(() => unwrapWasmResult('error: failed')).toThrow('failed')
    expect(() => unwrapWasmResult('invalid arguments')).toThrow('invalid arguments')
  })
})
