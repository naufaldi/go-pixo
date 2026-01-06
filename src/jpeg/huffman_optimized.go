package jpeg

import (
	"sort"

	"github.com/mac/go-pixo/src/compress"
)

// BuildOptimizedTables generates custom Huffman tables based on the actual image data.
func BuildOptimizedTables(pixels []byte, width, height int, colorType ColorType, subsampling Subsampling, qt *QuantizationTables) *HuffmanTables {
	// 1. Count frequencies
	dcLumFreq := make([]int, 12)
	dcChromFreq := make([]int, 12)
	acLumFreq := make([]int, 256)
	acChromFreq := make([]int, 256)

	prevDCY := int16(0)
	prevDCCb := int16(0)
	prevDCCr := int16(0)

	if subsampling == Subsampling420 && colorType == ColorRGB {
		for y := 0; y < height; y += 16 {
			for x := 0; x < width; x += 16 {
				yBlocks, cbBlock, crBlock := ExtractMCU420(pixels, width, height, x, y)
				for i := 0; i < 4; i++ {
					prevDCY = countBlockFreqs(yBlocks[i], prevDCY, dcLumFreq, acLumFreq, qt.Luminance)
				}
				prevDCCb = countBlockFreqs(cbBlock, prevDCCb, dcChromFreq, acChromFreq, qt.Chrominance)
				prevDCCr = countBlockFreqs(crBlock, prevDCCr, dcChromFreq, acChromFreq, qt.Chrominance)
			}
		}
	} else {
		for y := 0; y < height; y += 8 {
			for x := 0; x < width; x += 8 {
				yBlock, cbBlock, crBlock := ExtractBlock(pixels, width, height, x, y, colorType)
				prevDCY = countBlockFreqs(yBlock, prevDCY, dcLumFreq, acLumFreq, qt.Luminance)
				if colorType == ColorRGB {
					prevDCCb = countBlockFreqs(cbBlock, prevDCCb, dcChromFreq, acChromFreq, qt.Chrominance)
					prevDCCr = countBlockFreqs(crBlock, prevDCCr, dcChromFreq, acChromFreq, qt.Chrominance)
				}
			}
		}
	}

	// 2. Build tables from frequencies
	ht := &HuffmanTables{}
	ht.DCLumBits, ht.DCLumVals = buildJpegHuffmanTable(dcLumFreq, 12)
	ht.ACLumBits, ht.ACLumVals = buildJpegHuffmanTable(acLumFreq, 256)

	if colorType == ColorRGB {
		ht.DCChromBits, ht.DCChromVals = buildJpegHuffmanTable(dcChromFreq, 12)
		ht.ACChromBits, ht.ACChromVals = buildJpegHuffmanTable(acChromFreq, 256)
	}

	// 3. Build lookup tables for encoding
	ht.DCLumCodes = buildCodeTable12(ht.DCLumBits, ht.DCLumVals)
	ht.ACLumCodes = buildCodeTable256(ht.ACLumBits, ht.ACLumVals)
	if colorType == ColorRGB {
		ht.DCChromCodes = buildCodeTable12(ht.DCChromBits, ht.DCChromVals)
		ht.ACChromCodes = buildCodeTable256(ht.ACChromBits, ht.ACChromVals)
	}

	return ht
}

func countBlockFreqs(block [64]float32, prevDC int16, dcFreq []int, acFreq []int, qTable [64]float32) int16 {
	dct := ForwardDCT(block)
	quantized := QuantizeBlock(dct, qTable)
	zigzag := ZigzagReorder(quantized)

	// DC
	dc := zigzag[0]
	diff := dc - prevDC
	if diff < 0 {
		diff = -diff
	}
	category := uint8(0)
	for diff > 0 {
		category++
		diff >>= 1
	}
	dcFreq[category]++

	// AC
	acRuns := RunLengthEncode(zigzag)
	for _, run := range acRuns {
		symbol := (run.RunLength << 4) | run.Size
		acFreq[symbol]++
	}

	return dc
}

func buildJpegHuffmanTable(frequencies []int, numSymbols int) (bits [16]uint8, vals []uint8) {
	// Check how many symbols have non-zero frequency
	nonZeroSymbols := 0
	firstSymbol := -1
	for i, f := range frequencies {
		if f > 0 {
			nonZeroSymbols++
			if firstSymbol == -1 {
				firstSymbol = i
			}
		}
	}

	if nonZeroSymbols == 0 {
		// Just provide a dummy table with one symbol if empty
		bits[0] = 1
		vals = []uint8{0}
		return
	}

	if nonZeroSymbols == 1 {
		// Single symbol case: JPEG requires at least 1 bit
		bits[0] = 1
		vals = []uint8{uint8(firstSymbol)}
		return
	}

	// Build tree using compress package
	tree := compress.BuildTree(frequencies)
	if tree == nil {
		return
	}

	// Generate code lengths
	codesMap := compress.GenerateCodes(tree)

	// Convert to symbols sorted by length
	type symLen struct {
		sym int
		len int
	}
	var sorted []symLen
	for sym, code := range codesMap {
		if code.Length > 0 {
			l := code.Length
			if l > 16 {
				l = 16 // Simple clamping for now
			}
			sorted = append(sorted, symLen{sym: sym, len: l})
		}
	}

	// Sort by length, then by symbol value
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].len != sorted[j].len {
			return sorted[i].len < sorted[j].len
		}
		return sorted[i].sym < sorted[j].sym
	})

	// Fill bits and vals
	for _, sl := range sorted {
		if sl.len <= 16 {
			bits[sl.len-1]++
			vals = append(vals, uint8(sl.sym))
		}
	}

	return bits, vals
}
