package jpeg

// ScanSpec defines the parameters for a single progressive scan.
type ScanSpec struct {
	Components []uint8 // Component IDs (1=Y, 2=Cb, 3=Cr)
	SS         uint8   // Spectral selection start (0-63)
	SE         uint8   // Spectral selection end (0-63)
	AH         uint8   // Successive approximation high (bit position)
	AL         uint8   // Successive approximation low (bit position)
}

// DefaultProgressiveScript returns a standard progressive scan script.
func DefaultProgressiveScript() []ScanSpec {
	return []ScanSpec{
		// 1. DC scans
		{Components: []uint8{1, 2, 3}, SS: 0, SE: 0, AH: 0, AL: 0},
		// 2. AC scans for Y (luminance)
		{Components: []uint8{1}, SS: 1, SE: 5, AH: 0, AL: 0},
		{Components: []uint8{1}, SS: 6, SE: 63, AH: 0, AL: 0},
		// 3. AC scans for Cb/Cr (chrominance)
		{Components: []uint8{2}, SS: 1, SE: 63, AH: 0, AL: 0},
		{Components: []uint8{3}, SS: 1, SE: 63, AH: 0, AL: 0},
	}
}

// SimpleProgressiveScript returns a faster but still progressive script.
func SimpleProgressiveScript() []ScanSpec {
	return []ScanSpec{
		{Components: []uint8{1, 2, 3}, SS: 0, SE: 0, AH: 0, AL: 0},
		{Components: []uint8{1}, SS: 1, SE: 63, AH: 0, AL: 0},
		{Components: []uint8{2}, SS: 1, SE: 63, AH: 0, AL: 0},
		{Components: []uint8{3}, SS: 1, SE: 63, AH: 0, AL: 0},
	}
}

// WebOptimizedProgressiveScript returns a script optimized for web delivery.
// Prioritizes fast first preview and good overall compression.
func WebOptimizedProgressiveScript() []ScanSpec {
	return []ScanSpec{
		// First scan: DC for all components (fastest preview)
		{Components: []uint8{1, 2, 3}, SS: 0, SE: 0, AH: 0, AL: 0},
		// Second scan: Low frequency AC (rough shapes/details)
		{Components: []uint8{1}, SS: 1, SE: 10, AH: 0, AL: 0},
		{Components: []uint8{2}, SS: 1, SE: 5, AH: 0, AL: 0},
		{Components: []uint8{3}, SS: 1, SE: 5, AH: 0, AL: 0},
		// Third scan: High frequency AC (fine details)
		{Components: []uint8{1}, SS: 11, SE: 63, AH: 0, AL: 0},
		{Components: []uint8{2}, SS: 6, SE: 63, AH: 0, AL: 0},
		{Components: []uint8{3}, SS: 6, SE: 63, AH: 0, AL: 0},
	}
}

// HighQualityProgressiveScript returns a script optimized for maximum quality.
// Uses spectral selection to better organize coefficients.
func HighQualityProgressiveScript() []ScanSpec {
	return []ScanSpec{
		// DC coefficients
		{Components: []uint8{1, 2, 3}, SS: 0, SE: 0, AH: 0, AL: 0},
		// Low frequency AC (coefficients 1-15)
		{Components: []uint8{1}, SS: 1, SE: 15, AH: 0, AL: 0},
		{Components: []uint8{2}, SS: 1, SE: 15, AH: 0, AL: 0},
		{Components: []uint8{3}, SS: 1, SE: 15, AH: 0, AL: 0},
		// Medium frequency AC (coefficients 16-31)
		{Components: []uint8{1}, SS: 16, SE: 31, AH: 0, AL: 0},
		{Components: []uint8{2}, SS: 16, SE: 31, AH: 0, AL: 0},
		{Components: []uint8{3}, SS: 16, SE: 31, AH: 0, AL: 0},
		// High frequency AC (coefficients 32-63)
		{Components: []uint8{1}, SS: 32, SE: 63, AH: 0, AL: 0},
		{Components: []uint8{2}, SS: 32, SE: 63, AH: 0, AL: 0},
		{Components: []uint8{3}, SS: 32, SE: 63, AH: 0, AL: 0},
	}
}

