package compress

import (
	"bytes"
	"io"
)

// DeflateEncoder encodes data using DEFLATE compression.
type DeflateEncoder struct {
	lz77             *LZ77Encoder
	compressionLevel int
}

// NewDeflateEncoder creates a new DEFLATE encoder.
func NewDeflateEncoder() *DeflateEncoder {
	return &DeflateEncoder{
		lz77:             NewLZ77Encoder(),
		compressionLevel: 6,
	}
}

// SetCompressionLevel sets the compression level (1-9).
// Higher levels produce better compression but are slower.
func (enc *DeflateEncoder) SetCompressionLevel(level int) {
	if level < 1 {
		level = 1
	} else if level > 9 {
		level = 9
	}
	enc.compressionLevel = level
	enc.lz77.SetCompressionLevel(level)
}

// Encode compresses data using DEFLATE with the specified block type.
// If useDynamic is true, uses dynamic Huffman tables; otherwise uses fixed tables.
func (enc *DeflateEncoder) Encode(data []byte, useDynamic bool) ([]byte, error) {
	if len(data) == 0 {
		var buf bytes.Buffer
		if err := WriteStoredBlockDeflate(&buf, true, data); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	tokens := enc.lz77.Encode(data)

	var buf bytes.Buffer
	if useDynamic {
		if err := WriteDynamicBlock(&buf, true, tokens); err != nil {
			return nil, err
		}
	} else {
		if err := WriteFixedBlock(&buf, true, tokens); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// EncodeAuto compresses data using DEFLATE and automatically chooses
// between fixed and dynamic Huffman tables based on which produces smaller output.
// If dynamic encoding fails, it falls back to fixed encoding.
func (enc *DeflateEncoder) EncodeAuto(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return enc.Encode(data, false)
	}

	fixed, err := enc.Encode(data, false)
	if err != nil {
		return nil, err
	}

	dynamic, err := enc.Encode(data, true)
	if err != nil {
		return fixed, nil
	}

	if len(dynamic) < len(fixed) {
		return dynamic, nil
	}
	return fixed, nil
}

// EncodeOptimal compresses data using optimal DEFLATE with iterative refinement.
// This produces better compression at the cost of slower encoding.
func (enc *DeflateEncoder) EncodeOptimal(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return enc.Encode(data, false)
	}

	// For now, use multiple passes with increasing compression level
	// A full Zopfli implementation would use optimal parsing with cost model
	bestResult := data
	bestSize := len(data)

	// Try multiple iterations with increasing effort
	for iteration := 0; iteration < 5; iteration++ {
		// Increase compression level each iteration
		enc.SetCompressionLevel(enc.compressionLevel + iteration)
		if enc.compressionLevel > 9 {
			enc.SetCompressionLevel(9)
		}

		result, err := enc.EncodeAuto(data)
		if err != nil {
			continue
		}

		if len(result) < bestSize {
			bestResult = result
			bestSize = len(result)
		}
	}

	// Reset to original level
	enc.SetCompressionLevel(enc.compressionLevel)

	return bestResult, nil
}

// EncodeTo writes compressed DEFLATE data directly to the writer.
func (enc *DeflateEncoder) EncodeTo(w io.Writer, data []byte, useDynamic bool) error {
	compressed, err := enc.Encode(data, useDynamic)
	if err != nil {
		return err
	}
	_, err = w.Write(compressed)
	return err
}
