package png

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"image/color"
	stdpng "image/png"
	"io"
	"testing"

	"github.com/mac/go-pixo/src/compress"
)

type parsedChunk struct {
	Type string
	Data []byte
	CRC  uint32
}

func TestEncode1x1RGB(t *testing.T) {
	tests := []struct {
		name   string
		pixels []byte
	}{
		{
			name:   "red",
			pixels: []byte{0xFF, 0x00, 0x00},
		},
		{
			name:   "green",
			pixels: []byte{0x00, 0xFF, 0x00},
		},
		{
			name:   "blue",
			pixels: []byte{0x00, 0x00, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pngData := encodeTestImage(t, 1, 1, ColorRGB, tt.pixels)
			assertMinimalValidPNG(t, pngData, 1, 1, ColorRGB)
			assertIHDR(t, pngData, 1, 1, ColorRGB)
			assertIDATZlibScanlines(t, pngData, 1, 1, ColorRGB, tt.pixels)
			assertDecodedPixels(t, pngData, 1, 1, ColorRGB, tt.pixels)
		})
	}
}

func TestEncode1x1RGBA(t *testing.T) {
	tests := []struct {
		name   string
		pixels []byte
	}{
		{
			name:   "transparent",
			pixels: []byte{0x10, 0x20, 0x30, 0x00},
		},
		{
			name:   "semi_transparent",
			pixels: []byte{0x00, 0xFF, 0x00, 0x80},
		},
		{
			name:   "opaque",
			pixels: []byte{0xAA, 0xBB, 0xCC, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pngData := encodeTestImage(t, 1, 1, ColorRGBA, tt.pixels)
			assertMinimalValidPNG(t, pngData, 1, 1, ColorRGBA)
			assertIHDR(t, pngData, 1, 1, ColorRGBA)
			assertIDATZlibScanlines(t, pngData, 1, 1, ColorRGBA, tt.pixels)
			assertDecodedPixels(t, pngData, 1, 1, ColorRGBA, tt.pixels)
		})
	}
}

func TestEncode2x2RGB(t *testing.T) {
	tests := []struct {
		name   string
		pixels []byte
	}{
		{
			name: "rgb_corners",
			// Row-major: (0,0) (1,0) (0,1) (1,1)
			pixels: []byte{
				0xFF, 0x00, 0x00, // red
				0x00, 0xFF, 0x00, // green
				0x00, 0x00, 0xFF, // blue
				0xFF, 0xFF, 0xFF, // white
			},
		},
		{
			name: "rgb_grayscale_like",
			pixels: []byte{
				0x10, 0x10, 0x10,
				0x20, 0x20, 0x20,
				0x30, 0x30, 0x30,
				0x40, 0x40, 0x40,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pngData := encodeTestImage(t, 2, 2, ColorRGB, tt.pixels)
			assertMinimalValidPNG(t, pngData, 2, 2, ColorRGB)
			assertIHDR(t, pngData, 2, 2, ColorRGB)
			assertIDATZlibScanlines(t, pngData, 2, 2, ColorRGB, tt.pixels)
			assertDecodedPixels(t, pngData, 2, 2, ColorRGB, tt.pixels)
		})
	}
}

func TestEncode2x2RGBA(t *testing.T) {
	tests := []struct {
		name   string
		pixels []byte
	}{
		{
			name: "rgba_mixed_alpha",
			pixels: []byte{
				0xFF, 0x00, 0x00, 0xFF, // opaque red
				0x00, 0xFF, 0x00, 0x80, // semi green
				0x00, 0x00, 0xFF, 0x40, // more transparent blue
				0xFF, 0xFF, 0xFF, 0x00, // fully transparent white
			},
		},
		{
			name: "rgba_opaque_only",
			pixels: []byte{
				0x01, 0x02, 0x03, 0xFF,
				0x04, 0x05, 0x06, 0xFF,
				0x07, 0x08, 0x09, 0xFF,
				0x0A, 0x0B, 0x0C, 0xFF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pngData := encodeTestImage(t, 2, 2, ColorRGBA, tt.pixels)
			assertMinimalValidPNG(t, pngData, 2, 2, ColorRGBA)
			assertIHDR(t, pngData, 2, 2, ColorRGBA)
			assertIDATZlibScanlines(t, pngData, 2, 2, ColorRGBA, tt.pixels)
			assertDecodedPixels(t, pngData, 2, 2, ColorRGBA, tt.pixels)
		})
	}
}

