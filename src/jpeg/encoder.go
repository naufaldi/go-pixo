package jpeg

import (
	"bytes"
)

// Encoder represents a JPEG encoder.
type Encoder struct {
	Width       int
	Height      int
	ColorType   ColorType
	Quality     uint8
	Subsampling Subsampling
}

// NewEncoder creates a new JPEG encoder.
func NewEncoder(width, height int, colorType ColorType, quality uint8) (*Encoder, error) {
	if width <= 0 || height <= 0 || width > 65535 || height > 65535 {
		return nil, ErrInvalidDimensions
	}
	if colorType != ColorGrayscale && colorType != ColorRGB {
		return nil, ErrUnsupportedColorType
	}
	if quality < 1 || quality > 100 {
		return nil, ErrInvalidQuality
	}

	return &Encoder{
		Width:       width,
		Height:      height,
		ColorType:   colorType,
		Quality:     quality,
		Subsampling: Subsampling444, // Default for baseline
	}, nil
}

// Encode encodes raw pixel data as a JPEG file.
func (e *Encoder) Encode(pixels []byte) ([]byte, error) {
	// Validate pixel data length
	bpp := 1
	if e.ColorType == ColorRGB {
		bpp = 3
	}
	if len(pixels) != e.Width*e.Height*bpp {
		return nil, ErrInvalidDataLength
	}

	buf := new(bytes.Buffer)
	bw := NewBitWriter(buf)

	// Write SOI
	if err := WriteSOI(buf); err != nil {
		return nil, err
	}

	// Write APP0
	if err := WriteAPP0(buf); err != nil {
		return nil, err
	}

	// Create quantization tables
	qt := NewQuantizationTables(e.Quality)

	// Write DQT for Luminance
	if err := WriteDQT(buf, 0, qt.LuminanceTable); err != nil {
		return nil, err
	}

	// Write DQT for Chrominance (only for RGB)
	if e.ColorType == ColorRGB {
		if err := WriteDQT(buf, 1, qt.ChrominanceTable); err != nil {
			return nil, err
		}
	}

	// Write SOF0
	if err := WriteSOF0(buf, uint16(e.Width), uint16(e.Height), e.ColorType, e.Subsampling); err != nil {
		return nil, err
	}

	// Create Huffman tables
	ht := NewHuffmanTables()

	// Write DHT segments
	// DC Luminance
	if err := WriteDHT(buf, 0x00, ht.DCLumBits, ht.DCLumVals); err != nil {
		return nil, err
	}
	// AC Luminance
	if err := WriteDHT(buf, 0x10, ht.ACLumBits, ht.ACLumVals); err != nil {
		return nil, err
	}

	if e.ColorType == ColorRGB {
		// DC Chrominance
		if err := WriteDHT(buf, 0x01, ht.DCChromBits, ht.DCChromVals); err != nil {
			return nil, err
		}
		// AC Chrominance
		if err := WriteDHT(buf, 0x11, ht.ACChromBits, ht.ACChromVals); err != nil {
			return nil, err
		}
	}

	// Write SOS
	if err := WriteSOS(buf, e.ColorType); err != nil {
		return nil, err
	}

	// Encoding Loop
	prevDCY := int16(0)
	prevDCCb := int16(0)
	prevDCCr := int16(0)

	for y := 0; y < e.Height; y += 8 {
		for x := 0; x < e.Width; x += 8 {
			// Extract and process blocks
			yBlock, cbBlock, crBlock := ExtractBlock(pixels, e.Width, e.Height, x, y, e.ColorType)

			// Process Y channel
			prevDCY = e.encodeComponentBlock(bw, yBlock, prevDCY, true, ht, qt)

			if e.ColorType == ColorRGB {
				// Process Cb channel
				prevDCCb = e.encodeComponentBlock(bw, cbBlock, prevDCCb, false, ht, qt)
				// Process Cr channel
				prevDCCr = e.encodeComponentBlock(bw, crBlock, prevDCCr, false, ht, qt)
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

// encodeComponentBlock processes a single 8x8 float block (DCT, Quantize, Zigzag, Huffman).
// It returns the new DC value.
func (e *Encoder) encodeComponentBlock(bw *BitWriter, block [64]float32, prevDC int16, isLuminance bool, ht *HuffmanTables, qt *QuantizationTables) int16 {
	// 1. Forward DCT
	dct := ForwardDCT(block)

	// 2. Quantization
	var qTable [64]float32
	if isLuminance {
		qTable = qt.Luminance
	} else {
		qTable = qt.Chrominance
	}
	quantized := QuantizeBlock(dct, qTable)

	// 3. Zigzag Reorder
	zigzag := ZigzagReorder(quantized)

	// 4. Huffman Encode DC
	dc := zigzag[0]
	cat, bits, bitLen := EncodeDC(dc, prevDC)
	hCode, hLen := ht.EncodeDC(cat, isLuminance)
	bw.Write(uint32(hCode), hLen)
	if bitLen > 0 {
		bw.Write(uint32(bits), bitLen)
	}

	// 5. Huffman Encode AC
	acRuns := RunLengthEncode(zigzag)
	for _, run := range acRuns {
		hCode, hLen := ht.EncodeAC(run.RunLength, run.Size, isLuminance)
		bw.Write(uint32(hCode), hLen)
		if run.Size > 0 {
			bits, bitLen := EncodeValue(run.Value)
			bw.Write(uint32(bits), bitLen)
		}
	}

	return dc
}
