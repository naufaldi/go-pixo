package jpeg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	stdjpeg "image/jpeg"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// photoLike renders a deterministic image with gradients, texture, edges and
// sensor-like noise so that entropy coding behaves like a real photo.
func photoLike(width, height int, seed uint64) *image.RGBA {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fx, fy := float64(x)/float64(width), float64(y)/float64(height)
			base := 90 + 80*math.Sin(fx*7.1+fy*2.3) + 40*math.Cos(fy*11.7-fx*3.1)
			if (x/37+y/29)%5 == 0 {
				base += 45 // Hard edges.
			}
			noise := rng.NormFloat64() * 6
			r := base + 30*fx + noise
			g := base*0.8 + 50*fy + noise
			b := 200 - base*0.6 + rng.NormFloat64()*4
			img.SetRGBA(x, y, color.RGBA{clampByte(r), clampByte(g), clampByte(b), 255})
		}
	}
	return img
}

func clampByte(v float64) uint8 {
	return uint8(math.Max(0, math.Min(255, math.Round(v))))
}

func grayLike(width, height int, seed uint64) *image.Gray {
	rgba := photoLike(width, height, seed)
	img := image.NewGray(rgba.Bounds())
	for i := range img.Pix {
		img.Pix[i] = rgba.Pix[i*4+1]
	}
	return img
}

