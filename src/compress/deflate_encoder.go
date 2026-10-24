package compress

import (
	"bytes"
	"io"
)

// DeflateEncoder encodes data using DEFLATE compression.
type DeflateEncoder struct {
	lz77             *LZ77Encoder
	compressionLevel int
	ProgressCallback func(iteration, total int)
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
	return encodeDeflateTokens(tokens, useDynamic)
}

// encodeDeflateTokens codes an existing parse without repeating LZ77 matching.
func encodeDeflateTokens(tokens []Token, useDynamic bool) ([]byte, error) {
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

	tokens := enc.lz77.Encode(data)
	fixed, err := encodeDeflateTokens(tokens, false)
	if err != nil {
		return nil, err
	}

	dynamic, err := encodeDeflateTokens(tokens, true)
	if err != nil {
		return fixed, nil
	}

	if len(dynamic) < len(fixed) && validateDeflateRoundTrip(dynamic, data) {
		return dynamic, nil
	}
	return fixed, nil
}

// EncodeOptimal selects the best of distinct greedy LZ77 configurations.
// It is a bounded search, not a full Zopfli optimizer; see ZopfliConfig.
func (enc *DeflateEncoder) EncodeOptimal(data []byte) ([]byte, error) {
	config := DefaultZopfliConfig()
	return enc.EncodeOptimalWithConfig(data, config.Iterations, config.BlockSplitting)
}

// EncodeOptimalWithConfig selects distinct parses with the budget described by
// ZopfliConfig.Iterations. blockSplitting is retained for compatibility and ignored.
func (enc *DeflateEncoder) EncodeOptimalWithConfig(data []byte, iterations int, blockSplitting bool) ([]byte, error) {
	if len(data) == 0 {
		return enc.Encode(data, false)
	}

	zopfliConfig := ZopfliConfig{
		Iterations:       iterations,
		BlockSplitting:   blockSplitting,
		MaxBlockSize:     65535,
		ProgressCallback: enc.ProgressCallback,
	}

	return ZopfliEncode(data, zopfliConfig)
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

// EncodeStored encodes data using DEFLATE stored blocks (no compression).
// This is used as a fallback when DEFLATE compression doesn't reduce size.
// Stored blocks have minimal overhead: header (1 byte) + LEN/NLEN (4 bytes) + data.
// Large data is automatically split into multiple blocks of 65535 bytes each.
func (enc *DeflateEncoder) EncodeStored(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := WriteStoredBlocks(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeWithFallback tries DEFLATE compression first. If the result is not
// smaller than the input, it falls back to stored blocks (no compression).
// Expansion is bounded by stored-block overhead; framing still adds bytes.
func (enc *DeflateEncoder) EncodeWithFallback(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return enc.EncodeStored(data)
	}

	compressed, err := enc.EncodeAuto(data)
	if err != nil {
		return nil, err
	}

	if len(compressed) >= len(data) {
		return enc.EncodeStored(data)
	}
	return compressed, nil
}
