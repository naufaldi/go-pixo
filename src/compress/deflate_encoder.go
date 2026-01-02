package compress

import (
	"bytes"
	"io"
)

// DeflateEncoder encodes data using DEFLATE compression.
type DeflateEncoder struct {
	lz77 *LZ77Encoder
}

// NewDeflateEncoder creates a new DEFLATE encoder.
func NewDeflateEncoder() *DeflateEncoder {
	return &DeflateEncoder{
		lz77: NewLZ77Encoder(),
	}
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
		return nil, err
	}

	if len(dynamic) < len(fixed) {
		return dynamic, nil
	}
	return fixed, nil
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
