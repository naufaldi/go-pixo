package jpeg

import (
	"bytes"
	"errors"
)

// Parallel encoding splits an image into bands of whole MCU rows. An MCU
// reads only its own pixels (blocks past the right or bottom edge replicate
// the image's last column or row), and quantization, including trellis, works
// on one block at a time, so a band's coefficients never depend on another
// band. Everything that crosses blocks (DC prediction, Huffman statistics,
// progressive scans) runs once, in CoefficientFrame.Encode, over all bands.
// The output is therefore byte-identical to Encoder.Encode for any split.

// ErrInvalidBand reports a band that does not fit the image's MCU rows or
// coefficient layout.
var ErrInvalidBand = errors.New("jpeg: invalid coefficient band")

// MCUHeight returns the pixel height of one MCU row: 16 for 4:2:0 color,
// otherwise 8.
func (e *Encoder) MCUHeight() int {
	if e.Options.Subsampling == Subsampling420 && e.Options.ColorType == ColorRGB {
		return 16
	}
	return 8
}

// MCURows returns the number of MCU rows in the image.
func (e *Encoder) MCURows() int {
	h := e.MCUHeight()
	return (e.Options.Height + h - 1) / h
}

// BandPixelRows returns the first image row and the number of image rows
// that EncodeBand reads for MCU rows [firstRow, firstRow+rows).
func (e *Encoder) BandPixelRows(firstRow, rows int) (y, height int) {
	h := e.MCUHeight()
	y = firstRow * h
	return y, min((firstRow+rows)*h, e.Options.Height) - y
}

// EncodeBand quantizes MCU rows [firstRow, firstRow+rows). pixels holds only
// the image rows BandPixelRows reports, in the encoder's color type. The
// result holds, for each component in frame order, the band's block rows in
// zig-zag order; pass it to CoefficientFrame.SetBand with the same firstRow.
func (e *Encoder) EncodeBand(pixels []byte, firstRow, rows int) ([]int16, error) {
	if err := validateOptions(e.Options); err != nil {
		return nil, err
	}
	// Layout only: a band never holds the whole image's coefficients.
	frame, err := e.frameLayout()
	if err != nil {
		return nil, err
	}
	if err := frame.checkBand(firstRow, rows); err != nil {
		return nil, err
	}
	out := make([]int16, rows*frame.rowCoefficients())
	segments := make([][]int16, frame.d.nComp)
	offset := 0
	for ci := range segments {
		n := rows * frame.componentRowCoefficients(ci)
		segments[ci] = out[offset : offset+n : offset+n]
		offset += n
	}
	if err := e.encodeBandInto(pixels, firstRow, rows, frame.d, segments); err != nil {
		return nil, err
	}
	return out, nil
}

// encodeBandInto quantizes MCU rows [firstRow, firstRow+rows) of pixels (the
// band's image rows) into segments, one per component, each holding that
// component's block rows of the band.
func (e *Encoder) encodeBandInto(pixels []byte, firstRow, rows int, d *coeffDecoder, segments [][]int16) error {
	bpp := 1
	if e.Options.ColorType == ColorRGB {
		bpp = 3
	}
	_, height := e.BandPixelRows(firstRow, rows)
	width := e.Options.Width
	if len(pixels) != width*height*bpp {
		return ErrInvalidDataLength
	}
	qt := NewQuantizationTables(e.Options.Quality)
	store := func(ci, blockRow, blockCol int, block [64]float32, isLuminance bool) {
		zigzag := e.quantizeBlock(block, isLuminance, qt)
		at := (blockRow*d.comp[ci].blocksW + blockCol) * coeffBlockSize
		copy(segments[ci][at:at+coeffBlockSize], zigzag[:])
	}
	// Band-relative coordinates: the band's pixel rows start at y = 0, and
	// only the last band reaches the image's bottom edge, where clamping to
	// height-1 replicates the same row as a whole-image encode.
	if e.MCUHeight() == 16 {
		for my := 0; my < rows; my++ {
			for mx := 0; mx < d.mxx; mx++ {
				yBlocks, cb, cr := ExtractMCU420(pixels, width, height, mx*16, my*16)
				for i := range yBlocks {
					store(0, my*2+i/2, mx*2+i%2, yBlocks[i], true)
				}
				store(1, my, mx, cb, false)
				store(2, my, mx, cr, false)
			}
		}
		return nil
	}
	for my := 0; my < rows; my++ {
		for mx := 0; mx < d.mxx; mx++ {
			yBlock, cb, cr := ExtractBlock(pixels, width, height, mx*8, my*8, e.Options.ColorType)
			store(0, my, mx, yBlock, true)
			if e.Options.ColorType == ColorRGB {
				store(1, my, mx, cb, false)
				store(2, my, mx, cr, false)
			}
		}
	}
	return nil
}

// CoefficientFrame holds the quantized coefficients of a whole image while
// bands are filled in, then writes the JPEG.
type CoefficientFrame struct {
	e      *Encoder
	d      *coeffDecoder
	header []byte // SOI..SOF0, as encodeSequential writes them.
}

// NewCoefficientFrame allocates zeroed coefficient storage for the
// encoder's image.
func (e *Encoder) NewCoefficientFrame() (*CoefficientFrame, error) {
	frame, err := e.frameLayout()
	if err != nil {
		return nil, err
	}
	frame.d.coeffs = make([]int16, frame.d.totalBlocks*coeffBlockSize)
	return frame, nil
}

