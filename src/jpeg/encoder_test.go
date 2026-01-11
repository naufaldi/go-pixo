package jpeg

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"testing"
)

func TestEncoder_1x1(t *testing.T) {
	width, height := 1, 1
	pixels := []byte{255, 0, 0} // Single Red pixel
	opts := BalancedOptions(width, height, 75)
	encoder, _ := NewEncoder(opts)
	jpegBytes, err := encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Verify it can be decoded by standard library
	img, format, err := image.Decode(bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatalf("Failed to decode produced JPEG: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("expected format jpeg, got %s", format)
	}
	if img.Bounds().Dx() != 1 || img.Bounds().Dy() != 1 {
		t.Errorf("expected 1x1, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestEncoder_Grayscale(t *testing.T) {
	width, height := 8, 8
	pixels := make([]byte, width*height)
	for i := range pixels {
		pixels[i] = uint8(i * 4)
	}
	opts := BalancedOptions(width, height, 75)
	opts.ColorType = ColorGrayscale
	encoder, _ := NewEncoder(opts)
	jpegBytes, err := encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	_, format, err := image.Decode(bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatalf("Failed to decode produced JPEG: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("expected format jpeg, got %s", format)
	}
}

func TestEncoder_NonMultipleOf8(t *testing.T) {
	width, height := 10, 10
	pixels := make([]byte, width*height*3)
	opts := BalancedOptions(width, height, 75)
	encoder, _ := NewEncoder(opts)
	jpegBytes, err := encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 10 {
		t.Errorf("expected 10x10, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestEncoder_QualityLevels(t *testing.T) {
	width, height := 32, 32
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	optsLow := BalancedOptions(width, height, 10)
	eLow, _ := NewEncoder(optsLow)
	jpegLow, _ := eLow.Encode(pixels)

	optsHigh := BalancedOptions(width, height, 90)
	eHigh, _ := NewEncoder(optsHigh)
	jpegHigh, _ := eHigh.Encode(pixels)

	if len(jpegLow) >= len(jpegHigh) {
		t.Errorf("expected low quality JPEG (%d) to be smaller than high quality (%d)", len(jpegLow), len(jpegHigh))
	}
}

func TestOptionsBuilder(t *testing.T) {
	width, height := 64, 64
	builder := NewOptionsBuilder(width, height)
	opts := builder.Quality(85).
		Subsampling(Subsampling420).
		OptimizeHuffman(true).
		Progressive(true).
		Build()

	if opts.Quality != 85 {
		t.Errorf("expected quality 85, got %d", opts.Quality)
	}
	if opts.Subsampling != Subsampling420 {
		t.Errorf("expected subsampling 420")
	}
	if !opts.OptimizeHuffman {
		t.Errorf("expected OptimizeHuffman true")
	}
	if !opts.Progressive {
		t.Errorf("expected Progressive true")
	}
}

func TestPresets(t *testing.T) {
	width, height := 64, 64
	quality := uint8(75)

	fast := FastOptions(width, height, quality)
	if fast.Subsampling != Subsampling444 || fast.OptimizeHuffman || fast.Progressive {
		t.Errorf("Fast preset incorrect: %+v", fast)
	}

	balanced := BalancedOptions(width, height, quality)
	if balanced.Subsampling != Subsampling420 || balanced.OptimizeHuffman || balanced.Progressive {
		t.Errorf("Balanced preset incorrect: %+v", balanced)
	}

	max := MaxOptions(width, height, quality)
	if max.Subsampling != Subsampling420 || !max.OptimizeHuffman || !max.Progressive {
		t.Errorf("Max preset incorrect: %+v", max)
	}
}

func BenchmarkEncoder_ScalarDCT(b *testing.B) {
	width, height := 256, 256
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := BalancedOptions(width, height, 75)
	opts.UseSIMD = false
	encoder, _ := NewEncoder(opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder.Encode(pixels)
	}
}

func BenchmarkEncoder_SIMDDCT(b *testing.B) {
	width, height := 256, 256
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := BalancedOptions(width, height, 75)
	opts.UseSIMD = true
	encoder, _ := NewEncoder(opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder.Encode(pixels)
	}
}

func BenchmarkEncoder_Progressive_Scalar(b *testing.B) {
	width, height := 256, 256
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := BalancedOptions(width, height, 75)
	opts.UseSIMD = false
	opts.Progressive = true
	encoder, _ := NewEncoder(opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder.Encode(pixels)
	}
}

func BenchmarkEncoder_Progressive_SIMD(b *testing.B) {
	width, height := 256, 256
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := BalancedOptions(width, height, 75)
	opts.UseSIMD = true
	opts.Progressive = true
	encoder, _ := NewEncoder(opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder.Encode(pixels)
	}
}

func BenchmarkEncoder_OptimizedHuffman_Scalar(b *testing.B) {
	width, height := 256, 256
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := BalancedOptions(width, height, 75)
	opts.UseSIMD = false
	opts.OptimizeHuffman = true
	encoder, _ := NewEncoder(opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder.Encode(pixels)
	}
}

func BenchmarkEncoder_OptimizedHuffman_SIMD(b *testing.B) {
	width, height := 256, 256
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := BalancedOptions(width, height, 75)
	opts.UseSIMD = true
	opts.OptimizeHuffman = true
	encoder, _ := NewEncoder(opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder.Encode(pixels)
	}
}

func TestEncoder_CacheHit(t *testing.T) {
	width, height := 64, 64
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := BalancedOptions(width, height, 75)

	huffmanCache.ResetStats()

	encoder, _ := NewEncoder(opts)

	_, err := encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	hits, misses := encoder.GetGlobalCacheStats()
	t.Logf("After first encode: hits=%d, misses=%d", hits, misses)

	_, err = encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	hits, misses = encoder.GetGlobalCacheStats()
	hitRate := encoder.GetGlobalCacheHitRate()
	t.Logf("After second encode: hits=%d, misses=%d, hitRate=%.2f%%", hits, misses, hitRate)

	if hitRate < 50 {
		t.Errorf("expected hit rate > 50%% after cache warm, got %.2f%%", hitRate)
	}
}

func TestEncoder_CacheHitRateWithMultipleImages(t *testing.T) {
	width, height := 64, 64
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	huffmanCache.ResetStats()

	numEncodes := 10
	for i := 0; i < numEncodes; i++ {
		opts := BalancedOptions(width, height, 75)
		encoder, _ := NewEncoder(opts)
		_, err := encoder.Encode(pixels)
		if err != nil {
			t.Fatalf("Encode %d failed: %v", i, err)
		}
	}

	hits, misses := huffmanCache.Stats()
	hitRate := float64(hits) / float64(hits+misses) * 100
	t.Logf("After %d encodes: hits=%d, misses=%d, hitRate=%.2f%%", numEncodes, hits, misses, hitRate)

	if hitRate < 80 {
		t.Errorf("expected hit rate > 80%% with cache, got %.2f%%", hitRate)
	}
}

func TestEncoder_CacheHitRateDifferentQualities(t *testing.T) {
	width, height := 64, 64
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	huffmanCache.ResetStats()

	qualities := []uint8{50, 75, 90, 95}
	for _, quality := range qualities {
		for j := 0; j < 3; j++ {
			opts := BalancedOptions(width, height, quality)
			encoder, _ := NewEncoder(opts)
			_, err := encoder.Encode(pixels)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
		}
	}

	hits, misses := huffmanCache.Stats()
	hitRate := float64(hits) / float64(hits+misses) * 100
	t.Logf("After multiple quality encodes: hits=%d, misses=%d, hitRate=%.2f%%", hits, misses, hitRate)

	if hitRate < 70 {
		t.Errorf("expected hit rate > 70%% with repeated qualities, got %.2f%%", hitRate)
	}
}

func BenchmarkEncoder_CacheHitRate(b *testing.B) {
	width, height := 256, 256
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	b.ResetTimer()
	b.Run("with cache", func(b *testing.B) {
		huffmanCache.ResetStats()
		for i := 0; i < b.N; i++ {
			opts := BalancedOptions(width, height, 75)
			encoder, _ := NewEncoder(opts)
			encoder.Encode(pixels)
		}
	})

	hits, misses := huffmanCache.Stats()
	hitRate := float64(hits) / float64(hits+misses) * 100
	b.Logf("Cache hit rate: %.2f%% (hits=%d, misses=%d)", hitRate, hits, misses)
}

func TestEncoder_TrellisIntegration(t *testing.T) {
	width, height := 64, 64
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := BalancedOptions(width, height, 75)
	opts.TrellisQuant = true
	opts.TrellisLambda = 1.0
	opts.TrellisPerceptual = true

	encoder, err := NewEncoder(opts)
	if err != nil {
		t.Fatalf("NewEncoder failed: %v", err)
	}

	jpegBytes, err := encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode with trellis failed: %v", err)
	}

	_, format, err := image.Decode(bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatalf("Failed to decode JPEG with trellis: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("expected format jpeg, got %s", format)
	}

	t.Logf("Trellis enabled JPEG size: %d bytes", len(jpegBytes))
}

func TestEncoder_TrellisFileSizeComparison(t *testing.T) {
	width, height := 128, 128
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8((i * 3) % 256)
	}

	optsNoTrellis := BalancedOptions(width, height, 80)
	optsNoTrellis.TrellisQuant = false

	encoderNoTrellis, _ := NewEncoder(optsNoTrellis)
	jpegNoTrellis, err := encoderNoTrellis.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode without trellis failed: %v", err)
	}

	optsWithTrellis := BalancedOptions(width, height, 80)
	optsWithTrellis.TrellisQuant = true
	optsWithTrellis.TrellisLambda = 1.0
	optsWithTrellis.TrellisPerceptual = true

	encoderWithTrellis, _ := NewEncoder(optsWithTrellis)
	jpegWithTrellis, err := encoderWithTrellis.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode with trellis failed: %v", err)
	}

	t.Logf("Without trellis: %d bytes", len(jpegNoTrellis))
	t.Logf("With trellis: %d bytes", len(jpegWithTrellis))

	sizeDiff := float64(len(jpegNoTrellis)-len(jpegWithTrellis)) / float64(len(jpegNoTrellis)) * 100
	t.Logf("Size difference: %.2f%%", sizeDiff)
}

func TestEncoder_TrellisLambda(t *testing.T) {
	width, height := 64, 64
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := BalancedOptions(width, height, 75)
	opts.TrellisQuant = true
	opts.TrellisPerceptual = true

	testLambdas := []float64{0.5, 1.0, 2.0, 5.0}
	for _, lambda := range testLambdas {
		opts.TrellisLambda = lambda
		encoder, err := NewEncoder(opts)
		if err != nil {
			t.Fatalf("NewEncoder with lambda=%f failed: %v", lambda, err)
		}

		jpegBytes, err := encoder.Encode(pixels)
		if err != nil {
			t.Fatalf("Encode with lambda=%f failed: %v", lambda, err)
		}

		_, _, err = image.Decode(bytes.NewReader(jpegBytes))
		if err != nil {
			t.Fatalf("Failed to decode JPEG with lambda=%f: %v", lambda, err)
		}

		t.Logf("Lambda=%.1f: %d bytes", lambda, len(jpegBytes))
	}
}

func TestEncoder_TrellisPerceptual(t *testing.T) {
	width, height := 64, 64
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8((i * 7) % 256)
	}

	opts := BalancedOptions(width, height, 75)
	opts.TrellisQuant = true
	opts.TrellisLambda = 1.0

	optsPerceptual := opts
	optsPerceptual.TrellisPerceptual = true
	encoderPerceptual, _ := NewEncoder(optsPerceptual)
	jpegPerceptual, err := encoderPerceptual.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode with perceptual=true failed: %v", err)
	}

	optsNonPerceptual := opts
	optsNonPerceptual.TrellisPerceptual = false
	encoderNonPerceptual, _ := NewEncoder(optsNonPerceptual)
	jpegNonPerceptual, err := encoderNonPerceptual.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode with perceptual=false failed: %v", err)
	}

	t.Logf("Perceptual=true: %d bytes", len(jpegPerceptual))
	t.Logf("Perceptual=false: %d bytes", len(jpegNonPerceptual))
}

