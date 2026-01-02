---
name: "Phase 4.1-4.3: PNG Preset System and Lossless Optimizations"
overview: Implement preset system with configurable speed/size tradeoffs, alpha optimization, and color type analysis for the go-pixo PNG encoder.
todos:
  - id: 4-1-1
    content: Create src/png/options.go with Options struct, Preset type, and preset constructors
    status: completed
  - id: 4-1-2
    content: Create src/png/options_builder.go with chainable builder methods
    status: completed
  - id: 4-1-3
    content: Create src/png/options_builder_test.go with builder tests
    status: completed
  - id: 4-2-1
    content: Create src/png/alpha.go with HasAlpha and OptimizeAlpha functions
    status: completed
  - id: 4-2-2
    content: Create src/png/alpha_test.go with alpha optimization tests
    status: completed
  - id: 4-3-1
    content: Create src/png/color_analysis.go with color type detection functions
    status: completed
  - id: 4-3-2
    content: Create src/png/color_analysis_test.go with color analysis tests
    status: completed
  - id: 4-3-3
    content: Create src/png/color_reduce.go with color reduction functions
    status: completed
  - id: 4-3-4
    content: Create src/png/color_reduce_test.go with color reduction tests
    status: completed
  - id: 4-4-1
    content: Update src/png/encoder.go to accept Options and apply optimizations
    status: completed
  - id: 4-4-2
    content: Update src/png/idat_writer.go to support configurable compression level
    status: completed
  - id: 4-4-3
    content: Update src/png/filter_selector.go to use FilterStrategy from options
    status: completed
  - id: 4-5-1
    content: Run tests and verify all acceptance criteria
    status: completed
---

## Overview

This plan implements Phase 4.1-4.3 of the go-pixo project, adding a preset system for PNG compression with configurable optimization options. The design is inspired by the [pixo reference implementation](Users/mac/WebApps/projects/pixo/src/png/mod.rs) which demonstrates a proven approach with Fast/Balanced/Max presets.

## User Stories

### User Story 1: Preset Selection

**As a user**, I want to choose between fast compression (small files quickly) and best compression (smallest files, slower), so that I can balance speed and file size for my use case.

### User Story 2: Fine-Grained Control  

**As an advanced user**, I want to customize individual compression settings, so that I can fine-tune for my specific requirements.

### User Story 3: Alpha Optimization

**As a user** with images containing transparent pixels, I want those pixels to compress better, so that my PNG files are smaller.

### User Story 4: Color Type Reduction

**As a user** with grayscale or opaque images, I want them encoded efficiently, so that file sizes are smaller.

## Solution Comparison

### Solution A: Simple Preset Enum with Overridable Options (RECOMMENDED)

```go
type Preset int

const (
    PresetFast Preset = iota
    PresetBalanced
    PresetMax
)

type Options struct {
    Width           int
    Height          int
    ColorType       ColorType
    CompressionLevel int    // 1-9
    FilterStrategy  FilterStrategy
    OptimizeAlpha   bool
    ReduceColorType bool
    StripMetadata   bool
    OptimalDeflate  bool
}

type OptionsBuilder struct {
    opts Options
}

// Preset constructors
func FastOptions(width, height int) Options
func BalancedOptions(width, height int) Options  
func MaxOptions(width, height int) Options
```

**Pros:**

- Matches pixo's proven approach (simple, flexible)
- Easy to understand and use
- Backwards compatible
- Builder allows full customization after preset selection

**Cons:**

- Users may not understand which fields to tweak

### Solution B: Hierarchical Configuration with Validation

```go
type CompressionProfile struct {
    Level         int
    MinMatchLen   int
    ChainLength   int
    SearchDepth   int
    NiceLength    int
    LazyMatching  bool
}

type FilterProfile struct {
    Strategy    FilterStrategy
    MinSumRows  int  // For MinSum strategy
}

type OptimizationProfile struct {
    Alpha       bool
    ColorReduce bool
    Metadata    bool
}

type Options struct {
    Compression CompressionProfile
    Filtering   FilterProfile
    Optimization OptimizationProfile
}
```

**Pros:**

- Enforces valid combinations
- Clearer separation of concerns
- Advanced users get more control

**Cons:**

