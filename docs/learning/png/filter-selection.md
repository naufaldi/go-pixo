# Filter Selection: Why and How

This guide explains why PNG encoders select different filters for each row and how the selection algorithm works.

---

## Why Select Filters Per Row?

PNG allows **different filters** for each scanline. Encoders should choose the filter that produces the **smallest compressed size** for each row.

### Why Not Use the Same Filter Everywhere?

Different parts of an image have different patterns:

```
Example image:
┌─────────────────┐
│ Horizontal      │  ← Sub filter works best
│ gradient        │
├─────────────────┤
│ Vertical        │  ← Up filter works best
│ stripes         │
├─────────────────┤
│ Diagonal        │  ← Paeth filter works best
│ pattern         │
└─────────────────┘
```

Using a single filter for the entire image would miss optimization opportunities in rows with different patterns.

---

## The Selection Problem

**Goal**: For each row, choose the filter (0-4) that produces the smallest DEFLATE-compressed size.

**Challenge**: Computing DEFLATE compression for all 5 filters is **expensive**:
- LZ77 encoding: O(n) per filter
- Huffman table building: O(n log n) per filter
- Total: 5 × O(n log n) per row

For a 1920×1080 image, that's **5,184,000 filter evaluations**!

---

## The Heuristic: Sum of Absolute Values

Instead of full compression, encoders use a **fast heuristic**: choose the filter with the minimum **sum of absolute values** of filtered bytes.

### Why This Works

1. **More zeros → better compression**: Filters that produce many zero bytes compress well
2. **Smaller values → better compression**: Small differences compress better than large ones
3. **Correlates with DEFLATE**: Sum of absolute values is a good proxy for compressed size
4. **Fast to compute**: O(n) per filter, no compression needed

### Example

```
Row: [100, 101, 102, 103, 104]

Filter None: [100, 101, 102, 103, 104]
             Sum of abs = 100+101+102+103+104 = 510

Filter Sub:  [100,   1,   1,   1,   1]
             Sum of abs = 100+1+1+1+1 = 104  ← Better!

Filter Sub wins (smaller sum)
```

---

## The Selection Algorithm

```go
func SelectFilter(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
    bestFilter := FilterNone
    bestFiltered := FilterNone(row)
    bestScore := SumAbsoluteValues(bestFiltered)

    // Try all filters
    filters := []struct {
        typ FilterType
        fn  func() []byte
    }{
        {FilterSub, func() []byte { return FilterSub(row, bpp) }},
        {FilterUp, func() []byte { return FilterUp(row, prevRow) }},
        {FilterAverage, func() []byte { return FilterAverage(row, prevRow, bpp) }},
        {FilterPaeth, func() []byte { return FilterPaeth(row, prevRow, bpp) }},
    }

    for _, f := range filters {
        filtered := f.fn()
        score := SumAbsoluteValues(filtered)
        if score < bestScore {
            bestScore = score
            bestFilter = f.typ
            bestFiltered = filtered
        }
    }

    return bestFilter, bestFiltered
}
```

### Steps

1. **Start with Filter None** as baseline
2. **Try each filter** (Sub, Up, Average, Paeth)
3. **Compute score** (sum of absolute values) for each
4. **Choose minimum** score
5. **Return** filter type and filtered bytes

---

## Sum of Absolute Values Implementation

```go
func SumAbsoluteValues(filtered []byte) int {
    sum := 0
    for _, b := range filtered {
        signed := int(int8(b))  // Interpret as signed byte (-128 to 127)
        if signed < 0 {
            sum -= signed  // abs(negative) = -negative
        } else {
            sum += signed  // abs(positive) = positive
        }
    }
    return sum
}
```

### Why Signed Byte Interpretation?

Filtered bytes can be **negative** (when prediction > actual value):
- Example: `x = 10`, `predictor = 20` → `filtered = 10 - 20 = -10`
- Stored as byte: `uint8(-10) = 246`
- To compute absolute value, interpret as signed: `int8(246) = -10`
- Then take absolute: `abs(-10) = 10`

---

## When the Heuristic Fails

The sum-of-absolute-values heuristic is **usually correct** but can fail in edge cases:

### Case 1: Many Small Non-Zero Values

```
Filter A: [1, 1, 1, 1, 1]  → Sum = 5
Filter B: [0, 0, 0, 0, 5]  → Sum = 5

Heuristic: Tie (both sum = 5)
Reality: Filter B compresses better (4 zeros vs 0 zeros)
```

**Mitigation**: Prefer filters with more zeros when scores are close.

### Case 2: Large Values vs Many Small Values

```
Filter A: [100, 0, 0, 0, 0]  → Sum = 100
Filter B: [20, 20, 20, 20, 20]  → Sum = 100

Heuristic: Tie (both sum = 100)
Reality: Filter A compresses better (single large value vs repeated values)
```

**Mitigation**: Consider value distribution, not just sum.

### Case 3: Filter Overhead

The filter byte itself adds 1 byte per row. For very small rows, filter selection overhead can outweigh benefits.

**Mitigation**: Skip filter selection for rows smaller than a threshold (e.g., < 8 bytes).

---

## Performance Considerations

### Optimization: Early Exit

If a filter produces all zeros, it's optimal (can't be beaten):

```go
func SelectFilter(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
    // Try Up filter first (often best for repeated rows)
    filtered := FilterUp(row, prevRow)
    score := SumAbsoluteValues(filtered)
    if score == 0 {
        return FilterUp, filtered  // Perfect match!
    }
    
    // Continue with other filters...
}
```

### Optimization: Skip Filters

For first row, skip Up/Average/Paeth (no previous row):

```go
if len(prevRow) == 0 {
    // Only try None and Sub
    filters := []FilterType{FilterNone, FilterSub}
    // ...
}
```

---

## Real-World Results

### Typical Filter Distribution

For a typical photograph:
- **Paeth**: ~40-50% of rows
- **Sub**: ~20-30% of rows
- **Average**: ~10-20% of rows
- **Up**: ~5-10% of rows
- **None**: ~5-10% of rows

### Compression Improvement

Filter selection typically improves compression by **5-15%** compared to always using Filter None.

---

## Tradeoffs

| Approach | Accuracy | Speed | Use Case |
|----------|----------|-------|----------|
| **Full compression** | Perfect | Slow | Offline encoding |
| **Sum of abs** | Good (~95%) | Fast | Real-time encoding |
| **Fixed filter** | Poor | Fastest | Minimal encoding |

---

## Summary

1. **Filter selection** chooses the best filter per row for optimal compression
2. **Heuristic**: Sum of absolute values (fast, correlates with compression)
3. **Algorithm**: Try all filters, pick minimum score
4. **Performance**: O(n) per row (much faster than full compression)
5. **Tradeoff**: ~95% accuracy vs perfect selection, but 100x faster

---

## Related Documentation

- [PNG Filters](filters.md) - What filters are and how they work
- [Paeth Predictor](paeth.md) - Detailed explanation of Paeth algorithm
- [PNG Scanlines](scanlines.md) - How filters are applied to scanlines
