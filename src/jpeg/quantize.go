package jpeg

import (
	"math"
)

// Standard JPEG luminance quantization table.
var stdLuminanceTable = [64]uint8{
	16, 11, 10, 16, 24, 40, 51, 61,
	12, 12, 14, 19, 26, 58, 60, 55,
	14, 13, 16, 24, 40, 57, 69, 56,
	14, 17, 22, 29, 51, 87, 80, 62,
	18, 22, 37, 56, 68, 109, 103, 77,
	24, 35, 55, 64, 81, 104, 113, 92,
	49, 64, 78, 87, 103, 121, 120, 101,
	72, 92, 95, 98, 112, 100, 103, 99,
}

// Standard JPEG chrominance quantization table.
var stdChrominanceTable = [64]uint8{
	17, 18, 24, 47, 99, 99, 99, 99,
	18, 21, 26, 66, 99, 99, 99, 99,
	24, 26, 56, 99, 99, 99, 99, 99,
	47, 66, 99, 99, 99, 99, 99, 99,
	99, 99, 99, 99, 99, 99, 99, 99,
	99, 99, 99, 99, 99, 99, 99, 99,
	99, 99, 99, 99, 99, 99, 99, 99,
	99, 99, 99, 99, 99, 99, 99, 99,
}

// QuantizationTables stores scaled quantization tables for luminance and chrominance.
type QuantizationTables struct {
	// Tables in natural order as float32 for computation
	Luminance   [64]float32
	Chrominance [64]float32

	// Tables in natural order as uint8 for markers
	LuminanceTable   [64]uint8
	ChrominanceTable [64]uint8
}

// NewQuantizationTables creates scaled quantization tables for the given quality (1-100).
func NewQuantizationTables(quality uint8) *QuantizationTables {
	if quality < 1 {
		quality = 1
	}
	if quality > 100 {
		quality = 100
	}

	// Calculate scale factor (same formula as libjpeg)
	var scale uint32
	if quality < 50 {
		scale = 5000 / uint32(quality)
	} else {
		scale = 200 - 2*uint32(quality)
	}

	qt := &QuantizationTables{}
	for i := 0; i < 64; i++ {
		// Scale and clamp to 1-255
		lumVal := (uint32(stdLuminanceTable[i])*scale + 50) / 100
		if lumVal < 1 {
			lumVal = 1
		} else if lumVal > 255 {
			lumVal = 255
		}

		chromVal := (uint32(stdChrominanceTable[i])*scale + 50) / 100
		if chromVal < 1 {
			chromVal = 1
		} else if chromVal > 255 {
			chromVal = 255
		}

		qt.LuminanceTable[i] = uint8(lumVal)
		qt.ChrominanceTable[i] = uint8(chromVal)
		qt.Luminance[i] = float32(lumVal)
		qt.Chrominance[i] = float32(chromVal)
	}

	return qt
}

// QuantizeBlock divides each DCT coefficient by the corresponding quantization value and rounds to the nearest integer.
func QuantizeBlock(dct [64]float32, table [64]float32) [64]int16 {
	var result [64]int16
	for i := 0; i < 64; i++ {
		result[i] = int16(math.Round(float64(dct[i] / table[i])))
	}
	return result
}
