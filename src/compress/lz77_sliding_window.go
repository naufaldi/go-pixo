package compress

// SlidingWindow maintains a fixed-size buffer of recently seen bytes.
// It implements a circular buffer for efficient LZ77 matching.
type SlidingWindow struct {
	buffer []byte
	size   int
	pos    int
	full   bool
}

// NewSlidingWindow creates a new sliding window with the specified size.
// Size must be at most 32768 (DEFLATE maximum).
func NewSlidingWindow(size int) *SlidingWindow {
	if size > 32768 {
		size = 32768
	}
	return &SlidingWindow{
		buffer: make([]byte, size),
		size:   size,
		pos:    0,
		full:   false,
	}
}

// Write adds a byte to the window, overwriting the oldest byte if full.
func (sw *SlidingWindow) Write(b byte) {
	sw.buffer[sw.pos] = b
	sw.pos++
	if sw.pos >= sw.size {
		sw.pos = 0
		sw.full = true
	}
}

// Bytes returns all bytes currently in the window in chronological order.
// The most recent byte is at the end.
func (sw *SlidingWindow) Bytes() []byte {
	if !sw.full {
		return sw.buffer[:sw.pos]
	}
	result := make([]byte, sw.size)
	copy(result, sw.buffer[sw.pos:])
	copy(result[sw.size-sw.pos:], sw.buffer[:sw.pos])
	return result
}

// Len returns the number of bytes currently in the window.
func (sw *SlidingWindow) Len() int {
	if sw.full {
		return sw.size
	}
	return sw.pos
}

// WriteBytes adds multiple bytes to the window.
func (sw *SlidingWindow) WriteBytes(data []byte) {
	for _, b := range data {
		sw.Write(b)
	}
}
