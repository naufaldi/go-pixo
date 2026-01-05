package jpeg

import (
	"math"
)

// AAN DCT constants
const (
	a1 = 0.70710678118 // cos(4*pi/16) = 1/sqrt(2)
	a2 = 0.54119610014 // cos(6*pi/16) - cos(2*pi/16)
	a3 = 0.70710678118 // cos(4*pi/16) = 1/sqrt(2)
	a4 = 1.30656296487 // cos(2*pi/16) + cos(6*pi/16)
	a5 = 0.38268343236 // cos(6*pi/16)
)

// Post-scaling factors for the AAN algorithm
var sFactors = [8]float32{
	0.35355339059, // 1/(2*sqrt(2))
	0.25489778955, // 1/(4*cos(pi/16))
	0.27059805007, // 1/(4*cos(2*pi/16))
	0.30067244346, // 1/(4*cos(3*pi/16))
	0.35355339059, // 1/(4*cos(4*pi/16)) = 1/(2*sqrt(2))
	0.44998811156, // 1/(4*cos(5*pi/16))
	0.65328148243, // 1/(4*cos(6*pi/16))
	1.28145772387, // 1/(4*cos(7*pi/16))
}

// ForwardDCT performs a 2D DCT on an 8x8 block using the AAN fast DCT algorithm.
func ForwardDCT(block [64]float32) [64]float32 {
	var temp [64]float32
	var result [64]float32

	// 1D DCT on rows
	for row := 0; row < 8; row++ {
		rowStart := row * 8
		rowData := [8]float32{
			block[rowStart], block[rowStart+1], block[rowStart+2], block[rowStart+3],
			block[rowStart+4], block[rowStart+5], block[rowStart+6], block[rowStart+7],
		}
		aanDCT1D(&rowData)
		for i := 0; i < 8; i++ {
			temp[rowStart+i] = rowData[i]
		}
	}

	// 1D DCT on columns
	for col := 0; col < 8; col++ {
		var colData [8]float32
		for row := 0; row < 8; row++ {
			colData[row] = temp[row*8+col]
		}
		aanDCT1D(&colData)
		for row := 0; row < 8; row++ {
			result[row*8+col] = colData[row]
		}
	}

	return result
}

// aanDCT1D performs a 1D DCT on 8 values using the AAN algorithm.
func aanDCT1D(data *[8]float32) {
	// Stage 1
	tmp0 := data[0] + data[7]
	tmp7 := data[0] - data[7]
	tmp1 := data[1] + data[6]
	tmp6 := data[1] - data[6]
	tmp2 := data[2] + data[5]
	tmp5 := data[2] - data[5]
	tmp3 := data[3] + data[4]
	tmp4 := data[3] - data[4]

	// Stage 2: Even part
	tmp10 := tmp0 + tmp3
	tmp13 := tmp0 - tmp3
	tmp11 := tmp1 + tmp2
	tmp12 := tmp1 - tmp2

	data[0] = tmp10 + tmp11
	data[4] = tmp10 - tmp11

	z1 := (tmp12 + tmp13) * float32(a1)
	data[2] = tmp13 + z1
	data[6] = tmp13 - z1

	// Stage 3: Odd part
	tmp10 = tmp4 + tmp5
	tmp11 = tmp5 + tmp6
	tmp12 = tmp6 + tmp7

	z5 := (tmp10 - tmp12) * float32(a5)
	z2 := tmp10*float32(a2) + z5
	z4 := tmp12*float32(a4) + z5
	z3 := tmp11 * float32(a3)

	z11 := tmp7 + z3
	z13 := tmp7 - z3

	data[5] = z13 + z2
	data[3] = z13 - z2
	data[1] = z11 + z4
	data[7] = z11 - z4

	// Stage 4: Post-scaling
	for i := 0; i < 8; i++ {
		data[i] *= sFactors[i]
	}
}

// InverseDCT performs an inverse 2D DCT on an 8x8 block for testing purposes.
// It uses a naive O(N^2) implementation as precision is more important than speed here.
func InverseDCT(block [64]float32) [64]float32 {
	var temp [64]float32
	var result [64]float32

	// 1D IDCT on columns
	for col := 0; col < 8; col++ {
		var colIn [8]float32
		var colOut [8]float32
		for row := 0; row < 8; row++ {
			colIn[row] = block[row*8+col]
		}
		idct1D(colIn[:], colOut[:])
		for row := 0; row < 8; row++ {
			temp[row*8+col] = colOut[row]
		}
	}

	// 1D IDCT on rows
	for row := 0; row < 8; row++ {
		rowStart := row * 8
		idct1D(temp[rowStart:rowStart+8], result[rowStart:rowStart+8])
	}

	return result
}

var cosTable [8][8]float32

func init() {
	for n := 0; n < 8; n++ {
		for k := 0; k < 8; k++ {
			cosTable[n][k] = float32(math.Cos(float64(2*n+1) * float64(k) * math.Pi / 16.0))
		}
	}
}

func idct1D(input, output []float32) {
	alpha := [8]float32{float32(1.0 / math.Sqrt(2.0)), 1, 1, 1, 1, 1, 1, 1}

	for n := 0; n < 8; n++ {
		var sum float32
		for k := 0; k < 8; k++ {
			sum += alpha[k] * input[k] * cosTable[n][k]
		}
		output[n] = 0.5 * sum
	}
}
