// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Forked from Go's image/jpeg/reader.go and image/jpeg/scan.go for the
// lossless optimizer (Go's LICENSE is reproduced in NOTICE.md). Changes
// from upstream:
//   - quantized DCT coefficients are retained (zig-zag order, one int16 buffer
//     for all components) instead of being reconstructed into pixels;
//   - JFIF, ICC (APP2) and Adobe (APP14) segments are captured for re-emission;
//   - input and dimension limits are checked before coefficient allocation;
//   - CMYK/YCCK, 12-bit, arithmetic, lossless and hierarchical files are
//     rejected with sentinel errors;
//   - non-interleaved scans use the spec's per-component block extents and
//     count one block per MCU for restart intervals;
//   - restart markers, sequential EOB runs and quantization-table changes
//     between scans are rejected instead of being resynchronized or ignored.

package jpeg

import (
	"bytes"
	"errors"
	"io"

	"github.com/mac/go-pixo/src/internal/imagelimit"
)

// coeffFormatError reports that the input is not a valid JPEG.
type coeffFormatError string

func (e coeffFormatError) Error() string { return "jpeg: invalid JPEG input: " + string(e) }

// Is lets errors.Is match every format error against ErrInvalidJPEG.
func (e coeffFormatError) Is(target error) bool { return target == ErrInvalidJPEG }

const (
	coeffDCTable       = 0
	coeffACTable       = 1
	coeffMaxTc         = 1
	coeffMaxTh         = 3
	coeffMaxTq         = 3
	coeffMaxComponents = 3
	coeffBlockSize     = 64
)

const (
	mkSOF0  = 0xc0 // Start Of Frame (Baseline Sequential).
	mkSOF1  = 0xc1 // Start Of Frame (Extended Sequential).
	mkSOF2  = 0xc2 // Start Of Frame (Progressive).
	mkSOF3  = 0xc3 // Start Of Frame (Lossless).
	mkDHT   = 0xc4 // Define Huffman Table.
	mkJPG   = 0xc8 // Reserved for JPEG extensions.
	mkDAC   = 0xcc // Define Arithmetic Coding conditioning.
	mkSOF15 = 0xcf // Last SOF marker (lossless, differential, arithmetic).
	mkRST0  = 0xd0 // ReSTart (0).
	mkRST7  = 0xd7 // ReSTart (7).
	mkSOI   = 0xd8 // Start Of Image.
	mkEOI   = 0xd9 // End Of Image.
	mkSOS   = 0xda // Start Of Scan.
	mkDQT   = 0xdb // Define Quantization Table.
	mkDNL   = 0xdc // Define Number of Lines.
	mkDRI   = 0xdd // Define Restart Interval.
	mkDHP   = 0xde // Define Hierarchical Progression.
	mkEXP   = 0xdf // EXPand reference components.
	mkAPP0  = 0xe0
	mkAPP2  = 0xe2
	mkAPP14 = 0xee
	mkAPP15 = 0xef
	mkCOM   = 0xfe // COMment.
)

// coeffBits holds the unprocessed bits that have been taken from the
// byte-stream. The n least significant bits of a form the unread bits, to be
// read in MSB to LSB order.
type coeffBits struct {
	a uint32 // accumulator.
	m uint32 // mask. m==1<<(n-1) when n>0, with m==0 when n==0.
	n int32  // the number of unread bits in a.
}

// coeffComponent is a frame component (section B.2.2) plus its block layout.
type coeffComponent struct {
	c  uint8 // Component identifier.
	hv uint8 // Sampling factors as written in the SOF segment.
	tq uint8 // Quantization table destination selector.
	h  int   // Effective horizontal sampling factor.
	v  int   // Effective vertical sampling factor.
	// blocksW and blocksH span the padded MCU grid (stride of the storage).
	blocksW, blocksH int
	// compW and compH span the blocks that hold the component's samples;
	// non-interleaved scans only code these blocks.
	compW, compH int
	offset       int // First block of this component in coeffDecoder.coeffs.
	scanned      bool
}

