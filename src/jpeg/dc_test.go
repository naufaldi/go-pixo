package jpeg

import (
	"testing"
)

func TestCategory(t *testing.T) {
	tests := []struct {
		val      int16
		expected uint8
	}{
		{0, 0},
		{1, 1},
		{-1, 1},
		{2, 2},
		{-2, 2},
		{3, 2},
		{-3, 2},
		{4, 3},
		{-4, 3},
		{7, 3},
		{-7, 3},
		{8, 4},
		{255, 8},
		{-255, 8},
		{256, 9},
		{2047, 11},
		{-2047, 11},
	}

	for _, tt := range tests {
		got := Category(tt.val)
		if got != tt.expected {
			t.Errorf("Category(%d): got %d, want %d", tt.val, got, tt.expected)
		}
	}
}

func TestEncodeValue(t *testing.T) {
	tests := []struct {
		val      int16
		ebits    uint16
		elen     uint8
	}{
		{0, 0, 0},
		{1, 1, 1},
		{-1, 0, 1},
		{2, 2, 2},
		{3, 3, 2},
		{-2, 1, 2},
		{-3, 0, 2},
		{7, 7, 3},
		{-7, 0, 3},
	}

	for _, tt := range tests {
		bits, len := EncodeValue(tt.val)
		if bits != tt.ebits || len != tt.elen {
			t.Errorf("EncodeValue(%d): got (%d, %d), want (%d, %d)", tt.val, bits, len, tt.ebits, tt.elen)
		}
	}
}

func TestDC_RoundTrip(t *testing.T) {
	testValues := []int16{0, 1, -1, 2, -2, 3, -3, 4, -4, 100, -100, 1000, -1000, 2047, -2047}

	for _, val := range testValues {
		cat, bits, _ := EncodeDC(val, 0)
		recovered := DecodeDC(cat, bits)
		if recovered != val {
			t.Errorf("RoundTrip %d: got %d (cat %d, bits %b)", val, recovered, cat, bits)
		}
	}
}

func TestEncodeDC_Differential(t *testing.T) {
	// Test DC=100, prevDC=50 -> diff=50
	cat, bits, len := EncodeDC(100, 50)
	if cat != 6 {
		t.Errorf("cat: got %d, want 6", cat)
	}
	if len != 6 {
		t.Errorf("len: got %d, want 6", len)
	}
	if bits != 50 {
		t.Errorf("bits: got %d, want 50", bits)
	}

	// Test DC=50, prevDC=100 -> diff=-50
	cat, bits, len = EncodeDC(50, 100)
	if cat != 6 {
		t.Errorf("cat: got %d, want 6", cat)
	}
	// -50 in 6 bits one's complement: -50-1 = -51 = 11001101... masked to 6 bits = 001101 = 13
	if bits != 13 {
		t.Errorf("bits: got %d, want 13", bits)
	}
}
