# PNG Performance Bottlenecks Explained

Simple explanations of why your PNG encoder is slow, using everyday analogies.

---

## Table of Contents

1. [Palette Lookup O(n)](#1-palette-lookup-on)
2. [No K-means Refinement](#2-no-k-means-refinement)
3. [No Bigrams Filter](#3-no-bigrams-filter)
4. [Perceptual Distance](#4-perceptual-distance)
5. [Per-Row Allocations](#5-per-row-allocations)
6. [Sequential Filtering](#6-sequential-filtering)
7. [No Early Termination](#7-no-early-termination)
8. [Greedy LZ77 Matching](#8-greedy-lz77-matching)

---

## 1. Palette Lookup O(n)

### What It Means

Finding the nearest color in a palette by checking every color one by one.

### ELI5: Finding a Book in a Library

Imagine you have a library with 256 books (your palette colors), and you need to find the book most similar to the one you're holding.

**The slow way (O(n)):**
```
You: "Is book 1 similar? No. Is book 2 similar? No. Is book 3 similar? 
      No... (repeat 256 times) ... Is book 256 similar? YES!"
      
Time: ~256 checks
```

**The fast way (O(1)):**
```
You: Look at the index card for "Blue" → says books 47, 142, and 203 are blue
      Go directly to book 47 → Perfect match!

Time: ~1 check
```

### Real Impact

| Image Size | Palette Size | Distance Calculations |
|------------|--------------|----------------------|
| 100×100 = 10,000 pixels | 256 colors | 2.56 million |
| 1000×1000 = 1 million pixels | 256 colors | **256 million** |
| 4K = 8 million pixels | 256 colors | **2 billion** |

That's 2 billion operations just to find colors!

### The Fix

Create a pre-made "cheat sheet" (lookup table) that tells you instantly which palette color to use for any RGB value.

---

## 2. No K-means Refinement

### What It Means

The palette is created by mechanically splitting color space, without considering what colors actually appear in your image.

### ELI5: Sorting Crayons by Color

Imagine you have a huge box of crayons with thousands of colors, and you need to pick just 8 crayons that best represent all the colors in your picture.

**The median-cut way:**
```
You: "Let's look at all the crayons and find where the colors spread out the most.
      Split there. Now we have 2 groups. Split each group again...
      (keep splitting until we have 8 groups)
      
      Take the average color of each group as our 8 crayons."
      
Problem: The "average" might not be a real crayon! And some important
         colors in your picture might not be well-represented.
```

**The K-means way:**
```
You: "First, pick any 8 crayons as a guess.
      
      Step 1: For each color in your picture, find which of the 8 crayons
              is most similar. Group colors by their closest crayon.
      
      Step 2: For each group of colors, find the "center" color.
              Replace your crayon with this new center color.
      
      Step 3: Repeat Steps 1-2 a few times until the crayons stop changing.
      
Result: Your 8 crayons now perfectly represent the colors in your picture!
```

### Why It Matters

| Image Type | Median-Cut Result | With K-means Result |
|------------|-------------------|---------------------|
| Sunset gradient | Banding artifacts | Smooth transitions |
| Portrait | Flat skin tones | Natural skin tones |
| Sky | Color stepping | Smooth blue |

**K-means makes the palette "fit" your image like a glove.**

---

## 3. No Bigrams Filter

### What It Means

PNG uses "filters" to prepare image data for compression. The current filter selection doesn't consider how well the filtered data will compress.

### ELI5: Packing for a Trip

Imagine you're packing for a trip and you want to fit everything into the smallest suitcase possible.

**The current way (MinSum filter):**
```
You: "I'll pack things that are similar next to each other.
      This makes the suitcase look organized."
      
Problem: Just because it looks organized doesn't mean it compresses well!
         The packing method (filter) doesn't consider the compression algorithm.
```

**The Bigrams way:**
```
You: "I know the suitcase uses a special compression technique
      that works best when the same things appear multiple times.
      
      Let me try different packing methods and count how many UNIQUE
      pairs of items I create.
      
      The method with the FEWER unique pairs will compress best!"
```

### How It Works

A "bigram" is just a pair of adjacent items. DEFLATE compression loves repeated patterns:

```
Before filtering: [A][B][A][B][A][B][A][B]
                   └─┘ └─┘ └─┘ └─┘
                   Many unique bigrams: AB, BA, AB, BA... → Hard to compress

After filtering:  [A][0][0][0][0][0][0][0]
                   └─┘
                   One unique bigram repeated: AA → EASY to compress!
```

### The Fix

Add a filter strategy that counts distinct bigrams and chooses the filter that minimizes them.

---

## 4. Perceptual Distance

### What It Means

The way we measure "how different are these two colors" uses simple math instead of how human eyes actually perceive color.

### ELI5: Matching Paint Colors

Imagine you're trying to match a paint color for a wall, and you have several paint samples.

**The Euclidean way:**
```
You: "I'll measure the exact RGB values and pick the closest one.
      RGB distance: sqrt((R1-R2)² + (G1-G2)² + (B1-B2)²)"
      
Problem: This treats all colors equally, but human eyes are more
         sensitive to some colors than others!
```

**The Perceptual way (Redmean):**
```
You: "I know human eyes are most sensitive to GREEN.
      And sensitivity to RED and BLUE changes based on brightness.
      
      Let me weight the differences accordingly:
      - Green differences matter most
      - Red differences matter more in bright scenes
      - Blue differences matter more in dark scenes"
```

### Visual Example

```
Two colors that look almost identical:
- Color A: Pure red (255, 0, 0)
- Color B: Slightly orange-red (250, 10, 0)

Euclidean distance: sqrt(5² + 10² + 0²) = ~11.2

But these colors look very different because:
- Humans are more sensitive to red/orange tones
- The 10-unit difference in green matters more than the 5-unit difference in red

Perceptual distance correctly weights this as "more different"
```

### The Fix

Replace the simple Euclidean distance formula with a perceptually-weighted formula (Redmean).

---

## 5. Per-Row Allocations

### What It Means

Creating new memory buffers for every row of the image, instead of reusing existing buffers.

### ELI5: Using New Paper for Every Math Problem

Imagine you're solving 2000 math problems on separate sheets of paper.

**The current way:**
```
Problem 1: Take new paper → Solve → Throw away paper
Problem 2: Take new paper → Solve → Throw away paper
Problem 3: Take new paper → Solve → Throw away paper
...
Problem 2000: Take new paper → Solve → Throw away paper

Result: 
- Used 2000 sheets of paper
- Generated 2000 trash items
- The garbage truck (garbage collector) has to pick up all 2000 items!
```

**The reusable way:**
```
Problem 1: Use whiteboard → Erase → Reuse for Problem 2
Problem 2: Use whiteboard → Erase → Reuse for Problem 3
...
Problem 2000: Use whiteboard

Result:
- Used 1 whiteboard
- No trash to collect
- Much faster and cleaner!
```

### Real Impact

For a 4000×3000 image with 5 filter evaluations per row:

| Metric | Per-Row Allocations | Reusable Buffers |
|--------|--------------------|------------------|
| Memory allocations | ~60,000 | ~5-10 |
| Garbage collector work | High | Minimal |
| Speed | Slower | **2-5x faster** |

### The Fix

Use Go's `sync.Pool` to reuse scratch buffers across rows.

---

## 6. Sequential Filtering

### What It Means

Processing each row of the image one at a time, even though rows are independent and could be processed simultaneously.

### ELI5: One Worker vs Multiple Workers

Imagine you have 2000 identical tasks (filtering 2000 rows of pixels).

**Sequential (one worker):**
```
Worker 1: Row 1 → Row 2 → Row 3 → ... → Row 2000
          (takes 2000 seconds if each row takes 1 second)
```

**Parallel (8 workers):**
```
Worker 1: Row 1 → Row 9 → Row 17 → ...
Worker 2: Row 2 → Row 10 → Row 18 → ...
Worker 3: Row 3 → Row 11 → Row 19 → ...
...
Worker 8: Row 8 → Row 16 → Row 24 → ...

          (takes ~250 seconds with 8 workers)
          SPEEDUP: 8x!
```

### When Parallel Helps Most

| Image Shape | Rows | Parallel Speedup |
|-------------|------|------------------|
| Wide (10000×100) | 100 | Minimal (too few rows) |
| Square (1000×1000) | 1000 | 4-6x |
| Tall (200×4000) | 4000 | **8-10x** |

### The Fix

Use goroutines to process rows in parallel, then collect results.

---

## 7. No Early Termination

### What It Means

Evaluating all filter options even when a clearly best option is found early.

### ELI5: Finding the Best Pizza Place

Imagine you're trying to find the best pizza place in town by visiting all 5 options.

**Without early termination:**
```
Visit Pizza Place 1: 7/10 → Keep as best
Visit Pizza Place 2: 6/10 → Not better, keep 7/10
Visit Pizza Place 3: 8/10 → New best!
Visit Pizza Place 4: 5/10 → Not better, keep 8/10
Visit Pizza Place 5: 6/10 → Not better, keep 8/10

Result: Found 8/10, but visited ALL 5 places.
```

**With early termination:**
```
Visit Pizza Place 1: 7/10 → Keep as best, threshold = 2
Visit Pizza Place 2: 6/10 → Not better, keep 7/10
Visit Pizza Place 3: 9/10 → New best! threshold = 2.25

STOP! The score 9 is SO GOOD that nothing else could beat it.
We already found the best possible option!

Result: Found 9/10, only visited 3 places. Saved 40% effort!
```

### How It Applies to PNG

For each image row, we try 5 filters. If one filter produces a near-perfect score, we can skip the remaining 4 filters:

```
Row 1: Filter None → Score 5 (very good!)
        early_stop threshold = len(row) / 4 = 100
        5 < 100 → STOP! Skip other filters.
        
        Result: Evaluated 1 filter instead of 5. 5x faster!
```

### The Fix

Add an early-exit condition when the best score is already "good enough."

---

## 8. Greedy LZ77 Matching

### What It Means

The compression algorithm takes the first match it finds, rather than the best overall match.

### ELI5: Choosing Books to Read

Imagine you're looking at a bookshelf and want to find repeated patterns.

**Greedy approach:**
```
Bookshelf: [A][B][C][A][B][D][A][B][C][A][B][E]
            └─────┘
            Found "A,B" at position 0!
            Let's use this match and move on.
            
            ...wait, we could have matched [A][B][C] three times!
            But we already committed to the first "A,B" match.
```

**Optimal approach:**
```
Bookshelf: [A][B][C][A][B][D][A][B][C][A][B][E]
            Look ahead at ALL possible matches:
            - Match "A,B" at 0: saves 1 byte, affects positions 2-11
            - Match "A,B,C" at 0: saves 2 bytes, affects positions 3-11
            - Match "A,B" at 3: saves 1 byte, affects positions 5-11
            ...
            
            Choose the option that gives the BEST overall compression,
            not just the first option we find.
```

### Why It Matters

| Approach | Compression | Speed |
|----------|-------------|-------|
| Greedy | Good | Fast |
| Optimal | **5-10% better** | Slower |

### The Fix

Use dynamic programming to find the optimal sequence of matches, considering how each choice affects future matches.

---

## Summary Table

| Bottleneck | Problem | Impact | Fix |
|------------|---------|--------|-----|
| Palette lookup O(n) | Linear search through palette | 10-100x slower | Lookup table (LUT) |
| No K-means | Palette doesn't fit image | 5-15% quality loss | Iterative refinement |
| No Bigrams filter | Filter doesn't optimize for DEFLATE | 2-5% compression loss | Count bigrams |
| Perceptual distance | Color matching ignores human vision | Suboptimal quality | Redmean formula |
| Per-row allocations | Too much memory allocation | GC pressure, slower | Reuse buffers |
| Sequential filtering | Single-threaded processing | No multi-core speedup | Parallel processing |
| No early termination | Evaluates all options | 10-30% wasted effort | Exit early when optimal |
| Greedy LZ77 | First match, not best match | 3-8% compression loss | Optimal parsing |

---

## The Big Picture

Think of PNG encoding like preparing food:

| Bottleneck | Analogy |
|------------|---------|
| Palette lookup | Finding ingredients by checking each one |
| K-means | Adjusting recipes to match your taste |
| Bigrams filter | Arranging food for best storage |
| Perceptual distance | Understanding which flavors matter |
| Per-row allocations | Using new plates for each dish |
| Sequential filtering | One cook working alone |
| Early termination | Stopping when dish is perfect |
| Greedy LZ77 | Taking first storage box instead of optimal packing |

**The goal:** Make every step as efficient as possible while keeping the final result (image quality) unchanged or improved.

---

## What This Means for Your Code

Your current implementation is like a careful but slow chef:

- ✅ Makes delicious food (good image quality)
- ✅ Follows recipes exactly (PNG specification compliant)
- ❌ Checks every ingredient one by one (O(n) palette lookup)
- ❌ Doesn't adjust recipes based on diners' feedback (no K-means)
- ❌ Uses new pots and pans for every dish (per-row allocations)
- ❌ Works alone instead of with a team (sequential filtering)

**The optimizations are like giving the chef better tools and techniques:**
- A well-organized pantry (LUT)
- Feedback from diners (K-means)
- A prep team (parallel processing)
- Smart cooking strategies (early termination, bigrams)

Result: Same great food (quality), but much faster service (speed)!