// coeffDecoder reads a JPEG's quantized coefficients and the metadata the
// lossless optimizer preserves.
type coeffDecoder struct {
	r    io.Reader
	bits coeffBits
	// bytes is a byte buffer, similar to a bufio.Reader, except that it
	// has to be able to unread more than 1 byte, due to byte stuffing.
	// Byte stuffing is specified in section F.1.2.3.
	bytes struct {
		// buf[i:j] are the buffered bytes read from the underlying
		// io.Reader that haven't yet been passed further on.
		buf  [4096]byte
		i, j int
		// nUnreadable is the number of bytes to back up i after
		// overshooting. It can be 0, 1 or 2.
		nUnreadable int
	}
	width, height int

	ri    int // Restart Interval.
	nComp int

	baseline    bool
	progressive bool

	eobRun uint16 // End-of-Band run, specified in section G.1.2.2.

	comp         [coeffMaxComponents]coeffComponent
	hMax, vMax   int
	mxx, myy     int // MCUs per row and column of interleaved scans.
	totalBlocks  int
	coeffs       []int16 // Zig-zag ordered coefficients, 64 per block.
	huff         [coeffMaxTc + 1][coeffMaxTh + 1]coeffHuffman
	quant        [coeffMaxTq + 1][coeffBlockSize]uint16 // Zig-zag order.
	quantDefined [coeffMaxTq + 1]bool
	// approx records, per component and zig-zag index, 1 + the Al of the
	// last progressive scan that coded it (0 = not yet coded).
	approx         [coeffMaxComponents][coeffBlockSize]uint8
	scanSeen       bool
	jfif           []byte   // APP0 JFIF payload, if any.
	iccSegments    [][]byte // APP2 ICC_PROFILE payloads, in file order.
	adobe          []byte   // APP14 Adobe payload, if any.
	tmp            [2 * coeffBlockSize]byte
	segmentScratch []byte
	// headerOnly accepts a stream that ends after its frame header, for
	// coefficients the encoder fills in itself (see readFrameHeader).
	headerOnly bool
}

// fill fills up the d.bytes.buf buffer from the underlying io.Reader. It
// should only be called when there are no unread bytes in d.bytes.
func (d *coeffDecoder) fill() error {
	if d.bytes.i != d.bytes.j {
		return coeffFormatError("internal: fill called when unread bytes exist")
	}
	// Move the last 2 bytes to the start of the buffer, in case we need
	// to call unreadByteStuffedByte.
	if d.bytes.j > 2 {
		d.bytes.buf[0] = d.bytes.buf[d.bytes.j-2]
		d.bytes.buf[1] = d.bytes.buf[d.bytes.j-1]
		d.bytes.i, d.bytes.j = 2, 2
	}
	// Fill in the rest of the buffer.
	n, err := d.r.Read(d.bytes.buf[d.bytes.j:])
	d.bytes.j += n
	if n > 0 {
		return nil
	}
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return err
}

// unreadByteStuffedByte undoes the most recent readByteStuffedByte call,
// giving a byte of data back from d.bits to d.bytes. The Huffman look-up table
// requires at least 8 bits for look-up, which means that Huffman decoding can
// sometimes overshoot and read one or two too many bytes. Two-byte overshoot
// can happen when expecting to read a 0xff 0x00 byte-stuffed byte.
func (d *coeffDecoder) unreadByteStuffedByte() {
	d.bytes.i -= d.bytes.nUnreadable
	d.bytes.nUnreadable = 0
	if d.bits.n >= 8 {
		d.bits.a >>= 8
		d.bits.n -= 8
		d.bits.m >>= 8
	}
}

// readByte returns the next byte, whether buffered or not buffered. It does
// not care about byte stuffing.
func (d *coeffDecoder) readByte() (x byte, err error) {
	for d.bytes.i == d.bytes.j {
		if err = d.fill(); err != nil {
			return 0, err
		}
	}
	x = d.bytes.buf[d.bytes.i]
	d.bytes.i++
	d.bytes.nUnreadable = 0
	return x, nil
}

// errCoeffMissingFF00 means that readByteStuffedByte encountered an 0xff byte
// (a marker byte) that wasn't the expected byte-stuffed sequence 0xff, 0x00.
var errCoeffMissingFF00 = coeffFormatError("missing 0xff00 sequence")

// readByteStuffedByte is like readByte but is for byte-stuffed Huffman data.
func (d *coeffDecoder) readByteStuffedByte() (x byte, err error) {
	// Take the fast path if d.bytes.buf contains at least two bytes.
	if d.bytes.i+2 <= d.bytes.j {
		x = d.bytes.buf[d.bytes.i]
		d.bytes.i++
		d.bytes.nUnreadable = 1
		if x != 0xff {
			return x, err
		}
		if d.bytes.buf[d.bytes.i] != 0x00 {
			return 0, errCoeffMissingFF00
		}
		d.bytes.i++
		d.bytes.nUnreadable = 2
		return 0xff, nil
	}

	d.bytes.nUnreadable = 0

	x, err = d.readByte()
	if err != nil {
		return 0, err
	}
	d.bytes.nUnreadable = 1
	if x != 0xff {
		return x, nil
	}

	x, err = d.readByte()
	if err != nil {
		return 0, err
	}
	d.bytes.nUnreadable = 2
	if x != 0x00 {
		return 0, errCoeffMissingFF00
	}
	return 0xff, nil
}

