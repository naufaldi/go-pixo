// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Forked from Go's image/jpeg/huffman.go for the lossless coefficient reader
// (Go's LICENSE is reproduced in NOTICE.md).
// Changes: renamed identifiers and errors so they live beside the encoder.

package jpeg

import (
	"io"
)

// coeffMaxCodeLength is the maximum (inclusive) number of bits in a Huffman code.
const coeffMaxCodeLength = 16

// coeffMaxNCodes is the maximum (inclusive) number of codes in a Huffman tree.
const coeffMaxNCodes = 256

// coeffLutSize is the log-2 size of the Huffman decoder's look-up table.
const coeffLutSize = 8

// coeffHuffman is a Huffman decoder, specified in section C.
type coeffHuffman struct {
	// nCodes is the number of codes in the tree.
	nCodes int32
	// lut is the look-up table for the next coeffLutSize bits in the bit-stream.
	// The high 8 bits of the uint16 are the encoded value. The low 8 bits
	// are 1 plus the code length, or 0 if the value is too large to fit in
	// coeffLutSize bits.
	lut [1 << coeffLutSize]uint16
	// vals are the decoded values, sorted by their encoding.
	vals [coeffMaxNCodes]uint8
	// minCodes[i] is the minimum code of length i, or -1 if there are no
	// codes of that length.
	minCodes [coeffMaxCodeLength]int32
	// maxCodes[i] is the maximum code of length i, or -1 if there are no
	// codes of that length.
	maxCodes [coeffMaxCodeLength]int32
	// valsIndices[i] is the index into vals of minCodes[i].
	valsIndices [coeffMaxCodeLength]int32
}

// errCoeffShortHuffmanData means that an unexpected EOF occurred while
// decoding Huffman data.
var errCoeffShortHuffmanData = coeffFormatError("short Huffman data")

// ensureNBits reads bytes from the byte buffer to ensure that d.bits.n is at
// least n. For best performance (avoiding function calls inside hot loops),
// the caller is the one responsible for first checking that d.bits.n < n.
func (d *coeffDecoder) ensureNBits(n int32) error {
	for {
		c, err := d.readByteStuffedByte()
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				return errCoeffShortHuffmanData
			}
			return err
		}
		d.bits.a = d.bits.a<<8 | uint32(c)
		d.bits.n += 8
		if d.bits.m == 0 {
			d.bits.m = 1 << 7
		} else {
			d.bits.m <<= 8
		}
		if d.bits.n >= n {
			break
		}
	}
	return nil
}

// receiveExtend is the composition of RECEIVE and EXTEND, specified in section
// F.2.2.1.
func (d *coeffDecoder) receiveExtend(t uint8) (int32, error) {
	if d.bits.n < int32(t) {
		if err := d.ensureNBits(int32(t)); err != nil {
			return 0, err
		}
	}
	d.bits.n -= int32(t)
	d.bits.m >>= t
	s := int32(1) << t
	x := int32(d.bits.a>>uint8(d.bits.n)) & (s - 1)
	if x < s>>1 {
		x += ((-1) << t) + 1
	}
	return x, nil
}

