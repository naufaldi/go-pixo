package jpeg

import (
	"bytes"
	"math/bits"
)

// losslessScan describes one output scan over the input's coefficients.
// comps holds frame component indexes; ss/se/ah/al are the SOS parameters.
type losslessScan struct {
	comps          []int
	ss, se, ah, al int
}

type losslessScanKind uint8

const (
	scanSequential losslessScanKind = iota
	scanDCFirst
	scanDCRefine
	scanACFirst
	scanACRefine
)

func (s losslessScan) kind(progressive bool) losslessScanKind {
	switch {
	case !progressive:
		return scanSequential
	case s.ss == 0 && s.ah == 0:
		return scanDCFirst
	case s.ss == 0:
		return scanDCRefine
	case s.ah == 0:
		return scanACFirst
	default:
		return scanACRefine
	}
}

const (
	// maxEOBRun is the largest run an EOB14 symbol can code.
	maxEOBRun = 0x7fff
	// maxCorrectionBits bounds buffered refinement bits, as in libjpeg.
	maxCorrectionBits = 1000
	// Code length limits for 8-bit DCT data (table F.1 and F.2).
	maxDCBits = 11
	maxACBits = 10
)

var errCoefficientRange = coeffFormatError("coefficient exceeds the 8-bit JPEG range")

// huffmanSlot maps a frame component to a table destination: the first
// component (luma) uses tables 0, all others share tables 1. This keeps
// sequential output within the baseline two-table limit.
func huffmanSlot(component int) int {
	if component == 0 {
		return 0
	}
	return 1
}

// entropyEncoder emits scans for the lossless optimizer. The same code runs
// twice per scan: once with gather set to count symbol frequencies, then
// with optimized tables to write the entropy-coded data.
type entropyEncoder struct {
	d       *coeffDecoder
	gather  bool
	out     *bytes.Buffer
	bw      *BitWriter
	dcFreq  [2][256]int
	acFreq  [2][256]int
	dcCodes [2][256]HuffCode
	acCodes [2][256]HuffCode
	// Per-scan state.
	pred            [coeffMaxComponents]int32
	eobRun          int
	corrections     []byte               // Refinement bits buffered behind a pending EOB run.
	pending         [coeffBlockSize]byte // Correction bits of the current block.
	acSlot          int
	sequential      bool                   // Sequential scans end blocks with EOB and never use EOB runs.
	restartInterval int                    // MCUs per restart interval; 0 disables restarts.
	extraBits       int64                  // Non-Huffman bits counted during a gather pass.
	stats           map[scanKey]*scanStats // Cached counting passes; nil disables caching.
}

func (e *entropyEncoder) emitDC(slot int, symbol int) error {
	if e.gather {
		e.dcFreq[slot][symbol]++
		return nil
	}
	code := e.dcCodes[slot][symbol]
	if code.Length == 0 {
		return coeffFormatError("internal: missing DC Huffman code")
	}
	return e.bw.Write(uint32(code.Code), code.Length)
}

func (e *entropyEncoder) emitAC(symbol int) error {
	if e.gather {
		e.acFreq[e.acSlot][symbol]++
		return nil
	}
	code := e.acCodes[e.acSlot][symbol]
	if code.Length == 0 {
		return coeffFormatError("internal: missing AC Huffman code")
	}
	return e.bw.Write(uint32(code.Code), code.Length)
}

func (e *entropyEncoder) emitBits(value uint32, n int) error {
	if e.gather || n == 0 {
		e.extraBits += int64(n)
		return nil
	}
	return e.bw.Write(value, uint8(n))
}

func (e *entropyEncoder) emitCorrections(corrections []byte) error {
	if e.gather {
		e.extraBits += int64(len(corrections))
		return nil
	}
	for _, bit := range corrections {
		if err := e.bw.Write(uint32(bit), 1); err != nil {
			return err
		}
	}
	return nil
}

// magnitude returns the JPEG size category of v and the bits that code it.
func magnitude(v int32) (int, uint32) {
	if v < 0 {
		n := bits.Len32(uint32(-v))
		return n, uint32(v-1) & (1<<uint(n) - 1)
	}
	return bits.Len32(uint32(v)), uint32(v)
}

