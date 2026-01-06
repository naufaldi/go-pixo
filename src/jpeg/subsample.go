package jpeg

// Subsample420 performs 4:2:0 chroma subsampling on Cb and Cr planes.
// It averages 2x2 blocks of chroma samples into a single sample.
// The input planes should be full resolution (width x height).
// The output planes will be (width+1)/2 x (height+1)/2.
func Subsample420(cb, cr []byte, width, height int) (subCb, subCr []byte) {
	newWidth := (width + 1) / 2
	newHeight := (height + 1) / 2
	subCb = make([]byte, newWidth*newHeight)
	subCr = make([]byte, newWidth*newHeight)

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			// Coordinates in original image
			x0 := x * 2
			y0 := y * 2
			x1 := x0 + 1
			y1 := y0 + 1

			// Handle edge cases
			if x1 >= width {
				x1 = width - 1
			}
			if y1 >= height {
				y1 = height - 1
			}

			// Average 2x2 block
			cb00 := float32(cb[y0*width+x0])
			cb01 := float32(cb[y0*width+x1])
			cb10 := float32(cb[y1*width+x0])
			cb11 := float32(cb[y1*width+x1])
			subCb[y*newWidth+x] = byte((cb00 + cb01 + cb10 + cb11) / 4.0)

			cr00 := float32(cr[y0*width+x0])
			cr01 := float32(cr[y0*width+x1])
			cr10 := float32(cr[y1*width+x0])
			cr11 := float32(cr[y1*width+x1])
			subCr[y*newWidth+x] = byte((cr00 + cr01 + cr10 + cr11) / 4.0)
		}
	}

	return subCb, subCr
}
