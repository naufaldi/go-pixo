//go:build js && wasm

package wasm

import (
	"fmt"
	"syscall/js"
)

// HandleEncodePng converts JS arguments to Go and calls EncodePng.
// Expected arguments: (pixels: Uint8Array, width: number, height: number, colorType: number, preset: number, lossy: boolean, maxColors: number, allowRGBAFallback?: boolean)
func HandleEncodePng(this js.Value, args []js.Value) any {
	if err := checkPNGArguments(args, false); err != nil {
		return argumentError(err)
	}
	pixelsJS := args[0]
	width := args[1].Int()
	height := args[2].Int()
	colorType := args[3].Int()
	preset := args[4].Int()
	lossy := args[5].Bool()
	maxColors := args[6].Int()

	// Copy JS buffer to Go slice
	pixels, err := copyJSBytes(pixelsJS)
	if err != nil {
		return argumentError(err)
	}

	// Basic full-budget compression is opportunistic unless explicitly disabled.
	allowRGBAFallback := true
	if len(args) > 7 && args[7].Type() == js.TypeBoolean {
		allowRGBAFallback = args[7].Bool()
	}

	// Call the actual implementation
	output, err := EncodePng(pixels, width, height, colorType, preset, lossy, maxColors, allowRGBAFallback)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}

	// Copy Go slice back to JS
	dst := js.Global().Get("Uint8Array").New(len(output))
	js.CopyBytesToJS(dst, output)

	return dst
}

// HandleEncodePngAdvanced converts JS arguments to Go and calls EncodePngAdvanced.
// Expected arguments: (pixels: Uint8Array, width: number, height: number, colorType: number, preset: number, lossy: boolean, maxColors: number, dithering: boolean, ditherStrength: number, qualityTarget: number, zopfliIterations: number, [filterStrategy: number, usePerceptual: boolean, distanceMetric: number, optimalDeflate: boolean, filterEarlyTerm: boolean], [progressCallback: function], [allowRGBAFallback: boolean])
func HandleEncodePngAdvanced(this js.Value, args []js.Value) any {
	if err := checkPNGArguments(args, true); err != nil {
		return argumentError(err)
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

	filterStrategy := 0
	usePerceptual := false
	distanceMetric := 0
	optimalDeflate := false
	filterEarlyTerm := false

	if len(args) >= 16 {
		filterStrategy = args[11].Int()
		usePerceptual = args[12].Bool()
		distanceMetric = args[13].Int()
		optimalDeflate = args[14].Bool()
		filterEarlyTerm = args[15].Bool()
	}

	var progressFunc func(phase string, progress int)
	if len(args) > 16 && args[16].Type() == js.TypeFunction {
		cb := args[16]
		progressFunc = func(phase string, progress int) {
			cb.Invoke(phase, progress)
		}
	} else if len(args) > 11 && args[11].Type() == js.TypeFunction {
		cb := args[11]
		progressFunc = func(phase string, progress int) {
			cb.Invoke(phase, progress)
		}
	}

	// Copy JS buffer to Go slice
	pixels, err := copyJSBytes(pixelsJS)
	if err != nil {
		return argumentError(err)
	}

	// Preserve strict behavior for explicit advanced quantizer calls. The app
	// opts in after the existing argument-17 progress callback.
	allowRGBAFallback := len(args) > 17 && args[17].Type() == js.TypeBoolean && args[17].Bool()

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
		allowRGBAFallback,
	)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}

	// Copy Go slice back to JS
	dst := js.Global().Get("Uint8Array").New(len(output))
	js.CopyBytesToJS(dst, output)

	return dst
}

