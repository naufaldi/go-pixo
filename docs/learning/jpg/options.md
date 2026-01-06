# Encoder Options and Presets

`go-pixo` provides a flexible way to configure the JPEG encoder through an `Options` struct and a fluent `OptionsBuilder`.

## The `Options` Struct

The `Options` struct in `src/jpeg/options.go` controls every aspect of the encoding process:

```go
type Options struct {
    Width           int
    Height          int
    ColorType       ColorType
    Quality         uint8
    Subsampling     Subsampling
    OptimizeHuffman bool
    Progressive     bool
    TrellisQuant    bool
    RestartInterval *uint16
}
```

## Presets

For convenience, we provide three standard presets:

### 1. `FastOptions`
Optimized for encoding speed.
- Subsampling: **4:4:4**
- Huffman: Standard tables
- Encoding: Baseline

### 2. `BalancedOptions` (Default)
A good balance between speed, size, and quality.
- Subsampling: **4:2:0**
- Huffman: Standard tables
- Encoding: Baseline

### 3. `MaxOptions`
Optimized for the smallest possible file size.
- Subsampling: **4:2:0**
- Huffman: **Optimized tables** (Custom for every image)
- Encoding: **Progressive**

## Using the `OptionsBuilder`

The builder pattern allows for fine-grained control:

```go
opts := jpeg.NewOptionsBuilder(width, height).
    Quality(85).
    Subsampling(jpeg.Subsampling420).
    OptimizeHuffman(true).
    Progressive(true).
    Build()

encoder, _ := jpeg.NewEncoder(opts)
```

## Selecting the Right Options

| Use Case | Recommended Preset / Options |
| --- | --- |
| Real-time video/streaming | `FastOptions` |
| General web uploads | `BalancedOptions` |
| Static image hosting (CDN) | `MaxOptions` |
| High-res thumbnails | `BalancedOptions` + `Quality(60)` |
| Professional photography | `FastOptions` + `Quality(95)` |
