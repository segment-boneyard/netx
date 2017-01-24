package httpx

import (
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
)

// ContentEncoder is an interfae implemented by types that provide the
// implementation of a content encoding for HTTP responses.
type ContentEncoder interface {
	// Coding returns the format in which the content encoder's writers
	// can encode HTTP responses.
	Coding() string

	// NewWriter wraps w in a writer that applies an the content encoder's
	// format to all bytes it receives.
	NewWriter(w io.Writer) io.WriteCloser
}

// NewEncodingHandler wraps handler to support encoding the responses by
// negotiating the coding based on the given list of supported content encoders.
func NewEncodingHandler(handler http.Handler, contentEncoders ...ContentEncoder) http.Handler {
	encoders := make(map[string]ContentEncoder, len(contentEncoders))
	codings := make([]string, 0, len(contentEncoders))

	for _, encoder := range contentEncoders {
		coding := encoder.Coding()
		encoders[coding] = encoder
		codings = append(codings, coding)
	}

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		coding := NegotiateEncoding(req.Header.Get("Accept-Encoding"), codings...)

		if len(coding) != 0 {
			w := encoders[coding].NewWriter(res)
			defer w.Close()

			h := res.Header()
			h.Set("Content-Encoding", coding)

			res = contentEncoderWriter{res, w}
		}

		handler.ServeHTTP(res, req)
	})
}

type contentEncoderWriter struct {
	http.ResponseWriter
	io.Writer
}

func (w contentEncoderWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// DeflateEncoder implements the ContentEncoder interface for the deflate
// algorithm.
type DeflateEncoder struct {
	Level int
}

// NewDeflateEncoder creates a new content encoder with the default compression
// level.
func NewDeflateEncoder() *DeflateEncoder {
	return NewDeflateEncoderLevel(flate.DefaultCompression)
}

// NewDeflateEncoderLevel creates a new content encoder with the given
// compression level.
func NewDeflateEncoderLevel(level int) *DeflateEncoder {
	return &DeflateEncoder{
		Level: level,
	}
}

// Coding satsifies ContentEncoder.
func (e *DeflateEncoder) Coding() string {
	return "deflate"
}

// NewWriter satsifies ContentEncoder.
func (e *DeflateEncoder) NewWriter(w io.Writer) io.WriteCloser {
	z, err := flate.NewWriter(w, e.Level)
	if err != nil {
		panic(err)
	}
	return z
}

// GzipEncoder implements the ContentEncoder interface for the gzip
// algorithm.
type GzipEncoder struct {
	Level int
}

// NewGzipEncoder creates a new content encoder with the default compression
// level.
func NewGzipEncoder() *GzipEncoder {
	return NewGzipEncoderLevel(gzip.DefaultCompression)
}

// NewGzipEncoderLevel creates a new content encoder with the given
// compression level.
func NewGzipEncoderLevel(level int) *GzipEncoder {
	return &GzipEncoder{
		Level: level,
	}
}

// Coding satsifies ContentEncoder.
func (e *GzipEncoder) Coding() string {
	return "gzip"
}

// NewWriter satsifies ContentEncoder.
func (e *GzipEncoder) NewWriter(w io.Writer) io.WriteCloser {
	z, err := gzip.NewWriterLevel(w, e.Level)
	if err != nil {
		panic(err)
	}
	return z
}

// ZlibEncoder implements the ContentEncoder interface for the zlib
// algorithm.
type ZlibEncoder struct {
	Level int
}

// NewZlibEncoder creates a new content encoder with the default compression
// level.
func NewZlibEncoder() *ZlibEncoder {
	return NewZlibEncoderLevel(zlib.DefaultCompression)
}

// NewZlibEncoderLevel creates a new content encoder with the given
// compression level.
func NewZlibEncoderLevel(level int) *ZlibEncoder {
	return &ZlibEncoder{
		Level: level,
	}
}

// Coding satsifies ContentEncoder.
func (e *ZlibEncoder) Coding() string {
	return "zlib"
}

// NewWriter satsifies ContentEncoder.
func (e *ZlibEncoder) NewWriter(w io.Writer) io.WriteCloser {
	z, err := zlib.NewWriterLevel(w, e.Level)
	if err != nil {
		panic(err)
	}
	return z
}
