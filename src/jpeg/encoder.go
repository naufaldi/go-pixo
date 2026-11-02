// Package jpeg implements a baseline and progressive JPEG encoder with optional Huffman optimization and trellis quantization.
package jpeg

import (
	"bytes"

	"github.com/mac/go-pixo/src/internal/imagelimit"
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
	if err := validateOptions(opts); err != nil {
		return nil, err
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

func validateOptions(opts Options) error {
	if opts.Width <= 0 || opts.Height <= 0 || opts.Width > 65535 || opts.Height > 65535 {
		return ErrInvalidDimensions
	}
	if err := imagelimit.CheckDimensions(opts.Width, opts.Height); err != nil {
		return err
	}
	if opts.ColorType != ColorGrayscale && opts.ColorType != ColorRGB {
		return ErrUnsupportedColorType
	}
	if opts.Quality < 1 || opts.Quality > 100 {
		return ErrInvalidQuality
	}
	return nil
}

// Encode encodes raw pixel data as a JPEG file.
func (e *Encoder) Encode(pixels []byte) ([]byte, error) {
	// Options is public and may have changed since construction.
	if err := validateOptions(e.Options); err != nil {
		return nil, err
	}
	// Validate pixel data length
	bpp := 1
	if e.Options.ColorType == ColorRGB {
		bpp = 3
	}
	if len(pixels) != e.Options.Width*e.Options.Height*bpp {
		return nil, ErrInvalidDataLength
	}
	if !e.Options.Progressive && !e.Options.OptimizeHuffman {
		return e.encodeSequential(pixels)
	}
	// The pixel-domain progressive and optimized-table writers emitted
	// entropy-coded data that decoders reconstruct wrongly (PSNR ~10-20 dB).
	// Encode with standard tables, then re-code the same quantized
	// coefficients without changing pixels: Progressive yields a progressive
	// file and OptimizeHuffman alone a sequential one, each with optimized
	// tables and the smallest layout of that kind.
	baseline, err := e.encodeSequential(pixels)
	if err != nil {
		return nil, err
	}
	layout := sequentialLayouts
	if e.Options.Progressive {
		layout = progressiveLayouts
	}
	return optimizeLossless(baseline, layout)
}

// encodeSequential writes a baseline JPEG with standard Huffman tables.
func (e *Encoder) encodeSequential(pixels []byte) ([]byte, error) {
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

	if err := WriteSOF0(buf, uint16(e.Options.Width), uint16(e.Options.Height), e.Options.ColorType, e.Options.Subsampling); err != nil {
		return nil, err
	}

	ht := GetStandardHuffmanTables(e.Options.Quality, e.Options.Subsampling)

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

func (e *Encoder) quantizeBlock(block [64]float32, isLuminance bool, qt *QuantizationTables) [64]int16 {
	var dct [64]float32
	if e.dctSIMD != nil && e.dctSIMD.PreferredMethod != dctMethodScalar {
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
		quantized := trellisOptimize(dct, qTable, e.trellisConfig, isLuminance)
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
