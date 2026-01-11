package jpeg

import "testing"

func TestSubsample420(t *testing.T) {
	width, height := 4, 4
	cb := make([]byte, width*height)
	cr := make([]byte, width*height)

	// Fill with some values
	for i := range cb {
		cb[i] = uint8(i * 10)
		cr[i] = uint8(i*10 + 5)
	}

	subCb, subCr := Subsample420(cb, cr, width, height)

	expectedWidth := 2
	expectedHeight := 2
	if len(subCb) != expectedWidth*expectedHeight {
		t.Fatalf("expected subCb length %d, got %d", expectedWidth*expectedHeight, len(subCb))
	}
	if len(subCr) != expectedWidth*expectedHeight {
		t.Fatalf("expected subCr length %d, got %d", expectedWidth*expectedHeight, len(subCr))
	}

	// Check a value (average of (0,0), (1,0), (0,1), (1,1))
	// cb values: 0, 10, 40, 50. Average: (0+10+40+50)/4 = 100/4 = 25
	if subCb[0] != 25 {
		t.Errorf("expected subCb[0] to be 25, got %d", subCb[0])
	}
}

func TestSubsample420_OddDimensions(t *testing.T) {
	width, height := 3, 3
	cb := make([]byte, width*height)
	cr := make([]byte, width*height)

	subCb, _ := Subsample420(cb, cr, width, height)

	expectedWidth := 2
	expectedHeight := 2
	if len(subCb) != expectedWidth*expectedHeight {
		t.Fatalf("expected subCb length %d, got %d", expectedWidth*expectedHeight, len(subCb))
	}
}
