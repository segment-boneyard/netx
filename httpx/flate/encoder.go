package flate

import (
	"compress/flate"
	"io"
)

// ContentEncoder implements the httpx.ContentEncoder interface for the flate
// algorithm.
type ContentEncoder struct {
	Level int
}

// NewContentEncoder creates a new content encoder with the default compression
// level.
func NewContentEncoder() *ContentEncoder {
	return NewContentEncoderLevel(flate.DefaultCompression)
}

// NewContentEncoderLevel creates a new content encoder with the given
// compression level.
func NewContentEncoderLevel(level int) *ContentEncoder {
	return &ContentEncoder{
		Level: level,
	}
}

// Coding satsifies httpx.ContentEncoder.
func (e *ContentEncoder) Coding() string {
	return "deflate"
}

// NewWriter satsifies httpx.ContentEncoder.
func (e *ContentEncoder) NewWriter(w io.Writer) io.WriteCloser {
	z, err := flate.NewWriter(w, e.Level)
	if err != nil {
		panic(err)
	}
	return z
}
