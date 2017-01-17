package httpx

import (
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	httpxflate "github.com/segmentio/netx/httpx/flate"
	httpxgzip "github.com/segmentio/netx/httpx/gzip"
	httpxzlib "github.com/segmentio/netx/httpx/zlib"
)

func TestEncodingHandler(t *testing.T) {
	tests := []struct {
		coding    string
		newReader func(io.Reader) io.ReadCloser
	}{
		{
			coding:    "deflate",
			newReader: flate.NewReader,
		},
		{
			coding: "gzip",
			newReader: func(r io.Reader) io.ReadCloser {
				z, _ := gzip.NewReader(r)
				return z
			},
		},
		{
			coding: "zlib",
			newReader: func(r io.Reader) io.ReadCloser {
				z, _ := zlib.NewReader(r)
				return z
			},
		},
	}

	h := NewEncodingHandler(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("Hello World!"))
	}),
		httpxflate.NewContentEncoder(),
		httpxgzip.NewContentEncoder(),
		httpxzlib.NewContentEncoder(),
	)

	for _, test := range tests {
		t.Run(test.coding, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Accept-Encoding", test.coding)

			res := httptest.NewRecorder()

			h.ServeHTTP(res, req)
			res.Flush()

			r := test.newReader(res.Body)
			b, _ := ioutil.ReadAll(r)

			if res.Code != http.StatusOK {
				t.Error("bad status:", res.Code)
			}
			if coding := res.HeaderMap.Get("Content-Encoding"); coding != test.coding {
				t.Error("bad content encoding:", coding)
			}
			if s := string(b); s != "Hello World!" {
				t.Error("bad content:", s)
			}
		})
	}
}
