//go:build js && wasm

package wasm

import (
	"fmt"
	"syscall/js"

	"github.com/mac/go-pixo/src/png"
)

/**
 * HandleEncodePng converts JS arguments to Go and calls EncodePng.
 * Expected arguments: (pixels: Uint8Array, width: number, height: number, colorType: number, preset: number, lossy: boolean, maxColors: number)
 */
func HandleEncodePng(this js.Value, args []js.Value) any {
	if len(args) < 7 {
		return js.ValueOf("invalid arguments")
	}

	pixelsJS := args[0]
	width := args[1].Int()
	height := args[2].Int()
	colorType := args[3].Int()
	preset := args[4].Int()
	lossy := args[5].Bool()
	maxColors := args[6].Int()

	// Copy JS buffer to Go slice
	pixels := make([]byte, pixelsJS.Get("length").Int())
	js.CopyBytesToGo(pixels, pixelsJS)

	// Call the actual implementation
	output, err := EncodePng(pixels, width, height, colorType, preset, lossy, maxColors)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}

	// Copy Go slice back to JS
	dst := js.Global().Get("Uint8Array").New(len(output))
	js.CopyBytesToJS(dst, output)

	return dst
}

/**
 * HandleEncodePngAdvanced converts JS arguments to Go and calls EncodePngAdvanced.
 * Expected arguments: (pixels: Uint8Array, width: number, height: number, colorType: number, preset: number, lossy: boolean, maxColors: number, dithering: boolean, ditherStrength: number, qualityTarget: number, zopfliIterations: number, progressCallback: function)
 */
func HandleEncodePngAdvanced(this js.Value, args []js.Value) any {
	if len(args) < 11 {
		return js.ValueOf("invalid arguments: expected 11 arguments")
	}

	pixelsJS := args[0]
	width := args[1].Int()
	height := args[2].Int()
	colorType := args[3].Int()
	preset := args[4].Int()
	lossy := args[5].Bool()
	maxColors := args[6].Int()
	dithering := args[7].Bool()
	ditherStrength := args[8].Float()
	qualityTarget := args[9].Int()
	zopfliIterations := args[10].Int()

	var progressFunc func(phase string, progress int)
	if len(args) > 11 && args[11].Type() == js.TypeFunction {
		cb := args[11]
		progressFunc = func(phase string, progress int) {
			cb.Invoke(phase, progress)
		}
	}

	// Copy JS buffer to Go slice
	pixels := make([]byte, pixelsJS.Get("length").Int())
	js.CopyBytesToGo(pixels, pixelsJS)

	// Call the advanced implementation
	output, err := EncodePngAdvanced(pixels, width, height, colorType, preset, lossy, maxColors, dithering, ditherStrength, qualityTarget, zopfliIterations, progressFunc)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}

	// Copy Go slice back to JS
	dst := js.Global().Get("Uint8Array").New(len(output))
	js.CopyBytesToJS(dst, output)

	return dst
}

/**
 * HandleBytesPerPixel returns the bytes per pixel for a given color type.
 * Expected arguments: (colorType: number)
 */
func HandleBytesPerPixel(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return js.ValueOf(0)
	}
	colorType := args[0].Int()
	return js.ValueOf(BytesPerPixel(colorType))
}

/**
 * HandleQuantizeInfo returns quantization capabilities.
 * No arguments required.
 */
func HandleQuantizeInfo(this js.Value, args []js.Value) any {
	return js.ValueOf(map[string]interface{}{
		"maxColors":           256,
		"ditheringSupported":  true,
		"minColors":           2,
		"maxDitherStrength":   1.0,
		"minDitherStrength":   0.0,
		"qualityTargetMin":    0,
		"qualityTargetMax":    100,
		"zopfliIterationsMax": 50,
	})
}

/**
 * HandleGetPresets returns available compression presets.
 * No arguments required.
 */
func HandleGetPresets(this js.Value, args []js.Value) any {
	return js.ValueOf(map[string]interface{}{
		"fast":     0,
		"balanced": 1,
		"max":      2,
		"extreme":  3,
	})
}

/**
 * HandleRecompressPngLossless recompresses an input PNG (provided as file bytes) in lossless mode.
 * Expected arguments: (pngBytes: Uint8Array, preset: number, zopfliIterations: number, progressCallback?: function)
 */
func HandleRecompressPngLossless(this js.Value, args []js.Value) any {
	if len(args) < 3 {
		return js.ValueOf("invalid arguments: expected 3 arguments")
	}

	pngBytesJS := args[0]
	preset := args[1].Int()
	zopfliIterations := args[2].Int()

	var progressFunc func(phase string, progress int)
	if len(args) > 3 && args[3].Type() == js.TypeFunction {
		cb := args[3]
		progressFunc = func(phase string, progress int) {
			cb.Invoke(phase, progress)
		}
	}

	input := make([]byte, pngBytesJS.Get("length").Int())
	js.CopyBytesToGo(input, pngBytesJS)

	output, err := RecompressPngLossless(input, preset, zopfliIterations, progressFunc)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}

	dst := js.Global().Get("Uint8Array").New(len(output))
	js.CopyBytesToJS(dst, output)
	return dst
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
	// Behavior:
	// - Smaller (0): Maximum compression with quality preservation
	// - Balanced (1): Standard trade-off
	// - Faster (2): Fast encoding with size guarantee (output <= original)
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
	// Map preset values to Options
	// - Smaller (0): Maximum compression with quality preservation
	// - Balanced (1): Standard trade-off
	// - Faster (2): Fast with size guarantee
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
 * This function provides access to advanced features like:
 * - Zopfli iterations for better DEFLATE compression
 * - Configurable dithering strength
 * - Quality target for lossy compression
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

	// Map preset values to Options
	// - Smaller (0): Maximum compression with quality preservation
	// - Balanced (1): Standard trade-off
	// - Faster (2): Fast with size guarantee
	// - Extreme (3): Maximum compression with Zopfli iterations
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

	// Apply progress callback
	if progressFunc != nil {
		opts.ProgressCallback = progressFunc
	}

	// Apply advanced options
	if zopfliIterations > 0 {
		opts.ZopfliIterations = zopfliIterations
	}

	// Apply lossy options if enabled
	if lossy && maxColors > 0 && maxColors <= 256 {
		// Clamp quality target to valid range
		if qualityTarget < 0 {
			qualityTarget = 0
		}
		if qualityTarget > 100 {
			qualityTarget = 100
		}
		// Clamp dither strength to valid range
		if ditherStrength < 0 {
			ditherStrength = 0
		}
		if ditherStrength > 1 {
			ditherStrength = 1
		}

		opts.ApplyLossy(maxColors, qualityTarget, ditherStrength)
		opts.ColorType = png.ColorIndexed
	} else if dithering {
		// Lossless mode with dithering for indexed color
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
