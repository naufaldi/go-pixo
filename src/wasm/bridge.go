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
		"maxColors":          256,
		"ditheringSupported": true,
		"minColors":          2,
	})
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

	// Map ReScript presets to Go presets
	// ReScript: Smaller=0, Balanced=1, Faster=2
	// Go: PresetFast=0, PresetBalanced=1, PresetMax=2
	var opts png.Options
	switch preset {
	case 0: // Smaller
		opts = png.MaxOptions(width, height)
	case 1: // Balanced
		opts = png.BalancedOptions(width, height)
	case 2: // Faster
		opts = png.FastOptions(width, height)
	default:
		opts = png.BalancedOptions(width, height)
	}
	opts.ColorType = pngColorType

	// Apply lossy quantization if enabled
	if lossy && maxColors > 0 && maxColors <= 256 {
		opts.MaxColors = maxColors
		opts.Dithering = false
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
