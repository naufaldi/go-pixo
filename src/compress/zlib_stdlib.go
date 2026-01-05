package compress

import (
	"bytes"
	"compress/zlib"
)

// ZlibCompressStdlib compresses data using the Go standard library zlib writer.
// This produces a complete zlib stream (header + DEFLATE + Adler32).
func ZlibCompressStdlib(data []byte, level int) ([]byte, error) {
	if level < 1 {
		level = 1
	}
	if level > 9 {
		level = 9
	}

	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