// readFull reads exactly len(p) bytes into p. It does not care about byte
// stuffing.
func (d *coeffDecoder) readFull(p []byte) error {
	// Unread the overshot bytes, if any.
	if d.bytes.nUnreadable != 0 {
		if d.bits.n >= 8 {
			d.unreadByteStuffedByte()
		}
		d.bytes.nUnreadable = 0
	}

	for {
		n := copy(p, d.bytes.buf[d.bytes.i:d.bytes.j])
		p = p[n:]
		d.bytes.i += n
		if len(p) == 0 {
			break
		}
		if err := d.fill(); err != nil {
			return err
		}
	}
	return nil
}

// ignore ignores the next n bytes.
func (d *coeffDecoder) ignore(n int) error {
	// Unread the overshot bytes, if any.
	if d.bytes.nUnreadable != 0 {
		if d.bits.n >= 8 {
			d.unreadByteStuffedByte()
		}
		d.bytes.nUnreadable = 0
	}

	for {
		m := d.bytes.j - d.bytes.i
		if m > n {
			m = n
		}
		d.bytes.i += m
		n -= m
		if n == 0 {
			break
		}
		if err := d.fill(); err != nil {
			return err
		}
	}
	return nil
}

// Specified in section B.2.2.
func (d *coeffDecoder) processSOF(n int) error {
	if d.nComp != 0 {
		return coeffFormatError("multiple SOF markers")
	}
	switch n {
	case 6 + 3*1: // Grayscale image.
		d.nComp = 1
	case 6 + 3*3: // YCbCr or RGB image.
		d.nComp = 3
	case 6 + 3*4: // YCbCrK or CMYK image.
		return ErrUnsupportedJPEGComponents
	default:
		if n >= 6 {
			return ErrUnsupportedJPEGComponents
		}
		return coeffFormatError("SOF has wrong length")
	}
	if err := d.readFull(d.tmp[:n]); err != nil {
		return err
	}
	// We only support 8-bit precision.
	if d.tmp[0] != 8 {
		return ErrUnsupportedJPEGPrecision
	}
	d.height = int(d.tmp[1])<<8 + int(d.tmp[2])
	d.width = int(d.tmp[3])<<8 + int(d.tmp[4])
	if int(d.tmp[5]) != d.nComp {
		return coeffFormatError("SOF has wrong length")
	}
	// Bound the frame before any coefficient storage is sized. A zero height
	// would require a DNL marker, which is not supported.
	if err := imagelimit.CheckDimensions(d.width, d.height); err != nil {
		return err
	}

	for i := 0; i < d.nComp; i++ {
		d.comp[i].c = d.tmp[6+3*i]
		// Section B.2.2 states that "the value of C_i shall be different from
		// the values of C_1 through C_(i-1)".
		for j := 0; j < i; j++ {
			if d.comp[i].c == d.comp[j].c {
				return coeffFormatError("repeated component identifier")
			}
		}

		d.comp[i].tq = d.tmp[8+3*i]
		if d.comp[i].tq > coeffMaxTq {
			return coeffFormatError("bad Tq value")
		}

		hv := d.tmp[7+3*i]
		h, v := int(hv>>4), int(hv&0x0f)
		if h < 1 || 4 < h || v < 1 || 4 < v {
			return coeffFormatError("luma/chroma subsampling ratio")
		}
		if d.nComp == 1 {
			// Section A.2: single-component data is non-interleaved by
			// definition and the MCU is one data unit, regardless of H_1, V_1.
			h, v = 1, 1
		}
		d.comp[i].hv = hv
		d.comp[i].h = h
		d.comp[i].v = v
		d.hMax = max(d.hMax, h)
		d.vMax = max(d.vMax, v)
	}

	// Section A.1.1: component i has ceil(X * H_i / H_max) samples per line.
	d.mxx = (d.width + 8*d.hMax - 1) / (8 * d.hMax)
	d.myy = (d.height + 8*d.vMax - 1) / (8 * d.vMax)
	offset := 0
	for i := 0; i < d.nComp; i++ {
		c := &d.comp[i]
		c.blocksW = d.mxx * c.h
		c.blocksH = d.myy * c.v
		samplesW := (d.width*c.h + d.hMax - 1) / d.hMax
		samplesH := (d.height*c.v + d.vMax - 1) / d.vMax
		c.compW = (samplesW + 7) / 8
		c.compH = (samplesH + 7) / 8
		c.offset = offset
		// Dimensions are at most 16384 and 24 MP, so per-component padded
		// block counts stay well below the 32-bit int range used by WASM.
		offset += c.blocksW * c.blocksH
	}
	d.totalBlocks = offset
	return nil
}

