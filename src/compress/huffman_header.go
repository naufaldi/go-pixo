package compress

// CodeLengthOrder is the order in which code length codes are stored in DEFLATE dynamic headers.
var CodeLengthOrder = [19]int{
	16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15,
}

// WriteHLIT writes the HLIT field (5 bits): number of literal/length codes - 257.
func WriteHLIT(w *BitWriter, n int) error {
	if n < 257 || n > 286 {
		return ErrInvalidHLIT
	}
	value := uint16(n - 257)
	return w.Write(value, 5)
}

// WriteHDIST writes the HDIST field (5 bits): number of distance codes - 1.
func WriteHDIST(w *BitWriter, n int) error {
	if n < 1 || n > 30 {
		return ErrInvalidHDIST
	}
	value := uint16(n - 1)
	return w.Write(value, 5)
}

// WriteHCLEN writes the HCLEN field (4 bits): number of code length codes - 4.
func WriteHCLEN(w *BitWriter, n int) error {
	if n < 4 || n > 19 {
		return ErrInvalidHCLEN
	}
	value := uint16(n - 4)
	return w.Write(value, 4)
}

// WriteDynamicHeader writes a complete dynamic Huffman block header to the bit writer.
// It includes HLIT, HDIST, HCLEN, code length code lengths, and RLE-encoded literal/length/distance code lengths.
func WriteDynamicHeader(w *BitWriter, litLengths []int, distLengths []int) error {
	litCount := findLastNonZero(litLengths, 286)
	distCount := findLastNonZero(distLengths, 30)
	
	if err := WriteHLIT(w, litCount+1); err != nil {
		return err
	}
	if err := WriteHDIST(w, distCount+1); err != nil {
		return err
	}
	
	codeLengthLengths := buildCodeLengthLengths(litLengths, distLengths)
	clenCount := findLastNonZeroCodeLength(codeLengthLengths)
	
	if err := WriteHCLEN(w, clenCount+1); err != nil {
		return err
	}
	
	for i := 0; i <= clenCount; i++ {
		code := CodeLengthOrder[i]
		length := codeLengthLengths[code]
		if err := w.Write(uint16(length), 3); err != nil {
			return err
		}
	}
	
	codeLengthTable := buildCodeLengthTable(codeLengthLengths)
	
	if err := writeRLECodeLengths(w, litLengths[:litCount+1], codeLengthTable); err != nil {
		return err
	}
	
	if err := writeRLECodeLengths(w, distLengths[:distCount+1], codeLengthTable); err != nil {
		return err
	}
	
	return nil
}

// findLastNonZero finds the last non-zero index in lengths, up to max.
func findLastNonZero(lengths []int, max int) int {
	last := 0
	for i := 0; i < len(lengths) && i <= max; i++ {
		if lengths[i] > 0 {
			last = i
		}
	}
	return last
}

// findLastNonZeroCodeLength finds the last non-zero code length index in CodeLengthOrder.
func findLastNonZeroCodeLength(codeLengthLengths []int) int {
	last := 0
	for i := 0; i < len(CodeLengthOrder); i++ {
		code := CodeLengthOrder[i]
		if codeLengthLengths[code] > 0 {
			last = i
		}
	}
	return last
}

// buildCodeLengthLengths builds the code length code lengths from literal/length and distance code lengths.
func buildCodeLengthLengths(litLengths []int, distLengths []int) []int {
	allLengths := append(litLengths, distLengths...)
	freq := make([]int, 19)
	
	for _, length := range allLengths {
		if length < 16 {
			freq[length]++
		} else if length == 16 {
			freq[16]++
		} else if length == 17 {
			freq[17]++
		} else if length == 18 {
			freq[18]++
		}
	}
	
	tree := BuildTree(freq)
	if tree == nil {
		return make([]int, 19)
	}
	
	codesMap := GenerateCodes(tree)
	codes, lengths := Canonicalize(codesMap)
	
	result := make([]int, 19)
	for i := 0; i < 19; i++ {
		if i < len(lengths) && lengths[i] > 0 {
			result[i] = lengths[i]
		}
	}
	
	_ = codes
	return result
}

// buildCodeLengthTable builds a Huffman table for code length codes.
func buildCodeLengthTable(codeLengthLengths []int) Table {
	freq := make([]int, 19)
	for i, length := range codeLengthLengths {
		if length > 0 {
			freq[i] = 1
		}
	}
	
	tree := BuildTree(freq)
	if tree == nil {
		return Table{Codes: make([]Code, 19), MaxLength: 0}
	}
	
	codesMap := GenerateCodes(tree)
	codes, lengths := Canonicalize(codesMap)
	
	maxLength := 0
	for _, length := range lengths {
		if length > maxLength {
			maxLength = length
		}
	}
	
	return Table{Codes: codes, MaxLength: maxLength}
}

// writeRLECodeLengths writes code lengths using RLE encoding (symbols 16, 17, 18).
func writeRLECodeLengths(w *BitWriter, lengths []int, codeLengthTable Table) error {
	i := 0
	for i < len(lengths) {
		length := lengths[i]
		
		if length == 0 {
			zeroRun := 0
			start := i
			for i < len(lengths) && lengths[i] == 0 {
				zeroRun++
				i++
			}
			
			if zeroRun >= 3 {
				if zeroRun <= 10 {
					if err := EncodeLiteral(w, 17, codeLengthTable); err != nil {
						return err
					}
					if err := w.Write(uint16(zeroRun-3), 3); err != nil {
						return err
					}
				} else {
					if err := EncodeLiteral(w, 18, codeLengthTable); err != nil {
						return err
					}
					if err := w.Write(uint16(zeroRun-11), 7); err != nil {
						return err
					}
				}
			} else {
				for j := start; j < start+zeroRun; j++ {
					if err := EncodeLiteral(w, 0, codeLengthTable); err != nil {
						return err
					}
				}
			}
		} else {
			repeat := 1
			for i+repeat < len(lengths) && lengths[i+repeat] == length && repeat < 6 {
				repeat++
			}
			
			if repeat >= 3 {
				if err := EncodeLiteral(w, length, codeLengthTable); err != nil {
					return err
				}
				if err := EncodeLiteral(w, 16, codeLengthTable); err != nil {
					return err
				}
				if err := w.Write(uint16(repeat-3), 2); err != nil {
					return err
				}
				i += repeat
			} else {
				if err := EncodeLiteral(w, length, codeLengthTable); err != nil {
					return err
				}
				i++
			}
		}
	}
	
	return nil
}
