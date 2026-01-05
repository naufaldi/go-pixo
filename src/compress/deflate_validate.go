package compress

import (
	"bytes"
	"compress/flate"
	"io"
)

func validateDeflateRoundTrip(deflated []byte, original []byte) bool {
	r := flate.NewReader(bytes.NewReader(deflated))
	defer func() { _ = r.Close() }()

	decoded, err := io.ReadAll(r)
	if err != nil {
		return false
	}
	return bytes.Equal(decoded, original)
}
