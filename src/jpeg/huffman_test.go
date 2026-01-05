package jpeg

import (
	"testing"
)

func TestNewHuffmanTables(t *testing.T) {
	ht := NewHuffmanTables()

	t.Run("DC Luminance", func(t *testing.T) {
		// Category 0 (DC difference 0) in standard table should have some code
		code, length := ht.EncodeDC(0, true)
		if length == 0 {
			t.Error("DC luminance category 0 should have a code")
		}
		// Standard DC luminance: cat 0 is 2 bits, value 00 (binary)
		if length != 2 || code != 0 {
			t.Errorf("got (%d, %d), want (0, 2)", code, length)
		}
	})

	t.Run("AC Luminance", func(t *testing.T) {
		// EOB (run=0, size=0) in standard table
		code, length := ht.EncodeAC(0, 0, true)
		if length == 0 {
			t.Error("AC luminance EOB should have a code")
		}
		// Standard AC luminance: EOB is 4 bits, value 1010 (10)
		if length != 4 || code != 10 {
			t.Errorf("EOB: got (%d, %d), want (10, 4)", code, length)
		}

		// ZRL (run=15, size=0) in standard table
		code, length = ht.EncodeAC(15, 0, true)
		if length == 0 {
			t.Error("AC luminance ZRL should have a code")
		}
		// Standard AC luminance: ZRL is 11 bits, value 11111111001 (binary) = 2041
		if length != 11 || code != 2041 {
			t.Errorf("ZRL: got (%d, %d), want (2041, 11)", code, length)
		}
	})

	t.Run("Chrominance", func(t *testing.T) {
		_, length := ht.EncodeDC(0, false)
		if length == 0 {
			t.Error("DC chrominance category 0 should have a code")
		}
		_, length = ht.EncodeAC(0, 0, false)
		if length == 0 {
			t.Error("AC chrominance EOB should have a code")
		}
	})
}