// Specified in section B.2.4.1.
func (d *coeffDecoder) processDQT(n int) error {
	for n > 0 {
		n--
		x, err := d.readByte()
		if err != nil {
			return err
		}
		tq := x & 0x0f
		if tq > coeffMaxTq {
			return coeffFormatError("bad Tq value")
		}
		var table [coeffBlockSize]uint16
		switch x >> 4 {
		default:
			return coeffFormatError("bad Pq value")
		case 0:
			if n < coeffBlockSize {
				return coeffFormatError("DQT has wrong length")
			}
			n -= coeffBlockSize
			if err := d.readFull(d.tmp[:coeffBlockSize]); err != nil {
				return err
			}
			for i := range table {
				table[i] = uint16(d.tmp[i])
			}
		case 1:
			if n < 2*coeffBlockSize {
				return coeffFormatError("DQT has wrong length")
			}
			n -= 2 * coeffBlockSize
			if err := d.readFull(d.tmp[:2*coeffBlockSize]); err != nil {
				return err
			}
			for i := range table {
				table[i] = uint16(d.tmp[2*i])<<8 | uint16(d.tmp[2*i+1])
			}
		}
		if d.scanSeen && d.quantDefined[tq] && d.quant[tq] != table && d.quantInUse(tq) {
			// Decoders disagree on whether a redefinition applies to
			// components that were already scanned; refuse rather than guess.
			return ErrUnsupportedJPEGProcess
		}
		d.quant[tq] = table
		d.quantDefined[tq] = true
	}
	return nil
}

func (d *coeffDecoder) quantInUse(tq uint8) bool {
	for i := 0; i < d.nComp; i++ {
		if d.comp[i].tq == tq && d.comp[i].scanned {
			return true
		}
	}
	return false
}

// Specified in section B.2.4.4.
func (d *coeffDecoder) processDRI(n int) error {
	if n != 2 {
		return coeffFormatError("DRI has wrong length")
	}
	if err := d.readFull(d.tmp[:2]); err != nil {
		return err
	}
	d.ri = int(d.tmp[0])<<8 + int(d.tmp[1])
	return nil
}

// readSegment returns the next n bytes in a scratch buffer that is reused by
// the next call.
func (d *coeffDecoder) readSegment(n int) ([]byte, error) {
	if cap(d.segmentScratch) < n {
		d.segmentScratch = make([]byte, n)
	}
	segment := d.segmentScratch[:n]
	if err := d.readFull(segment); err != nil {
		return nil, err
	}
	return segment, nil
}

// processAppMarker captures the APPn payloads that affect how decoders render
// the image (JFIF, ICC profile, Adobe transform) and skips everything else.
func (d *coeffDecoder) processAppMarker(marker uint8, n int) error {
	switch marker {
	case mkAPP0, mkAPP2, mkAPP14:
	default:
		return d.ignore(n)
	}
	segment, err := d.readSegment(n)
	if err != nil {
		return err
	}
	switch {
	case marker == mkAPP0 && d.jfif == nil && bytes.HasPrefix(segment, []byte("JFIF\x00")):
		d.jfif = bytes.Clone(segment)
	case marker == mkAPP2 && bytes.HasPrefix(segment, []byte("ICC_PROFILE\x00")):
		d.iccSegments = append(d.iccSegments, bytes.Clone(segment))
	case marker == mkAPP14 && d.adobe == nil && len(segment) >= 12 && bytes.HasPrefix(segment, []byte("Adobe")):
		d.adobe = bytes.Clone(segment)
	}
	return nil
}

