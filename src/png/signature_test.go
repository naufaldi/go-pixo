package png

import "testing"

func TestIsValidSignature_ValidSignature(t *testing.T) {
	validSig := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if !IsValidSignature(validSig) {
		t.Errorf("IsValidSignature() = false, want true for valid PNG signature")
	}
}

func TestIsValidSignature_CorruptedSignature(t *testing.T) {
	corruptedSig := []byte{0x00, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if IsValidSignature(corruptedSig) {
		t.Errorf("IsValidSignature() = true, want false for corrupted signature")
	}
}

func TestIsValidSignature_ShortBuffer(t *testing.T) {
	shortBuf := []byte{0x89, 0x50}
	if IsValidSignature(shortBuf) {
		t.Errorf("IsValidSignature() = true, want false for buffer shorter than 8 bytes")
	}
}

func TestIsValidSignature_EmptyBuffer(t *testing.T) {
	emptyBuf := []byte{}
	if IsValidSignature(emptyBuf) {
		t.Errorf("IsValidSignature() = true, want false for empty buffer")
	}
}

func TestSignature(t *testing.T) {
	sig := Signature()
	expected := PNG_SIGNATURE[:]
	if len(sig) != len(expected) {
		t.Errorf("Signature() length = %d, want %d", len(sig), len(expected))
	}
	for i := range sig {
		if sig[i] != expected[i] {
			t.Errorf("Signature()[%d] = 0x%02x, want 0x%02x", i, sig[i], expected[i])
		}
	}
}

func TestConstants(t *testing.T) {
	if ChunkIHDR != "IHDR" {
		t.Errorf("ChunkIHDR = %q, want %q", ChunkIHDR, "IHDR")
	}
	if ColorRGBA != 6 {
		t.Errorf("ColorRGBA = %d, want %d", ColorRGBA, 6)
	}
	if FilterPaeth != 4 {
		t.Errorf("FilterPaeth = %d, want %d", FilterPaeth, 4)
	}
}
