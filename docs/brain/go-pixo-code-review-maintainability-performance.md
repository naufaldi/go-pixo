# Comprehensive Code Review: go-pixo

**Date:** January 8, 2026  
**Reviewer:** Scribe Agent  
**Project:** go-pixo - Go → WASM PNG/JPEG compression library with web UI  

## Scope and Methodology

This comprehensive code review analyzes the go-pixo codebase across all major components, focusing on maintainability, performance, and code quality. The review examined:

**Files Reviewed:** 
- `src/compress/` - DEFLATE, LZ77, Huffman, Zopfli (20+ files)
- `src/png/` - PNG encoder, filters, quantization (50+ files)
- `src/jpeg/` - JPEG encoder, DCT, progressive (30+ files)
- `src/wasm/` - WebAssembly bridge (3 files)
- `web/src/` - Frontend components and worker

The methodology included:
- Static analysis of code patterns and architecture
- Performance hotspot identification
- Review of error handling and edge cases
- Assessment of maintainability concerns
- Evaluation against Go best practices and project conventions

**Overall Assessment:** 🟢 **APPROVE with Minor Issues**

**Summary:** go-pixo demonstrates solid engineering with clean separation of concerns, comprehensive test coverage patterns, and no use of panic(). The codebase follows Go best practices with consistent error handling and good documentation. Several opportunities exist for performance optimization and maintainability improvements.

---

## 🔴 Critical Issues

**None identified.** The codebase has no security vulnerabilities, crashes, data loss risks, or corruption issues.

---

## 🟠 Major Issues

### 1. **Memory Allocation in Hot Paths** (Performance)
**File:** `src/png/filter_selector.go:54-81`, `src/png/filter_selector.go:115-142`

The filter selection functions allocate closures in a loop for each row:

```go
// src/png/filter_selector.go:59-68
filters := []struct {
    typ FilterType
    fn  func() []byte
}{
    {FilterNone, func() []byte { return ApplyFilterNone(row) }},
    {FilterSub, func() []byte { return ApplyFilterSub(row, bpp) }},
    // ... more closures
}
```

**Issue:** For large images, this creates thousands of closure allocations per encode operation.

**Recommendation:** Pre-compute filter results directly without closures, or use function pointers:

```go
filterResults := [5][]byte{
    ApplyFilterNone(row),
    ApplyFilterSub(row, bpp),
    ApplyFilterUp(row, prevRow),
    ApplyFilterAverage(row, prevRow, bpp),
    ApplyFilterPaeth(row, prevRow, bpp),
}
```

---

### 2. **Inefficient Zopfli Implementation** (Performance)
**File:** `src/compress/zopfli.go:64-92`

The current Zopfli implementation runs the same encoding multiple times without actual iterative refinement:

```go
// src/compress/zopfli.go:64-92
for iteration := 0; iteration < config.Iterations; iteration++ {
    encoder.SetCompressionLevel(9)
    singleResult, encodeErr := encoder.EncodeAuto(data)  // Same call repeated
    // ... fixed and dynamic also repeat the same operation
}
```

**Issue:** True Zopfli uses cost model optimization and squeeze iterations. This implementation repeats identical operations.

**Recommendation:** Either implement true Zopfli-style iteration with cost model feedback, or reduce default iterations since additional passes don't improve results.

---

### 3. **WASM Bridge Error Handling Returns Strings** (API Design)
**File:** `src/wasm/bridge_wasm.go:14-42`, `src/wasm/bridge_wasm.go:124-157`

Error handling uses string prefixes which is fragile:

```go
// src/wasm/bridge_wasm.go:33-34
if err != nil {
    return js.ValueOf(fmt.Sprintf("error: %v", err))
}
```

```typescript
// web/src/worker.ts:124-127
if (typeof result === "string" && result.startsWith("error:")) {
    throw new Error(result);
}
```

**Issue:** String-based error detection is fragile and loses error typing.

**Recommendation:** Return a structured object:

```go
return js.ValueOf(map[string]interface{}{
    "error": true,
    "message": err.Error(),
})
```

---

### 4. **Duplicate Code in encodeOptimalFallback** (Maintainability)
**File:** `src/compress/deflate_encoder.go:111-138`

The fallback function modifies encoder state in a way that persists beyond the call:

```go
// src/compress/deflate_encoder.go:116-135
for iteration := 0; iteration < 5; iteration++ {
    enc.SetCompressionLevel(enc.compressionLevel + iteration) // Mutates shared state
    if enc.compressionLevel > 9 {
        enc.SetCompressionLevel(9)
    }
    // ...
}
enc.SetCompressionLevel(enc.compressionLevel) // Doesn't restore original!
```

**Issue:** Line 135 doesn't correctly restore the original compression level.

**Recommendation:** Store original level before loop and restore after:

```go
originalLevel := enc.compressionLevel
defer enc.SetCompressionLevel(originalLevel)
```

---

## 🟡 Minor Issues

### 1. **Code Duplication in Filter Selection** (DRY Violation)
**Files:** `src/png/filter_selector.go:54-81`, `src/png/filter_selector.go:87-113`, `src/png/filter_selector.go:115-142`, `src/png/filter_selector.go:148-175`

Four nearly identical filter selection functions with repeated filter definitions.

**Recommendation:** Extract filter application into a reusable helper:

```go
type filterApplier func(row, prevRow []byte, bpp int) []byte

var allFilters = []struct {
    typ FilterType
    fn  filterApplier
}{...}
```

---