func TestEncoder_TrellisWithProgressive(t *testing.T) {
	width, height := 64, 64
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := MaxOptions(width, height, 80)
	encoder, _ := NewEncoder(opts)

	jpegBytes, err := encoder.Encode(pixels)
	if err != nil {
		t.Fatalf("Encode with trellis+progressive failed: %v", err)
	}

	_, format, err := image.Decode(bytes.NewReader(jpegBytes))
	if err != nil {
		t.Fatalf("Failed to decode JPEG with trellis+progressive: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("expected format jpeg, got %s", format)
	}

	t.Logf("Trellis+Progressive JPEG size: %d bytes", len(jpegBytes))
}

func BenchmarkEncoder_Trellis(b *testing.B) {
	width, height := 256, 256
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	opts := BalancedOptions(width, height, 75)
	opts.TrellisQuant = true
	opts.TrellisLambda = 1.0
	opts.TrellisPerceptual = true
	encoder, _ := NewEncoder(opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder.Encode(pixels)
	}
}

func BenchmarkEncoder_TrellisComparison(b *testing.B) {
	width, height := 256, 256
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	b.Run("without trellis", func(b *testing.B) {
		opts := BalancedOptions(width, height, 75)
		opts.TrellisQuant = false
		encoder, _ := NewEncoder(opts)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder.Encode(pixels)
		}
	})

	b.Run("with trellis", func(b *testing.B) {
		opts := BalancedOptions(width, height, 75)
		opts.TrellisQuant = true
		opts.TrellisLambda = 1.0
		opts.TrellisPerceptual = true
		encoder, _ := NewEncoder(opts)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoder.Encode(pixels)
		}
	})
}
