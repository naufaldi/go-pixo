package jpeg

// JPEG Markers
const (
	SOI  uint16 = 0xFFD8 // Start of Image
	EOI  uint16 = 0xFFD9 // End of Image
	APP0 uint16 = 0xFFE0 // JFIF Application Segment
	DQT  uint16 = 0xFFDB // Define Quantization Table
	SOF0 uint16 = 0xFFC0 // Start of Frame - Baseline DCT
	SOF2 uint16 = 0xFFC2 // Start of Frame - Progressive DCT
	DHT  uint16 = 0xFFC4 // Define Huffman Table
	SOS  uint16 = 0xFFDA // Start of Scan
	DRI  uint16 = 0xFFDD // Define Restart Interval
)

// ColorType represents the color space of the image.
type ColorType uint8

const (
	ColorGrayscale ColorType = 1 // JPEG format value for 1 component
	ColorRGB       ColorType = 3 // JPEG format value for 3 components
)

// Subsampling represents the chroma subsampling mode.
type Subsampling uint8

const (
	Subsampling444 Subsampling = iota // 4:4:4, no subsampling
	Subsampling420                    // 4:2:0, 2x2 chroma downsample
)
