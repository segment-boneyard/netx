package httpxtest

import (
	"bytes"
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
