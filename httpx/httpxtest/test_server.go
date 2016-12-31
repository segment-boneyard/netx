package httpxtest

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

// ServerConfig is used to configure the HTTP server started by MakeServer.
type ServerConfig struct {
	Handler        http.Handler
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
}

// MakeServer is a function called by the TestServer test suite to create a new
// HTTP server that runs the given config.
// The function must return the URL at which the server can be accessed and a
// closer function to terminate the server.
type MakeServer func(ServerConfig) (url string, close func())

// TestServer is a test suite for HTTP servers, inspired from
// golang.org/x/net/nettest.TestConn.
func TestServer(t *testing.T, f MakeServer) {
	run := func(name string, test func(*testing.T, MakeServer)) {
		t.Run(name, func(t *testing.T) { test(t, f) })
	}
	run("Basic", testServerBasic)
	run("Transfer-Encoding:chunked", testServerTransferEncodingChunked)
	run("ErrBodyNotAllowed", testServerErrBodyNotAllowed)
	run("ErrContentLength", testServerErrContentLength)
}

func testServerBasic(t *testing.T, f MakeServer) {
	url, close := f(ServerConfig{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Length", "12")
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte("Hello World!"))
		}),
	})
	defer close()

	res, err := http.Get(url + "/")
	if err != nil {
		t.Error(err)
		return
	}

	buf := &bytes.Buffer{}
	buf.ReadFrom(res.Body)

	if err := res.Body.Close(); err != nil {
		t.Error("error closing the response body:", err)
	}

	if res.StatusCode != http.StatusAccepted {
		t.Error("bad response code:", res.StatusCode)
	}

	if s := buf.String(); s != "Hello World!" {
		t.Error("bad response body:", s)
	}
}

func testServerTransferEncodingChunked(t *testing.T, f MakeServer) {
	url, close := f(ServerConfig{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// No Content-Length is set, the server should be using
			// "Transfer-Encoding: chunked" in the response.
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte("Hello World!"))
		}),
	})
	defer close()

	res, err := http.Get(url + "/")
	if err != nil {
		t.Error(err)
		return
	}

	buf := &bytes.Buffer{}
	buf.ReadFrom(res.Body)

	if err := res.Body.Close(); err != nil {
		t.Error("error closing the response body:", err)
	}

	if res.StatusCode != http.StatusAccepted {
		t.Error("bad response code:", res.StatusCode)
	}

	if s := buf.String(); s != "Hello World!" {
		t.Error("bad response body:", s)
	}
}

func testServerErrBodyNotAllowed(t *testing.T, f MakeServer) {
	tests := []struct {
		reason  string
		handler http.Handler
	}{
		{
			reason: "101 Switching Protocols",
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusSwitchingProtocols)
			}),
		},
		{
			reason: "101 Switching Protocols",
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}),
		},
		{
			reason: "101 Switching Protocols",
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusNotModified)
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.reason, func(t *testing.T) {
			url, close := f(ServerConfig{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					test.handler.ServeHTTP(w, req)
					// No body is allowed on this response, the Write method
					// must return an error indicating that the program is
					// misbehaving.
					if _, err := w.Write([]byte("Hello World!")); err != http.ErrBodyNotAllowed {
						t.Errorf("expected http.ErrBodyNotAllowed but got %v", err)
					}
				}),
			})
			defer close()

			res, err := http.Get(url + "/")
			if err != nil {
				t.Error(err)
				return
			}

			r := &countReader{R: res.Body}
			io.Copy(ioutil.Discard, r)
			res.Body.Close()

			if r.N != 0 {
				t.Error("expected no body in the response but received %d bytes", r.N)
			}
		})
	}
}

func testServerErrContentLength(t *testing.T, f MakeServer) {
	url, close := f(ServerConfig{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Length", "1")
			w.WriteHeader(http.StatusOK)
			// The program writes too many bytes to the response, it must be
			// notified by getting an error on the Write call.
			if _, err := w.Write([]byte("Hello World!")); err != http.ErrContentLength {
				t.Error("expected http.ErrContentLength but got %v", err)
			}
		}),
	})
	defer close()

	res, err := http.Get(url + "/")
	if err != nil {
		t.Error(err)
		return
	}

	r := &countReader{R: res.Body}
	io.Copy(ioutil.Discard, r)
	res.Body.Close()

	if r.N != 1 {
		t.Error("expected at 1 byte in the response but received %d bytes", r.N)
	}
}

type countReader struct {
	R io.Reader
	N int
}

func (r *countReader) Read(b []byte) (n int, err error) {
	if n, err = r.R.Read(b); n > 0 {
		r.N += n
	}
	return
}