// processDHT processes a Define Huffman Table marker, and initializes a huffman
// struct from its contents. Specified in section B.2.4.2.
func (d *coeffDecoder) processDHT(n int) error {
	for n > 0 {
		if n < 17 {
			return coeffFormatError("DHT has wrong length")
		}
		if err := d.readFull(d.tmp[:17]); err != nil {
			return err
		}
		tc := d.tmp[0] >> 4
		if tc > coeffMaxTc {
			return coeffFormatError("bad Tc value")
		}
		th := d.tmp[0] & 0x0f
		// The baseline th <= 1 restriction is specified in table B.5.
		if th > coeffMaxTh || (d.baseline && th > 1) {
			return coeffFormatError("bad Th value")
		}
		h := &d.huff[tc][th]

		// Read nCodes and h.vals (and derive h.nCodes).
		// nCodes[i] is the number of codes with code length i.
		// h.nCodes is the total number of codes.
		h.nCodes = 0
		var nCodes [coeffMaxCodeLength]int32
		for i := range nCodes {
			nCodes[i] = int32(d.tmp[i+1])
			h.nCodes += nCodes[i]
		}
		if h.nCodes == 0 {
			return coeffFormatError("Huffman table has zero length")
		}
		if h.nCodes > coeffMaxNCodes {
			return coeffFormatError("Huffman table has excessive length")
		}
		n -= int(h.nCodes) + 17
		if n < 0 {
			return coeffFormatError("DHT has wrong length")
		}
		if err := d.readFull(h.vals[:h.nCodes]); err != nil {
			return err
		}

		// Derive the look-up table.
		clear(h.lut[:])
		var x, code uint32
		for i := uint32(0); i < coeffLutSize; i++ {
			code <<= 1
			for j := int32(0); j < nCodes[i]; j++ {
				// The codeLength is 1+i, so shift code by 8-(1+i) to
				// calculate the high bits for every 8-bit sequence
				// whose codeLength's high bits matches code.
				// The high 8 bits of lutValue are the encoded value.
				// The low 8 bits are 1 plus the codeLength.
				base := uint8(code << (7 - i))
				lutValue := uint16(h.vals[x])<<8 | uint16(2+i)
				for k := uint8(0); k < 1<<(7-i); k++ {
					h.lut[base|k] = lutValue
				}
				code++
				x++
			}
		}

		// Derive minCodes, maxCodes, and valsIndices.
		var c, index int32
		for i, n := range nCodes {
			if n == 0 {
				h.minCodes[i] = -1
				h.maxCodes[i] = -1
				h.valsIndices[i] = -1
			} else {
				h.minCodes[i] = c
				h.maxCodes[i] = c + n - 1
				h.valsIndices[i] = index
				c += n
				index += n
			}
			c <<= 1
		}
	}
	return nil
}

// decodeHuffman returns the next Huffman-coded value from the bit-stream,
// decoded according to h.
func (d *coeffDecoder) decodeHuffman(h *coeffHuffman) (uint8, error) {
	if h.nCodes == 0 {
		return 0, coeffFormatError("uninitialized Huffman table")
	}

	if d.bits.n < 8 {
		if err := d.ensureNBits(8); err != nil {
			if err != errCoeffMissingFF00 && err != errCoeffShortHuffmanData {
				return 0, err
			}
			// There are no more bytes of data in this segment, but we may still
			// be able to read the next symbol out of the previously read bits.
			// First, undo the readByte that the ensureNBits call made.
			if d.bytes.nUnreadable != 0 {
				d.unreadByteStuffedByte()
			}
			goto slowPath
		}
	}
	if v := h.lut[(d.bits.a>>uint32(d.bits.n-coeffLutSize))&0xff]; v != 0 {
		n := (v & 0xff) - 1
		d.bits.n -= int32(n)
		d.bits.m >>= n
		return uint8(v >> 8), nil
	}

slowPath:
	for i, code := 0, int32(0); i < coeffMaxCodeLength; i++ {
		if d.bits.n == 0 {
			if err := d.ensureNBits(1); err != nil {
				return 0, err
			}
		}
		if d.bits.a&d.bits.m != 0 {
			code |= 1
		}
		d.bits.n--
		d.bits.m >>= 1
		if code <= h.maxCodes[i] {
			return h.vals[h.valsIndices[i]+code-h.minCodes[i]], nil
		}
		code <<= 1
	}
	return 0, coeffFormatError("bad Huffman code")
}

func (d *coeffDecoder) decodeBit() (bool, error) {
	if d.bits.n == 0 {
		if err := d.ensureNBits(1); err != nil {
			return false, err
		}
	}
	ret := d.bits.a&d.bits.m != 0
	d.bits.n--
	d.bits.m >>= 1
	return ret, nil
}

func (d *coeffDecoder) decodeBits(n int32) (uint32, error) {
	if d.bits.n < n {
		if err := d.ensureNBits(n); err != nil {
			return 0, err
		}
	}
	ret := d.bits.a >> uint32(d.bits.n-n)
	ret &= (1 << uint32(n)) - 1
	d.bits.n -= n
	d.bits.m >>= uint32(n)
	return ret, nil
}