// frameLayout returns a frame with the encoder's layout and no coefficient
// storage.
func (e *Encoder) frameLayout() (*CoefficientFrame, error) {
	if err := validateOptions(e.Options); err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if _, err := e.writeFrameHeader(buf); err != nil {
		return nil, err
	}
	// Parse the header this encoder writes rather than rebuilding its
	// layout, so the frame matches what the lossless optimizer reads back.
	d, err := readFrameHeader(buf.Bytes())
	if err != nil {
		return nil, err
	}
	return &CoefficientFrame{e: e, d: d, header: buf.Bytes()}, nil
}

// MCURows returns the number of MCU rows in the frame.
func (f *CoefficientFrame) MCURows() int { return f.d.myy }

// componentRowCoefficients returns the coefficients one MCU row holds for a
// component.
func (f *CoefficientFrame) componentRowCoefficients(ci int) int {
	c := &f.d.comp[ci]
	return c.v * c.blocksW * coeffBlockSize
}

// rowCoefficients returns the coefficients one MCU row holds.
func (f *CoefficientFrame) rowCoefficients() int {
	n := 0
	for ci := 0; ci < f.d.nComp; ci++ {
		n += f.componentRowCoefficients(ci)
	}
	return n
}

// BandLength returns the number of coefficients EncodeBand returns for rows
// MCU rows.
func (f *CoefficientFrame) BandLength(rows int) int { return rows * f.rowCoefficients() }

func (f *CoefficientFrame) checkBand(firstRow, rows int) error {
	if firstRow < 0 || rows < 1 || rows > f.d.myy || firstRow > f.d.myy-rows {
		return ErrInvalidBand
	}
	return nil
}

// BandSegments returns views of the frame's storage for MCU rows
// [firstRow, firstRow+rows), one per component in frame order. EncodeBand's
// result is these segments concatenated, so a caller may copy a band into
// them directly instead of calling SetBand.
func (f *CoefficientFrame) BandSegments(firstRow, rows int) ([][]int16, error) {
	if err := f.checkBand(firstRow, rows); err != nil {
		return nil, err
	}
	segments := make([][]int16, f.d.nComp)
	for ci := range segments {
		c := &f.d.comp[ci]
		start := (c.offset + firstRow*c.v*c.blocksW) * coeffBlockSize
		end := start + rows*f.componentRowCoefficients(ci)
		segments[ci] = f.d.coeffs[start:end:end]
	}
	return segments, nil
}

// SetBand copies a band returned by EncodeBand for MCU rows starting at
// firstRow into the frame.
func (f *CoefficientFrame) SetBand(firstRow int, band []int16) error {
	rowLength := f.rowCoefficients()
	if len(band) == 0 || len(band)%rowLength != 0 {
		return ErrInvalidBand
	}
	segments, err := f.BandSegments(firstRow, len(band)/rowLength)
	if err != nil {
		return err
	}
	for _, segment := range segments {
		band = band[copy(segment, band):]
	}
	return nil
}

// Encode writes the frame as a JPEG with the encoder's options: optimized
// Huffman tables and the smallest sequential or progressive layout when
// OptimizeHuffman or Progressive is set, otherwise a baseline file with the
// standard tables. Rows without a band encode as zero coefficients (flat
// gray); callers must fill every row.
func (f *CoefficientFrame) Encode() ([]byte, error) {
	if !f.e.Options.Progressive && !f.e.Options.OptimizeHuffman {
		return f.encodeBaseline()
	}
	layout := sequentialLayouts
	if f.e.Options.Progressive {
		layout = progressiveLayouts
	}
	// The encoder writes no EXIF, so orientation is the default.
	return optimizeCoefficients(f.d, 1, layout)
}

// encodeBaseline writes the same file as encodeSequential from the stored
// coefficients.
func (f *CoefficientFrame) encodeBaseline() ([]byte, error) {
	d := f.d
	buf := new(bytes.Buffer)
	buf.Grow(len(f.header) + len(d.coeffs)/4)
	buf.Write(f.header)
	ht, err := f.e.writeStandardTables(buf)
	if err != nil {
		return nil, err
	}
	if err = WriteSOS(buf, f.e.Options.ColorType); err != nil {
		return nil, err
	}
	bw := NewBitWriter(buf)
	var prevDC [coeffMaxComponents]int16
	var block [coeffBlockSize]int16
	for my := 0; my < d.myy; my++ {
		for mx := 0; mx < d.mxx; mx++ {
			for ci := 0; ci < d.nComp; ci++ {
				c := &d.comp[ci]
				for by := 0; by < c.v; by++ {
					for bx := 0; bx < c.h; bx++ {
						at := (c.offset + (my*c.v+by)*c.blocksW + mx*c.h + bx) * coeffBlockSize
						copy(block[:], d.coeffs[at:at+coeffBlockSize])
						if prevDC[ci], err = writeBlock(bw, &block, prevDC[ci], ci == 0, ht); err != nil {
							return nil, err
						}
					}
				}
			}
		}
	}
	if err := bw.Flush(); err != nil {
		return nil, err
	}
	if err := WriteEOI(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeBands writes the JPEG for bands returned by EncodeBand that cover
// every MCU row in order.
func (e *Encoder) EncodeBands(bands [][]int16) ([]byte, error) {
	frame, err := e.NewCoefficientFrame()
	if err != nil {
		return nil, err
	}
	row := 0
	for _, band := range bands {
		if err := frame.SetBand(row, band); err != nil {
			return nil, err
		}
		row += len(band) / frame.rowCoefficients()
	}
	if row != frame.MCURows() {
		return nil, ErrInvalidBand
	}
	return frame.Encode()
}
