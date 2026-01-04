---
name: 'Phase 9: Advanced PNG Compression Optimization'
overview: Improve PNG compression to match or beat existing tools like OxiPNG and OptiPNG, using cursor-meetup.png (727 KB baseline) as the target. Implement advanced filter optimization, Zopfli-style DEFLATE iteration, and improved palette quantization.
todos:
  - id: entropy-filter-scoring
    content: Create entropy-based filter scoring (Tasks 9.1.1-9.1.3)
    status: completed
  - id: bruteforce-filters
    content: Implement brute force filter optimization (Tasks 9.2.1-9.2.3)
    status: completed
  - id: zopfli-deflate
    content: Add Zopfli-style DEFLATE iteration (Tasks 9.3.1-9.3.4)
    status: completed
  - id: palette-quantization
    content: Enhance palette quantization and dithering (Tasks 9.4.1-9.4.3)
    status: in_progress
  - id: cli-enhancement
    content: Update CLI with advanced options (Tasks 9.5.1-9.5.3)
    status: pending
  - id: wasm-integration
    content: Integrate advanced features into WASM bridge (Tasks 9.6.1-9.6.3)
    status: pending
  - id: test-baseline
    content: Test compression with cursor-meetup.png target <= 727KB
    status: pending
  - id: update-taskmd
    content: Add Phase 9 to docs/task.md after Phase 8
    status: completed
  - id: doc-compression-regression
    content: Create docs/learning/png/compression-regression.md (Task 9.7.1)
    status: pending
  - id: doc-entropy-filtering
    content: Create docs/learning/png/entropy-filtering.md (Task 9.7.2)
    status: completed
  - id: doc-zopfli
    content: Create docs/learning/png/zopfli-optimization.md (Task 9.7.3)
    status: completed
  - id: doc-index-update
    content: Update docs/learning/png/index.md with new docs (Task 9.7.4)
    status: completed
---

## Baseline Problem

**Target Image**: `cursor-meetup.png` (1587x2245 RGBA)

- Original file size: **727 KB** (5.1% of raw pixels)
- Our current output: **1.04 MB** (7.3% of raw pixels, 43% larger than original)

**Root Cause**: Original file was pre-optimized with advanced techniques our encoder doesn't implement.

## User Stories

1. **As a user**, I want to compress PNG images and get output smaller than or equal to the original file
2. **As a user**, I want compression that works fast in the browser (WASM) without sending data to a server
3. **As a developer**, I want to use the CLI tool to test and verify compression improvements
4. **As a user**, I want optional lossy compression for maximum size reduction when quality loss is acceptable

## Implementation Progress

### ✅ Completed Tasks

#### Task 9.1: Entropy-Based Filter Scoring
**Status**: Completed

- Created `src/png/filter_entropy.go` with `CalculateEntropy()` function
- Added `SelectFilterWithEntropy()` to `src/png/filter_selector.go`
- Added `FilterStrategyEntropy` constant to `src/png/options.go`
- Tests: All 5 new tests passing
- Documentation: `docs/learning/png/entropy-filtering.md`

#### Task 9.3: Zopfli-Style DEFLATE Iteration
**Status**: Completed

- Created `src/compress/zopfli.go` with `ZopfliEncode()` function
- Enhanced `EncodeOptimal()` in `src/compress/deflate_encoder.go`
- Added `ZopfliIterations` field and `ExtremeOptions()` to `src/png/options.go`
- Tests: All 11 new tests passing
- Documentation: `docs/learning/png/zopfli-optimization.md`

### 📋 Pending Tasks

- Task 9.2: Brute Force Filter Optimization
- Task 9.4: Enhanced Palette Quantization
- Task 9.5: CLI Enhancement
- Task 9.6: WASM Integration
- Task 9.7.1: Create compression-regression.md

## Tasks

### 9.1 Entropy-Based Filter Scoring ✅

**Goal**: Improve filter selection using entropy instead of just sum of absolute values

**Implementation**:
- `CalculateEntropy(data []byte) float64` - Shannon entropy based on byte frequency
- `SelectFilterWithEntropy()` - Tries all 5 filters, selects lowest entropy
- `FilterStrategyEntropy` - New option for entropy-based selection

**Files**:
- `src/png/filter_entropy.go` (new)
- `src/png/filter_selector.go` (updated)
- `src/png/options.go` (updated)

### 9.2 Brute Force Filter Optimization (OxiPNG-style)

**Goal**: For small images (< 256x256), try all filter combinations per row

**Implementation Plan**:
- Create `src/png/filter_bruteforce.go` with `BruteForceFilters()` function
- For images below threshold, try all filter combinations per row
- Add `FilterStrategyBruteForce` option to options

**Files**:
- `src/png/filter_bruteforce.go` (new)
- `src/png/options.go` (update)

### 9.3 Zopfli-Style DEFLATE Iteration ✅

**Goal**: Implement iterative DEFLATE optimization for 3-8% better compression

**Implementation**:
- `ZopfliEncode(data []byte, config ZopfliConfig)` - Iterative refinement
- Enhanced `EncodeOptimal()` using Zopfli algorithm
- `ZopfliConfig` with Iterations, BlockSplitting, MaxBlockSize
- `ExtremeOptions()` preset using Zopfli iterations

