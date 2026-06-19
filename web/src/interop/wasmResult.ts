export function unwrapWasmResult(result: Uint8Array | string): Uint8Array {
  if (typeof result !== 'string') {
    return result
  }

  throw new Error(result.startsWith('error:') ? result.slice('error:'.length).trim() : result)
}
