package imagelimit

import "testing"

func TestPixelBytesChecksBeforeMultiplication(t *testing.T) {
	for _, tt := range []struct{ w, h, bpp, want int }{
		{6000, 4000, 4, 96000000}, {16384, 1, 8, 131072}, {1, 1, 3, 3},
		{6000, 4001, 4, 0}, {16385, 1, 1, 0}, {0, 1, 4, 0}, {-1, 1, 4, 0},
		{int(^uint(0) >> 1), 8, 4, 0}, {1, 1, 0, 0}, {1, 1, 9, 0},
	} {
		got, err := PixelBytes(tt.w, tt.h, tt.bpp)
		if tt.want == 0 {
			if err == nil {
				t.Errorf("%+v: accepted unsafe size", tt)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("%+v: got %d, %v", tt, got, err)
		}
	}
}

func TestEncodedSizeLimit(t *testing.T) {
	for _, tt := range []struct {
		n     int
		valid bool
	}{{0, false}, {-1, false}, {1, true}, {64 * 1024 * 1024, true}, {64*1024*1024 + 1, false}} {
		if err := CheckEncodedSize(tt.n); (err == nil) != tt.valid {
			t.Errorf("%d: got %v", tt.n, err)
		}
	}
}
