---
name: Improved PNG Quantization
overview: Enhance pixo's default palette quantization to excel at both photographic and graphic content by adding perceptual color handling and K-means refinement. No new modes or API changes - just better defaults.
todos:
  - id: perceptual-distance
    content: Add perceptual_distance_sq() function and update nearest_palette_index() to use it
    status: completed
  - id: perceptual-splitting
    content: Update ColorBox::range() to use perceptually-weighted channel importance
    status: completed
  - id: kmeans-refinement
    content: Add refine_palette_kmeans() to iterate palette colors after median-cut (2 iterations)
    status: completed
  - id: update-palette-lut
    content: Rebuild PaletteLut using perceptual distance
    status: completed
  - id: test-perceptual-distance
    content: Add unit tests for perceptual_distance_sq() function
    status: completed
  - id: test-kmeans-refinement
    content: Add unit tests for refine_palette_kmeans() function
    status: completed
  - id: test-quantization-quality
    content: Add integration test verifying improved palette quality on photographic images
    status: completed
  - id: run-tests
    content: Run cargo test to ensure all tests pass
    status: completed
  - id: run-benchmarks
    content: Run benchmarks on avatar-color.png and rocket.png to verify improvements
    status: pending
---

# Improved PNG Quantization for Photographic and Graphic Content

## Problem Statement

pixo's current median-cut quantization excels at graphic content but underperforms on photographs:

- **Graphic content (rocket.png)**: pixo wins by 27% over pngquant
- **Photographic content (avatar-color.png)**: pngquant wins by 9%

The key differences between pixo and pngquant (libimagequant):

| Aspect | pixo (current) | pngquant/libimagequant |

|--------|----------------|------------------------|

| Color distance | Euclidean RGB | Perceptual (CIELAB-like) |

| Box splitting | Largest range | Variance-weighted |

| Palette refinement | None | K-means iterations |

| Dithering | Basic Floyd-Steinberg | Edge-aware error diffusion |

## Proposed Solution: Improved Default Quantization

Since pixo is already fast, we'll simply improve the default quantization algorithm rather than adding modes. The overhead is minimal:

- Perceptual distance: ~3 extra multiplications per comparison
- K-means refinement: 2 iterations on the histogram (not pixels)

## Implementation Plan

### 1. Replace Euclidean with Perceptual Color Distance

Update `nearest_palette_index()` in [`src/png/mod.rs`](src/png/mod.rs):

```rust
/// Perceptual color distance using weighted RGB (approximates human perception)
/// Based on "Color metric" by Thiadmer Riemersma (compuphase.com/cmetric.htm)
fn perceptual_distance_sq(c1: [u8; 4], c2: [u8; 4]) -> u32 {
    let r_mean = (c1[0] as i32 + c2[0] as i32) / 2;
    let dr = c1[0] as i32 - c2[0] as i32;
    let dg = c1[1] as i32 - c2[1] as i32;
    let db = c1[2] as i32 - c2[2] as i32;
    let da = c1[3] as i32 - c2[3] as i32;

    // Weight factors based on r_mean (red-sensitive vs blue-sensitive)
    let r_weight = 2 + (r_mean >> 8);
    let g_weight = 4;
    let b_weight = 2 + ((255 - r_mean) >> 8);

    (r_weight * dr * dr + g_weight * dg * dg + b_weight * db * db + da * da) as u32
}
```

This fast approximation is much better than Euclidean RGB for skin tones and gradients while adding negligible overhead.

### 2. Improve Box Splitting with Perceptual Weighting

Update `ColorBox::range()` to weight green channel higher (human eyes most sensitive to green):

```rust
fn range(&self) -> (u8, u16) {
    let r_range = self.r_max - self.r_min;
    let g_range = self.g_max - self.g_min;
    let b_range = self.b_max - self.b_min;
    let a_range = self.a_max - self.a_min;

    // Perceptual weights: G > R > B (matches human perception)
    let r_score = r_range as u16 * 2;
    let g_score = g_range as u16 * 4;  // Green most important
    let b_score = b_range as u16 * 1;
    let a_score = a_range as u16 * 3;  // Alpha important for transparency

    // Return channel with highest perceptual score
    ...
}
```

### 3. Add K-means Refinement

After median-cut produces initial palette, run 2 iterations of K-means on the color histogram:

```rust
fn refine_palette_kmeans(palette: &mut [[u8; 4]], colors: &[ColorCount]) {
    for _ in 0..2 {
        // 1. Assign each histogram color to nearest palette entry (using perceptual distance)
        // 2. Recalculate palette entries as weighted mean of assigned colors
    }
}
```

This operates on the histogram (at most ~8K colors after sampling), not pixels, so it's fast.

### 4. Update PaletteLut

Rebuild the LUT using perceptual distance for lookups.

## File Changes

**[`src/png/mod.rs`](src/png/mod.rs)** only:

- Add `perceptual_distance_sq()` function
- Update `nearest_palette_index()` to use perceptual distance
- Update `ColorBox::range()` to use perceptual weighting
- Add `refine_palette_kmeans()` after `median_cut_palette()`
- Rebuild `PaletteLut` with perceptual distance

**No API changes, no new CLI flags, no WASM changes needed.**

## Performance Impact

Estimated overhead: **10-20%** on quantization step (which is already fast).

| Component | Current | After |

|-----------|---------|-------|

| Distance calc | 4 ops | ~10 ops |

| Box splitting | Range-based | Weighted range |

| Palette refinement | None | 2 K-means iterations on histogram |

The DEFLATE step remains the bottleneck, so overall encode time impact should be negligible.

## Test Coverage

Add unit tests for the new functions in the `mod tests` section of [`src/png/mod.rs`](src/png/mod.rs):

### Unit Tests

```rust
#[test]
fn test_perceptual_distance_identical() {
    // Identical colors should have distance 0
    let c = [128, 64, 192, 255];
    assert_eq!(perceptual_distance_sq(c, c), 0);
}

#[test]
fn test_perceptual_distance_green_weighted() {
    // Green differences should contribute more than red/blue
    let base = [128, 128, 128, 255];
    let red_diff = [138, 128, 128, 255];   // +10 red
    let green_diff = [128, 138, 128, 255]; // +10 green
    let blue_diff = [128, 128, 138, 255];  // +10 blue

    let d_red = perceptual_distance_sq(base, red_diff);
    let d_green = perceptual_distance_sq(base, green_diff);
    let d_blue = perceptual_distance_sq(base, blue_diff);

    assert!(d_green > d_red, "green should be weighted higher than red");
    assert!(d_red > d_blue, "red should be weighted higher than blue");
}

#[test]
fn test_kmeans_refinement_converges() {
    // Palette should move toward cluster centers
    ...
}
```

### Integration Tests

```rust
#[test]
fn test_quantization_photographic_quality() {
    // Quantize avatar-color.png and verify palette represents skin tones well
    ...
}
```

## Validation

### Run Tests

```bash
cargo test
```

### Run Benchmarks

```bash
cargo bench --bench comparison -- "PNG Lossy"
```

Compare results against baseline:

| Image | Before | After | Target |

|-------|--------|-------|--------|

| avatar-color.png | +9% vs pngquant | ? | ~+2-3% |

| rocket.png | -27% vs pngquant | ? | maintain -27% |
