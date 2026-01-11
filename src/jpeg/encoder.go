package jpeg

import (
	"bytes"
)

// Encoder represents a JPEG encoder.
type Encoder struct {
	Options       Options
	trellisConfig TrellisConfig
	dctSIMD       *DCTSIMD
	cacheStats    *CacheStats
}

// CacheStats tracks cache hit/miss statistics for Huffman tables.
type CacheStats struct {
	Hits   int64
	Misses int64
}

// NewEncoder creates a new JPEG encoder with the given options.
func NewEncoder(opts Options) (*Encoder, error) {
	if opts.Width <= 0 || opts.Height <= 0 || opts.Width > 65535 || opts.Height > 65535 {
		return nil, ErrInvalidDimensions
	}
	if opts.ColorType != ColorGrayscale && opts.ColorType != ColorRGB {
		return nil, ErrUnsupportedColorType
	}
	if opts.Quality < 1 || opts.Quality > 100 {
		return nil, ErrInvalidQuality
	}

	var dct *DCTSIMD
	if opts.UseSIMD {
		dct = DetectCPUFeatures()
	}

	trellisConfig := TrellisConfig{
		Lambda:        opts.TrellisLambda,
		MaxIterations: 64,
		UsePerceptual: opts.TrellisPerceptual,
	}

	return &Encoder{
		Options:       opts,
		trellisConfig: trellisConfig,
		dctSIMD:       dct,
		cacheStats:    &CacheStats{},
	}, nil
}

// Encode encodes raw pixel data as a JPEG file.
func (e *Encoder) Encode(pixels []byte) ([]byte, error) {
	// Validate pixel data length
	bpp := 1
	if e.Options.ColorType == ColorRGB {
		bpp = 3
	}
	if len(pixels) != e.Options.Width*e.Options.Height*bpp {
		return nil, ErrInvalidDataLength
	}

	buf := new(bytes.Buffer)

	// Write SOI
	if err := WriteSOI(buf); err != nil {
		return nil, err
	}

	// Write APP0
	if err := WriteAPP0(buf); err != nil {
		return nil, err
	}

	// Create quantization tables
	qt := NewQuantizationTables(e.Options.Quality)

	// Write DQT for Luminance
	if err := WriteDQT(buf, 0, qt.LuminanceTable); err != nil {
		return nil, err
	}

	// Write DQT for Chrominance (only for RGB)
	if e.Options.ColorType == ColorRGB {
		if err := WriteDQT(buf, 1, qt.ChrominanceTable); err != nil {
			return nil, err
		}
	}

	// Write SOF
	if e.Options.Progressive {
		if err := WriteSOF2(buf, uint16(e.Options.Width), uint16(e.Options.Height), e.Options.ColorType, e.Options.Subsampling); err != nil {
			return nil, err
		}
	} else {
		if err := WriteSOF0(buf, uint16(e.Options.Width), uint16(e.Options.Height), e.Options.ColorType, e.Options.Subsampling); err != nil {
			return nil, err
		}
	}

	// Create Huffman tables
	// Note: When trellis quantization is enabled, we use standard Huffman tables
	// because the optimized tables are built based on standard quantization,
	// but trellis produces different coefficients that would mismatch.
	var ht *HuffmanTables
	if e.Options.OptimizeHuffman && !e.Options.TrellisQuant {
		ht = BuildOptimizedTables(pixels, e.Options.Width, e.Options.Height, e.Options.ColorType, e.Options.Subsampling, qt)
	} else {
		ht = GetStandardHuffmanTables(e.Options.Quality, e.Options.Subsampling)
	}

	// Update cache stats
	if e.cacheStats != nil {
		hits, misses := huffmanCache.Stats()
		e.cacheStats.Hits = hits
		e.cacheStats.Misses = misses
	}

	// Write DHT segments
	if err := e.writeDHTSegments(buf, ht); err != nil {
		return nil, err
	}

	if e.Options.Progressive {
		return e.encodeProgressive(buf, pixels, ht, qt)
	}
	return e.encodeBaseline(buf, pixels, ht, qt)
}

