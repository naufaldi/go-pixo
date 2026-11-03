//go:build js && wasm

package wasm

import (
	"fmt"
	"syscall/js"
	"unsafe"

	"github.com/mac/go-pixo/src/internal/imagelimit"
)

// maxJpegBands bounds the band list of HandleEncodeJpegBands; the app uses
// at most a handful of band workers.
const maxJpegBands = 64

// int16Bytes views coefficients as bytes in the WASM's little-endian order.
// Band buffers only travel between instances of this module, so the order
// is the same on both sides.
func int16Bytes(values []int16) []byte {
	if len(values) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&values[0])), 2*len(values))
}

func jpegFrameRules() []argumentRule {
	return []argumentRule{
		integerArg(1, imagelimit.MaxDimension), integerArg(1, imagelimit.MaxDimension), integerArg(1, 3), integerArg(1, 100),
		integerArg(0, 1), booleanArg, booleanArg, booleanArg, integerArg(0, 4),
	}
}

// HandleEncodeJpegBand quantizes one band of MCU rows for a parallel encode.
// Expected arguments: (pixels: Uint8Array, width, height, colorType, quality,
// subsampling, progressive, trellis, optimizeHuffman, preset, firstRow, rows),
// where pixels holds only the band's image rows and the other options match
// encodeJpegAdvanced. Returns a Uint8Array of the band's int16 coefficients,
// for encodeJpegBands in another instance of this module.
func HandleEncodeJpegBand(this js.Value, args []js.Value) any {
	rules := append(jpegFrameRules(), integerArg(0, imagelimit.MaxDimension), integerArg(1, imagelimit.MaxDimension))
	if err := checkArguments(args, rules); err != nil {
		return argumentError(err)
	}
	width, height, colorType, quality := args[1].Int(), args[2].Int(), args[3].Int(), args[4].Int()
	subsampling, progressive, trellis, optimizeHuffman, preset := args[5].Int(), args[6].Bool(), args[7].Bool(), args[8].Bool(), args[9].Int()
	firstRow, rows := args[10].Int(), args[11].Int()

	encoder, err := newJpegAdvancedEncoder(width, height, colorType, uint8(quality), subsampling, progressive, trellis, optimizeHuffman, preset)
	if err != nil {
		return argumentError(err)
	}
	if firstRow+rows > encoder.MCURows() {
		return argumentError(fmt.Errorf("band rows exceed the image's %d MCU rows", encoder.MCURows()))
	}
	_, bandHeight := encoder.BandPixelRows(firstRow, rows)
	expected, err := imagelimit.PixelBytes(width, bandHeight, colorType)
	if err != nil {
		return argumentError(err)
	}
	if actual, lengthErr := byteArrayLength(args[0]); lengthErr != nil {
		return argumentError(lengthErr)
	} else if actual != expected {
		return argumentError(fmt.Errorf("band pixel data length %d does not match expected %d", actual, expected))
	}
	pixels, err := copyJSBytes(args[0])
	if err != nil {
		return argumentError(err)
	}

	band, err := encoder.EncodeBand(pixels, firstRow, rows)
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}
	raw := int16Bytes(band)
	dst := js.Global().Get("Uint8Array").New(len(raw))
	js.CopyBytesToJS(dst, raw)
	return dst
}

// HandleEncodeJpegBands writes the JPEG for bands from encodeJpegBand.
// Expected arguments: (bands: Uint8Array[], width, height, colorType, quality,
// subsampling, progressive, trellis, optimizeHuffman, preset). Bands are in
// row order and must cover every MCU row; the output equals
// encodeJpegAdvanced for the same pixels and options.
func HandleEncodeJpegBands(this js.Value, args []js.Value) any {
	if err := checkArguments(args, jpegFrameRules()); err != nil {
		return argumentError(err)
	}
	if !js.Global().Get("Array").Call("isArray", args[0]).Bool() {
		return argumentError(fmt.Errorf("bands must be an array"))
	}
	bands := args[0]
	count := bands.Length()
	if count < 1 || count > maxJpegBands {
		return argumentError(fmt.Errorf("expected 1 to %d bands", maxJpegBands))
	}
	frame, err := NewJpegCoefficientFrame(args[1].Int(), args[2].Int(), args[3].Int(), uint8(args[4].Int()),
		args[5].Int(), args[6].Bool(), args[7].Bool(), args[8].Bool(), args[9].Int())
	if err != nil {
		return argumentError(err)
	}
	rowBytes := 2 * frame.BandLength(1)
	row := 0
	for i := 0; i < count; i++ {
		band := bands.Index(i)
		n, lengthErr := byteArrayLength(band)
		if lengthErr != nil {
			return argumentError(lengthErr)
		}
		if n == 0 || n%rowBytes != 0 {
			return argumentError(fmt.Errorf("band %d length %d is not a whole number of MCU rows", i, n))
		}
		rows := n / rowBytes
		segments, segmentErr := frame.BandSegments(row, rows)
		if segmentErr != nil {
			return argumentError(fmt.Errorf("band %d: %w", i, segmentErr))
		}
		// Copy each component's rows straight into the frame's storage.
		offset := 0
		for _, segment := range segments {
			raw := int16Bytes(segment)
			if copyErr := copyJSBytesInto(raw, band, offset); copyErr != nil {
				return argumentError(copyErr)
			}
			offset += len(raw)
		}
		row += rows
	}
	if row != frame.MCURows() {
		return argumentError(fmt.Errorf("bands cover %d of %d MCU rows", row, frame.MCURows()))
	}

	output, err := frame.Encode()
	if err != nil {
		return js.ValueOf(fmt.Sprintf("error: %v", err))
	}
	dst := js.Global().Get("Uint8Array").New(len(output))
	js.CopyBytesToJS(dst, output)
	return dst
}
