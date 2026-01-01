package compress

import (
	"bytes"
	"fmt"
	"math/bits"
	"testing"
)

func TestWriteCMFValidWindowSizes(t *testing.T) {
	for windowSize := 256; windowSize <= 32768; windowSize *= 2 {
		windowSize := windowSize
		t.Run(fmt.Sprintf("windowSize=%d", windowSize), func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteCMF(&buf, windowSize)
			if err != nil {
				t.Fatalf("WriteCMF(%d) failed: %v", windowSize, err)
			}
			if buf.Len() != 1 {
				t.Fatalf("WriteCMF(%d) wrote %d bytes, want 1", windowSize, buf.Len())
			}

			wlog := bits.TrailingZeros(uint(windowSize))
			cinfo := wlog - 8
			expected := byte(0x08 | byte(cinfo<<4))

			if got := buf.Bytes()[0]; got != expected {
				t.Fatalf("WriteCMF(%d) = 0x%02X, want 0x%02X", windowSize, got, expected)
			}
		})
	}
}

func TestWriteCMFInvalidWindowSize(t *testing.T) {
	invalidSizes := []int{0, 1, 255, 257, 1234, 65536}

	for _, windowSize := range invalidSizes {
		windowSize := windowSize
		t.Run(fmt.Sprintf("windowSize=%d", windowSize), func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteCMF(&buf, windowSize)
			if err != ErrInvalidWindowSize {
				t.Fatalf("WriteCMF(%d) error = %v, want %v", windowSize, err, ErrInvalidWindowSize)
			}
		})
	}
}

func TestWriteFLGComputesFCHECK(t *testing.T) {
	testCases := []struct {
		level   uint8
		expect  byte
		name    string
		window  int
		wantCMF byte
	}{
		{level: 0, expect: 0x01, name: "level=0 (0x78 0x01)", window: 32768, wantCMF: 0x78},
		{level: 1, expect: 0x5E, name: "level=1 (0x78 0x5E)", window: 32768, wantCMF: 0x78},
		{level: 2, expect: 0x9C, name: "level=2 (0x78 0x9C)", window: 32768, wantCMF: 0x78},
		{level: 3, expect: 0xDA, name: "level=3 (0x78 0xDA)", window: 32768, wantCMF: 0x78},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteCMF(&buf, tc.window); err != nil {
				t.Fatalf("WriteCMF(%d) failed: %v", tc.window, err)
			}
			cmf := buf.Bytes()[0]
			if err := WriteFLG(&buf, cmf, tc.level); err != nil {
				t.Fatalf("WriteFLG(%d) failed: %v", tc.level, err)
			}

			if buf.Len() != 2 {
				t.Fatalf("zlib header should be 2 bytes, got %d", buf.Len())
			}

			gotCMF := buf.Bytes()[0]
			flg := buf.Bytes()[1]

			if gotCMF != tc.wantCMF {
				t.Fatalf("CMF = 0x%02X, want 0x%02X", gotCMF, tc.wantCMF)
			}
			if flg != tc.expect {
				t.Fatalf("FLG = 0x%02X, want 0x%02X", flg, tc.expect)
			}

			combined := int(gotCMF)<<8 | int(flg)
			if combined%31 != 0 {
				t.Fatalf("(CMF*256+FLG) %% 31 = %d, want 0", combined%31)
			}

			if (flg & 0x20) != 0 {
				t.Fatalf("FDICT bit set in FLG=0x%02X", flg)
			}
			if ((flg >> 6) & 3) != tc.level {
				t.Fatalf("FLEVEL = %d, want %d", (flg>>6)&3, tc.level)
			}
		})
	}
}

func TestWriteFLGWithCMF(t *testing.T) {
	var buf bytes.Buffer
	cmf := byte(0x78)
	err := WriteFLG(&buf, cmf, 0)
	if err != nil {
		t.Fatalf("WriteFLG with CMF error = %v, want nil", err)
	}
	if buf.Len() != 1 {
		t.Fatalf("WriteFLG wrote %d bytes, want 1", buf.Len())
	}
}

func TestWriteFLGInvalidCompressionLevel(t *testing.T) {
	levels := []uint8{4, 7, 255}

	for _, level := range levels {
		level := level
		t.Run(fmt.Sprintf("level=%d", level), func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteCMF(&buf, 32768); err != nil {
				t.Fatalf("WriteCMF failed: %v", err)
			}
			cmf := buf.Bytes()[0]
			err := WriteFLG(&buf, cmf, level)
			if err != ErrInvalidCompressionLevel {
				t.Fatalf("WriteFLG(%d) error = %v, want %v", level, err, ErrInvalidCompressionLevel)
			}
		})
	}
}

func TestZlibHeaderCMFAndFLGDivisibleBy31ForAllValidWindows(t *testing.T) {
	for windowSize := 256; windowSize <= 32768; windowSize *= 2 {
		windowSize := windowSize
		t.Run(fmt.Sprintf("windowSize=%d", windowSize), func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteCMF(&buf, windowSize); err != nil {
				t.Fatalf("WriteCMF(%d) failed: %v", windowSize, err)
			}
			cmf := buf.Bytes()[0]
			if err := WriteFLG(&buf, cmf, 0); err != nil {
				t.Fatalf("WriteFLG failed: %v", err)
			}

			gotCMF := buf.Bytes()[0]
			flg := buf.Bytes()[1]
			combined := int(gotCMF)<<8 | int(flg)
			if combined%31 != 0 {
				t.Fatalf("(CMF*256+FLG) %% 31 = %d, want 0", combined%31)
			}
		})
	}
}