// BalancedProgressiveScript returns a balanced script for general use.
// Good compromise between encoding speed and compression efficiency.
func BalancedProgressiveScript() []ScanSpec {
	return []ScanSpec{
		// DC coefficients
		{Components: []uint8{1, 2, 3}, SS: 0, SE: 0, AH: 0, AL: 0},
		// Low frequency AC
		{Components: []uint8{1}, SS: 1, SE: 20, AH: 0, AL: 0},
		{Components: []uint8{2}, SS: 1, SE: 10, AH: 0, AL: 0},
		{Components: []uint8{3}, SS: 1, SE: 10, AH: 0, AL: 0},
		// High frequency AC
		{Components: []uint8{1}, SS: 21, SE: 63, AH: 0, AL: 0},
		{Components: []uint8{2}, SS: 11, SE: 63, AH: 0, AL: 0},
		{Components: []uint8{3}, SS: 11, SE: 63, AH: 0, AL: 0},
	}
}

// GetProgressiveScript returns the appropriate progressive scan script based on quality.
func GetProgressiveScript(quality uint8) []ScanSpec {
	if quality >= 90 {
		return HighQualityProgressiveScript()
	} else if quality >= 70 {
		return BalancedProgressiveScript()
	} else {
		return WebOptimizedProgressiveScript()
	}
}

// EncodeDCScan encodes DC coefficients for a scan.
func EncodeDCScan(bw *BitWriter, scan *ScanSpec, mcuCoeffs [][64]int16, compIdx int, ht *HuffmanTables, prevDC *int16, isLuminance bool) error {
	for _, coeffs := range mcuCoeffs {
		dc := coeffs[0]

		// In progressive mode, DC is encoded differently if AL/AH are used.
		// For simplicity in this implementation, we mostly use AL=0, AH=0 for first DC scan.
		if scan.AH == 0 {
			// First scan of DC
			cat, bits, bitLen := EncodeDC(dc, *prevDC)
			hCode, hLen := ht.EncodeDC(cat, isLuminance)
			if err := bw.Write(uint32(hCode), hLen); err != nil {
				return err
			}
			if bitLen > 0 {
				if err := bw.Write(uint32(bits), bitLen); err != nil {
					return err
				}
			}
			*prevDC = dc
		} else {
			// Refinement scan of DC (not implemented in scripts for now)
			bit := uint32((dc >> scan.AL) & 1)
			if err := bw.Write(bit, 1); err != nil {
				return err
			}
		}
	}
	return nil
}

// EncodeACFirstScan encodes AC coefficients for the first scan of a spectral range.
func EncodeACFirstScan(bw *BitWriter, scan *ScanSpec, mcuCoeffs [][64]int16, ht *HuffmanTables, isLuminance bool) error {
	for _, coeffs := range mcuCoeffs {
		lastNonZero := int(scan.SS) - 1
		for k := int(scan.SS); k <= int(scan.SE); k++ {
			if (coeffs[k] >> scan.AL) != 0 {
				lastNonZero = k
			}
		}

		if lastNonZero < int(scan.SS) {
			// Write EOB0
			hCode, hLen := ht.EncodeAC(0, 0, isLuminance)
			if err := bw.Write(uint32(hCode), hLen); err != nil {
				return err
			}
			continue
		}

		run := 0
		for k := int(scan.SS); k <= lastNonZero; k++ {
			val := coeffs[k] >> scan.AL
			if val == 0 {
				run++
				if run == 16 {
					// ZRL
					hCode, hLen := ht.EncodeAC(15, 0, isLuminance)
					if err := bw.Write(uint32(hCode), hLen); err != nil {
						return err
					}
					run = 0
				}
			} else {
				// Encode non-zero
				absVal := val
				if absVal < 0 {
					absVal = -absVal
				}
				size := uint8(0)
				for absVal > 0 {
					size++
					absVal >>= 1
				}

				hCode, hLen := ht.EncodeAC(uint8(run), size, isLuminance)
				if err := bw.Write(uint32(hCode), hLen); err != nil {
					return err
				}

				bits, bitLen := EncodeValue(int16(val))
				if err := bw.Write(uint32(bits), bitLen); err != nil {
					return err
				}
				run = 0
			}
		}

		// Write EOB0 if there are zeros remaining in the spectral range
		if lastNonZero < int(scan.SE) {
			hCode, hLen := ht.EncodeAC(0, 0, isLuminance)
			if err := bw.Write(uint32(hCode), hLen); err != nil {
				return err
			}
		}
	}

	return nil
}
