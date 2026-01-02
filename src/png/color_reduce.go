package png

import (
	"errors"
)

var ErrCannotReduceColorType = errors.New("png: cannot reduce color type for given pixels")

func ReduceToGrayscale(pixels []byte, width, height int, colorType ColorType) ([]byte, ColorType, error) {
	if !CanReduceToGrayscale(pixels, width, height, colorType) {
		return nil, colorType, ErrCannotReduceColorType
	}

	switch colorType {
	case ColorGrayscale:
		return pixels, ColorGrayscale, nil
	case ColorRGB:
		return reduceRGBToGrayscale(pixels, width, height), ColorGrayscale, nil
	case ColorRGBA:
		return reduceRGBAToGrayscale(pixels, width, height), ColorGrayscale, nil
	default:
		return nil, colorType, ErrCannotReduceColorType
	}
}

func reduceRGBToGrayscale(pixels []byte, width, height int) []byte {
	result := make([]byte, width*height)
	for i := 0; i < width*height; i++ {
		offset := i * 3
		result[i] = pixels[offset]
	}
	return result
}

func reduceRGBAToGrayscale(pixels []byte, width, height int) []byte {
	result := make([]byte, width*height)
	for i := 0; i < width*height; i++ {
		offset := i * 4
		result[i] = pixels[offset]
	}
	return result
}

func ReduceToRGB(pixels []byte, width, height int) ([]byte, ColorType, error) {
	if !CanReduceToRGB(pixels, width, height) {
		return nil, ColorRGBA, ErrCannotReduceColorType
	}

	result := make([]byte, width*height*3)
	for i := 0; i < width*height; i++ {
		srcOffset := i * 4
		dstOffset := i * 3
		result[dstOffset] = pixels[srcOffset]
		result[dstOffset+1] = pixels[srcOffset+1]
		result[dstOffset+2] = pixels[srcOffset+2]
	}
	return result, ColorRGB, nil
}