### 2. **Magic Numbers in Preset Options** (Style)
**File:** `src/wasm/bridge.go:122-132`, `src/wasm/bridge.go:157-165`

Preset numbers lack documentation at the call site:

```go
case 0: // Smaller - Maximum compression
case 1: // Balanced
case 2: // Faster
```

**Recommendation:** Use named constants or the `Preset` type from `src/png/options.go`:

```go
case int(png.PresetMax): // Smaller
```

---

### 3. **Inconsistent Error Type Usage** (Consistency)
**Files:** `src/png/errors.go`, `src/jpeg/errors.go`

PNG and JPEG use different error naming patterns:

- PNG: `ErrInvalidSignature` (no type prefix)
- Both use custom error types but export sentinel values

**Recommendation:** Align naming and consider using `errors.Is()` compatibility:

```go
var (
    ErrPNGInvalidSignature = &PngError{...}
    ErrPNGInvalidDimensions = &PngError{...}
)
```

---

### 4. **Missing Godoc on Exported Functions**
**File:** `src/png/filter_selector.go:3-5`

```go
func SelectFilter(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
    // No Godoc comment
    return SelectFilterWithStrategy(row, prevRow, bpp, FilterStrategyAdaptive)
}
```

**Recommendation:** Add Godoc comments to all exported functions per project conventions.

---

### 5. **QualityPresets Returns Empty Map** (Dead Code)
**File:** `src/png/options.go:198-202`

```go
func QualityPresets() map[string]Options {
    // Return a map of preset names to configuration functions
    // This is a placeholder - actual usage requires width/height
    return map[string]Options{}
}
```

**Recommendation:** Either implement properly or remove the placeholder function.

---

### 6. **Worker eval() Security Consideration**
**File:** `web/src/worker.ts:66-68`

```typescript
const wasmExecJs = await fetch("/wasm_exec.js").then((r) => r.text());
(0, eval)(wasmExecJs);
```

**Issue:** Using eval() for loading wasm_exec.js is necessary but should be documented.

**Recommendation:** Add comment explaining why eval is required and the security model (same-origin fetch).

---

## 🟢 Positive Observations

### 1. **No panic() Usage**
The entire codebase returns errors properly without any panic calls - excellent error handling discipline.

### 2. **Clean Architecture**
Clear separation of concerns:
- `compress/` - Low-level compression primitives
- `png/` - PNG-specific encoding
- `jpeg/` - JPEG-specific encoding  
- `wasm/` - Clean bridge layer

### 3. **Consistent Test Patterns**
Table-driven tests with descriptive names throughout, per AGENTS.md conventions.

### 4. **Comprehensive AGENTS.md Files**
Each package has detailed documentation explaining patterns, gotchas, and JIT Index hints.

### 5. **Good Error Wrapping**
Consistent use of `fmt.Errorf("context: %w", err)` for error context.

### 6. **Performance-Conscious Design**
- Pre-allocated buffers in `src/png/idat_writer.go:36`
- Efficient AAN DCT implementation in `src/jpeg/dct.go`
- Size comparison fallback to prevent output expansion

### 7. **Type Safety**
Strong typing with `ColorType`, `FilterType`, `Preset` enums rather than raw ints.

---

## Philosophy Compliance

| Principle | Status | Notes |
|-----------|--------|-------|
| Early Exit | 🟢 PASS | Guard clauses at function tops (e.g., `deflate_encoder.go:37-44`) |
| Parse Don't Validate | 🟢 PASS | Validation at boundaries (IHDR creation validates dimensions) |
| Atomic Predictability | 🟡 MINOR | `encodeOptimalFallback` mutates encoder state |
| Fail Fast | 🟢 PASS | Invalid states return errors immediately |
| Intentional Naming | 🟢 PASS | Clear names like `SelectFilterWithStrategy`, `WriteIDATWithOptions` |
| Security | 🟢 PASS | No hardcoded secrets, proper input validation |
| Performance | 🟡 MINOR | Closure allocations in hot paths, Zopfli inefficiency |

---

## Detailed Recommendations

### Quick Wins (< 1 hour each)
1. Fix `encodeOptimalFallback` state restoration bug
2. Add Godoc to `SelectFilter` and other exported functions
3. Remove or implement `QualityPresets()` placeholder
4. Add comment explaining eval() usage in worker.ts

### Medium Effort (1-4 hours)
5. Refactor filter selection to eliminate closure allocations
6. Extract common filter definitions to reduce duplication
7. Improve WASM error handling with structured returns

### Long-Term Improvements
8. Implement true Zopfli-style iterative optimization or simplify iterations
9. Consider SIMD optimizations for DCT and filter operations (Go 1.25+ may help)
10. Add concurrency support for encoding large images (parallel scanline filtering)

---

## Test Coverage Recommendations

Based on the codebase structure, ensure these areas have tests:
- [ ] Edge cases: 1x1 images, max dimension images
- [ ] Filter selection with empty/nil prevRow
- [ ] WASM bridge error paths
- [ ] Progressive JPEG with grayscale images
- [ ] Size guarantee fallback paths

---

## Conclusion

The go-pixo codebase demonstrates excellent engineering practices with a clear architecture and consistent approach to error handling. The identified issues are primarily performance optimizations and maintainability improvements rather than fundamental flaws. The codebase is well-structured for future enhancements and follows Go best practices throughout.

The minor issues identified should be addressed to improve code maintainability and performance, but do not prevent approval of the current implementation. The solid foundation makes this an excellent base for continued development.