// readCoefficients parses a complete JPEG and returns its coefficients.
// Dimensions are bounded before allocation; callers bound untrusted input size.
func readCoefficients(input []byte) (*coeffDecoder, error) {
	d := &coeffDecoder{r: bytes.NewReader(input)}
	if err := d.decode(); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, coeffFormatError("truncated input")
		}
		return nil, err
	}
	return d, nil
}

// readFrameHeader parses SOI..SOF segments (without scans), so that the
// frame layout, quantization tables and JFIF header are exactly what
// readCoefficients reports for a file starting with them, and allocates
// zeroed coefficient storage for the frame.
func readFrameHeader(header []byte) (*coeffDecoder, error) {
	stream := make([]byte, 0, len(header)+2)
	stream = append(append(stream, header...), 0xff, mkEOI)
	d := &coeffDecoder{r: bytes.NewReader(stream), headerOnly: true}
	if err := d.decode(); err != nil {
		return nil, err
	}
	d.coeffs = make([]int16, d.totalBlocks*coeffBlockSize)
	for i := 0; i < d.nComp; i++ {
		d.comp[i].scanned = true
	}
	return d, nil
}

func (d *coeffDecoder) decode() error {
	// Check for the Start Of Image marker.
	if err := d.readFull(d.tmp[:2]); err != nil {
		return err
	}
	if d.tmp[0] != 0xff || d.tmp[1] != mkSOI {
		return coeffFormatError("missing SOI marker")
	}

	// Process the remaining segments until the End Of Image marker.
	for {
		err := d.readFull(d.tmp[:2])
		if err != nil {
			return err
		}
		for d.tmp[0] != 0xff {
			// Like libjpeg and image/jpeg, silently skip extraneous bytes
			// between segments. They never carry coefficient data.
			d.tmp[0] = d.tmp[1]
			d.tmp[1], err = d.readByte()
			if err != nil {
				return err
			}
		}
		marker := d.tmp[1]
		if marker == 0 {
			// Treat "\xff\x00" as extraneous data.
			continue
		}
		for marker == 0xff {
			// Section B.1.1.2 says, "Any marker may optionally be preceded by any
			// number of fill bytes, which are bytes assigned code X'FF'".
			marker, err = d.readByte()
			if err != nil {
				return err
			}
		}
		if marker == mkEOI { // End Of Image.
			break
		}
		if mkRST0 <= marker && marker <= mkRST7 {
			// A trailing restart marker after the final entropy-coded
			// segment is a harmless encoder bug; it has no payload.
			continue
		}

		// Read the 16-bit length of the segment. The value includes the 2 bytes for the
		// length itself, so we subtract 2 to get the number of remaining bytes.
		if err = d.readFull(d.tmp[:2]); err != nil {
			return err
		}
		n := int(d.tmp[0])<<8 + int(d.tmp[1]) - 2
		if n < 0 {
			return coeffFormatError("short segment length")
		}

		switch {
		case marker == mkSOF0 || marker == mkSOF1 || marker == mkSOF2:
			d.baseline = marker == mkSOF0
			d.progressive = marker == mkSOF2
			err = d.processSOF(n)
		case marker == mkDHT:
			err = d.processDHT(n)
		case marker == mkDQT:
			err = d.processDQT(n)
		case marker == mkSOS:
			err = d.processSOS(n)
		case marker == mkDRI:
			err = d.processDRI(n)
		case mkSOF0+9 <= marker && marker <= mkSOF15: // SOF9-SOF15 and DAC.
			err = ErrUnsupportedJPEGArithmetic
		case marker == mkSOF3 || (mkSOF0+5 <= marker && marker <= mkSOF0+7) || marker == mkDHP || marker == mkEXP || marker == mkDNL || marker == mkJPG:
			err = ErrUnsupportedJPEGProcess
		case mkAPP0 <= marker && marker <= mkAPP15:
			err = d.processAppMarker(marker, n)
		case marker == mkCOM:
			err = d.ignore(n)
		case marker < 0xc0: // See Table B.1 "Marker code assignments".
			err = coeffFormatError("unknown marker")
		default:
			err = ErrUnsupportedJPEGProcess
		}
		if err != nil {
			return err
		}
	}

	if d.nComp == 0 {
		return coeffFormatError("missing SOF marker")
	}
	if d.headerOnly {
		if d.scanSeen {
			return coeffFormatError("internal: unexpected scan in frame header")
		}
		return nil
	}
	if !d.scanSeen {
		return coeffFormatError("missing SOS marker")
	}
	for i := 0; i < d.nComp; i++ {
		if !d.comp[i].scanned {
			return coeffFormatError("component has no scan data")
		}
	}
	return nil
}

