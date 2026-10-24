package wasm

import (
	"fmt"

	"github.com/mac/go-pixo/src/jpeg"
	"github.com/mac/go-pixo/src/png"
)

// EncodePng encodes pixels with preset compression. A full 256-color budget is
// opportunistic: transparent RGBA falls back to sample-preserving compression.
// Pass false as allowRGBAFallback to require a palette even at that budget.
// Reduced budgets always require an opaque palette-compatible input.
func EncodePng(pixels []byte, width, height int, colorType, preset int, lossy bool, maxColors int, allowRGBAFallback ...bool) ([]byte, error) {
	var pngColorType png.ColorType
	switch colorType {
	case 0:
		pngColorType = png.ColorGrayscale
	case 2:
		pngColorType = png.ColorRGB
	case 6:
		pngColorType = png.ColorRGBA
	default:
		return nil, fmt.Errorf("unsupported color type: %d", colorType)
	}

	var opts png.Options
	switch preset {
	case 0:
		opts = png.SmallerOptions(width, height)
	case 1:
		opts = png.BalancedOptions(width, height)
	case 2:
		opts = png.FasterOptions(width, height)
	case 4:
		opts = png.UltraOptions(width, height)
	default:
		opts = png.BalancedOptions(width, height)
	}
	opts.ColorType = pngColorType

	if lossy && maxColors > 0 && maxColors <= 256 {
		opts.ApplyLossy(maxColors, 75, 0.5)
	} else {
		opts.ApplyLossless()
		opts.OptimizeAlpha = false
	}

	allowFallback := len(allowRGBAFallback) == 0 || allowRGBAFallback[0]
	applyPngRGBAFallback(&opts, pixels, allowFallback)

	encoder, err := png.NewEncoderWithOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}

	pngBytes, err := encoder.Encode(pixels)
	if err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return pngBytes, nil
}

// EncodePngAdvanced encodes pixels with explicit quantizer and compression controls.
// Palette requests are strict by default. App presets may pass true as the optional
// allowRGBAFallback argument to permit sample-preserving RGBA at a 256-color budget.
// On that fallback only palette controls are inapplicable; filter/DEFLATE controls
// remain active. Reduced palette budgets remain strict even with this opt-in.
func EncodePngAdvanced(pixels []byte, width, height int, colorType, preset int, lossy bool, maxColors int, dithering bool, ditherStrength float64, qualityTarget int, zopfliIterations int, filterStrategy int, usePerceptual bool, distanceMetric int, optimalDeflate bool, filterEarlyTerm bool, progressFunc func(string, int), allowRGBAFallback ...bool) ([]byte, error) {
	var pngColorType png.ColorType
	switch colorType {
	case 0:
		pngColorType = png.ColorGrayscale
	case 2:
		pngColorType = png.ColorRGB
	case 6:
		pngColorType = png.ColorRGBA
	default:
		return nil, fmt.Errorf("unsupported color type: %d", colorType)
	}

	var opts png.Options
	switch preset {
	case 0:
		opts = png.SmallerOptions(width, height)
	case 1:
		opts = png.BalancedOptions(width, height)
	case 2:
		opts = png.FasterOptions(width, height)
	case 3:
		opts = png.ExtremeOptions(width, height)
	case 4:
		opts = png.UltraOptions(width, height)
	default:
		opts = png.BalancedOptions(width, height)
	}
	opts.ColorType = pngColorType

	if progressFunc != nil {
		opts.ProgressCallback = progressFunc
	}

	if zopfliIterations > 0 {
		opts.ZopfliIterations = zopfliIterations
	}

	if optimalDeflate {
		opts.OptimalDeflate = true
	}

	if filterStrategy >= int(png.FilterStrategyNone) {
		opts.FilterStrategy = png.FilterStrategy(filterStrategy)
	}
	if usePerceptual {
		opts.UsePerceptualDistance = true
	}
	if distanceMetric >= int(png.DistanceMetricEuclidean) {
		opts.DistanceMetric = png.DistanceMetric(distanceMetric)
	}
	opts.FilterEarlyTermination = filterEarlyTerm

	if lossy && maxColors > 0 && maxColors <= 256 {
		if qualityTarget < 0 {
			qualityTarget = 0
		}
		if qualityTarget > 100 {
			qualityTarget = 100
		}
		if ditherStrength < 0 {
			ditherStrength = 0
		}
		if ditherStrength > 1 {
			ditherStrength = 1
		}

		opts.ApplyLossy(maxColors, qualityTarget, ditherStrength)
		opts.Dithering = dithering
	} else {
		opts.ApplyLossless()
		opts.OptimizeAlpha = false
	}

	applyPngRGBAFallback(&opts, pixels, len(allowRGBAFallback) > 0 && allowRGBAFallback[0])

	encoder, err := png.NewEncoderWithOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}

	pngBytes, err := encoder.Encode(pixels)
	if err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return pngBytes, nil
}