func stdEncode(t testing.TB, img image.Image, quality int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := stdjpeg.Encode(&buf, img, &stdjpeg.Options{Quality: quality}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// repoEncode uses this package's encoder to write 4:4:4 baseline and
// progressive files, which image/jpeg cannot.
func repoEncode(t testing.TB, img *image.RGBA, progressive bool) []byte {
	t.Helper()
	b := img.Bounds()
	pixels := make([]byte, 0, b.Dx()*b.Dy()*3)
	for i := 0; i < len(img.Pix); i += 4 {
		pixels = append(pixels, img.Pix[i:i+3]...)
	}
	enc, err := NewEncoder(Options{
		Width: b.Dx(), Height: b.Dy(), Quality: 85, ColorType: ColorRGB,
		Subsampling: Subsampling444, Progressive: progressive,
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := enc.Encode(pixels)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func readTestdata(t testing.TB, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// reencode rewrites a fixture's coefficients with this package's writer to
// craft layouts (restart intervals, other scan scripts) that no available
// encoder produces. The fixture is then checked against image/jpeg on its own.
func reencode(t testing.TB, input []byte, layout func(*coeffDecoder) losslessCandidate, restartInterval int) []byte {
	t.Helper()
	d, err := readCoefficients(input)
	if err != nil {
		t.Fatal(err)
	}
	header, err := losslessHeader(d, 1)
	if err != nil {
		t.Fatal(err)
	}
	out, err := encodeLossless(d, header, layout(d), restartInterval, nil)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func sequentialLayout(d *coeffDecoder) losslessCandidate {
	return losslessCandidate{scans: sequentialScript(d)}
}

func jpegtranLayout(d *coeffDecoder) losslessCandidate {
	return losslessCandidate{progressive: true, scans: progressiveScript(d, true)}
}

// pixelBytes returns the decoder's raw sample planes, so equality means the
// decoded buffers are byte-identical.
func pixelBytes(t testing.TB, data []byte) (image.Rectangle, []byte) {
	t.Helper()
	img, err := stdjpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("image/jpeg decode: %v", err)
	}
	b := img.Bounds()
	var planes []byte
	// Only samples that map to visible pixels count; MCU padding may differ.
	switch m := img.(type) {
	case *image.YCbCr:
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				planes = append(planes, m.Y[m.YOffset(x, y)], m.Cb[m.COffset(x, y)], m.Cr[m.COffset(x, y)])
			}
		}
	case *image.Gray:
		for y := b.Min.Y; y < b.Max.Y; y++ {
			planes = append(planes, m.Pix[m.PixOffset(b.Min.X, y):m.PixOffset(b.Max.X, y)]...)
		}
	case *image.RGBA:
		for y := b.Min.Y; y < b.Max.Y; y++ {
			planes = append(planes, m.Pix[m.PixOffset(b.Min.X, y):m.PixOffset(b.Max.X, y)]...)
		}
	default:
		t.Fatalf("unexpected decoded type %T", img)
	}
	return b, planes
}

// assertLossless checks decoded pixels and, independently of any decoder, the
// quantized coefficients that cover the image.
func assertLossless(t *testing.T, input, output []byte) {
	t.Helper()
	if !bytes.HasPrefix(output, []byte{0xff, 0xd8}) || !bytes.HasSuffix(output, []byte{0xff, 0xd9}) {
		t.Fatal("output is not framed by SOI/EOI")
	}
	inBounds, inPixels := pixelBytes(t, input)
	outBounds, outPixels := pixelBytes(t, output)
	if inBounds != outBounds {
		t.Fatalf("bounds changed: %v -> %v", inBounds, outBounds)
	}
	if !bytes.Equal(inPixels, outPixels) {
		t.Fatal("decoded pixels differ")
	}

	a, err := readCoefficients(input)
	if err != nil {
		t.Fatal(err)
	}
	b, err := readCoefficients(output)
	if err != nil {
		t.Fatalf("output is not readable: %v", err)
	}
	if a.nComp != b.nComp || a.width != b.width || a.height != b.height {
		t.Fatal("frame changed")
	}
	for i := 0; i < a.nComp; i++ {
		ca, cb := a.comp[i], b.comp[i]
		if ca.c != cb.c || ca.hv != cb.hv || ca.tq != cb.tq || a.quant[ca.tq] != b.quant[cb.tq] {
			t.Fatalf("component %d parameters changed", i)
		}
		for by := 0; by < ca.compH; by++ {
			for bx := 0; bx < ca.compW; bx++ {
				ia := (ca.offset + by*ca.blocksW + bx) * coeffBlockSize
				ib := (cb.offset + by*cb.blocksW + bx) * coeffBlockSize
				if !slicesEqual(a.coeffs[ia:ia+coeffBlockSize], b.coeffs[ib:ib+coeffBlockSize]) {
					t.Fatalf("component %d block (%d,%d) coefficients differ", i, bx, by)
				}
			}
		}
	}
}

func slicesEqual(a, b []int16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestOptimizeLosslessPreservesPixels(t *testing.T) {
	photo := func(w, h int) *image.RGBA { return photoLike(w, h, uint64(w*131+h)) }
	q50_422 := readTestdata(t, "video-001.q50.422.progressive.jpeg")
	tests := []struct {
		name  string
		input func(t *testing.T) []byte
	}{
		{"baseline 4:2:0", func(t *testing.T) []byte { return stdEncode(t, photo(96, 64), 80) }},
		{"baseline 4:2:0 q100", func(t *testing.T) []byte { return stdEncode(t, photo(64, 48), 100) }},
		{"baseline 4:2:0 q5", func(t *testing.T) []byte { return stdEncode(t, photo(64, 48), 5) }},
		{"baseline 4:2:0 1x1", func(t *testing.T) []byte { return stdEncode(t, photo(1, 1), 90) }},
		{"baseline 4:2:0 17x9", func(t *testing.T) []byte { return stdEncode(t, photo(17, 9), 90) }},
		{"baseline 4:2:0 33x65", func(t *testing.T) []byte { return stdEncode(t, photo(33, 65), 90) }},
		{"baseline 4:4:4 33x65", func(t *testing.T) []byte { return repoEncode(t, photo(33, 65), false) }},
		{"baseline 4:4:4 1x1", func(t *testing.T) []byte { return repoEncode(t, photo(1, 1), false) }},
		{"baseline 4:4:4 17x9", func(t *testing.T) []byte { return repoEncode(t, photo(17, 9), false) }},
		{"baseline 4:2:2 libjpeg", func(t *testing.T) []byte { return reencode(t, q50_422, sequentialLayout, 0) }},
		{"baseline 4:1:1 libjpeg", func(t *testing.T) []byte { return readTestdata(t, "video-001.q50.411.jpeg") }},
		{"baseline 2x2/1x2 libjpeg", func(t *testing.T) []byte { return readTestdata(t, "video-001.221212.jpeg") }},
		{"baseline RGB components", func(t *testing.T) []byte { return readTestdata(t, "video-001.rgb.jpeg") }},
		{"grayscale", func(t *testing.T) []byte { return stdEncode(t, grayLike(80, 56, 1), 85) }},
		{"grayscale 1x1", func(t *testing.T) []byte { return stdEncode(t, grayLike(1, 1, 2), 85) }},
		{"grayscale 17x9", func(t *testing.T) []byte { return stdEncode(t, grayLike(17, 9, 3), 85) }},
		{"grayscale 33x65", func(t *testing.T) []byte { return stdEncode(t, grayLike(33, 65, 4), 85) }},
		{"progressive 4:4:4 libjpeg", func(t *testing.T) []byte { return readTestdata(t, "video-001.progressive.jpeg") }},
		{"progressive 4:2:0 libjpeg", func(t *testing.T) []byte { return readTestdata(t, "video-001.q50.420.progressive.jpeg") }},
		{"progressive 4:2:2 libjpeg", func(t *testing.T) []byte { return q50_422 }},
		{"progressive 4:4:0 libjpeg", func(t *testing.T) []byte { return readTestdata(t, "video-001.q50.440.progressive.jpeg") }},
		{"progressive 4:1:0 libjpeg", func(t *testing.T) []byte { return readTestdata(t, "video-001.q50.410.progressive.jpeg") }},
		{"progressive 4:1:1 libjpeg", func(t *testing.T) []byte { return readTestdata(t, "video-001.q50.411.progressive.jpeg") }},
		{"progressive with missing final scans libjpeg", func(t *testing.T) []byte {
			return readTestdata(t, "video-001.progressive.truncated.jpeg")
		}},
		{"progressive separate DC scans", func(t *testing.T) []byte {
			return readTestdata(t, "video-001.separate.dc.progression.progressive.jpeg")
		}},
		{"progressive grayscale 2x2 sampling", func(t *testing.T) []byte {
			return readTestdata(t, "video-005.gray.q50.2x2.progressive.jpeg")
		}},
		{"progressive 4:4:4 33x65 repo encoder", func(t *testing.T) []byte { return repoEncode(t, photo(33, 65), true) }},
		{"progressive 4:2:0 17x9 successive approximation", func(t *testing.T) []byte {
			return reencode(t, stdEncode(t, photo(17, 9), 90), jpegtranLayout, 0)
		}},
		{"progressive grayscale 33x65 successive approximation", func(t *testing.T) []byte {
			return reencode(t, stdEncode(t, grayLike(33, 65, 5), 90), jpegtranLayout, 0)
		}},
		{"restart interval libjpeg", func(t *testing.T) []byte { return readTestdata(t, "video-001.restart2.jpeg") }},
		{"restart every MCU 4:2:0", func(t *testing.T) []byte {
			return reencode(t, stdEncode(t, photo(33, 65), 80), sequentialLayout, 1)
		}},
		{"restart progressive 4:4:4", func(t *testing.T) []byte {
			return reencode(t, repoEncode(t, photo(40, 24), false), jpegtranLayout, 3)
		}},
		{"restart progressive grayscale", func(t *testing.T) []byte {
			return reencode(t, stdEncode(t, grayLike(33, 65, 6), 75), jpegtranLayout, 2)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input(t)
			output, err := OptimizeLossless(input)
			if err != nil {
				t.Fatalf("OptimizeLossless: %v", err)
			}
			assertLossless(t, input, output)

			// Optimizing the output again must stay lossless too, which also
			// exercises the reader on every layout the writer produces.
			again, err := OptimizeLossless(output)
			if err != nil {
				t.Fatalf("second pass: %v", err)
			}
			assertLossless(t, input, again)
		})
	}
}

func TestOptimizeLosslessCandidatesAreLossless(t *testing.T) {
	inputs := map[string][]byte{
		"4:2:0":     stdEncode(t, photoLike(57, 41, 9), 88),
		"grayscale": stdEncode(t, grayLike(41, 23, 9), 70),
		"4:2:2":     readTestdata(t, "video-001.q50.422.progressive.jpeg"),
	}
	for name, input := range inputs {
		d, err := readCoefficients(input)
		if err != nil {
			t.Fatal(err)
		}
		for i, candidate := range losslessCandidates(d) {
			t.Run(name+"/candidate "+string(rune('0'+i)), func(t *testing.T) {
				output := reencode(t, input, func(*coeffDecoder) losslessCandidate { return candidate }, 0)
				assertLossless(t, input, output)
			})
		}
	}
}

func TestOptimizeLosslessBeatsStandardTables(t *testing.T) {
	input := stdEncode(t, photoLike(512, 384, 42), 85)
	output, err := OptimizeLossless(input)
	if err != nil {
		t.Fatal(err)
	}
	assertLossless(t, input, output)
	savings := 1 - float64(len(output))/float64(len(input))
	t.Logf("standard tables %d bytes -> %d bytes (%.1f%% smaller)", len(input), len(output), savings*100)
	if savings < 0.03 {
		t.Fatalf("expected at least 3%% savings, got %.2f%%", savings*100)
	}
}

// insertSegments places raw marker segments right after SOI.
func insertSegments(input []byte, segments ...[]byte) []byte {
	out := append([]byte{}, input[:2]...)
	for _, segment := range segments {
		out = append(out, segment...)
	}
	return append(out, input[2:]...)
}

func rawSegment(marker byte, payload []byte) []byte {
	segment := make([]byte, 4, 4+len(payload))
	segment[0], segment[1] = 0xff, marker
	binary.BigEndian.PutUint16(segment[2:], uint16(len(payload)+2))
	return append(segment, payload...)
}

// segmentsOf lists the marker segments before the first SOS.
func segmentsOf(t *testing.T, data []byte) (markers []byte, payloads [][]byte) {
	t.Helper()
	for offset := 2; offset+4 <= len(data); {
		if data[offset] != 0xff {
			t.Fatalf("expected marker at %d", offset)
		}
		marker := data[offset+1]
		length := int(binary.BigEndian.Uint16(data[offset+2:]))
		markers = append(markers, marker)
		payloads = append(payloads, data[offset+4:offset+2+length])
		if marker == mkSOS {
			break
		}
		offset += 2 + length
	}
	return markers, payloads
}

func TestOptimizeLosslessMetadata(t *testing.T) {
	base := stdEncode(t, photoLike(40, 30, 7), 85)
	iccChunk1 := append([]byte("ICC_PROFILE\x00\x01\x02"), bytes.Repeat([]byte("profile-part-one"), 40)...)
	iccChunk2 := append([]byte("ICC_PROFILE\x00\x02\x02"), []byte("profile-part-two")...)
	adobe := []byte{'A', 'd', 'o', 'b', 'e', 0, 100, 0, 0, 0, 0, 1}
	jfifWithThumb := append([]byte{'J', 'F', 'I', 'F', 0, 1, 2, 1, 0, 72, 0, 72, 1, 1}, 1, 2, 3)
	secrets := [][]byte{
		rawSegment(mkCOM, []byte("secret comment")),
		rawSegment(mkAPP0+1, []byte("http://ns.adobe.com/xap/1.0/\x00secret xmp")),
		rawSegment(mkAPP0+13, []byte("Photoshop 3.0\x00secret iptc")),
		rawSegment(mkAPP2, []byte("MPF\x00secret mpf")),
	}

	tests := []struct {
		name        string
		segments    [][]byte
		orientation uint16
		wantICC     [][]byte
		wantAdobe   bool
	}{
		{"exif orientation 6 big endian", [][]byte{exifOrientationSegment(6, binary.BigEndian)}, 6, nil, false},
		{"exif orientation 6 little endian", [][]byte{exifOrientationSegment(6, binary.LittleEndian)}, 6, nil, false},
		{"exif orientation 8 with other metadata", append([][]byte{exifOrientationSegment(8, binary.LittleEndian)}, secrets...), 8, nil, false},
		{"exif orientation 1 dropped", [][]byte{exifOrientationSegment(1, binary.BigEndian)}, 1, nil, false},
		{"icc chunks preserved", [][]byte{rawSegment(mkAPP2, iccChunk1), rawSegment(mkAPP2, iccChunk2)}, 1, [][]byte{iccChunk1, iccChunk2}, false},
		{"adobe preserved", [][]byte{rawSegment(mkAPP14, adobe)}, 1, nil, true},
		{"jfif thumbnail dropped", [][]byte{rawSegment(mkAPP0, jfifWithThumb)}, 1, nil, false},
		{"everything", append([][]byte{
			rawSegment(mkAPP0, jfifWithThumb), exifOrientationSegment(3, binary.BigEndian),
			rawSegment(mkAPP2, iccChunk1), rawSegment(mkAPP14, adobe),
		}, secrets...), 3, [][]byte{iccChunk1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := insertSegments(base, tt.segments...)
			output, err := OptimizeLossless(input)
			if err != nil {
				t.Fatal(err)
			}
			assertLossless(t, input, output)

			orientation, err := jpegOrientation(output)
			if err != nil {
				t.Fatal(err)
			}
			if orientation != tt.orientation {
				t.Fatalf("orientation = %d, want %d", orientation, tt.orientation)
			}
			if bytes.Contains(output, []byte("secret")) {
				t.Fatal("output still contains stripped metadata")
			}

			markers, payloads := segmentsOf(t, output)
			var icc [][]byte
			adobeSeen, exifSeen := false, false
			for i, marker := range markers {
				switch marker {
				case mkAPP0:
					if !bytes.Equal(payloads[i], []byte{'J', 'F', 'I', 'F', 0, 1, 2, 1, 0, 72, 0, 72, 0, 0}) {
						t.Fatalf("JFIF payload = %v", payloads[i])
					}
				case mkAPP0 + 1:
					exifSeen = true
				case mkAPP2:
					icc = append(icc, payloads[i])
				case mkAPP14:
					adobeSeen = bytes.Equal(payloads[i], adobe)
				case mkCOM, mkAPP0 + 13:
					t.Fatalf("marker %#x should be stripped", marker)
				}
			}
			if exifSeen != (tt.orientation != 1) {
				t.Fatalf("EXIF present = %v for orientation %d", exifSeen, tt.orientation)
			}
			if len(icc) != len(tt.wantICC) {
				t.Fatalf("got %d ICC segments, want %d", len(icc), len(tt.wantICC))
			}
			for i := range icc {
				if !bytes.Equal(icc[i], tt.wantICC[i]) {
					t.Fatalf("ICC segment %d changed", i)
				}
			}
			if adobeSeen != tt.wantAdobe {
				t.Fatalf("Adobe preserved = %v, want %v", adobeSeen, tt.wantAdobe)
			}
		})
	}
}

// patchSOF applies fn to the first SOF segment's payload.
func patchSOF(t *testing.T, input []byte, fn func(marker *byte, payload []byte)) []byte {
	t.Helper()
	out := bytes.Clone(input)
	for offset := 2; offset+4 <= len(out); {
		marker := out[offset+1]
		length := int(binary.BigEndian.Uint16(out[offset+2:]))
		if marker >= mkSOF0 && marker <= mkSOF2 {
			fn(&out[offset+1], out[offset+4:offset+2+length])
			return out
		}
		offset += 2 + length
	}
	t.Fatal("missing SOF")
	return nil
}

func TestOptimizeLosslessRejectsUnsupported(t *testing.T) {
	base := stdEncode(t, photoLike(24, 16, 3), 80)
	setMarker := func(m byte) func(*byte, []byte) { return func(marker *byte, _ []byte) { *marker = m } }
	tests := []struct {
		name  string
		input []byte
		want  error
	}{
		{"CMYK", readTestdata(t, "video-001.cmyk.jpeg"), ErrUnsupportedJPEGComponents},
		{"arithmetic sequential", patchSOF(t, base, setMarker(0xc9)), ErrUnsupportedJPEGArithmetic},
		{"arithmetic progressive", patchSOF(t, base, setMarker(0xca)), ErrUnsupportedJPEGArithmetic},
		{"lossless", patchSOF(t, base, setMarker(mkSOF3)), ErrUnsupportedJPEGProcess},
		{"hierarchical", patchSOF(t, base, setMarker(0xc5)), ErrUnsupportedJPEGProcess},
		{"12-bit", patchSOF(t, base, func(_ *byte, p []byte) { p[0] = 12 }), ErrUnsupportedJPEGPrecision},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := OptimizeLossless(tt.input)
			if !errors.Is(err, tt.want) || !errors.Is(err, ErrUnsupportedJPEG) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestOptimizeLosslessRejectsMalformed(t *testing.T) {
	valid := stdEncode(t, photoLike(40, 24, 11), 80)
	progressive := readTestdata(t, "video-001.q50.420.progressive.jpeg")
	sos := bytes.Index(valid, []byte{0xff, mkSOS})
	tests := []struct {
		name  string
		input []byte
	}{
		{"not a JPEG", []byte("GIF89a not a jpeg")},
		{"SOI only", []byte{0xff, 0xd8}},
		{"missing EOI", valid[:len(valid)-2]},
		{"header only", valid[:sos]},
		{"bad segment length", append(append([]byte{}, valid[:2]...), 0xff, 0xe0, 0, 1)},
		{"duplicate SOF", insertSegments(valid, valid[bytes.Index(valid, []byte{0xff, mkSOF0}):sos])},
		{"marker inside entropy data", append(append(bytes.Clone(valid[:sos+20]), 0xff, 0xc4), valid[sos+22:]...)},
		{"progressive cut in SOI", progressive[:3]},
		{"progressive cut in header", progressive[:20]},
		{"progressive cut at a third", progressive[:len(progressive)/3]},
		{"progressive cut at half", progressive[:len(progressive)/2]},
		{"progressive cut before EOI", progressive[:len(progressive)-3]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := OptimizeLossless(tt.input)
			if !errors.Is(err, ErrInvalidJPEG) {
				t.Fatalf("want ErrInvalidJPEG, got %v (%d bytes)", err, len(out))
			}
		})
	}
	t.Run("zero height (DNL)", func(t *testing.T) {
		input := patchSOF(t, valid, func(_ *byte, p []byte) { p[1], p[2] = 0, 0 })
		if _, err := OptimizeLossless(input); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestOptimizeLosslessEveryPrefixErrors(t *testing.T) {
	inputs := [][]byte{
		stdEncode(t, photoLike(20, 12, 5), 75),
		readTestdata(t, "video-005.gray.q50.2x2.progressive.jpeg"),
	}
	for _, input := range inputs {
		for n := 0; n < len(input)-1; n += 7 {
			if _, err := OptimizeLossless(input[:n]); err == nil {
				t.Fatalf("prefix of %d/%d bytes was accepted", n, len(input))
			}
		}
	}
}

func TestOptimizeLosslessCorruptInputNeverPanics(t *testing.T) {
	seeds := [][]byte{
		stdEncode(t, photoLike(24, 24, 1), 80),
		readTestdata(t, "video-001.q50.420.progressive.jpeg"),
		readTestdata(t, "video-001.restart2.jpeg"),
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 600; i++ {
		input := bytes.Clone(seeds[i%len(seeds)])
		for flips := 1 + rng.IntN(4); flips > 0; flips-- {
			input[2+rng.IntN(len(input)-2)] = byte(rng.Uint32())
		}
		checkCorruptInput(t, input)
	}
}

// checkCorruptInput requires an error or a self-consistent, decodable result.
func checkCorruptInput(t *testing.T, input []byte) {
	t.Helper()
	output, err := OptimizeLossless(input)
	if err != nil {
		return
	}
	again, err := OptimizeLossless(output)
	if err != nil {
		t.Fatalf("optimized output is not readable: %v", err)
	}
	a, errA := readCoefficients(output)
	b, errB := readCoefficients(again)
	if errA != nil || errB != nil {
		t.Fatalf("re-read failed: %v %v", errA, errB)
	}
	for i := 0; i < a.nComp; i++ {
		c := a.comp[i]
		for by := 0; by < c.compH; by++ {
			start := (c.offset + by*c.blocksW) * coeffBlockSize
			end := start + c.compW*coeffBlockSize
			if !slicesEqual(a.coeffs[start:end], b.coeffs[start:end]) {
				t.Fatal("second optimization changed coefficients")
			}
		}
	}
	if _, err := stdjpeg.Decode(bytes.NewReader(input)); err == nil {
		if _, err := stdjpeg.Decode(bytes.NewReader(output)); err != nil {
			t.Fatalf("image/jpeg decodes the input but not the output: %v", err)
		}
	}
}

func FuzzOptimizeLossless(f *testing.F) {
	f.Add(stdEncode(f, photoLike(16, 16, 1), 75))
	f.Add(stdEncode(f, grayLike(9, 7, 1), 75))
	for _, name := range []string{"video-001.q50.420.progressive.jpeg", "video-001.restart2.jpeg", "video-005.gray.q50.2x2.progressive.jpeg"} {
		f.Add(readTestdata(f, name))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		checkCorruptInput(t, input)
	})
}

func TestOptimizeLosslessRejectsOversizedBeforeAllocation(t *testing.T) {
	valid := stdEncode(t, photoLike(16, 16, 1), 80)
	withSize := func(width, height int) []byte {
		return patchSOF(t, valid, func(_ *byte, p []byte) {
			binary.BigEndian.PutUint16(p[1:], uint16(height))
			binary.BigEndian.PutUint16(p[3:], uint16(width))
		})
	}
	tests := []struct {
		name  string
		input []byte
	}{
		{"25 megapixels", withSize(5000, 5000)},
		{"wider than 16384", withSize(16385, 1)},
		{"64 MiB plus one", nil},
		{"empty", []byte{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			if input == nil {
				input = make([]byte, 64*1024*1024+1)
				copy(input, valid)
			}
			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			_, err := OptimizeLossless(input)
			runtime.ReadMemStats(&after)
			if err == nil || !bytes.Contains([]byte(err.Error()), []byte("limit")) {
				t.Fatalf("want a limit error, got %v", err)
			}
			if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 1<<20 {
				t.Fatalf("allocated %d bytes before rejecting", allocated)
			}
		})
	}
}

func TestOptimizeLosslessAllowsLargeHeaderWithoutScanData(t *testing.T) {
	// A 16 MP frame header without scan data must fail without allocating
	// coefficient storage, which is deferred to the first SOS.
	valid := stdEncode(t, photoLike(16, 16, 1), 80)
	input := patchSOF(t, valid, func(_ *byte, p []byte) {
		binary.BigEndian.PutUint16(p[1:], 4000)
		binary.BigEndian.PutUint16(p[3:], 4000)
	})
	input = input[:bytes.Index(input, []byte{0xff, mkSOS})]
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := OptimizeLossless(input)
	runtime.ReadMemStats(&after)
	if !errors.Is(err, ErrInvalidJPEG) {
		t.Fatalf("err = %v", err)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 1<<20 {
		t.Fatalf("allocated %d bytes for a header-only file", allocated)
	}
}

func TestOptimalHuffmanTable(t *testing.T) {
	tests := []struct {
		name string
		freq map[int]int
	}{
		{"empty", nil},
		{"single symbol", map[int]int{0x00: 9}},
		{"two symbols", map[int]int{0x00: 1, 0x11: 1}},
		{"skewed", map[int]int{0x00: 1 << 20, 0x01: 1, 0x02: 1, 0x03: 2, 0x04: 3, 0x05: 5, 0x06: 8, 0x07: 13, 0x08: 21, 0x09: 34, 0x0a: 55, 0x11: 89, 0x12: 144, 0x13: 233, 0x14: 377, 0x15: 610, 0x16: 987, 0x17: 1597, 0x18: 2584, 0x19: 4181}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var freq [256]int
			for symbol, count := range tt.freq {
				freq[symbol] = count
			}
			bitCounts, vals, err := optimalHuffmanTable(&freq)
			if err != nil {
				t.Fatal(err)
			}
			total := 0
			for _, count := range bitCounts {
				total += int(count)
			}
			if total != len(vals) || total == 0 {
				t.Fatalf("bits sum %d, %d values", total, len(vals))
			}
			for symbol := range tt.freq {
				if bytes.IndexByte(vals, byte(symbol)) < 0 {
					t.Fatalf("symbol %#x missing", symbol)
				}
			}
			// No code may be all ones: the Kraft sum must stay below one.
			kraft := 0.0
			for length, count := range bitCounts {
				kraft += float64(count) / float64(uint(1)<<(length+1))
			}
			if kraft >= 1 {
				t.Fatalf("Kraft sum %v leaves no reserved all-ones code", kraft)
			}
		})
	}
}

func BenchmarkOptimizeLossless(b *testing.B) {
	// About 12 MP, encoded by image/jpeg with its standard Huffman tables.
	input := stdEncode(b, photoLike(4000, 3000, 12), 90)
	b.SetBytes(int64(len(input)))
	b.ReportAllocs()
	var output []byte
	for b.Loop() {
		var err error
		if output, err = OptimizeLossless(input); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(100*(1-float64(len(output))/float64(len(input))), "%saved")
}

// huffmanCost returns the optimal unrestricted Huffman cost of weights and
// its longest code length.
func huffmanCost(weights []int) (cost int, longest int) {
	type node struct{ weight, depth int }
	var nodes []node
	for _, w := range weights {
		if w > 0 {
			nodes = append(nodes, node{w, 0})
		}
	}
	for len(nodes) > 1 {
		// Repeatedly merge the two lightest nodes; depth tracks the height.
		for i := 0; i < 2; i++ {
			for j := i + 1; j < len(nodes); j++ {
				if nodes[j].weight < nodes[i].weight {
					nodes[i], nodes[j] = nodes[j], nodes[i]
				}
			}
		}
		merged := node{nodes[0].weight + nodes[1].weight, max(nodes[0].depth, nodes[1].depth) + 1}
		cost += merged.weight
		nodes = append(nodes[2:], merged)
	}
	return cost, nodes[0].depth
}

func TestOptimalHuffmanTableRandomized(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 8))
	for iteration := 0; iteration < 400; iteration++ {
		var freq [256]int
		symbols := 1 + rng.IntN(60)
		skewed := iteration%4 == 0
		for i := 0; i < symbols; i++ {
			symbol := rng.IntN(255) // Never the reserved 0xff.
			if skewed {
				freq[symbol] += 1 << rng.IntN(24) // Forces the 16-bit limit.
			} else {
				freq[symbol] += 1 + rng.IntN(1000)
			}
		}
		bitCounts, vals, err := optimalHuffmanTable(&freq)
		if err != nil {
			t.Fatal(err)
		}
		lengths := make(map[uint8]int)
		index, kraft, cost := 0, 0.0, 0
		for length, count := range bitCounts {
			for ; count > 0; count-- {
				lengths[vals[index]] = length + 1
				cost += 2 * freq[vals[index]] * (length + 1)
				kraft += 1 / float64(uint(1)<<(length+1))
				index++
			}
		}
		if index != len(vals) || kraft >= 1 {
			t.Fatalf("iteration %d: %d values, Kraft %v", iteration, len(vals), kraft)
		}
		for symbol, f := range freq {
			if f > 0 && lengths[uint8(symbol)] == 0 {
				t.Fatalf("iteration %d: symbol %#x has no code", iteration, symbol)
			}
		}
		weights := make([]int, 256)
		for i, f := range freq {
			weights[i] = 2 * f
		}
		weights[reservedHuffmanSymbol] = 1
		want, longest := huffmanCost(weights)
		// The reserved symbol's single count is not part of cost.
		if longest <= 16 && cost+longestCodeLength(bitCounts) != want {
			t.Fatalf("iteration %d: cost %d is not optimal (want %d)", iteration, cost, want)
		}
	}
}

func longestCodeLength(bitCounts [16]uint8) int {
	longest := 0
	for length, count := range bitCounts {
		if count > 0 {
			longest = length + 1
		}
	}
	return longest
}
