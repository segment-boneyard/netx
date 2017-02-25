package httpx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultMaxAttempts is the default number of attempts used by RetryHandler
	// and RetryTransport.
	DefaultMaxAttempts = 10
)

// A RetryHandler is a http.Handler which retries calls to its sub-handler if
// they fail with a 5xx code. When a request is retried the handler will apply
// an exponential backoff to maximize the chances of success (because it is
// usually unlikely that a failed request will succeed right away).
//
// Note that only idempotent methods are retried, because the handler doesn't
// have enough context about why it failed, it wouldn't be safe to retry other
// HTTP methods.
type RetryHandler struct {
	// Handler is the sub-handler that the RetryHandler delegates requests to.
	//
	// ServeHTTP will panic if Handler is nil.
	Handler http.Handler

	// MaxAttampts is the maximum number of attempts that the handler will make
	// at handling a single request.
	// Zero means to use a default value.
	MaxAttempts int
}

// ServeHTTP satisfies the http.Handler interface.
func (h *RetryHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body := &retryRequestBody{ReadCloser: req.Body}
	req.Body = body

	res := &retryResponseWriter{ResponseWriter: w}
	max := h.MaxAttempts
	if max == 0 {
		max = DefaultMaxAttempts
	}

	for attempt := 0; true; {
		res.status = 0
		res.header = make(http.Header, 10)
		res.buffer.Reset()

		h.Handler.ServeHTTP(res, req)

		if res.status < 500 {
			return // success
		}

		if body.n != 0 {
			break
		}

		if !isRetriable(res.status) {
			break
		}

		if !isIdempotent(req.Method) {
			break
		}

		if attempt++; attempt >= max {
			break
		}

		if sleep(req.Context(), backoff(attempt)) != nil {
			break
		}
	}

	if res.status == 0 {
		res.status = http.StatusServiceUnavailable
	}

	// 5xx error, write the buffered response to the original writer.
	copyHeader(w.Header(), res.header)
	w.WriteHeader(res.status)
	res.buffer.WriteTo(w)
}

// RetryTransport is a http.RoundTripper which retries calls to its sub-handler
// if they failed with connection or server errors. When a request is retried
// the handler will apply an exponential backoff to maximize the chances of
// success (because it is usually unlikely that a failed request will succeed
// right away).
//
// Note that only idempotent methods are retried, because the handler doesn't
// have enough context about why it failed, it wouldn't be safe to retry other
// HTTP methods.
type RetryTransport struct {
	// Transport is the sub-transport that the RetryTransport delegates requests
	// to.
	//
	// http.DefaultTransport is used if Transport is nil.
	Transport http.RoundTripper

	// MaxAttampts is the maximum number of attempts that the handler will make
	// at handling a single request.
	// Zero means to use a default value.
	MaxAttempts int
}

// RoundTrip satisfies the http.RoundTripper interface.
func (t *RetryTransport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	body := &retryRequestBody{ReadCloser: req.Body}
	req.Body = body

	max := t.MaxAttempts
	if max == 0 {
		max = DefaultMaxAttempts
	}

	for attempt := 0; true; {
		if res, err = transport.RoundTrip(req); err == nil {
			if res.StatusCode < 500 || !isRetriable(res.StatusCode) {
				break // success
			}
		}

		if body.n != 0 {
			err = fmt.Errorf("%s %s: failed and cannot be retried because %d bytes of the body have already been sent", req.Method, req.URL.Path, body.n)
			break
		}

		if !isIdempotent(req.Method) {
			err = fmt.Errorf("%s %s: failed and cannot be retried because the method is not idempotent", req.Method, req.URL.Path)
			break
		}

		if attempt++; attempt >= max {
			err = fmt.Errorf("%s %s: failed %d times: %s", req.Method, req.URL.Path, attempt, err)
			break
		}

		if err = sleep(req.Context(), backoff(attempt)); err != nil {
			break
		}
	}

	return
}

// retryResponseWriter is a http.ResponseWriter which captures 5xx responses.
type retryResponseWriter struct {
	http.ResponseWriter
	status int
	header http.Header
	buffer bytes.Buffer
}

// Header satisfies the http.ResponseWriter interface.
func (w *retryResponseWriter) Header() http.Header {
	return w.header
}

// WriteHeader satisfies the http.ResponseWriter interface.
func (w *retryResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		if status < 500 {
			copyHeader(w.ResponseWriter.Header(), w.header)
			w.ResponseWriter.WriteHeader(status)
		}
	}
}

// Write satisfies the http.ResponseWriter interface.
func (w *retryResponseWriter) Write(b []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	if w.status >= 500 {
		return w.buffer.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// retryRequestBody is a io.ReadCloser wrapper which counts how many bytes were
// processed by a request body.
type retryRequestBody struct {
	io.ReadCloser
	n int
}

// Read satisfies the io.Reader interface.
func (r *retryRequestBody) Read(b []byte) (n int, err error) {
	if n, err = r.ReadCloser.Read(b); n > 0 {
		r.n += n
	}
	return
}

// backoff returns the amount of time a goroutine should wait before retrying
// what it was doing considering that it already made n attempts.
func backoff(n int) time.Duration {
	return time.Duration(n*n) * 10 * time.Millisecond
}

// sleep puts the goroutine to sleep until either ctx is canceled or d amount of
// time elapses.
func sleep(ctx context.Context, d time.Duration) (err error) {
	if d != 0 {
		timer := time.NewTimer(d)
		select {
		case <-timer.C:
		case <-ctx.Done():
			err = ctx.Err()
		}
		timer.Stop()
	}
	return
}
