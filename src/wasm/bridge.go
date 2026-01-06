package wasm

import (
	"fmt"

	"github.com/mac/go-pixo/src/jpeg"
	"github.com/mac/go-pixo/src/png"
)

/**
 * EncodeJpeg encodes pixels as a JPEG image using the go-pixo JPEG encoder.
 */
func EncodeJpeg(pixels []byte, width, height int, colorType int, quality uint8) ([]byte, error) {
	var jpegColorType jpeg.ColorType
	switch colorType {
	case 1:
		jpegColorType = jpeg.ColorGrayscale
	case 3:
		jpegColorType = jpeg.ColorRGB
	default:
		return nil, fmt.Errorf("unsupported JPEG color type: %d", colorType)
	}

	opts := jpeg.BalancedOptions(width, height, quality)
	opts.ColorType = jpegColorType

	encoder, err := jpeg.NewEncoder(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create JPEG encoder: %w", err)
	}

	jpegBytes, err := encoder.Encode(pixels)
	if err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return jpegBytes, nil
}

/**
 * EncodeJpegAdvanced encodes pixels as a JPEG image with full control over all options.
 * Parameters:
 *   - pixels: raw pixel data
 *   - width, height: image dimensions
 *   - colorType: 1=grayscale, 3=RGB
 *   - quality: 1-100 quality level
 *   - subsampling: 0=4:2:0, 1=4:4:4
 *   - progressive: enable progressive encoding
 *   - trellis: enable trellis quantization
 *   - optimizeHuffman: use optimized Huffman tables
 *   - preset: 0=fast, 1=balanced, 2=max
 */
func EncodeJpegAdvanced(pixels []byte, width, height int, colorType int, quality uint8, subsampling int, progressive bool, trellis bool, optimizeHuffman bool, preset int) ([]byte, error) {
	var jpegColorType jpeg.ColorType
	switch colorType {
	case 1:
		jpegColorType = jpeg.ColorGrayscale
	case 3:
		jpegColorType = jpeg.ColorRGB
	default:
		return nil, fmt.Errorf("unsupported JPEG color type: %d", colorType)
	}

	// Create options based on preset
	var opts jpeg.Options
	switch preset {
	case 0: // Fast
		opts = jpeg.FastOptions(width, height, quality)
	case 2: // Max
		opts = jpeg.MaxOptions(width, height, quality)
	default: // Balanced
		opts = jpeg.BalancedOptions(width, height, quality)
	}

	// Apply color type
	opts.ColorType = jpegColorType

	// Apply subsampling
	if subsampling == 1 {
		opts.Subsampling = jpeg.Subsampling444
	} else {
		opts.Subsampling = jpeg.Subsampling420
	}

	// Apply advanced options
	opts.Progressive = progressive
	opts.TrellisQuant = trellis
	opts.OptimizeHuffman = optimizeHuffman

	encoder, err := jpeg.NewEncoder(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create JPEG encoder: %w", err)
	}

	jpegBytes, err := encoder.Encode(pixels)
	if err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return jpegBytes, nil
}

/**
 * EncodePng encodes pixels as a PNG image using the go-pixo PNG encoder.
 * Returns PNG file bytes ready to be written to a file or used in a browser.
 */
func EncodePng(pixels []byte, width, height int, colorType, preset int, lossy bool, maxColors int) ([]byte, error) {
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

	// Map ReScript presets to Go options
	// ReScript: Smaller=0, Balanced=1, Faster=2
	var opts png.Options
	switch preset {
	case 0: // Smaller - Maximum compression
		opts = png.SmallerOptions(width, height)
	case 1: // Balanced
		opts = png.BalancedOptions(width, height)
	case 2: // Faster - Fast with size guarantee
		opts = png.FasterOptions(width, height)
	default:
		opts = png.BalancedOptions(width, height)
	}
	opts.ColorType = pngColorType

	// Apply lossy quantization if enabled
	if lossy && maxColors > 0 && maxColors <= 256 {
		opts.ApplyLossy(maxColors, 75, 0.5)
		opts.ColorType = png.ColorIndexed
	}

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

func RecompressPngLossless(inputPNG []byte, preset int, zopfliIterations int, progressFunc func(string, int)) ([]byte, error) {
	var opts png.Options
	switch preset {
	case 0: // Smaller
		opts = png.SmallerOptions(1, 1)
	case 1: // Balanced
		opts = png.BalancedOptions(1, 1)
	case 2: // Faster - Fast with size guarantee
		opts = png.FasterOptions(1, 1)
	default:
		opts = png.BalancedOptions(1, 1)
	}

	if progressFunc != nil {
		opts.ProgressCallback = progressFunc
	}
	if zopfliIterations > 0 {
		opts.ZopfliIterations = zopfliIterations
	}

	return png.RecompressPNGBytesLossless(inputPNG, opts)
}

/**
 * EncodePngAdvanced encodes pixels with full control over all compression options.
 */
func EncodePngAdvanced(pixels []byte, width, height int, colorType, preset int, lossy bool, maxColors int, dithering bool, ditherStrength float64, qualityTarget int, zopfliIterations int, progressFunc func(string, int)) ([]byte, error) {
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
	case 0: // Smaller
		opts = png.SmallerOptions(width, height)
	case 1: // Balanced
		opts = png.BalancedOptions(width, height)
	case 2: // Faster - Fast with size guarantee
		opts = png.FasterOptions(width, height)
	case 3: // Extreme
		opts = png.ExtremeOptions(width, height)
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
		opts.ColorType = png.ColorIndexed
	} else if dithering {
		opts.Dithering = true
		opts.DitheringStrength = ditherStrength
	}

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

/**
 * BytesPerPixel returns bytes per pixel based on color type.
 * 0 = Grayscale (1), 2 = RGB (3), 6 = RGBA (4), 3 = Indexed (1)
 */
func BytesPerPixel(colorType int) int {
	switch colorType {
	case 0: // Grayscale
		return 1
	case 2: // RGB
		return 3
	case 3: // Indexed
		return 1
	case 6: // RGBA
		return 4
	default:
		return 4
	}
}
