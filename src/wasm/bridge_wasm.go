//go:build js && wasm

package wasm

import (
	"fmt"
	"syscall/js"
)

/**
 * HandleEncodePng converts JS arguments to Go and calls EncodePng.
 * Expected arguments: (pixels: Uint8Array, width: number, height: number, colorType: number, preset: number, lossy: boolean, maxColors: number)
 */
func HandleEncodePng(this js.Value, args []js.Value) any {
	if len(args) < 7 {
		return js.ValueOf("error: invalid arguments: expected 7 arguments")
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
 * Expected arguments: (pixels: Uint8Array, width: number, height: number, colorType: number, preset: number, lossy: boolean, maxColors: number, dithering: boolean, ditherStrength: number, qualityTarget: number, zopfliIterations: number, filterStrategy: number, usePerceptual: boolean, distanceMetric: number, optimalDeflate: boolean, filterEarlyTerm: boolean, progressCallback?: function)
 */
func HandleEncodePngAdvanced(this js.Value, args []js.Value) any {
	if len(args) < 16 {
		return js.ValueOf("error: invalid arguments: expected at least 16 arguments")
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

	filterStrategy := args[11].Int()
	usePerceptual := args[12].Bool()
	distanceMetric := args[13].Int()
	optimalDeflate := args[14].Bool()
	filterEarlyTerm := args[15].Bool()

	var progressFunc func(phase string, progress int)
	if len(args) > 16 && args[16].Type() == js.TypeFunction {
		cb := args[16]
		progressFunc = func(phase string, progress int) {
			cb.Invoke(phase, progress)
		}
	}

	// Copy JS buffer to Go slice
	pixels := make([]byte, pixelsJS.Get("length").Int())
	js.CopyBytesToGo(pixels, pixelsJS)

	// Call the advanced implementation
	output, err := EncodePngAdvanced(
		pixels, width, height, colorType, preset,
		lossy, maxColors,
		dithering, ditherStrength, qualityTarget,
		zopfliIterations,
		filterStrategy,
		usePerceptual,
		distanceMetric,
		optimalDeflate,
		filterEarlyTerm,
		progressFunc,
	)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}

	// Copy Go slice back to JS
	dst := js.Global().Get("Uint8Array").New(len(output))
	js.CopyBytesToJS(dst, output)

	return dst
}

/**
 * HandleEncodeJpeg converts JS arguments to Go and calls EncodeJpeg.
 * Expected arguments: (pixels: Uint8Array, width: number, height: number, colorType: number, quality: number)
 */
func HandleEncodeJpeg(this js.Value, args []js.Value) any {
	if len(args) < 5 {
		return js.ValueOf("error: invalid arguments: expected 5 arguments")
	}

	pixelsJS := args[0]
	width := args[1].Int()
	height := args[2].Int()
	colorType := args[3].Int()
	quality := args[4].Int()

	// Copy JS buffer to Go slice
	pixels := make([]byte, pixelsJS.Get("length").Int())
	js.CopyBytesToGo(pixels, pixelsJS)

	// Call the actual implementation
	output, err := EncodeJpeg(pixels, width, height, colorType, uint8(quality))
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}

	// Copy Go slice back to JS
	dst := js.Global().Get("Uint8Array").New(len(output))
	js.CopyBytesToJS(dst, output)

	return dst
}

/**
 * HandleEncodeJpegAdvanced converts JS arguments to Go and calls EncodeJpegAdvanced.
 * Expected arguments: (pixels: Uint8Array, width: number, height: number, colorType: number, quality: number, subsampling: number, progressive: boolean, trellis: boolean, optimizeHuffman: boolean, preset: number)
 */
func HandleEncodeJpegAdvanced(this js.Value, args []js.Value) any {
	if len(args) < 10 {
		return js.ValueOf("error: invalid arguments: expected 10 arguments")
	}

	pixelsJS := args[0]
	width := args[1].Int()
	height := args[2].Int()
	colorType := args[3].Int()
	quality := args[4].Int()
	subsampling := args[5].Int()
	progressive := args[6].Bool()
	trellis := args[7].Bool()
	optimizeHuffman := args[8].Bool()
	preset := args[9].Int()

	// Copy JS buffer to Go slice
	pixels := make([]byte, pixelsJS.Get("length").Int())
	js.CopyBytesToGo(pixels, pixelsJS)

	// Call the advanced implementation
	output, err := EncodeJpegAdvanced(pixels, width, height, colorType, uint8(quality), subsampling, progressive, trellis, optimizeHuffman, preset)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}

	// Copy Go slice back to JS
	dst := js.Global().Get("Uint8Array").New(len(output))
	js.CopyBytesToJS(dst, output)

	return dst
}

/**
 * HandleRecompressJpeg recompresses an input JPEG (provided as file bytes).
 * Expected args: (jpegBytes: Uint8Array, preset: number, quality: number, progressive: boolean, trellis: boolean, optimizeHuffman: boolean)
 */
func HandleRecompressJpeg(this js.Value, args []js.Value) any {
	if len(args) < 6 {
		return js.ValueOf("error: invalid arguments: expected 6 arguments")
	}

	jpegBytesJS := args[0]
	preset := args[1].Int()
	quality := args[2].Int()
	progressive := args[3].Bool()
	trellis := args[4].Bool()
	optimizeHuffman := args[5].Bool()

	input := make([]byte, jpegBytesJS.Get("length").Int())
	js.CopyBytesToGo(input, jpegBytesJS)

	output, err := RecompressJpeg(input, preset, uint8(quality), progressive, trellis, optimizeHuffman)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}

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
		return js.ValueOf("error: invalid arguments: expected 3 arguments")
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