// flushEOBRun emits a pending EOB run and the correction bits buffered
// behind it (section G.1.2.2 / G.1.2.3).
func (e *entropyEncoder) flushEOBRun() error {
	if e.eobRun == 0 {
		return nil
	}
	n := bits.Len(uint(e.eobRun)) - 1
	if err := e.emitAC(n << 4); err != nil {
		return err
	}
	if err := e.emitBits(uint32(e.eobRun), n); err != nil {
		return err
	}
	e.eobRun = 0
	if err := e.emitCorrections(e.corrections); err != nil {
		return err
	}
	e.corrections = e.corrections[:0]
	return nil
}

// encodeScan runs one pass of a scan over every block it covers.
func (e *entropyEncoder) encodeScan(scan losslessScan, kind losslessScanKind) error {
	d := e.d
	e.pred = [coeffMaxComponents]int32{}
	e.eobRun = 0
	e.corrections = e.corrections[:0]
	e.sequential = kind == scanSequential
	if kind == scanACFirst || kind == scanACRefine {
		e.acSlot = huffmanSlot(scan.comps[0])
	}
	rst := 0
	mcu := 0

	if len(scan.comps) == 1 {
		ci := scan.comps[0]
		c := &d.comp[ci]
		total := c.compW * c.compH
		for by := 0; by < c.compH; by++ {
			row := c.offset + by*c.blocksW
			for bx := 0; bx < c.compW; bx++ {
				start := (row + bx) * coeffBlockSize
				if err := e.encodeBlock(scan, kind, ci, d.coeffs[start:start+coeffBlockSize]); err != nil {
					return err
				}
				mcu++
				if err := e.maybeRestart(mcu, total, &rst); err != nil {
					return err
				}
			}
		}
		return e.flushEOBRun()
	}

	total := d.mxx * d.myy
	for my := 0; my < d.myy; my++ {
		for mx := 0; mx < d.mxx; mx++ {
			for _, ci := range scan.comps {
				c := &d.comp[ci]
				for j := 0; j < c.h*c.v; j++ {
					bx := c.h*mx + j%c.h
					by := c.v*my + j/c.h
					start := (c.offset + by*c.blocksW + bx) * coeffBlockSize
					if err := e.encodeBlock(scan, kind, ci, d.coeffs[start:start+coeffBlockSize]); err != nil {
						return err
					}
				}
			}
			mcu++
			if err := e.maybeRestart(mcu, total, &rst); err != nil {
				return err
			}
		}
	}
	return e.flushEOBRun()
}

func (e *entropyEncoder) maybeRestart(mcu, total int, rst *int) error {
	if e.restartInterval == 0 || mcu%e.restartInterval != 0 || mcu >= total {
		return nil
	}
	if err := e.flushEOBRun(); err != nil {
		return err
	}
	e.pred = [coeffMaxComponents]int32{}
	if e.gather {
		return nil
	}
	if err := e.bw.Flush(); err != nil {
		return err
	}
	e.out.WriteByte(0xff)
	e.out.WriteByte(byte(mkRST0 + *rst))
	*rst = (*rst + 1) & 7
	return nil
}

func (e *entropyEncoder) encodeBlock(scan losslessScan, kind losslessScanKind, ci int, block []int16) error {
	switch kind {
	case scanSequential:
		e.acSlot = huffmanSlot(ci)
		if err := e.encodeDC(ci, int32(block[0])); err != nil {
			return err
		}
		return e.encodeACFirst(block, 1, 63, 0)
	case scanDCFirst:
		return e.encodeDC(ci, int32(block[0])>>uint(scan.al))
	case scanDCRefine:
		return e.emitBits(uint32(block[0]>>uint(scan.al))&1, 1)
	case scanACFirst:
		return e.encodeACFirst(block, scan.ss, scan.se, scan.al)
	default:
		return e.encodeACRefine(block, scan.ss, scan.se, scan.al)
	}
}

func (e *entropyEncoder) encodeDC(ci int, value int32) error {
	diff := value - e.pred[ci]
	e.pred[ci] = value
	n, b := magnitude(diff)
	if n > maxDCBits {
		return errCoefficientRange
	}
	if err := e.emitDC(huffmanSlot(ci), n); err != nil {
		return err
	}
	return e.emitBits(b, n)
}