func TestEncodeSignature(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		height    int
		colorType ColorType
		pixels    []byte
	}{
		{
			name:      "1x1_rgb",
			width:     1,
			height:    1,
			colorType: ColorRGB,
			pixels:    []byte{0x12, 0x34, 0x56},
		},
		{
			name:      "1x1_rgba",
			width:     1,
			height:    1,
			colorType: ColorRGBA,
			pixels:    []byte{0x12, 0x34, 0x56, 0x78},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pngData := encodeTestImage(t, tt.width, tt.height, tt.colorType, tt.pixels)
			if len(pngData) < len(PNG_SIGNATURE) {
				t.Fatalf("encoded PNG too short: got %d bytes, want at least %d", len(pngData), len(PNG_SIGNATURE))
			}
			if !bytes.Equal(pngData[:8], PNG_SIGNATURE[:]) {
				t.Fatalf("signature = % x, want % x", pngData[:8], PNG_SIGNATURE[:])
			}
		})
	}
}

func TestEncodeChunkOrder(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		height    int
		colorType ColorType
		pixels    []byte
	}{
		{
			name:      "1x1_rgb",
			width:     1,
			height:    1,
			colorType: ColorRGB,
			pixels:    []byte{0x00, 0x00, 0x00},
		},
		{
			name:      "2x2_rgba",
			width:     2,
			height:    2,
			colorType: ColorRGBA,
			pixels: []byte{
				0x00, 0x00, 0x00, 0xFF,
				0xFF, 0x00, 0x00, 0xFF,
				0x00, 0xFF, 0x00, 0xFF,
				0x00, 0x00, 0xFF, 0xFF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pngData := encodeTestImage(t, tt.width, tt.height, tt.colorType, tt.pixels)
			chunks := parsePNGChunks(t, pngData)

			if len(chunks) < 3 {
				t.Fatalf("expected at least 3 chunks (IHDR, IDAT, IEND), got %d", len(chunks))
			}

			if chunks[0].Type != "IHDR" {
				t.Fatalf("first chunk = %q, want %q", chunks[0].Type, "IHDR")
			}

			if chunks[len(chunks)-1].Type != "IEND" {
				t.Fatalf("last chunk = %q, want %q", chunks[len(chunks)-1].Type, "IEND")
			}

			seenIHDR := 0
			seenIDAT := 0
			seenIEND := 0
			for i, c := range chunks {
				switch c.Type {
				case "IHDR":
					seenIHDR++
					if i != 0 {
						t.Fatalf("IHDR chunk at index %d, want index 0", i)
					}
				case "IDAT":
					seenIDAT++
				case "IEND":
					seenIEND++
					if i != len(chunks)-1 {
						t.Fatalf("IEND chunk at index %d, want last index %d", i, len(chunks)-1)
					}
				default:
					t.Fatalf("unexpected chunk type %q at index %d", c.Type, i)
				}
			}

			if seenIHDR != 1 {
				t.Fatalf("IHDR count = %d, want 1", seenIHDR)
			}
			if seenIDAT < 1 {
				t.Fatalf("IDAT count = %d, want at least 1", seenIDAT)
			}
			if seenIEND != 1 {
				t.Fatalf("IEND count = %d, want 1", seenIEND)
			}
		})
	}
}

