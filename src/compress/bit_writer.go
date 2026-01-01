package compress

import "io"

// BitWriter writes bits to an underlying io.Writer in LSB-first order (DEFLATE format).
// Bits are accumulated in a buffer and written to the writer when a full byte is formed.
type BitWriter struct {
	w     io.Writer
	buf   byte
	nbits int
}

// NewBitWriter creates a new BitWriter that writes to w.
func NewBitWriter(w io.Writer) *BitWriter {
	return &BitWriter{w: w}
}

// Write writes the n least-significant bits from bits to the writer.
// Bits are written LSB-first (least significant bit first).
// For example, Write(0b101, 3) writes bits in order: 1, 0, 1.
func (bw *BitWriter) Write(bits uint16, n int) error {
	if n == 0 {
		return nil
	}
	
	for i := 0; i < n; i++ {
		bit := (bits >> uint(i)) & 1
		bw.buf |= byte(bit) << uint(bw.nbits)
		bw.nbits++
		
		if bw.nbits == 8 {
			if err := bw.flushByte(); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// Flush writes any remaining bits in the buffer, padding with zeros to the next byte boundary.
func (bw *BitWriter) Flush() error {
	if bw.nbits > 0 {
		return bw.flushByte()
	}
	return nil
}

// flushByte writes the current byte buffer and resets it.
func (bw *BitWriter) flushByte() error {
	if bw.nbits == 0 {
		return nil
	}
	
	_, err := bw.w.Write([]byte{bw.buf})
	if err != nil {
		return err
	}
	
	bw.buf = 0
	bw.nbits = 0
	return nil
}