// Keep this policy at the public compression boundary. The low-level encoder's
// MaxColors contract always means a required RGB palette and stays strict.
func applyPngRGBAFallback(opts *png.Options, pixels []byte, allow bool) {
	if !allow || opts.MaxColors != 256 || opts.ColorType != png.ColorRGBA {
		return
	}
	for i := 3; i < len(pixels); i += 4 {
		if pixels[i] != 255 {
			opts.ApplyLossless()
			opts.OptimizeAlpha = false // Preserve RGB even under zero alpha.
			return
		}
	}
}

func EncodeJpeg(pixels []byte, width, height int, colorType int, quality uint8) ([]byte, error) {
	var jpegColorType jpeg.ColorType
	switch colorType {
	case 1:
		jpegColorType = jpeg.ColorGrayscale
	case 3:
		jpegColorType = jpeg.ColorRGB
	default:
		return nil, fmt.Errorf("unsupported color type: %d", colorType)
	}

	opts := jpeg.BalancedOptions(width, height, quality)
	opts.ColorType = jpegColorType
	if jpegColorType == jpeg.ColorGrayscale {
		opts.Subsampling = jpeg.Subsampling444
	}

	encoder, err := jpeg.NewEncoder(opts)
	if err != nil {
		return nil, err
	}

	return encoder.Encode(pixels)
}

func EncodeJpegAdvanced(pixels []byte, width, height int, colorType int, quality uint8, subsampling int, progressive bool, trellis bool, optimizeHuffman bool, preset int) ([]byte, error) {
	var jpegColorType jpeg.ColorType
	switch colorType {
	case 1:
		jpegColorType = jpeg.ColorGrayscale
	case 3:
		jpegColorType = jpeg.ColorRGB
	default:
		return nil, fmt.Errorf("unsupported color type: %d", colorType)
	}

	var opts jpeg.Options
	switch preset {
	case 0:
		opts = jpeg.FastOptions(width, height, quality)
	case 1:
		opts = jpeg.BalancedOptions(width, height, quality)
	case 2:
		opts = jpeg.MaxOptions(width, height, quality)
	case 4:
		opts = jpeg.MaxOptions(width, height, quality)
	default:
		opts = jpeg.BalancedOptions(width, height, quality)
	}

	opts.ColorType = jpegColorType
	opts.Progressive = progressive
	opts.TrellisQuant = trellis
	if trellis {
		optimizeHuffman = false
	}
	opts.OptimizeHuffman = optimizeHuffman

	if subsampling == 1 {
		opts.Subsampling = jpeg.Subsampling444
	} else {
		opts.Subsampling = jpeg.Subsampling420
	}
	if jpegColorType == jpeg.ColorGrayscale {
		opts.Subsampling = jpeg.Subsampling444
	}

	encoder, err := jpeg.NewEncoder(opts)
	if err != nil {
		return nil, err
	}

	return encoder.Encode(pixels)
}

func RecompressPngLossless(inputPNG []byte, preset int, zopfliIterations int, progressFunc func(string, int)) ([]byte, error) {
	var opts png.Options
	switch preset {
	case 0:
		opts = png.SmallerOptions(1, 1)
	case 1:
		opts = png.BalancedOptions(1, 1)
	case 2:
		opts = png.FasterOptions(1, 1)
	case 3:
		opts = png.ExtremeOptions(1, 1)
	case 4:
		opts = png.UltraOptions(1, 1)
	default:
		opts = png.BalancedOptions(1, 1)
	}

	if zopfliIterations > 0 {
		opts.ZopfliIterations = zopfliIterations
	}
	if progressFunc != nil {
		opts.ProgressCallback = progressFunc
	}

	return png.RecompressPNGBytesLossless(inputPNG, opts)
}

func RecompressJpeg(inputJPEG []byte, preset int, quality uint8, progressive bool, trellis bool, optimizeHuffman bool) ([]byte, error) {
	var opts jpeg.Options
	switch preset {
	case 0:
		opts = jpeg.FastOptions(1, 1, quality)
	case 1:
		opts = jpeg.BalancedOptions(1, 1, quality)
	case 2:
		opts = jpeg.MaxOptions(1, 1, quality)
	case 4:
		opts = jpeg.MaxOptions(1, 1, quality)
	default:
		opts = jpeg.BalancedOptions(1, 1, quality)
	}

	opts.Progressive = progressive
	opts.TrellisQuant = trellis
	if trellis {
		optimizeHuffman = false
	}
	opts.OptimizeHuffman = optimizeHuffman

	return jpeg.RecompressJPEGBytes(inputJPEG, opts)
}

func BytesPerPixel(colorType int) int {
	switch colorType {
	case 0:
		return 1
	case 2:
		return 3
	case 3:
		return 1
	case 6:
		return 4
	default:
		return 4
	}
}