func TestEncodeValidation(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		height    int
		colorType ColorType
		pixels    []byte
		wantErr   bool
	}{
		{
			name:      "valid_1x1_rgb",
			width:     1,
			height:    1,
			colorType: ColorRGB,
			pixels:    []byte{0x01, 0x02, 0x03},
			wantErr:   false,
		},
		{
			name:      "valid_2x2_rgba",
			width:     2,
			height:    2,
			colorType: ColorRGBA,
			pixels:    make([]byte, 2*2*4),
			wantErr:   false,
		},
		{
			name:      "too_short_rgb",
			width:     2,
			height:    2,
			colorType: ColorRGB,
			pixels:    make([]byte, 2*2*3-1),
			wantErr:   true,
		},
		{
			name:      "too_long_rgb",
			width:     2,
			height:    2,
			colorType: ColorRGB,
			pixels:    make([]byte, 2*2*3+1),
			wantErr:   true,
		},
		{
			name:      "too_short_rgba",
			width:     1,
			height:    1,
			colorType: ColorRGBA,
			pixels:    []byte{0x00, 0x00, 0x00},
			wantErr:   true,
		},
		{
			name:      "nil_pixels",
			width:     1,
			height:    1,
			colorType: ColorRGB,
			pixels:    nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewEncoder(tt.width, tt.height, tt.colorType)
			if err != nil {
				// Construction errors (e.g., invalid dimensions) are not the focus
				// of this test. Fail loudly so the encoder API stays predictable.
				t.Fatalf("NewEncoder() error = %v", err)
			}

			_, encodeErr := enc.Encode(tt.pixels)
			if tt.wantErr {
				if encodeErr == nil {
					t.Fatalf("Encode() error = nil, want error")
				}
			} else {
				if encodeErr != nil {
					t.Fatalf("Encode() error = %v, want nil", encodeErr)
				}
			}
		})
	}
}

func encodeTestImage(t *testing.T, width, height int, colorType ColorType, pixels []byte) []byte {
	t.Helper()

	enc, err := NewEncoder(width, height, colorType)
	if err != nil {
		t.Fatalf("NewEncoder() error = %v", err)
	}

	pngData, err := enc.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	return pngData
}

func assertMinimalValidPNG(t *testing.T, pngData []byte, width, height int, colorType ColorType) {
	t.Helper()

	if len(pngData) < 8 {
		t.Fatalf("encoded PNG too short: got %d bytes", len(pngData))
	}

	if !IsValidSignature(pngData[:8]) {
		t.Fatalf("encoded PNG signature invalid: % x", pngData[:8])
	}

	chunks := parsePNGChunks(t, pngData)
	if len(chunks) < 3 {
		t.Fatalf("expected at least 3 chunks (IHDR, IDAT, IEND), got %d", len(chunks))
	}

	_ = width
	_ = height
	_ = colorType
}

func assertIHDR(t *testing.T, pngData []byte, width, height int, colorType ColorType) {
	t.Helper()

	chunks := parsePNGChunks(t, pngData)
	ihdr := findFirstChunk(t, chunks, "IHDR")

	if len(ihdr.Data) != 13 {
		t.Fatalf("IHDR data length = %d, want 13", len(ihdr.Data))
	}

	gotWidth := binary.BigEndian.Uint32(ihdr.Data[0:4])
	gotHeight := binary.BigEndian.Uint32(ihdr.Data[4:8])
	gotBitDepth := ihdr.Data[8]
	gotColorType := ihdr.Data[9]
	gotCompression := ihdr.Data[10]
	gotFilter := ihdr.Data[11]
	gotInterlace := ihdr.Data[12]

	if gotWidth != uint32(width) {
		t.Fatalf("IHDR width = %d, want %d", gotWidth, width)
	}
	if gotHeight != uint32(height) {
		t.Fatalf("IHDR height = %d, want %d", gotHeight, height)
	}
	if gotBitDepth != 8 {
		t.Fatalf("IHDR bit depth = %d, want 8", gotBitDepth)
	}
	if gotColorType != uint8(colorType) {
		t.Fatalf("IHDR color type = %d, want %d", gotColorType, uint8(colorType))
	}
	if gotCompression != 0 {
		t.Fatalf("IHDR compression = %d, want 0", gotCompression)
	}
	if gotFilter != 0 {
		t.Fatalf("IHDR filter = %d, want 0", gotFilter)
	}
	if gotInterlace != 0 {
		t.Fatalf("IHDR interlace = %d, want 0", gotInterlace)
	}
}

func assertIDATZlibScanlines(t *testing.T, pngData []byte, width, height int, colorType ColorType, pixels []byte) {
	t.Helper()

	chunks := parsePNGChunks(t, pngData)
	idatData := concatChunkData(chunks, "IDAT")
	if len(idatData) == 0 {
		t.Fatalf("missing IDAT data")
	}

	zr, err := zlib.NewReader(bytes.NewReader(idatData))
	if err != nil {
		t.Fatalf("IDAT zlib.NewReader() error = %v", err)
	}
	defer zr.Close()

	raw, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("IDAT zlib decompression error = %v", err)
	}

	bpp := BytesPerPixel(colorType)
	wantRaw := buildRawScanlines(width, height, bpp, pixels)

	if !bytes.Equal(raw, wantRaw) {
		t.Fatalf("decompressed scanlines mismatch\nraw:  % x\nwant: % x", raw, wantRaw)
	}
}