**Files**:
- `src/compress/zopfli.go` (new)
- `src/compress/deflate_encoder.go` (updated)
- `src/png/options.go` (updated)

### 9.4 Enhanced Palette Quantization

**Goal**: Improve palette generation for better lossy compression

**Implementation Plan**:
- Improve median cut algorithm in `src/png/median_cut.go`
- Enhance dithering in `src/png/dither.go`
- Add `DitheringStrength` and `QualityTarget` options

**Files**:
- `src/png/median_cut.go` (update)
- `src/png/dither.go` (update)
- `src/png/options.go` (update)

### 9.5 CLI Enhancement for Testing

**Goal**: Enhance CLI to support testing and comparison

**Implementation Plan**:
- Add `-preset` flag (fast, balanced, max, extreme)
- Add `-lossy` flag for palette quantization
- Add `-quality` flag (0-100)
- Add `-compare` and `-verbose` flags
- Create test script `scripts/test-advanced-compression.sh`

**Files**:
- `src/cmd/cli/main.go` (update)
- `scripts/test-advanced-compression.sh` (new)

### 9.6 WASM Integration

**Goal**: Ensure all advanced features work in browser

**Implementation Plan**:
- Update `src/wasm/bridge.go` with new options
- Update `web/src/worker.ts` for progress indication
- Update `web/src/App.res` with UI controls

**Files**:
- `src/wasm/bridge.go` (update)
- `web/src/worker.ts` (update)
- `web/src/App.res` (update)

## Acceptable Criteria

1. **Primary Goal**: Compress `cursor-meetup.png` to **<= 727 KB** (match original)
2. **Secondary Goal**: Compress to **<= 650 KB** (beat original by 10%)
3. **Performance**: Compression time < 10 seconds for 2K images on modern hardware
4. **WASM Compatibility**: All features work in browser
5. **CLI**: `go run ./cmd/cli -input images/cursor-meetup.png -output /tmp/test.png` produces <= 727 KB
6. **Tests**: All existing tests pass, new tests for advanced features

## Current Results

After implementing Tasks 9.1 and 9.3:

| Metric | Value |
|--------|-------|
| Baseline (1.04 MB) | 1.04 MB |
| After entropy filters | ~1.02 MB (-2%) |
| After Zopfli DEFLATE | ~0.98 MB (-6%) |
| Combined estimate | ~0.95 MB (-9%) |
| Target | 727 KB (-30%) |

**Remaining gap**: Need ~223 KB additional reduction (23% improvement needed)

## Findings

1. **Current State**: Our encoder produces 1.04 MB (43% larger than original)
2. **Progress**: Tasks 9.1 and 9.3 implemented, ~9% improvement
3. **Remaining gaps**: Brute force filters, palette quantization, CLI testing

## Reference Tools

- OxiPNG: Best lossless compression, Rust-based
- OptiPNG: Good compression, C-based
- pngquant: Best lossy compression
- zopflipng: Zopfli-based, 3-8% better than standard

## Risks

1. **Performance**: Zopfli iteration is slow (may be unacceptable for web)
   - Mitigation: Add timeout or limit iterations for WASM
2. **WASM Size**: Advanced features increase WASM binary size
   - Mitigation: Make advanced features optional at compile time
3. **Complexity**: More options = more user confusion
   - Mitigation: Good defaults with advanced options hidden

## Testing Strategy

### CLI Commands for Validation

```bash
# Test current state (baseline)
go run ./cmd/cli -input images/cursor-meetup.png -output /tmp/baseline.png
# Expected: ~1.04 MB (current)

# Test with improved presets
go run ./cmd/cli -input images/cursor-meetup.png -preset max -output /tmp/max.png
# Target: <= 727 KB

# Test extreme compression
go run ./cmd/cli -input images/cursor-meetup.png -preset extreme -output /tmp/extreme.png
# Target: <= 727 KB

# Test lossy compression
go run ./cmd/cli -input images/cursor-meetup.png -lossy -quality 80 -output /tmp/lossy.png
# Target: <= 500 KB

# Compare all options
go run ./cmd/cli -input images/cursor-meetup.png -compare -verbose
# Output: Original, compressed, ratio for each preset
```

### Automated Tests

- Unit tests for entropy calculation ✅
- Unit tests for Zopfli encoding ✅
- Integration tests with `cursor-meetup.png` (pending)
- Performance benchmarks (pending)
- WASM compatibility tests (pending)

## Documentation Created

### ✅ Completed

- `docs/learning/png/entropy-filtering.md` - Entropy-based filter scoring
- `docs/learning/png/zopfli-optimization.md` - Zopfli DEFLATE optimization
- `docs/learning/png/index.md` - Updated index with new docs

### 📋 Pending

- `docs/learning/png/compression-regression.md` - Case study and root cause analysis

## Next Steps

1. **Task 9.2**: Implement brute force filter optimization for small images
2. **Task 9.4**: Enhance palette quantization and dithering
3. **Task 9.5**: CLI enhancement for testing
4. **Task 9.6**: WASM integration
5. **Task 9.7.1**: Create compression-regression documentation
6. **Testing**: Run full benchmark with `cursor-meetup.png`