- More complex to implement and understand
- Steeper learning curve
- Over-engineering for most use cases

**Recommendation: Solution A** - Simpler implementation, matches pixo's proven pattern, flexible enough for advanced use cases via builder.

## Preset Configuration

| Setting | Fast | Balanced | Max |

|---------|------|----------|-----|

| Compression Level | 2 | 6 | 9 |

| Filter Strategy | AdaptiveFast | Adaptive | MinSum |

| Optimize Alpha | false | true | true |

| Reduce Color Type | false | true | true |

| Strip Metadata | false | true | true |

| Optimal Deflate | false | false | true (5 iterations) |

## Compression Level Parameters (Inspired by pixo/lz77.rs)

| Level | Chain Length | Search Depth | Nice Length | Lazy |

|-------|--------------|--------------|-------------|------|

| 1 | 4 | 4 | 32 | false |

| 2 | 8 | 6 | 10 | false |

| 3 | 16 | 12 | 14 | false |

| 4 | 32 | 16 | 30 | false |

| 5 | 64 | 16 | 30 | true |

| 6 | 128 | 35 | 65 | true |

| 7 | 256 | 100 | 130 | true |

| 8 | 1024 | 300 | 258 | true |

| 9 | 4096 | 600 | 258 | true |

## Implementation Tasks

### Task 4.1.1: Create Options Structure

**File:** `src/png/options.go`

```go
package png

type Preset int

const (
    PresetFast Preset = iota
    PresetBalanced
    PresetMax
)

type FilterStrategy int

const (
    FilterStrategyNone FilterStrategy = iota
    FilterStrategySub
    FilterStrategyUp
    FilterStrategyAverage
    FilterStrategyPaeth
    FilterStrategyMinSum
    FilterStrategyAdaptive
    FilterStrategyAdaptiveFast
)

type Options struct {
    Width            int
    Height           int
    ColorType        ColorType
    CompressionLevel int    // 1-9
    FilterStrategy   FilterStrategy
    OptimizeAlpha    bool
    ReduceColorType  bool
    StripMetadata    bool
    OptimalDeflate   bool
}

// Preset constructors
func FastOptions(width, height int) Options
func BalancedOptions(width, height int) Options
func MaxOptions(width, height int) Options
```

**Tests:**

- Verify preset values match specification
- Validate compression level range (1-9)
- Test filter strategy enum values

### Task 4.1.2: Create Options Builder

**File:** `src/png/options_builder.go`

```go
type OptionsBuilder struct {
    opts Options
}

func NewOptionsBuilder(width, height int) *OptionsBuilder
func (b *OptionsBuilder) Fast() *OptionsBuilder
func (b *OptionsBuilder) Balanced() *OptionsBuilder
func (b *OptionsBuilder) Max() *OptionsBuilder
func (b *OptionsBuilder) CompressionLevel(level int) *OptionsBuilder
func (b *OptionsBuilder) FilterStrategy(strategy FilterStrategy) *OptionsBuilder
func (b *OptionsBuilder) OptimizeAlpha(enabled bool) *OptionsBuilder
func (b *OptionsBuilder) ReduceColorType(enabled bool) *OptionsBuilder
func (b *OptionsBuilder) StripMetadata(enabled bool) *OptionsBuilder
func (b *OptionsBuilder) OptimalDeflate(enabled bool) *OptionsBuilder
func (b *OptionsBuilder) Build() Options
```

**Tests:**

- Verify builder produces correct Options
- Test chaining order independence
- Test default values
- Validate compression level clamping

### Task 4.2.1: Alpha Optimization

**File:** `src/png/alpha.go`

```go
// HasAlpha checks if the pixel data contains any transparent pixels
func HasAlpha(pixels []byte, colorType ColorType) bool

// OptimizeAlpha sets RGB to 0 when alpha is 0 (undefined RGB becomes defined)
// This reduces entropy and improves compression
func OptimizeAlpha(pixels []byte, colorType ColorType) []byte
```

**Algorithm:**

- For RGBA: if alpha byte == 0, set R=G=B=0
- For color types without alpha, return unchanged
- O(n) single pass through pixel data

**Tests:**

- Test RGBA with some transparent pixels
- Test RGBA with all opaque pixels (no change)
- Test RGB color type (no change)
- Verify output decodes correctly

