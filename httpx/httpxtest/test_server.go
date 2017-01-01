package httpxtest

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/segmentio/netx"
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
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			test(t, f)
		})
	}
	run("Basic", testServerBasic)
	run("Transfer-Encoding:chunked", testServerTransferEncodingChunked)
	run("ErrBodyNotAllowed", testServerErrBodyNotAllowed)
	run("ErrContentLength", testServerErrContentLength)
	run("ReadTimeout", testServerReadTimeout)
	run("WriteTimeout", testServerWriteTimeout)
}

// tests that basic features of the http server are working as expected, setting
// a content length and a response status should work fine.
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

// test that a chunked transfer encoding on the connection works as expected,
// this is done by sending a huge payload via multiple calls to Write.
func testServerTransferEncodingChunked(t *testing.T, f MakeServer) {
	b := make([]byte, 128)

	url, close := f(ServerConfig{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// No Content-Length is set, the server should be using
			// "Transfer-Encoding: chunked" in the response.
			for i := 0; i != 100; i++ {
				if _, err := w.Write(b); err != nil {
					t.Error(err)
					return
				}
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

	if err := res.Body.Close(); err != nil {
		t.Error("error closing the response body:", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Error("bad response code:", res.StatusCode)
	}
	if r.N != (100 * len(b)) {
		t.Error("bad response body length:", r.N)
	}
}

// test that the server's response writer returns http.ErrBodyNotAllowed when
// the program attempts to write a body on a response that doesn't allow one.
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

// test that the server's response writer returns http.ErrContentLength when the
// program attempts to write more data in the response than it previously set on
// the Content-Length header.
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

// test that the server properly closes connections when reading a request takes
// too much time.
func testServerReadTimeout(t *testing.T, f MakeServer) {
	url, close := f(ServerConfig{
		Handler:     http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {}),
		ReadTimeout: 100 * time.Millisecond,
	})
	defer close()

	conn, err := net.Dial("tcp", url[7:]) // trim "http://"
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	// Write the beginning of a request but doesn't terminate it, the server
	// should timeout the connection after 100ms.
	if _, err := conn.Write([]byte("GET / HTTP/1.1")); err != nil {
		t.Error(err)
		return
	}

	var b [128]byte
	if n, err := conn.Read(b[:]); err != io.EOF {
		t.Errorf("expected io.EOF on the read operation but got %v (%d bytes)", err, n)
	}
}

// test that the server properly closes connections when the client doesn't read
// the response.
func testServerWriteTimeout(t *testing.T, f MakeServer) {
	b := make([]byte, 1<<22) // 4MB

	url, close := f(ServerConfig{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if _, err := w.Write(b); !netx.IsTimeout(err) {
				t.Error(err)
			}
		}),
		WriteTimeout: 100 * time.Millisecond,
	})
	defer close()

	conn, err := net.Dial("tcp", url[7:]) // trim "http://"
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Write(w)

	if err := w.Flush(); err != nil {
		t.Error(err)
		return
	}

	// Wait so the server can timeout the request.
	time.Sleep(200 * time.Millisecond)

	res, err := http.ReadResponse(r, req)
	if err != nil {
		return // OK, nothing was sent
	}

	body := &countReader{R: res.Body}
	io.Copy(ioutil.Discard, body)
	res.Body.Close()

	if body.N >= len(b) {
		t.Errorf("the server shouldn't have been able to send the entire response body of %d bytes", body.N)
	}
}

// countReader is an io.Reader which counts how many bytes were read.
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
