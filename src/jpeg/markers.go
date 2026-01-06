package jpeg

import (
	"encoding/binary"
	"io"
)

// WriteSOI writes the Start of Image marker.
func WriteSOI(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, SOI)
}

// WriteEOI writes the End of Image marker.
func WriteEOI(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, EOI)
}

// WriteAPP0 writes the JFIF APP0 segment.
func WriteAPP0(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, APP0); err != nil {
		return err
	}
	// Length (16), Identifier ("JFIF\0"), Version (1.1), Units (0), Xdensity (1), Ydensity (1), Xthumbnail (0), Ythumbnail (0)
	length := uint16(16)
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}
	if _, err := w.Write([]byte("JFIF\x00")); err != nil {
		return err
	}
	// Version 1.1, Units 0, Xdensity 1, Ydensity 1, Xthumb 0, Ythumb 0
	if _, err := w.Write([]byte{0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00}); err != nil {
		return err
	}
	return nil
}

// WriteDQT writes a Define Quantization Table segment.
func WriteDQT(w io.Writer, tableID uint8, table [64]uint8) error {
	if err := binary.Write(w, binary.BigEndian, DQT); err != nil {
		return err
	}
	// Length (2 + 1 + 64 = 67)
	length := uint16(67)
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}
	// Pq (0 for 8-bit), Tq (table ID)
	if err := binary.Write(w, binary.BigEndian, tableID&0x0F); err != nil {
		return err
	}
	// Write table in zigzag order as per spec
	for i := 0; i < 64; i++ {
		if err := binary.Write(w, binary.BigEndian, table[Zigzag[i]]); err != nil {
			return err
		}
	}
	return nil
}

// WriteSOF0 writes a Start of Frame (Baseline DCT) segment.
func WriteSOF0(w io.Writer, width, height uint16, colorType ColorType, subsampling Subsampling) error {
	return writeSOF(w, SOF0, width, height, colorType, subsampling)
}

// WriteSOF2 writes a Start of Frame (Progressive DCT) segment.
func WriteSOF2(w io.Writer, width, height uint16, colorType ColorType, subsampling Subsampling) error {
	return writeSOF(w, SOF2, width, height, colorType, subsampling)
}

func writeSOF(w io.Writer, marker uint16, width, height uint16, colorType ColorType, subsampling Subsampling) error {
	if err := binary.Write(w, binary.BigEndian, marker); err != nil {
		return err
	}

	numComponents := uint8(colorType)
	// Length (2 + 1 + 2 + 2 + 1 + numComponents * 3)
	length := uint16(8 + uint16(numComponents)*3)
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}

	// Data precision (8 bits)
	if err := binary.Write(w, binary.BigEndian, uint8(8)); err != nil {
		return err
	}
	// Image height
	if err := binary.Write(w, binary.BigEndian, height); err != nil {
		return err
	}
	// Image width
	if err := binary.Write(w, binary.BigEndian, width); err != nil {
		return err
	}
	// Number of components
	if err := binary.Write(w, binary.BigEndian, numComponents); err != nil {
		return err
	}

	if colorType == ColorGrayscale {
		// Component ID (1), Subsampling (1x1), QTable ID (0)
		if _, err := w.Write([]byte{0x01, 0x11, 0x00}); err != nil {
			return err
		}
	} else {
		// Component Y: ID (1), Subsampling (2x2 or 1x1), QTable ID (0)
		ySub := uint8(0x11)
		if subsampling == Subsampling420 {
			ySub = 0x22
		}
		if _, err := w.Write([]byte{0x01, ySub, 0x00}); err != nil {
			return err
		}
		// Component Cb: ID (2), Subsampling (1x1), QTable ID (1)
		if _, err := w.Write([]byte{0x02, 0x11, 0x01}); err != nil {
			return err
		}
		// Component Cr: ID (3), Subsampling (1x1), QTable ID (1)
		if _, err := w.Write([]byte{0x03, 0x11, 0x01}); err != nil {
			return err
		}
	}

	return nil
}

