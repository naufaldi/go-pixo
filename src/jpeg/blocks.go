package jpeg

// ExtractBlock extracts an 8x8 block from the image data at the given block coordinates.
// It handles edge padding by replicating the last pixel.
// For RGB images, it converts to YCbCr and returns level-shifted values (-128 to 127).
// For Grayscale images, it extracts the gray value and returns level-shifted values.
func ExtractBlock(data []byte, width, height, blockX, blockY int, colorType ColorType) (yBlock, cbBlock, crBlock [64]float32) {
	for dy := 0; dy < 8; dy++ {
		for dx := 0; dx < 8; dx++ {
			x := blockX + dx
			y := blockY + dy

			// Handle edge padding
			if x >= width {
				x = width - 1
			}
			if y >= height {
				y = height - 1
			}

			idx := dy*8 + dx
			switch colorType {
			case ColorGrayscale:
				pixelIdx := y*width + x
				gray := data[pixelIdx]
				yBlock[idx] = float32(gray) - 128.0
				cbBlock[idx] = 0.0
				crBlock[idx] = 0.0
			case ColorRGB:
				pixelIdx := (y*width + x) * 3
				r := data[pixelIdx]
				g := data[pixelIdx+1]
				b := data[pixelIdx+2]
				yc, cb, cr := RGBToYCbCr(r, g, b)
				yBlock[idx] = float32(yc) - 128.0
				cbBlock[idx] = float32(cb) - 128.0
				crBlock[idx] = float32(cr) - 128.0
			}
		}
	}
	return yBlock, cbBlock, crBlock
}