### Task 4.3.1: Color Analysis

**File:** `src/png/color_analysis.go`

```go
// IsGrayscale checks if RGB values are equal for all pixels
func IsGrayscale(pixels []byte, colorType ColorType) bool

// CanReduceToGrayscale returns true if all pixels have R=G=B
func CanReduceToGrayscale(pixels []byte, width, height int) bool

// CanReduceToRGB returns true if all alpha values are 255
func CanReduceToRGB(pixels []byte, width, height int) bool
```

**Tests:**

- Test RGB grayscale image
- Test RGB non-grayscale image
- Test RGBA all opaque
- Test RGBA mixed alpha
- Edge cases: 1x1 image, large images

### Task 4.3.2: Color Reduction

**File:** `src/png/color_reduce.go`

```go
// ReduceToGrayscale converts RGB/RGBA to Grayscale if all pixels are grayscale
func ReduceToGrayscale(pixels []byte, width, height int, colorType ColorType) ([]byte, ColorType, error)

// ReduceToRGB converts RGBA to RGB if all alpha values are 255
func ReduceToRGB(pixels []byte, width, height int) ([]byte, ColorType, error)
```

**Algorithm:**

- Validate all pixels qualify for reduction
- Create new pixel buffer with reduced color type
- Copy/convert pixel values
- Return new buffer, new color type

**Tests:**

- Test RGBA→RGB reduction
- Test RGB→Grayscale reduction
- Test RGBA→Grayscale reduction
- Verify output PNG is valid
- Verify decoded image matches original visually

## Integration Points

### Update encoder.go

Modify `Encoder.Encode` to accept `Options` parameter:

```go
func (e *Encoder) EncodeWithOptions(pixels []byte, opts Options) ([]byte, error)
```

Apply optimizations in order:

1. Color type reduction (if ReduceColorType enabled)
2. Alpha optimization (if OptimizeAlpha enabled)
3. Filter selection
4. DEFLATE compression (using CompressionLevel from options)

### Update idat_writer.go

Add compression level to `buildZlibData`:

```go
func (w *IDATWriter) SetCompressionLevel(level int)
```

Use level in `DeflateEncoder` initialization.

### Update filter_selector.go

Use `FilterStrategy` from options:

```go
func SelectFilter(row []byte, prevRow []byte, bpp int, strategy FilterStrategy) FilterType
```

## Acceptance Criteria

### Preset Selection

- Fast preset: <100ms for 1MB image, compression within 20% of Balanced
- Balanced preset: Default, 10-20% better than Fast
- Max preset: Best compression, within 5% of oxipng max
- All presets produce valid PNG output

### Fine-Grained Control

- Builder pattern works correctly
- Invalid combinations are validated
- Default values match Balanced preset
- All options are accessible and modifiable

### Alpha Optimization

- Pixels with alpha=0 have RGB=0
- No visual change for transparent pixels
- 5-15% size reduction for images with transparency
- Fully opaque images unchanged

### Color Type Reduction

- RGB→Grayscale when all pixels are grayscale
- RGBA→RGB when all alpha=255
- RGBA→Grayscale when all opaque and grayscale
- Reduction is lossless (100% of pixels qualify)
- Output PNG is valid and decodes correctly

## Files to Create

1. `src/png/options.go` - Options struct, Preset type, preset constructors
2. `src/png/options_builder.go` - Builder pattern implementation
3. `src/png/alpha.go` - Alpha optimization functions
4. `src/png/alpha_test.go` - Tests for alpha optimization
5. `src/png/color_analysis.go` - Color type detection
6. `src/png/color_analysis_test.go`
7. `src/png/color_reduce.go` - Color reduction implementation
8. `src/png/color_reduce_test.go`

## Files to Modify

1. `src/png/encoder.go` - Accept Options, apply optimizations
2. `src/png/idat_writer.go` - Support configurable compression level
3. `src/png/filter_selector.go` - Use FilterStrategy from options

## Verification Commands

```bash
# Run all tests
go test ./src/png/... -v

# Run specific tests
go test -run TestPreset ./src/png/...
go test -run TestAlpha ./src/png/...
go test -run TestColorReduce ./src/png/...

# Format code
go fmt ./src/png/...

# Lint
go vet ./src/png/...
```