// WriteDHT writes a Define Huffman Table segment.
func WriteDHT(w io.Writer, tableID uint8, bits [16]uint8, vals []uint8) error {
	if err := binary.Write(w, binary.BigEndian, DHT); err != nil {
		return err
	}

	// Length (2 + 1 + 16 + len(vals))
	length := uint16(19 + len(vals))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}

	// Table class and ID
	if err := binary.Write(w, binary.BigEndian, tableID); err != nil {
		return err
	}

	// Number of codes for each bit length
	if _, err := w.Write(bits[:]); err != nil {
		return err
	}

	// Huffman values
	if _, err := w.Write(vals); err != nil {
		return err
	}

	return nil
}

// WriteSOS writes a Start of Scan segment.
func WriteSOS(w io.Writer, colorType ColorType) error {
	numComponents := uint8(colorType)
	// Length (2 + 1 + numComponents * 2 + 3)
	length := uint16(6 + uint16(numComponents)*2)

	if err := binary.Write(w, binary.BigEndian, SOS); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, numComponents); err != nil {
		return err
	}

	if colorType == ColorGrayscale {
		// Component ID (1), DC/AC Huffman Table ID (0/0)
		if _, err := w.Write([]byte{0x01, 0x00}); err != nil {
			return err
		}
	} else {
		// Component Y: ID (1), DC/AC Table (0/0)
		if _, err := w.Write([]byte{0x01, 0x00}); err != nil {
			return err
		}
		// Component Cb: ID (2), DC/AC Table (1/1)
		if _, err := w.Write([]byte{0x02, 0x11}); err != nil {
			return err
		}
		// Component Cr: ID (3), DC/AC Table (1/1)
		if _, err := w.Write([]byte{0x03, 0x11}); err != nil {
			return err
		}
	}

	// Ss (0), Se (63), Ah/Al (0) - Baseline markers
	if _, err := w.Write([]byte{0x00, 0x3F, 0x00}); err != nil {
		return err
	}

	return nil
}

// WriteSOSProgressive writes a Start of Scan segment for a progressive scan.
func WriteSOSProgressive(w io.Writer, scan *ScanSpec, colorType ColorType) error {
	numComponents := uint8(len(scan.Components))
	// Length (2 + 1 + numComponents * 2 + 3)
	length := uint16(6 + uint16(numComponents)*2)

	if err := binary.Write(w, binary.BigEndian, SOS); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, numComponents); err != nil {
		return err
	}

	for _, compID := range scan.Components {
		if err := binary.Write(w, binary.BigEndian, compID); err != nil {
			return err
		}
		// Huffman table IDs
		// Bits 7-4: DC entropy coding table destination selector
		// Bits 3-0: AC entropy coding table destination selector
		var selector uint8
		if scan.SS == 0 {
			// DC scan
			dcID := uint8(0)
			if compID > 1 {
				dcID = 1
			}
			selector = (dcID << 4)
		} else {
			// AC scan
			acID := uint8(0)
			if compID > 1 {
				acID = 1
			}
			selector = acID
		}

		if err := binary.Write(w, binary.BigEndian, selector); err != nil {
			return err
		}
	}

	// Ss, Se, Ah/Al
	if _, err := w.Write([]byte{scan.SS, scan.SE, (scan.AH << 4) | scan.AL}); err != nil {
		return err
	}

	return nil
}

// WriteDRI writes a Define Restart Interval segment.
func WriteDRI(w io.Writer, interval uint16) error {
	if err := binary.Write(w, binary.BigEndian, DRI); err != nil {
		return err
	}
	// Length (4)
	if err := binary.Write(w, binary.BigEndian, uint16(4)); err != nil {
		return err
	}
	// Restart interval
	return binary.Write(w, binary.BigEndian, interval)
}
