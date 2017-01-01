package httpxtest

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MakeTransport constructs a new HTTP transport used by a single sub-test of
// TestTransport.
type MakeTransport func() http.RoundTripper

// TestTransport is a test suite for HTTP transports, inspired by
// golang.org/x/net/nettest.TestConn.
func TestTransport(t *testing.T, f MakeTransport) {
	run := func(name string, test func(*testing.T, MakeTransport)) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			test(t, f)
		})
	}
	run("Basic", testTransportHEAD)
}

func testTransportHEAD(t *testing.T, f MakeTransport) {
	tests := []struct {
		method  string
		path    string
		reqBody string
		resBody string
	}{
		{
			method: "HEAD",
			path:   "/",
		},
		{
			method:  "GET",
			path:    "/",
			resBody: "Hello World!",
		},
		{
			method:  "POST",
			path:    "/hello/world",
			reqBody: "answer",
			resBody: "42",
		},
		{
			method:  "PUT",
			path:    "/hello/world",
			reqBody: "answer=42",
			resBody: "",
		},
		{
			method: "DELETE",
			path:   "/hello/world",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.method, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if test.method != req.Method {
					t.Errorf("bad method received by the server, expected %s but got %s", test.method, req.Method)
				}
				if test.path != req.URL.Path {
					t.Errorf("bad path received by the server, expected %s but got %s", test.path, req.URL.Path)
				}

				b, err := ioutil.ReadAll(req.Body)
				req.Body.Close()

				if err != nil {
					t.Errorf("the server got an error while reading the request body: %v", err)
				}
				if s := string(b); s != test.reqBody {
					t.Errorf("bad request body received by the server, expected %#v but got %#v", test.reqBody, s)
				}

				h := w.Header()
				h.Set("Content-Type", "text/plain")

				if _, err := w.Write([]byte(test.resBody)); err != nil {
					t.Errorf("the server got an error while writing the response body: %v", err)
				}
			}))
			defer server.Close()

			req, err := http.NewRequest(test.method, server.URL+test.path, strings.NewReader(test.reqBody))
			if err != nil {
				t.Error(err)
				return
			}
			req.Header.Set("Content-Type", "text/plain")

			res, err := f().RoundTrip(req)
			if err != nil {
				t.Error(err)
				return
			}

			b, err := ioutil.ReadAll(res.Body)
			res.Body.Close()

			if err != nil {
				t.Error(err)
				return
			}
			if s := string(b); s != test.resBody {
				t.Errorf("bad body received by the client, expected %#v but got %#v", test.resBody, s)
			}
		})
	}
}
