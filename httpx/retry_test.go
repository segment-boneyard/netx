package httpx

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/netx/httpx/httpxtest"
)

func TestRetryTransportDefault(t *testing.T) {
	httpxtest.TestTransport(t, func() http.RoundTripper {
		return &RetryTransport{}
	})
}

func TestRetryTransportConfigured(t *testing.T) {
	httpxtest.TestTransport(t, func() http.RoundTripper {
		return &RetryTransport{
			MaxAttempts: 1,
		}
	})
}

func TestRetryHandler(t *testing.T) {
	tests := []struct {
		method      string
		body        string
		status      int
		maxAttempts int
	}{
		{ // HEAD + default max attempts
			method: "HEAD",
			status: http.StatusOK,
		},
		{ // GET + default max attempts
			method: "GET",
			status: http.StatusOK,
		},
		{ // PUT + default max attempts
			method: "PUT",
			status: http.StatusOK,
		},
		{ // DELETE + default max attempts
			method: "DELETE",
			status: http.StatusOK,
		},
		{ // POST (not idempotent) + default max attempts
			method: "POST",
			status: http.StatusInternalServerError,
		},
		{ // GET + low max attempts
			method:      "GET",
			status:      http.StatusInternalServerError,
			maxAttempts: 1,
		},
		{ // PUT + non-empty request boyd
			method: "PUT",
			body:   "Hello World!",
			status: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			req := httptest.NewRequest(test.method, "/", strings.NewReader(test.body))
			res := httptest.NewRecorder()

			attempt := 0
			handler := &RetryHandler{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					io.Copy(ioutil.Discard, req.Body)
					req.Body.Close()
					if attempt == 0 {
						w.WriteHeader(http.StatusInternalServerError)
					} else {
						w.WriteHeader(http.StatusOK)
					}
					attempt++
				}),
				MaxAttempts: test.maxAttempts,
			}
			handler.ServeHTTP(res, req)

			if res.Code != test.status {
				t.Errorf("bad status code: expected %d but got %d", test.status, res.Code)
			}
		})
	}
}

func TestSleepTimeout(t *testing.T) {
	ctx := context.Background()
	t0 := time.Now()
	err := sleep(ctx, 10*time.Millisecond)
	t1 := time.Now()

	if err != nil {
		t.Error(err)
	}

	if t1.Sub(t0) < (10 * time.Millisecond) {
		t.Error("sleep returned too early")
	}
}

func TestSleepCanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if err := sleep(ctx, 1*time.Second); err != context.DeadlineExceeded {
		t.Error(err)
	}
}
