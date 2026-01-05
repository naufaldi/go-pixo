package jpeg

// ExtractMCU420 extracts a 16x16 Minimum Coded Unit (MCU) from the image data
// and performs 4:2:0 chroma subsampling.
// It returns 4 Y blocks (8x8 each), one Cb block (8x8), and one Cr block (8x8).
// Values are level-shifted to -128..127.
func ExtractMCU420(data []byte, width, height, mcuX, mcuY int) (yBlocks [4][64]float32, cbBlock, crBlock [64]float32) {
	// cbAccum and crAccum are used to average chroma over 2x2 pixels
	var cbAccum, crAccum [64]float32

	for by := 0; by < 2; by++ {
		for bx := 0; bx < 2; bx++ {
			yBlockIdx := by*2 + bx
			for dy := 0; dy < 8; dy++ {
				for dx := 0; dx < 8; dx++ {
					x := mcuX + bx*8 + dx
					y := mcuY + by*8 + dy

					// Handle edge padding
					if x >= width {
						x = width - 1
					}
					if y >= height {
						y = height - 1
					}

					pixelIdx := (y*width + x) * 3
					r := data[pixelIdx]
					g := data[pixelIdx+1]
					b := data[pixelIdx+2]

					yc, cb, cr := RGBToYCbCr(r, g, b)

					yIdx := dy*8 + dx
					yBlocks[yBlockIdx][yIdx] = float32(yc) - 128.0

					// Calculate global position within the 16x16 MCU
					mcuInternalX := bx*8 + dx
					mcuInternalY := by*8 + dy

					// Chroma is subsampled 2:1 in each dimension
					cbCrX := mcuInternalX / 2
					cbCrY := mcuInternalY / 2
					cbCrIdx := cbCrY*8 + cbCrX

					cbAccum[cbCrIdx] += float32(cb)
					crAccum[cbCrIdx] += float32(cr)
				}
			}
		}
	}

	// Average chroma over 2x2 regions and level-shift
	for i := 0; i < 64; i++ {
		cbBlock[i] = (cbAccum[i] * 0.25) - 128.0
		crBlock[i] = (crAccum[i] * 0.25) - 128.0
	}

	return yBlocks, cbBlock, crBlock
}
