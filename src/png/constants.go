package png

var pngSignature = [8]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

// ChunkType identifies a PNG chunk type (for example, "IHDR").
type ChunkType string

const (
	// ChunkIHDR is the IHDR chunk type.
	ChunkIHDR ChunkType = "IHDR"
	// ChunkIDAT is the IDAT chunk type.
	ChunkIDAT ChunkType = "IDAT"
	// ChunkIEND is the IEND chunk type.
	ChunkIEND ChunkType = "IEND"
)

// ColorType identifies a PNG color type.
type ColorType uint8

const (
	// ColorGrayscale is PNG color type 0.
	ColorGrayscale ColorType = 0
	// ColorRGB is PNG color type 2.
	ColorRGB ColorType = 2
	// ColorRGBA is PNG color type 6.
	ColorRGBA ColorType = 6
	// ColorIndexed is PNG color type 3.
	ColorIndexed ColorType = 3
)
