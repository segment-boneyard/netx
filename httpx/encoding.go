package httpx

import (
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"strings"
)

// ContentEncoding is an interfae implemented by types that provide the
// implementation of a content encoding for HTTP responses.
type ContentEncoding interface {
	// Coding returns the format in which the content encoding's writers
	// can encode HTTP responses.
	Coding() string

	// NewReader wraps r in a reader that supports the content encoding's
	// format.
	NewReader(r io.Reader) (io.ReadCloser, error)

	// NewWriter wraps w in a writer that applies the content encoding's
	// format.
	NewWriter(w io.Writer) (io.WriteCloser, error)
}

// NewEncodingTransport wraps transport to support decoding the responses with
// specified content encodings.
//
// If contentEncodings is nil (no arguments were passed) the returned transport
// uses DefaultEncodings.
func NewEncodingTransport(transport http.RoundTripper, contentEncodings ...ContentEncoding) http.RoundTripper {
	if contentEncodings == nil {
		contentEncodings = defaultEncodings()
	}

	encodings := make(map[string]ContentEncoding, len(contentEncodings))
	codings := make([]string, 0, len(contentEncodings))

	for _, encoding := range contentEncodings {
		coding := encoding.Coding()
		codings = append(codings, coding)
		encodings[coding] = encoding
	}

	acceptEncoding := strings.Join(codings, ", ")

	return RoundTripperFunc(func(req *http.Request) (res *http.Response, err error) {
		req.Header["Accept-Encoding"] = []string{acceptEncoding}

		if res, err = transport.RoundTrip(req); err != nil {
			return
		}

		if coding := res.Header.Get("Content-Encoding"); len(coding) != 0 {
			if encoding := encodings[coding]; encoding != nil {
				var decoder io.ReadCloser

				if decoder, err = encoding.NewReader(res.Body); err != nil {
					res.Body.Close()
					return
				}

				res.Body = &contentEncodingReader{
					decoder: decoder,
					body:    res.Body,
				}

				delete(res.Header, "Content-Encoding")
			}
		}

		return
	})
}

type contentEncodingReader struct {
	decoder io.ReadCloser
	body    io.ReadCloser
}

func (r *contentEncodingReader) Read(b []byte) (int, error) {
	return r.decoder.Read(b)
}

func (r *contentEncodingReader) Close() error {
	r.decoder.Close()
	r.body.Close()
	return nil
}

// NewEncodingHandler wraps handler to support encoding the responses by
// negotiating the coding based on the given list of supported content encodings.
//
// If contentEncodings is nil (no arguments were passed) the returned handler
// uses DefaultEncodings.
func NewEncodingHandler(handler http.Handler, contentEncodings ...ContentEncoding) http.Handler {
	if contentEncodings == nil {
		contentEncodings = defaultEncodings()
	}

	encodings := make(map[string]ContentEncoding, len(contentEncodings))
	codings := make([]string, 0, len(contentEncodings))

	for _, encoding := range contentEncodings {
		coding := encoding.Coding()
		encodings[coding] = encoding
		codings = append(codings, coding)
	}

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		coding := NegotiateEncoding(req.Header.Get("Accept-Encoding"), codings...)

		if len(coding) != 0 {
			if w, err := encodings[coding].NewWriter(res); err == nil {
				defer w.Close()

				h := res.Header()
				h.Set("Content-Encoding", coding)

				res = &contentEncodingWriter{res, w}
				delete(req.Header, "Accept-Encoding")
			}
		}

		handler.ServeHTTP(res, req)
	})
}

type contentEncodingWriter struct {
	http.ResponseWriter
	io.Writer
}

func (w *contentEncodingWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// DeflateEncoding implements the ContentEncoding interface for the deflate
// algorithm.
type DeflateEncoding struct {
	Level int
}

// NewDeflateEncoding creates a new content encoding with the default compression
// level.
func NewDeflateEncoding() *DeflateEncoding {
	return NewDeflateEncodingLevel(flate.DefaultCompression)
}

// NewDeflateEncodingLevel creates a new content encoding with the given
// compression level.
func NewDeflateEncodingLevel(level int) *DeflateEncoding {
	return &DeflateEncoding{
		Level: level,
	}
}

// Coding satsifies the ContentEncoding interface.
func (e *DeflateEncoding) Coding() string {
	return "deflate"
}

// NewReader satisfies the ContentEncoding interface.
func (e *DeflateEncoding) NewReader(r io.Reader) (io.ReadCloser, error) {
	return flate.NewReader(r), nil
}

// NewWriter satsifies the ContentEncoding interface.
func (e *DeflateEncoding) NewWriter(w io.Writer) (io.WriteCloser, error) {
	return flate.NewWriter(w, e.Level)
}

// GzipEncoding implements the ContentEncoding interface for the gzip
// algorithm.
type GzipEncoding struct {
	Level int
}

// NewGzipEncoding creates a new content encoding with the default compression
// level.
func NewGzipEncoding() *GzipEncoding {
	return NewGzipEncodingLevel(gzip.DefaultCompression)
}

// NewGzipEncodingLevel creates a new content encoding with the given
// compression level.
func NewGzipEncodingLevel(level int) *GzipEncoding {
	return &GzipEncoding{
		Level: level,
	}
}

// Coding satsifies the ContentEncoding interface.
func (e *GzipEncoding) Coding() string {
	return "gzip"
}

// NewReader satisfies the ContentEncoding interface.
func (e *GzipEncoding) NewReader(r io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}

// NewWriter satsifies the ContentEncoding interface.
func (e *GzipEncoding) NewWriter(w io.Writer) (io.WriteCloser, error) {
	return gzip.NewWriterLevel(w, e.Level)
}

// ZlibEncoding implements the ContentEncoding interface for the zlib
// algorithm.
type ZlibEncoding struct {
	Level int
}

// NewZlibEncoding creates a new content encoding with the default compression
// level.
func NewZlibEncoding() *ZlibEncoding {
	return NewZlibEncodingLevel(zlib.DefaultCompression)
}

// NewZlibEncodingLevel creates a new content encoding with the given
// compression level.
func NewZlibEncodingLevel(level int) *ZlibEncoding {
	return &ZlibEncoding{
		Level: level,
	}
}

// Coding satsifies the ContentEncoding interface.
func (e *ZlibEncoding) Coding() string {
	return "zlib"
}

// NewReader satisfies the ContentEncoding interface.
func (e *ZlibEncoding) NewReader(r io.Reader) (io.ReadCloser, error) {
	return zlib.NewReader(r)
}

// NewWriter satsifies the ContentEncoding interface.
func (e *ZlibEncoding) NewWriter(w io.Writer) (io.WriteCloser, error) {
	return zlib.NewWriterLevel(w, e.Level)
}

func defaultEncodings() []ContentEncoding {
	return []ContentEncoding{
		NewGzipEncoding(),
		NewZlibEncoding(),
		NewDeflateEncoding(),
	}
}
