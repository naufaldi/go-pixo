package png

import (
	"bytes"
	"fmt"
	"testing"
)

func TestWriteScanline(t *testing.T) {
	tests := []struct {
		name        string
		filter      FilterType
		pixels      []byte
		expectErr   bool
		expectBytes []byte
	}{
		{
			name:        "FilterNone with single pixel RGB",
			filter:      FilterNone,
			pixels:      []byte{0xFF, 0x00, 0x00},
			expectBytes: []byte{0x00, 0xFF, 0x00, 0x00},
		},
		{
			name:        "FilterSub with two pixels RGB",
			filter:      FilterSub,
			pixels:      []byte{0xFF, 0x00, 0x00, 0x00, 0xFF, 0x00},
			expectBytes: []byte{0x01, 0xFF, 0x00, 0x00, 0x00, 0xFF, 0x00},
		},
		{
			name:        "FilterUp with RGBA",
			filter:      FilterUp,
			pixels:      []byte{0xFF, 0x00, 0x00, 0xFF},
			expectBytes: []byte{0x02, 0xFF, 0x00, 0x00, 0xFF},
		},
		{
			name:        "FilterAverage",
			filter:      FilterAverage,
			pixels:      []byte{0x80, 0x80, 0x80},
			expectBytes: []byte{0x03, 0x80, 0x80, 0x80},
		},
		{
			name:        "FilterPaeth",
			filter:      FilterPaeth,
			pixels:      []byte{0x12, 0x34, 0x56},
			expectBytes: []byte{0x04, 0x12, 0x34, 0x56},
		},
		{
			name:      "empty pixels",
			filter:    FilterNone,
			pixels:    []byte{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteScanline(&buf, tt.filter, tt.pixels)

			if tt.expectErr {
				if err != ErrEmptyScanline {
					t.Errorf("WriteScanline() error = %v, want %v", err, ErrEmptyScanline)
				}
				return
			}

			if err != nil {
				t.Fatalf("WriteScanline() error = %v, want nil", err)
			}

			if !bytes.Equal(buf.Bytes(), tt.expectBytes) {
				t.Errorf("WriteScanline() = %v, want %v", buf.Bytes(), tt.expectBytes)
			}
		})
	}
}

func TestScanlineBytes(t *testing.T) {
	tests := []struct {
		name        string
		filter      FilterType
		pixels      []byte
		expectErr   bool
		expectBytes []byte
	}{
		{
			name:        "valid RGB scanline",
			filter:      FilterNone,
			pixels:      []byte{0xFF, 0x00, 0x00},
			expectBytes: []byte{0x00, 0xFF, 0x00, 0x00},
		},
		{
			name:        "valid RGBA scanline",
			filter:      FilterSub,
			pixels:      []byte{0xFF, 0x00, 0x00, 0xFF},
			expectBytes: []byte{0x01, 0xFF, 0x00, 0x00, 0xFF},
		},
		{
			name:      "empty pixels",
			filter:    FilterNone,
			pixels:    []byte{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ScanlineBytes(tt.filter, tt.pixels)

			if tt.expectErr {
				if err != ErrEmptyScanline {
					t.Errorf("ScanlineBytes() error = %v, want %v", err, ErrEmptyScanline)
				}
				return
			}

			if err != nil {
				t.Fatalf("ScanlineBytes() error = %v, want nil", err)
			}

			if !bytes.Equal(result, tt.expectBytes) {
				t.Errorf("ScanlineBytes() = %v, want %v", result, tt.expectBytes)
			}
		})
	}
}

func TestBytesPerPixel(t *testing.T) {
	tests := []struct {
		colorType ColorType
		expect    int
	}{
		{ColorGrayscale, 1},
		{ColorRGB, 3},
		{ColorRGBA, 4},
		{ColorType(99), 1}, // Unknown color type defaults to 1
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("colorType=%d", tt.colorType), func(t *testing.T) {
			got := BytesPerPixel(tt.colorType)
			if got != tt.expect {
				t.Errorf("BytesPerPixel(%d) = %d, want %d", tt.colorType, got, tt.expect)
			}
		})
	}
}

