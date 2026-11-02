package wasm

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // Register the bounded fallback decoders.
	_ "image/png"

	_ "golang.org/x/image/webp" // Register WebP for validation without re-encoding.

	"github.com/mac/go-pixo/src/internal/imagelimit"
)

// ValidateImageBytes fully decodes bounded PNG, JPEG or WebP input without re-encoding it.
func ValidateImageBytes(input []byte) error {
	if err := imagelimit.CheckEncodedSize(len(input)); err != nil {
		return err
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(input))
	if err != nil {
		return fmt.Errorf("image: decode configuration: %w", err)
	}
	if format != "png" && format != "jpeg" && format != "webp" {
		return fmt.Errorf("image: unsupported validation format %q", format)
	}
	if err = imagelimit.CheckDimensions(config.Width, config.Height); err != nil {
		return err
	}
	decoded, _, err := image.Decode(bytes.NewReader(input))
	if err != nil {
		return fmt.Errorf("image: decode input: %w", err)
	}
	if decoded.Bounds().Dx() != config.Width || decoded.Bounds().Dy() != config.Height {
		return fmt.Errorf("image: decoded dimensions disagree with header")
	}
	return nil
}
