package compress

import "testing"

func TestFindMatchNoMatch(t *testing.T) {
	window := NewSlidingWindow(32)
	window.WriteBytes([]byte("ABC"))
	lookahead := []byte("XYZ")
	match, found := FindMatch(window, lookahead, 0)
	if found {
		t.Errorf("FindMatch found match when none should exist: %+v", match)
	}
}

func TestFindMatchSimpleRepeat(t *testing.T) {
	window := NewSlidingWindow(32)
	window.WriteBytes([]byte("ABC"))
	lookahead := []byte("ABCABC")
	match, found := FindMatch(window, lookahead, 0)
	if !found {
		t.Fatal("FindMatch should find match for 'ABCABC'")
	}
	if match.Distance != 3 {
		t.Errorf("Match.Distance = %d, want 3", match.Distance)
	}
	if match.Length != 3 {
		t.Errorf("Match.Length = %d, want 3", match.Length)
	}
}

func TestFindMatchMinLength(t *testing.T) {
	window := NewSlidingWindow(32)
	window.WriteBytes([]byte("AB"))
	lookahead := []byte("AB")
	match, found := FindMatch(window, lookahead, 0)
	if found {
		t.Errorf("FindMatch found match of length 2, but min is 3: %+v", match)
	}

	window2 := NewSlidingWindow(32)
	window2.WriteBytes([]byte("ABC"))
	lookahead2 := []byte("ABC")
	match2, found2 := FindMatch(window2, lookahead2, 0)
	if !found2 {
		t.Fatal("FindMatch should find match of length 3")
	}
	if match2.Length != 3 {
		t.Errorf("Match.Length = %d, want 3", match2.Length)
	}
}

func TestFindMatchMaxLength(t *testing.T) {
	window := NewSlidingWindow(32768)
	longPattern := make([]byte, 300)
	for i := range longPattern {
		longPattern[i] = byte('A' + (i % 26))
	}
	window.WriteBytes(longPattern)

	lookahead := make([]byte, 300)
	copy(lookahead, longPattern)
	match, found := FindMatch(window, lookahead, 0)
	if !found {
		t.Fatal("FindMatch should find match")
	}
	if match.Length > maxMatchLength {
		t.Errorf("Match.Length = %d, exceeds max %d", match.Length, maxMatchLength)
	}
	if match.Length < maxMatchLength && match.Length < 300 {
		t.Logf("Match.Length = %d (limited by maxMatchLength)", match.Length)
	}
}

func TestFindMatchMaxDistance(t *testing.T) {
	window := NewSlidingWindow(32768)
	pattern := make([]byte, 100)
	for i := range pattern {
		pattern[i] = byte('A')
	}
	window.WriteBytes(pattern)

	lookahead := make([]byte, 100)
	copy(lookahead, pattern)
	match, found := FindMatch(window, lookahead, 0)
	if !found {
		t.Fatal("FindMatch should find match")
	}
	if match.Distance > maxDistance {
		t.Errorf("Match.Distance = %d, exceeds max %d", match.Distance, maxDistance)
	}
}

func TestFindMatchLongestMatch(t *testing.T) {
	window := NewSlidingWindow(32)
	window.WriteBytes([]byte("ABC"))
	lookahead := []byte("ABCABCDEF")
	match, found := FindMatch(window, lookahead, 0)
	if !found {
		t.Fatal("FindMatch should find match")
	}
	if match.Length != 3 {
		t.Errorf("Match.Length = %d, want 3", match.Length)
	}

	window2 := NewSlidingWindow(32)
	window2.WriteBytes([]byte("ABCABC"))
	lookahead2 := []byte("ABCABC")
	match2, found2 := FindMatch(window2, lookahead2, 0)
	if !found2 {
		t.Fatal("FindMatch should find match")
	}
	if match2.Length != 6 {
		t.Errorf("Match.Length = %d, want 6", match2.Length)
	}
}
