package httpx

import (
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