// lastSignificant returns the index in band of the last coefficient whose
// magnitude is at least 1<<al, or -1 when the band codes as all zeros.
func lastSignificant(band []int16, al int) int {
	threshold := int16(1) << uint(al)
	for k := len(band) - 1; k >= 0; k-- {
		if v := band[k]; v >= threshold || v <= -threshold {
			return k
		}
	}
	return -1
}

// endBand counts a block whose remaining coefficients are all zero: an EOB
// symbol in sequential scans, or one more block in the pending EOB run.
func (e *entropyEncoder) endBand() error {
	if e.sequential {
		return e.emitAC(0x00)
	}
	e.eobRun++
	if e.eobRun == maxEOBRun {
		return e.flushEOBRun()
	}
	return nil
}

// encodeACFirst codes the first (or only) pass of a spectral band. Sequential
// scans call it with ss=1, se=63, al=0 and never carry EOB runs.
func (e *entropyEncoder) encodeACFirst(block []int16, ss, se, al int) error {
	band := block[ss : se+1]
	last := lastSignificant(band, al)
	if last < 0 {
		return e.endBand()
	}
	r := 0
	for _, coefficient := range band[:last+1] {
		v := int32(coefficient)
		// Successive approximation shifts the magnitude, rounding toward
		// zero (section G.1.2.1), not an arithmetic shift.
		if v < 0 {
			v = -((-v) >> uint(al))
		} else {
			v >>= uint(al)
		}
		if v == 0 {
			r++
			continue
		}
		if e.eobRun > 0 {
			if err := e.flushEOBRun(); err != nil {
				return err
			}
		}
		for r > 15 {
			if err := e.emitAC(0xf0); err != nil {
				return err
			}
			r -= 16
		}
		n, b := magnitude(v)
		if n > maxACBits {
			return errCoefficientRange
		}
		if err := e.emitAC(r<<4 | n); err != nil {
			return err
		}
		if err := e.emitBits(b, n); err != nil {
			return err
		}
		r = 0
	}
	if last < len(band)-1 {
		return e.endBand()
	}
	return nil
}

// encodeACRefine codes a successive-approximation refinement pass
// (section G.1.2.3), following libjpeg's encode_mcu_AC_refine.
func (e *entropyEncoder) encodeACRefine(block []int16, ss, se, al int) error {
	band := block[ss : se+1]
	last := lastSignificant(band, al)
	if last < 0 {
		return e.endBand()
	}
	// eob is the index of the last newly-nonzero coefficient (magnitude 1
	// at this bit position); later nonzero ones only need correction bits.
	eob := -1
	for k, coefficient := range band[:last+1] {
		v := int32(coefficient)
		if v < 0 {
			v = -v
		}
		if v>>uint(al) == 1 {
			eob = k
		}
	}

	pending := e.pending[:0]
	r := 0
	for k, coefficient := range band[:last+1] {
		v := int32(coefficient)
		if v < 0 {
			v = -v
		}
		v >>= uint(al)
		if v == 0 {
			r++
			continue
		}
		// Emit any required ZRLs, but not if they can be folded into EOB.
		for r > 15 && k <= eob {
			if err := e.flushEOBRun(); err != nil {
				return err
			}
			if err := e.emitAC(0xf0); err != nil {
				return err
			}
			r -= 16
			if err := e.emitCorrections(pending); err != nil {
				return err
			}
			pending = pending[:0]
		}
		// A previously nonzero coefficient only needs a correction bit.
		if v > 1 {
			pending = append(pending, byte(v&1))
			continue
		}
		if err := e.flushEOBRun(); err != nil {
			return err
		}
		if err := e.emitAC(r<<4 | 1); err != nil {
			return err
		}
		sign := uint32(1)
		if coefficient < 0 {
			sign = 0
		}
		if err := e.emitBits(sign, 1); err != nil {
			return err
		}
		if err := e.emitCorrections(pending); err != nil {
			return err
		}
		pending = pending[:0]
		r = 0
	}
	if r > 0 || len(pending) > 0 || last < len(band)-1 {
		e.eobRun++
		e.corrections = append(e.corrections, pending...)
		if e.eobRun == maxEOBRun || len(e.corrections) > maxCorrectionBits-coeffBlockSize+1 {
			return e.flushEOBRun()
		}
	}
	return nil
}