func (e *Encoder) writeDHTSegments(buf *bytes.Buffer, ht *HuffmanTables) error {
	// DC Luminance
	if err := WriteDHT(buf, 0x00, ht.DCLumBits, ht.DCLumVals); err != nil {
		return err
	}
	// AC Luminance
	if err := WriteDHT(buf, 0x10, ht.ACLumBits, ht.ACLumVals); err != nil {
		return err
	}

	if e.Options.ColorType == ColorRGB {
		// DC Chrominance
		if err := WriteDHT(buf, 0x01, ht.DCChromBits, ht.DCChromVals); err != nil {
			return err
		}
		// AC Chrominance
		if err := WriteDHT(buf, 0x11, ht.ACChromBits, ht.ACChromVals); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeBaseline(buf *bytes.Buffer, pixels []byte, ht *HuffmanTables, qt *QuantizationTables) ([]byte, error) {
	bw := NewBitWriter(buf)

	// Write SOS
	if err := WriteSOS(buf, e.Options.ColorType); err != nil {
		return nil, err
	}

	// Encoding Loop
	prevDCY := int16(0)
	prevDCCb := int16(0)
	prevDCCr := int16(0)

	if e.Options.Subsampling == Subsampling420 && e.Options.ColorType == ColorRGB {
		// 4:2:0 encoding loop (16x16 MCUs)
		for y := 0; y < e.Options.Height; y += 16 {
			for x := 0; x < e.Options.Width; x += 16 {
				yBlocks, cbBlock, crBlock := ExtractMCU420(pixels, e.Options.Width, e.Options.Height, x, y)

				// Encode 4 Y blocks
				for i := 0; i < 4; i++ {
					var err error
					prevDCY, err = e.encodeComponentBlock(bw, yBlocks[i], prevDCY, true, ht, qt)
					if err != nil {
						return nil, err
					}
				}

				// Encode Cb block
				var err error
				prevDCCb, err = e.encodeComponentBlock(bw, cbBlock, prevDCCb, false, ht, qt)
				if err != nil {
					return nil, err
				}

				// Encode Cr block
				prevDCCr, err = e.encodeComponentBlock(bw, crBlock, prevDCCr, false, ht, qt)
				if err != nil {
					return nil, err
				}
			}
		}
	} else {
		// 4:4:4 or Grayscale encoding loop (8x8 blocks)
		for y := 0; y < e.Options.Height; y += 8 {
			for x := 0; x < e.Options.Width; x += 8 {
				// Extract and process blocks
				yBlock, cbBlock, crBlock := ExtractBlock(pixels, e.Options.Width, e.Options.Height, x, y, e.Options.ColorType)

				// Process Y channel
				var err error
				prevDCY, err = e.encodeComponentBlock(bw, yBlock, prevDCY, true, ht, qt)
				if err != nil {
					return nil, err
				}

				if e.Options.ColorType == ColorRGB {
					// Process Cb channel
					prevDCCb, err = e.encodeComponentBlock(bw, cbBlock, prevDCCb, false, ht, qt)
					if err != nil {
						return nil, err
					}
					// Process Cr channel
					prevDCCr, err = e.encodeComponentBlock(bw, crBlock, prevDCCr, false, ht, qt)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	// Flush remaining bits
	if err := bw.Flush(); err != nil {
		return nil, err
	}

	// Write EOI
	if err := WriteEOI(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (e *Encoder) encodeProgressive(buf *bytes.Buffer, pixels []byte, ht *HuffmanTables, qt *QuantizationTables) ([]byte, error) {
	// 1. Collect all quantized DCT coefficients
	var yCoeffs [][64]int16
	var cbCoeffs [][64]int16
	var crCoeffs [][64]int16

	if e.Options.Subsampling == Subsampling420 && e.Options.ColorType == ColorRGB {
		for y := 0; y < e.Options.Height; y += 16 {
			for x := 0; x < e.Options.Width; x += 16 {
				yBlocks, cbBlock, crBlock := ExtractMCU420(pixels, e.Options.Width, e.Options.Height, x, y)
				for i := 0; i < 4; i++ {
					yCoeffs = append(yCoeffs, e.quantizeBlock(yBlocks[i], true, qt))
				}
				cbCoeffs = append(cbCoeffs, e.quantizeBlock(cbBlock, false, qt))
				crCoeffs = append(crCoeffs, e.quantizeBlock(crBlock, false, qt))
			}
		}
	} else {
		for y := 0; y < e.Options.Height; y += 8 {
			for x := 0; x < e.Options.Width; x += 8 {
				yBlock, cbBlock, crBlock := ExtractBlock(pixels, e.Options.Width, e.Options.Height, x, y, e.Options.ColorType)
				yCoeffs = append(yCoeffs, e.quantizeBlock(yBlock, true, qt))
				if e.Options.ColorType == ColorRGB {
					cbCoeffs = append(cbCoeffs, e.quantizeBlock(cbBlock, false, qt))
					crCoeffs = append(crCoeffs, e.quantizeBlock(crBlock, false, qt))
				}
			}
		}
	}

	// 2. Perform multiple scans
	script := GetProgressiveScript(e.Options.Quality)
	prevDC := make([]int16, 4) // Indices 1, 2, 3 for components

	for _, scan := range script {
		// Filter scan components to only include those present in the image
		activeComponents := []uint8{}
		for _, compID := range scan.Components {
			if e.Options.ColorType == ColorGrayscale && compID > 1 {
				continue
			}
			activeComponents = append(activeComponents, compID)
		}

		if len(activeComponents) == 0 {
			continue
		}

		// Update scan with active components for this image
		actualScan := scan
		actualScan.Components = activeComponents

		if err := WriteSOSProgressive(buf, &actualScan, e.Options.ColorType); err != nil {
			return nil, err
		}
		bw := NewBitWriter(buf)

		if actualScan.SS == 0 {
			// DC scan
			// Note: This logic needs to match the interleaving of components in the scan

			// We need to iterate by MCUs
			numMCUs := 0
			if e.Options.Subsampling == Subsampling420 && e.Options.ColorType == ColorRGB {
				numMCUs = len(cbCoeffs) // 1 MCU = 4Y + 1Cb + 1Cr
				for i := 0; i < numMCUs; i++ {
					for _, compID := range actualScan.Components {
						switch compID {
						case 1:
							// Y1, Y2, Y3, Y4
							for j := 0; j < 4; j++ {
								if err := EncodeDCScan(bw, &actualScan, [][64]int16{yCoeffs[i*4+j]}, 0, ht, &prevDC[1], true); err != nil {
									return nil, err
								}
							}
						case 2:
							// Cb
							if err := EncodeDCScan(bw, &actualScan, [][64]int16{cbCoeffs[i]}, 0, ht, &prevDC[2], false); err != nil {
								return nil, err
							}
						case 3:
							// Cr
							if err := EncodeDCScan(bw, &actualScan, [][64]int16{crCoeffs[i]}, 0, ht, &prevDC[3], false); err != nil {
								return nil, err
							}
						}
					}
				}
			} else {
				numMCUs = len(yCoeffs)
				for i := 0; i < numMCUs; i++ {
					for _, compID := range actualScan.Components {
						switch compID {
						case 1:
							if err := EncodeDCScan(bw, &actualScan, [][64]int16{yCoeffs[i]}, 0, ht, &prevDC[1], true); err != nil {
								return nil, err
							}
						case 2:
							if err := EncodeDCScan(bw, &actualScan, [][64]int16{cbCoeffs[i]}, 0, ht, &prevDC[2], false); err != nil {
								return nil, err
							}
						case 3:
							if err := EncodeDCScan(bw, &actualScan, [][64]int16{crCoeffs[i]}, 0, ht, &prevDC[3], false); err != nil {
								return nil, err
							}
						}
					}
				}
			}
		} else {
			// AC scan (must be only ONE component per scan)
			compID := actualScan.Components[0]
			if e.Options.Subsampling == Subsampling420 && e.Options.ColorType == ColorRGB {
				numMCUs := len(cbCoeffs)
				for i := 0; i < numMCUs; i++ {
					switch compID {
					case 1:
						for j := 0; j < 4; j++ {
							if err := EncodeACFirstScan(bw, &actualScan, [][64]int16{yCoeffs[i*4+j]}, ht, true); err != nil {
								return nil, err
							}
						}
					case 2:
						if err := EncodeACFirstScan(bw, &actualScan, [][64]int16{cbCoeffs[i]}, ht, false); err != nil {
							return nil, err
						}
					case 3:
						if err := EncodeACFirstScan(bw, &actualScan, [][64]int16{crCoeffs[i]}, ht, false); err != nil {
							return nil, err
						}
					}
				}
			} else {
				numMCUs := len(yCoeffs)
				for i := 0; i < numMCUs; i++ {
					switch compID {
					case 1:
						if err := EncodeACFirstScan(bw, &actualScan, [][64]int16{yCoeffs[i]}, ht, true); err != nil {
							return nil, err
						}
					case 2:
						if err := EncodeACFirstScan(bw, &actualScan, [][64]int16{cbCoeffs[i]}, ht, false); err != nil {
							return nil, err
						}
					case 3:
						if err := EncodeACFirstScan(bw, &actualScan, [][64]int16{crCoeffs[i]}, ht, false); err != nil {
							return nil, err
						}
					}
				}
			}
		}

		if err := bw.Flush(); err != nil {
			return nil, err
		}
	}

	// 3. Write EOI
	if err := WriteEOI(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (e *Encoder) quantizeBlock(block [64]float32, isLuminance bool, qt *QuantizationTables) [64]int16 {
	var dct [64]float32
	if e.dctSIMD != nil && e.dctSIMD.PreferredMethod != "scalar" {
		dct = ForwardDCTSIMD(block)
	} else {
		dct = ForwardDCT(block)
	}
	var qTable [64]float32
	if isLuminance {
		qTable = qt.Luminance
	} else {
		qTable = qt.Chrominance
	}

	if e.Options.TrellisQuant {
		quantized := TrellisOptimizeWithConfig(dct, qTable, e.trellisConfig)
		return ZigzagReorder(quantized)
	}

	quantized := QuantizeBlock(dct, qTable)
	return ZigzagReorder(quantized)
}

// encodeComponentBlock processes a single 8x8 float block (DCT, Quantize, Zigzag, Huffman).
// It returns the new DC value and any error encountered.
func (e *Encoder) encodeComponentBlock(bw *BitWriter, block [64]float32, prevDC int16, isLuminance bool, ht *HuffmanTables, qt *QuantizationTables) (int16, error) {
	// 1. Quantize and Zigzag
	zigzag := e.quantizeBlock(block, isLuminance, qt)

	// 2. Huffman Encode DC
	dc := zigzag[0]
	cat, bits, bitLen := EncodeDC(dc, prevDC)
	hCode, hLen := ht.EncodeDC(cat, isLuminance)
	if err := bw.Write(uint32(hCode), hLen); err != nil {
		return 0, err
	}
	if bitLen > 0 {
		if err := bw.Write(uint32(bits), bitLen); err != nil {
			return 0, err
		}
	}

	// 3. Huffman Encode AC
	acRuns := RunLengthEncode(zigzag)
	for _, run := range acRuns {
		hCode, hLen := ht.EncodeAC(run.RunLength, run.Size, isLuminance)
		if err := bw.Write(uint32(hCode), hLen); err != nil {
			return 0, err
		}
		if run.Size > 0 {
			bits, bitLen := EncodeValue(run.Value)
			if err := bw.Write(uint32(bits), bitLen); err != nil {
				return 0, err
			}
		}
	}

	return dc, nil
}

// GetCacheStats returns the current cache statistics for this encoder.
func (e *Encoder) GetCacheStats() CacheStats {
	if e.cacheStats == nil {
		return CacheStats{}
	}
	return *e.cacheStats
}

// GetGlobalCacheHitRate returns the global cache hit rate.
func (e *Encoder) GetGlobalCacheHitRate() float64 {
	return huffmanCache.HitRate()
}

// GetGlobalCacheStats returns the global cache statistics.
func (e *Encoder) GetGlobalCacheStats() (hits, misses int64) {
	return huffmanCache.Stats()
}