// Specified in section B.2.3.
func (d *coeffDecoder) processSOS(n int) error {
	if d.nComp == 0 {
		return coeffFormatError("missing SOF marker")
	}
	if n < 6 || 4+2*d.nComp < n || n%2 != 0 {
		return coeffFormatError("SOS has wrong length")
	}
	if err := d.readFull(d.tmp[:n]); err != nil {
		return err
	}
	nComp := int(d.tmp[0])
	if n != 4+2*nComp {
		return coeffFormatError("SOS length inconsistent with number of components")
	}
	var scan [coeffMaxComponents]struct {
		compIndex uint8
		td        uint8 // DC table selector.
		ta        uint8 // AC table selector.
	}
	totalHV := 0
	for i := 0; i < nComp; i++ {
		cs := d.tmp[1+2*i] // Component selector.
		compIndex := -1
		for j, comp := range d.comp[:d.nComp] {
			if cs == comp.c {
				compIndex = j
			}
		}
		if compIndex < 0 {
			return coeffFormatError("unknown component selector")
		}
		scan[i].compIndex = uint8(compIndex)
		// Section B.2.3 states that "the value of Cs_j shall be different from
		// the values of Cs_1 through Cs_(j-1)". Since we have previously
		// verified that a frame's component identifiers (C_i values in section
		// B.2.2) are unique, it suffices to check that the implicit indexes
		// into d.comp are unique.
		for j := 0; j < i; j++ {
			if scan[i].compIndex == scan[j].compIndex {
				return coeffFormatError("repeated component selector")
			}
		}
		totalHV += d.comp[compIndex].h * d.comp[compIndex].v

		// The baseline t <= 1 restriction is specified in table B.3.
		scan[i].td = d.tmp[2+2*i] >> 4
		if t := scan[i].td; t > coeffMaxTh || (d.baseline && t > 1) {
			return coeffFormatError("bad Td value")
		}
		scan[i].ta = d.tmp[2+2*i] & 0x0f
		if t := scan[i].ta; t > coeffMaxTh || (d.baseline && t > 1) {
			return coeffFormatError("bad Ta value")
		}
	}
	// Section B.2.3 states that if there is more than one component then the
	// total H*V values in a scan must be <= 10.
	if nComp > 1 && totalHV > 10 {
		return coeffFormatError("total sampling factors too large")
	}

	// zigStart and zigEnd are the spectral selection bounds.
	// ah and al are the successive approximation high and low values.
	// The spec calls these values Ss, Se, Ah and Al. For sequential JPEGs,
	// these parameters are hard-coded to 0/63/0/0, as per table B.3.
	zigStart, zigEnd, ah, al := int32(0), int32(coeffBlockSize-1), uint32(0), uint32(0)
	if d.progressive {
		zigStart = int32(d.tmp[1+2*nComp])
		zigEnd = int32(d.tmp[2+2*nComp])
		ah = uint32(d.tmp[3+2*nComp] >> 4)
		al = uint32(d.tmp[3+2*nComp] & 0x0f)
		if (zigStart == 0 && zigEnd != 0) || zigStart > zigEnd || coeffBlockSize <= zigEnd {
			return coeffFormatError("bad spectral selection bounds")
		}
		if zigStart != 0 && nComp != 1 {
			return coeffFormatError("progressive AC coefficients for more than one component")
		}
		if ah != 0 && ah != al+1 {
			return coeffFormatError("bad successive approximation values")
		}
		if al > 13 {
			return coeffFormatError("bad successive approximation values")
		}
	}

	for i := 0; i < nComp; i++ {
		ci := scan[i].compIndex
		c := &d.comp[ci]
		if !d.quantDefined[c.tq] {
			return coeffFormatError("missing quantization table")
		}
		if err := d.checkProgression(ci, zigStart, zigEnd, ah, al); err != nil {
			return err
		}
		c.scanned = true
	}
	d.scanSeen = true

	if d.coeffs == nil {
		// Dimensions were bounded in processSOF; this is the only
		// allocation proportional to image size.
		d.coeffs = make([]int16, d.totalBlocks*coeffBlockSize)
	}

	d.bits = coeffBits{}
	d.eobRun = 0
	mcu, expectedRST := 0, uint8(mkRST0)
	var dc [coeffMaxComponents]int32
	params := scanParams{zigStart: zigStart, zigEnd: zigEnd, ah: ah, al: al}

	if nComp == 1 {
		// Non-interleaved scans code each of the component's blocks as one
		// MCU, left to right and top to bottom (section A.2.2).
		ci := int(scan[0].compIndex)
		c := &d.comp[ci]
		totalMCU := c.compW * c.compH
		dcTable := &d.huff[coeffDCTable][scan[0].td]
		acTable := &d.huff[coeffACTable][scan[0].ta]
		for by := 0; by < c.compH; by++ {
			for bx := 0; bx < c.compW; bx++ {
				if err := d.decodeBlock(c.offset+by*c.blocksW+bx, &dc[ci], dcTable, acTable, params); err != nil {
					return err
				}
				mcu++
				if err := d.restart(mcu, totalMCU, &expectedRST, &dc); err != nil {
					return err
				}
			}
		}
		return nil
	}

	totalMCU := d.mxx * d.myy
	for my := 0; my < d.myy; my++ {
		for mx := 0; mx < d.mxx; mx++ {
			for i := 0; i < nComp; i++ {
				ci := int(scan[i].compIndex)
				c := &d.comp[ci]
				dcTable := &d.huff[coeffDCTable][scan[i].td]
				acTable := &d.huff[coeffACTable][scan[i].ta]
				for j := 0; j < c.h*c.v; j++ {
					// For 4:2:0, the Y blocks of a 32x16 image are visited as
					//	0 1 4 5
					//	2 3 6 7
					bx := c.h*mx + j%c.h
					by := c.v*my + j/c.h
					if err := d.decodeBlock(c.offset+by*c.blocksW+bx, &dc[ci], dcTable, acTable, params); err != nil {
						return err
					}
				}
			}
			mcu++
			if err := d.restart(mcu, totalMCU, &expectedRST, &dc); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkProgression rejects scans that re-code coefficients out of order
// (section G.1.1.1.1). Decoders disagree on such files: libjpeg skips an
// already-refined bit while Go's image/jpeg applies it again, so accepting
// them would make "lossless" output decode differently elsewhere.
func (d *coeffDecoder) checkProgression(ci uint8, zigStart, zigEnd int32, ah, al uint32) error {
	if !d.progressive {
		if d.comp[ci].scanned {
			return coeffFormatError("component coded by more than one sequential scan")
		}
		return nil
	}
	approx := &d.approx[ci]
	if zigStart > 0 && approx[0] == 0 {
		return coeffFormatError("progressive AC scan before the component's first DC scan")
	}
	for k := zigStart; k <= zigEnd; k++ {
		first, previous := ah == 0, approx[k]
		if (first && previous != 0) || (!first && uint32(previous) != ah+1) {
			return coeffFormatError("inconsistent successive approximation progression")
		}
		approx[k] = uint8(al + 1)
	}
	return nil
}

// restart consumes the RSTn marker that must follow every restart interval
// except the last one, then resets the decoder state (sections F.2.1.3.1 and
// G.1.2.2).
func (d *coeffDecoder) restart(mcu, totalMCU int, expectedRST *uint8, dc *[coeffMaxComponents]int32) error {
	if d.ri == 0 || mcu%d.ri != 0 || mcu >= totalMCU {
		return nil
	}
	if err := d.readFull(d.tmp[:2]); err != nil {
		return err
	}
	if d.tmp[0] != 0xff || d.tmp[1] != *expectedRST {
		return coeffFormatError("bad RST marker")
	}
	*expectedRST++
	if *expectedRST == mkRST7+1 {
		*expectedRST = mkRST0
	}
	d.bits = coeffBits{}
	*dc = [coeffMaxComponents]int32{}
	d.eobRun = 0
	return nil
}

type scanParams struct {
	zigStart, zigEnd int32
	ah, al           uint32
}

// decodeBlock decodes one block's contribution to the current scan and stores
// it in zig-zag order.
func (d *coeffDecoder) decodeBlock(blockIndex int, dc *int32, dcTable, acTable *coeffHuffman, p scanParams) error {
	stored := d.coeffs[blockIndex*coeffBlockSize : (blockIndex+1)*coeffBlockSize]
	var b [coeffBlockSize]int32
	if d.progressive {
		for zig := p.zigStart; zig <= p.zigEnd; zig++ {
			b[zig] = int32(stored[zig])
		}
	}

	if p.ah != 0 {
		if err := d.refine(&b, acTable, p.zigStart, p.zigEnd, 1<<p.al); err != nil {
			return err
		}
	} else {
		zig := p.zigStart
		if zig == 0 {
			zig++
			// Decode the DC coefficient, as specified in section F.2.2.1.
			value, err := d.decodeHuffman(dcTable)
			if err != nil {
				return err
			}
			if value > 16 {
				return coeffFormatError("excessive DC component")
			}
			dcDelta, err := d.receiveExtend(value)
			if err != nil {
				return err
			}
			*dc += dcDelta
			b[0] = *dc << p.al
		}

		if zig <= p.zigEnd && d.eobRun > 0 {
			d.eobRun--
		} else {
			// Decode the AC coefficients, as specified in section F.2.2.2.
			for ; zig <= p.zigEnd; zig++ {
				value, err := d.decodeHuffman(acTable)
				if err != nil {
					return err
				}
				val0 := value >> 4
				val1 := value & 0x0f
				if val1 != 0 {
					zig += int32(val0)
					if zig > p.zigEnd {
						break
					}
					ac, err := d.receiveExtend(val1)
					if err != nil {
						return err
					}
					b[zig] = ac << p.al
				} else {
					if val0 != 0x0f {
						if !d.progressive && val0 != 0 {
							// EOBn runs only exist in progressive scans.
							return coeffFormatError("EOB run in sequential scan")
						}
						d.eobRun = uint16(1 << val0)
						if val0 != 0 {
							bits, err := d.decodeBits(int32(val0))
							if err != nil {
								return err
							}
							d.eobRun |= uint16(bits)
						}
						d.eobRun--
						break
					}
					zig += 0x0f
				}
			}
		}
	}

	for zig := p.zigStart; zig <= p.zigEnd; zig++ {
		v := b[zig]
		if v < -32768 || v > 32767 {
			return coeffFormatError("coefficient out of range")
		}
		stored[zig] = int16(v)
	}
	return nil
}

// refine decodes a successive approximation refinement block, as specified in
// section G.1.2.
func (d *coeffDecoder) refine(b *[coeffBlockSize]int32, h *coeffHuffman, zigStart, zigEnd, delta int32) error {
	// Refining a DC component is trivial.
	if zigStart == 0 {
		if zigEnd != 0 {
			return coeffFormatError("bad spectral selection bounds")
		}
		bit, err := d.decodeBit()
		if err != nil {
			return err
		}
		if bit {
			b[0] |= delta
		}
		return nil
	}

	// Refining AC components is more complicated; see sections G.1.2.2 and G.1.2.3.
	zig := zigStart
	if d.eobRun == 0 {
	loop:
		for ; zig <= zigEnd; zig++ {
			z := int32(0)
			value, err := d.decodeHuffman(h)
			if err != nil {
				return err
			}
			val0 := value >> 4
			val1 := value & 0x0f

			switch val1 {
			case 0:
				if val0 != 0x0f {
					d.eobRun = uint16(1 << val0)
					if val0 != 0 {
						var bits uint32
						bits, err = d.decodeBits(int32(val0))
						if err != nil {
							return err
						}
						d.eobRun |= uint16(bits)
					}
					break loop
				}
			case 1:
				z = delta
				var bit bool
				bit, err = d.decodeBit()
				if err != nil {
					return err
				}
				if !bit {
					z = -z
				}
			default:
				return coeffFormatError("unexpected Huffman code")
			}

			zig, err = d.refineNonZeroes(b, zig, zigEnd, int32(val0), delta)
			if err != nil {
				return err
			}
			if zig > zigEnd {
				return coeffFormatError("too many coefficients")
			}
			if z != 0 {
				b[zig] = z
			}
		}
	}
	if d.eobRun > 0 {
		d.eobRun--
		if _, err := d.refineNonZeroes(b, zig, zigEnd, -1, delta); err != nil {
			return err
		}
	}
	return nil
}

// refineNonZeroes refines non-zero entries of b in zig-zag order. If nz >= 0,
// the first nz zero entries are skipped over.
func (d *coeffDecoder) refineNonZeroes(b *[coeffBlockSize]int32, zig, zigEnd, nz, delta int32) (int32, error) {
	for ; zig <= zigEnd; zig++ {
		if b[zig] == 0 {
			if nz == 0 {
				break
			}
			nz--
			continue
		}
		bit, err := d.decodeBit()
		if err != nil {
			return 0, err
		}
		if !bit {
			continue
		}
		if b[zig] >= 0 {
			b[zig] += delta
		} else {
			b[zig] -= delta
		}
	}
	return zig, nil
}
