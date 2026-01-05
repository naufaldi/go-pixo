package jpeg

import (
	"io"
)

// BitWriter writes bits to an underlying io.Writer in MSB-first order (JPEG format).
// It also handles byte stuffing (0xFF -> 0xFF 0x00).
type BitWriter struct {
	w     io.Writer
	buf   uint32 // Bit buffer
	nbits int    // Number of bits in the buffer
}

// NewBitWriter creates a new BitWriter that writes to w.
func NewBitWriter(w io.Writer) *BitWriter {
	return &BitWriter{w: w}
}

// Write writes the n bits from bits to the writer.
// Bits are written MSB-first (most significant bit first).
func (bw *BitWriter) Write(bits uint32, n uint8) error {
	if n == 0 {
		return nil
	}

	// Add bits to buffer (from MSB to LSB of the n bits)
	// We shift the buffer left and add the new bits
	bw.buf = (bw.buf << uint32(n)) | (bits & ((1 << uint32(n)) - 1))
	bw.nbits += int(n)

	// Flush full bytes
	for bw.nbits >= 8 {
		bw.nbits -= 8
		b := byte((bw.buf >> uint32(bw.nbits)) & 0xFF)
		if err := bw.writeByte(b); err != nil {
			return err
		}
	}

	return nil
}

// writeByte writes a single byte and performs stuffing if needed.
func (bw *BitWriter) writeByte(b byte) error {
	if _, err := bw.w.Write([]byte{b}); err != nil {
		return err
	}
	if b == 0xFF {
		if _, err := bw.w.Write([]byte{0x00}); err != nil {
			return err
		}
	}
	return nil
}

// Flush writes any remaining bits in the buffer, padding with 1s to the next byte boundary.
// Padding with 1s is standard for JPEG to avoid generating inadvertent markers.
func (bw *BitWriter) Flush() error {
	if bw.nbits > 0 {
		// Pad with 1s
		padBits := uint8(8 - bw.nbits)
		if err := bw.Write((1<<uint32(padBits))-1, padBits); err != nil {
			return err
		}
	}
	return nil
}