func TestScanlineLength(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		colorType ColorType
		expect    int
	}{
		{
			name:      "1x1 grayscale",
			width:     1,
			colorType: ColorGrayscale,
			expect:    2, // 1 filter + 1 pixel
		},
		{
			name:      "2x2 grayscale",
			width:     2,
			colorType: ColorGrayscale,
			expect:    3, // 1 filter + 2 pixels
		},
		{
			name:      "1x1 RGB",
			width:     1,
			colorType: ColorRGB,
			expect:    4, // 1 filter + 3 pixels
		},
		{
			name:      "4x4 RGB",
			width:     4,
			colorType: ColorRGB,
			expect:    13, // 1 filter + 12 pixels
		},
		{
			name:      "1x1 RGBA",
			width:     1,
			colorType: ColorRGBA,
			expect:    5, // 1 filter + 4 pixels
		},
		{
			name:      "10x10 RGBA",
			width:     10,
			colorType: ColorRGBA,
			expect:    41, // 1 filter + 40 pixels
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScanlineLength(tt.width, tt.colorType)
			if got != tt.expect {
				t.Errorf("ScanlineLength(%d, %d) = %d, want %d", tt.width, tt.colorType, got, tt.expect)
			}
		})
	}
}

func TestValidateScanlineData(t *testing.T) {
	tests := []struct {
		name      string
		pixels    []byte
		width     int
		colorType ColorType
		expectErr bool
	}{
		{
			name:      "valid 1x1 RGB",
			pixels:    []byte{0x00, 0xFF, 0x00, 0x00}, // filter 0 + RGB
			width:     1,
			colorType: ColorRGB,
			expectErr: false,
		},
		{
			name:      "valid 2x2 RGB",
			pixels:    []byte{0x00, 0xFF, 0x00, 0x00, 0x00, 0xFF, 0x00}, // filter 0 + 2 RGB pixels
			width:     2,
			colorType: ColorRGB,
			expectErr: false,
		},
		{
			name:      "valid 1x1 RGBA",
			pixels:    []byte{0x00, 0xFF, 0x00, 0x00, 0xFF}, // filter 0 + RGBA
			width:     1,
			colorType: ColorRGBA,
			expectErr: false,
		},
		{
			name:      "wrong length for RGB",
			pixels:    []byte{0x00, 0xFF, 0x00}, // 3 bytes, should be 4 (filter + 3 RGB)
			width:     1,
			colorType: ColorRGB,
			expectErr: true,
		},
		{
			name:      "wrong length for RGBA",
			pixels:    []byte{0x00, 0xFF, 0x00, 0x00}, // 4 bytes, should be 5 (filter + 4 RGBA)
			width:     1,
			colorType: ColorRGBA,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScanlineData(tt.pixels, tt.width, tt.colorType)

			if tt.expectErr && err == nil {
				t.Errorf("ValidateScanlineData() expected error, got nil")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("ValidateScanlineData() unexpected error: %v", err)
			}
		})
	}
}

func TestScanlineBytes_consistency(t *testing.T) {
	// Test that ScanlineBytes matches WriteScanline output
	filters := []FilterType{FilterNone, FilterSub, FilterUp, FilterAverage, FilterPaeth}
	pixels := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC}

	for i, filter := range filters {
		t.Run(fmt.Sprintf("filter=%d", i), func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteScanline(&buf, filter, pixels)
			if err != nil {
				t.Fatalf("WriteScanline() error = %v", err)
			}

			bytesResult, err := ScanlineBytes(filter, pixels)
			if err != nil {
				t.Fatalf("ScanlineBytes() error = %v", err)
			}

			if !bytes.Equal(buf.Bytes(), bytesResult) {
				t.Errorf("ScanlineBytes() = %v, WriteScanline() = %v", bytesResult, buf.Bytes())
			}
		})
	}
}
