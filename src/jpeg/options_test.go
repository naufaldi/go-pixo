package jpeg

import "testing"

func TestOptions_Presets(t *testing.T) {
	width, height := 100, 100
	quality := uint8(80)

	fast := FastOptions(width, height, quality)
	if fast.Subsampling != Subsampling444 {
		t.Errorf("FastOptions should use 4:4:4")
	}
	if fast.OptimizeHuffman {
		t.Errorf("FastOptions should not optimize Huffman")
	}

	balanced := BalancedOptions(width, height, quality)
	if balanced.Subsampling != Subsampling420 {
		t.Errorf("BalancedOptions should use 4:2:0")
	}

	max := MaxOptions(width, height, quality)
	if !max.OptimizeHuffman {
		t.Errorf("MaxOptions should optimize Huffman")
	}
	if !max.Progressive {
		t.Errorf("MaxOptions should be progressive")
	}
}

func TestOptionsBuilder_Chain(t *testing.T) {
	opts := NewOptionsBuilder(200, 150).
		Quality(95).
		Subsampling(Subsampling444).
		OptimizeHuffman(false).
		Progressive(false).
		Build()

	if opts.Width != 200 || opts.Height != 150 {
		t.Errorf("Width/Height mismatch")
	}
	if opts.Quality != 95 {
		t.Errorf("Quality mismatch")
	}
	if opts.Subsampling != Subsampling444 {
		t.Errorf("Subsampling mismatch")
	}
}