// HandleEncodeJpeg converts JS arguments to Go and calls EncodeJpeg.
// Expected arguments: (pixels: Uint8Array, width: number, height: number, colorType: number, quality: number)
func HandleEncodeJpeg(this js.Value, args []js.Value) any {
	if err := checkJPEGArguments(args, false); err != nil {
		return argumentError(err)
	}
	pixelsJS := args[0]
	width := args[1].Int()
	height := args[2].Int()
	colorType := args[3].Int()
	quality := args[4].Int()

	// Copy JS buffer to Go slice
	pixels, err := copyJSBytes(pixelsJS)
	if err != nil {
		return argumentError(err)
	}

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

// HandleEncodeJpegAdvanced converts JS arguments to Go and calls EncodeJpegAdvanced.
// Expected arguments: (pixels: Uint8Array, width: number, height: number, colorType: number, quality: number, subsampling: number, progressive: boolean, trellis: boolean, optimizeHuffman: boolean, preset: number)
func HandleEncodeJpegAdvanced(this js.Value, args []js.Value) any {
	if err := checkJPEGArguments(args, true); err != nil {
		return argumentError(err)
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
	pixels, err := copyJSBytes(pixelsJS)
	if err != nil {
		return argumentError(err)
	}

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

// HandleRecompressJpeg recompresses an input JPEG (provided as file bytes).
// Expected args: (jpegBytes: Uint8Array, preset: number, quality: number, progressive: boolean, trellis: boolean, optimizeHuffman: boolean)
func HandleRecompressJpeg(this js.Value, args []js.Value) any {
	if err := checkArguments(args, []argumentRule{integerArg(0, 4), integerArg(1, 100), booleanArg, booleanArg, booleanArg}); err != nil {
		return argumentError(err)
	}
	if err := checkEncodedBytes(args[0]); err != nil {
		return argumentError(err)
	}
	jpegBytesJS := args[0]
	preset := args[1].Int()
	quality := args[2].Int()
	progressive := args[3].Bool()
	trellis := args[4].Bool()
	optimizeHuffman := args[5].Bool()

	input, err := copyJSBytes(jpegBytesJS)
	if err != nil {
		return argumentError(err)
	}

	output, err := RecompressJpeg(input, preset, uint8(quality), progressive, trellis, optimizeHuffman)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}

	dst := js.Global().Get("Uint8Array").New(len(output))
	js.CopyBytesToJS(dst, output)
	return dst
}

// HandleBytesPerPixel returns the bytes per pixel for a given color type.
// Expected arguments: (colorType: number)
func HandleBytesPerPixel(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return js.ValueOf(0)
	}
	if args[0].Type() != js.TypeNumber {
		return argumentError(fmt.Errorf("color type must be an integer"))
	}
	value := args[0].Float()
	if value != 0 && value != 2 && value != 3 && value != 6 {
		return argumentError(fmt.Errorf("unsupported color type"))
	}
	return js.ValueOf(BytesPerPixel(int(value)))
}

// HandleQuantizeInfo returns quantization capabilities.
// No arguments required.
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

// HandleGetPresets returns available compression presets.
// No arguments required.
func HandleGetPresets(this js.Value, args []js.Value) any {
	return js.ValueOf(map[string]interface{}{
		"fast":     0,
		"balanced": 1,
		"max":      2,
		"extreme":  3,
	})
}

// HandleRecompressPngLossless recompresses an input PNG (provided as file bytes) in lossless mode.
// Expected arguments: (pngBytes: Uint8Array, preset: number, zopfliIterations: number, progressCallback?: function)
func HandleRecompressPngLossless(this js.Value, args []js.Value) any {
	if err := checkArguments(args, []argumentRule{integerArg(0, 4), integerArg(0, 50)}); err != nil {
		return argumentError(err)
	}
	if err := checkCallback(args, 3); err != nil {
		return argumentError(err)
	}
	if err := checkEncodedBytes(args[0]); err != nil {
		return argumentError(err)
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

	input, err := copyJSBytes(pngBytesJS)
	if err != nil {
		return argumentError(err)
	}

	output, err := RecompressPngLossless(input, preset, zopfliIterations, progressFunc)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}

	dst := js.Global().Get("Uint8Array").New(len(output))
	js.CopyBytesToJS(dst, output)
	return dst
}
