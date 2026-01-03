package png

var PNG_SIGNATURE = [8]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

type ChunkType string

const (
	ChunkIHDR ChunkType = "IHDR"
	ChunkIDAT ChunkType = "IDAT"
	ChunkIEND ChunkType = "IEND"
)

type ColorType uint8

const (
	ColorGrayscale ColorType = 0
	ColorRGB       ColorType = 2
	ColorRGBA      ColorType = 6
	ColorIndexed   ColorType = 3
)