func assertDecodedPixels(t *testing.T, pngData []byte, width, height int, colorType ColorType, pixels []byte) {
	t.Helper()

	img, err := stdpng.Decode(bytes.NewReader(pngData))
	if err != nil {
		t.Fatalf("image/png.Decode() error = %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != width || bounds.Dy() != height {
		t.Fatalf("decoded bounds = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), width, height)
	}

	bpp := BytesPerPixel(colorType)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bpp
			want := color.NRGBA{
				R: pixels[offset+0],
				G: pixels[offset+1],
				B: pixels[offset+2],
				A: 0xFF,
			}
			if bpp == 4 {
				want.A = pixels[offset+3]
			}

			got := color.NRGBAModel.Convert(img.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.NRGBA)
			if got != want {
				t.Fatalf("pixel(%d,%d) = %#v, want %#v", x, y, got, want)
			}
		}
	}
}

func buildRawScanlines(width, height, bytesPerPixel int, pixels []byte) []byte {
	rowBytes := width * bytesPerPixel
	want := make([]byte, 0, height*(1+rowBytes))
	var prevRow []byte

	for y := 0; y < height; y++ {
		rowStart := y * rowBytes
		row := pixels[rowStart : rowStart+rowBytes]
		filterType, filteredRow := SelectFilter(row, prevRow, bytesPerPixel)
		want = append(want, byte(filterType))
		want = append(want, filteredRow...)
		prevRow = row
	}

	return want
}

func parsePNGChunks(t *testing.T, pngData []byte) []parsedChunk {
	t.Helper()

	if len(pngData) < 8 {
		t.Fatalf("PNG too short: got %d bytes", len(pngData))
	}
	if !bytes.Equal(pngData[:8], PNG_SIGNATURE[:]) {
		t.Fatalf("PNG signature mismatch: got % x, want % x", pngData[:8], PNG_SIGNATURE[:])
	}

	off := 8
	chunks := make([]parsedChunk, 0, 4)

	for {
		if off == len(pngData) {
			break
		}
		if off+12 > len(pngData) {
			t.Fatalf("truncated chunk header at offset %d", off)
		}

		length := binary.BigEndian.Uint32(pngData[off : off+4])
		chunkType := string(pngData[off+4 : off+8])

		dataStart := off + 8
		dataEnd := dataStart + int(length)
		crcStart := dataEnd
		crcEnd := crcStart + 4

		if dataEnd < dataStart || crcEnd < crcStart || crcEnd > len(pngData) {
			t.Fatalf("invalid chunk bounds for %q at offset %d (length=%d)", chunkType, off, length)
		}

		data := pngData[dataStart:dataEnd]
		crc := binary.BigEndian.Uint32(pngData[crcStart:crcEnd])

		expectedCRC := compress.CRC32(append([]byte(chunkType), data...))
		if crc != expectedCRC {
			t.Fatalf("%s CRC = 0x%08x, want 0x%08x", chunkType, crc, expectedCRC)
		}

		chunks = append(chunks, parsedChunk{Type: chunkType, Data: data, CRC: crc})
		off = crcEnd

		if chunkType == "IEND" {
			break
		}
	}

	if len(chunks) == 0 {
		t.Fatalf("no chunks parsed")
	}

	// The file should end immediately after IEND.
	last := chunks[len(chunks)-1]
	if last.Type != "IEND" {
		t.Fatalf("missing IEND chunk")
	}
	if off != len(pngData) {
		t.Fatalf("trailing bytes after IEND: %d", len(pngData)-off)
	}

	return chunks
}

func findFirstChunk(t *testing.T, chunks []parsedChunk, typ string) parsedChunk {
	t.Helper()
	for _, c := range chunks {
		if c.Type == typ {
			return c
		}
	}
	t.Fatalf("missing %s chunk", typ)
	return parsedChunk{}
}

func concatChunkData(chunks []parsedChunk, typ string) []byte {
	var out []byte
	for _, c := range chunks {
		if c.Type == typ {
			out = append(out, c.Data...)
		}
	}
	return out
}
