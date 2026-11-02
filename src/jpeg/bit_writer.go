package jpeg

import (
	"io"
)

// BitWriter writes bits to an underlying io.Writer in MSB-first order (JPEG format).
// It also handles byte stuffing (0xFF -> 0xFF 0x00).
type BitWriter struct {
	w       io.Writer
	bytesW  io.ByteWriter // Set when w supports WriteByte, avoiding a slice per byte.
	buf     uint32        // Bit buffer
	nbits   int           // Number of bits in the buffer
	scratch [2]byte
}

// NewBitWriter creates a new BitWriter that writes to w.
func NewBitWriter(w io.Writer) *BitWriter {
	bw := &BitWriter{w: w}
	if byteWriter, ok := w.(io.ByteWriter); ok {
		bw.bytesW = byteWriter
	}
	return bw
}

// Write writes the n bits from bits to the writer.
// Bits are written MSB-first (most significant bit first).
func (bw *BitWriter) Write(bits uint32, n uint8) error {
	if n == 0 {
		return nil
	}

	// Add bits to buffer
	bw.buf = (bw.buf << uint32(n)) | (bits & ((1 << uint32(n)) - 1))
	bw.nbits += int(n)

	// Flush full bytes
	for bw.nbits >= 8 {
		b := byte((bw.buf >> uint32(bw.nbits-8)) & 0xFF)
		if err := bw.writeByte(b); err != nil {
			return err
		}
		bw.nbits -= 8
	}
	// Mask buffer to only keep remaining bits
	bw.buf &= (1 << uint32(bw.nbits)) - 1

	return nil
}

// writeByte writes a single byte and performs stuffing if needed.
func (bw *BitWriter) writeByte(b byte) error {
	if bw.bytesW != nil {
		if err := bw.bytesW.WriteByte(b); err != nil {
			return err
		}
		if b == 0xFF {
			return bw.bytesW.WriteByte(0x00)
		}
		return nil
	}
	bw.scratch[0] = b
	n := 1
	if b == 0xFF {
		bw.scratch[1] = 0x00
		n = 2
	}
	_, err := bw.w.Write(bw.scratch[:n])
	return err
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
