//go:build js && wasm

package wasm

import (
	"fmt"
	"syscall/js"
)

/**
 * HandleEncodePng converts JS arguments to Go and calls EncodePng.
 * Expected arguments: (pixels: Uint8Array, width: number, height: number, colorType: number, preset: number, lossy: boolean)
 */
func HandleEncodePng(this js.Value, args []js.Value) any {
	if len(args) < 6 {
		return js.ValueOf("invalid arguments")
	}

	pixelsJS := args[0]
	width := args[1].Int()
	height := args[2].Int()
	colorType := args[3].Int()
	preset := args[4].Int()
	lossy := args[5].Bool()

	// Copy JS buffer to Go slice
	pixels := make([]byte, pixelsJS.Get("length").Int())
	js.CopyBytesToGo(pixels, pixelsJS)

	// Call the actual implementation (placeholder for now)
	output, err := EncodePng(pixels, width, height, colorType, preset, lossy)
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
 * EncodePng is a placeholder for the actual implementation in Phase 1.
 * Currently returns input pixels to verify data transfer.
 */
func EncodePng(pixels []byte, width, height int, colorType, preset int, lossy bool) ([]byte, error) {
	// TODO: Implement real PNG encoding in Phase 1
	return pixels, nil
}

/**
 * BytesPerPixel returns bytes per pixel based on color type.
 * 2 = RGB, 6 = RGBA
 */
func BytesPerPixel(colorType int) int {
	switch colorType {
	case 2: // RGB
		return 3
	case 6: // RGBA
		return 4
	default:
		return 4
	}
}
