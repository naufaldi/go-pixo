package jpeg

// RGBToYCbCr converts an RGB pixel to YCbCr using ITU-R BT.601 coefficients
// with fixed-point arithmetic for performance.
// Formula:
// Y  = (77*R + 150*G + 29*B + 128) >> 8
// Cb = ((-43*R - 85*G + 128*B + 128) >> 8) + 128
// Cr = ((128*R - 107*G - 21*B + 128) >> 8) + 128
func RGBToYCbCr(r, g, b uint8) (y, cb, cr uint8) {
	ri, gi, bi := int32(r), int32(g), int32(b)

	// Fixed-point coefficients (scaled by 256)
	// +128 for rounding before right shift
	yi := (77*ri + 150*gi + 29*bi + 128) >> 8
	cbi := ((-43*ri - 85*gi + 128*bi + 128) >> 8) + 128
	cri := ((128*ri - 107*gi - 21*bi + 128) >> 8) + 128

	return uint8(clamp(yi)), uint8(clamp(cbi)), uint8(clamp(cri))
}

// YCbCrToRGB converts a YCbCr pixel to RGB for testing purposes.
// Formula:
// R = Y + 1.402 * (Cr - 128)
// G = Y - 0.344136 * (Cb - 128) - 0.714136 * (Cr - 128)
// B = Y + 1.772 * (Cb - 128)
func YCbCrToRGB(y, cb, cr uint8) (r, g, b uint8) {
	yi := int32(y)
	cbi := int32(cb) - 128
	cri := int32(cr) - 128

	// Fixed-point coefficients (scaled by 65536 for more precision in inverse)
	// R = Y + 91881 * (Cr - 128) >> 16
	// G = Y - 22554 * (Cb - 128) >> 16 - 46802 * (Cr - 128) >> 16
	// B = Y + 116130 * (Cb - 128) >> 16

	ri := yi + (91881*cri+32768)>>16
	gi := yi - (22554*cbi+32768)>>16 - (46802*cri+32768)>>16
	bi := yi + (116130*cbi+32768)>>16

	return uint8(clamp(ri)), uint8(clamp(gi)), uint8(clamp(bi))
}

func clamp(v int32) int32 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}
