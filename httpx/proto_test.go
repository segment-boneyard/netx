package httpx

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestProtoEqual(t *testing.T) {
	tests := []struct {
		req *http.Request
		maj int
		min int
		res bool
	}{
		{
			req: &http.Request{},
			maj: 0,
			min: 0,
			res: true,
		},
		{
			req: &http.Request{ProtoMajor: 1, ProtoMinor: 0},
			maj: 1,
			min: 0,
			res: true,
		},
		{
			req: &http.Request{ProtoMajor: 1, ProtoMinor: 1},
			maj: 1,
			min: 1,
			res: true,
		},
		{
			req: &http.Request{ProtoMajor: 0, ProtoMinor: 9},
			maj: 1,
			min: 0,
			res: false,
		},
		{
			req: &http.Request{ProtoMajor: 1, ProtoMinor: 0},
			maj: 1,
			min: 1,
			res: false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("HTTP/%d.%d", test.maj, test.min), func(t *testing.T) {
			if res := protoEqual(test.req, test.maj, test.min); res != test.res {
				t.Error(res)
			}
		})
	}
}

func TestProtoVersion(t *testing.T) {
	tests := []struct {
		req *http.Request
		out string
	}{
		{
			req: &http.Request{},
			out: "",
		},
		{
			req: &http.Request{Proto: "bad"},
			out: "bad",
		},
		{
			req: &http.Request{Proto: "HTTP/1.0"},
			out: "1.0",
		},
		{
			req: &http.Request{Proto: "HTTP/1.1"},
			out: "1.1",
		},
	}

	for _, test := range tests {
		t.Run(test.out, func(t *testing.T) {
			if s := protoVersion(test.req); s != test.out {
				t.Error(s)
			}
		})
	}
}

func TestSplitProtoAddr(t *testing.T) {
	tests := []struct {
		uri   string
		proto string
		addr  string
	}{
		{"localhost:80", "", "localhost:80"},
		{"http://localhost:80", "http", "localhost:80"},
	}

	for _, test := range tests {
		t.Run(test.uri, func(t *testing.T) {
			proto, addr := splitProtoAddr(test.uri)

			if proto != test.proto {
				t.Error("bad protocol:", proto)
			}

			if addr != test.addr {
				t.Error("bad address:", addr)
			}
		})
	}
}

func TestCopyHeader(t *testing.T) {
	h1 := http.Header{"Content-Type": {"text/html"}}
	h2 := http.Header{"Content-Type": {"text/html"}, "Content-Length": {"42"}}

	copyHeader(h1, h2)

	if !reflect.DeepEqual(h1, h2) {
		t.Error(h1)
	}
}

func TestDleeteHopFields(t *testing.T) {
	h := http.Header{
		"Connection":          {"Upgrade", "Other"},
		"Keep-Alive":          {},
		"Proxy-Authenticate":  {},
		"Proxy-Authorization": {},
		"Proxy-Connection":    {},
		"Te":                  {},
		"Trailer":             {},
		"Transfer-Encoding":   {},
		"Upgrade":             {},
		"Other":               {},
		"Content-Type":        {"text/html"},
	}

	deleteHopFields(h)

	if !reflect.DeepEqual(h, http.Header{
		"Content-Type": {"text/html"},
	}) {
		t.Error(h)
	}
}

func TestTranslateXForwarded(t *testing.T) {
	tests := []struct {
		in  http.Header
		out http.Header
	}{
		{
			in:  http.Header{},
			out: http.Header{},
		},

		{
			in: http.Header{
				"X-Forwarded-For": {"127.0.0.1"},
			},
			out: http.Header{
				"Forwarded": {"for=127.0.0.1"},
			},
		},

		{
			in: http.Header{
				"X-Forwarded-For":  {"127.0.0.1"},
				"X-Forwarded-Port": {"56789"},
			},
			out: http.Header{
				"Forwarded": {`for="127.0.0.1:56789"`},
			},
		},

		{
			in: http.Header{
				"X-Forwarded-For":   {"127.0.0.1"},
				"X-Forwarded-Port":  {"56789"},
				"X-Forwarded-Proto": {"https"},
			},
			out: http.Header{
				"Forwarded": {`proto=https;for="127.0.0.1:56789"`},
			},
		},

		{
			in: http.Header{
				"X-Forwarded-For":   {"127.0.0.1"},
				"X-Forwarded-Port":  {"56789"},
				"X-Forwarded-Proto": {"https"},
				"X-Forwarded-By":    {"localhost"},
			},
			out: http.Header{
				"Forwarded": {`proto=https;for="127.0.0.1:56789";by="localhost"`},
			},
		},

		{
			in: http.Header{
				"X-Forwarded-For":   {"212.53.1.6, 127.0.0.1"},
				"X-Forwarded-Port":  {"56789"},
				"X-Forwarded-Proto": {"https"},
				"X-Forwarded-By":    {"localhost"},
			},
			out: http.Header{
				"Forwarded": {`for=212.53.1.6, for=127.0.0.1`},
			},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			translateXForwarded(test.in)

			if !reflect.DeepEqual(test.in, test.out) {
				t.Error(test.in)
			}
		})
	}
}

func TestQuoteForwarded(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"", `""`},
		{"127.0.0.1", `127.0.0.1`},
		{"2001:db8:cafe::17", `"[2001:db8:cafe::17]"`},
		{"[2001:db8:cafe::17]", `"[2001:db8:cafe::17]"`},
		{"_gazonk", `"_gazonk"`},
	}

	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			if s := quoteForwarded(test.in); s != test.out {
				t.Error(s)
			}
		})
	}
}

func TestMakeForwarded(t *testing.T) {
	tests := []struct {
		proto     string
		forAddr   string
		byAddr    string
		forwarded string
	}{
		{ /* all zero-values */ },

		{
			proto:     "http",
			forwarded: "proto=http",
		},

		{
			forAddr:   "127.0.0.1",
			forwarded: "for=127.0.0.1",
		},

		{
			byAddr:    "127.0.0.1",
			forwarded: "by=127.0.0.1",
		},

		{
			proto:     "http",
			forAddr:   "212.53.1.6",
			byAddr:    "127.0.0.1",
			forwarded: "proto=http;for=212.53.1.6;by=127.0.0.1",
		},
	}

	for _, test := range tests {
		if s := makeForwarded(test.proto, test.forAddr, test.byAddr); s != test.forwarded {
			t.Error(s)
		}
	}
}

func TestAddForwarded(t *testing.T) {
	tests := []struct {
		header  http.Header
		proto   string
		forAddr string
		byAddr  string
		result  http.Header
	}{
		{
			header:  http.Header{},
			proto:   "http",
			forAddr: "127.0.0.1:56789",
			byAddr:  "127.0.0.1:80",
			result: http.Header{"Forwarded": {
				`proto=http;for="127.0.0.1:56789";by="127.0.0.1:80"`,
			}},
		},

		{
			header: http.Header{"Forwarded": {
				`proto=https;for="212.53.1.6:54387";by="127.0.0.1:443"`,
			}},
			proto:   "http",
			forAddr: "127.0.0.1:56789",
			byAddr:  "127.0.0.1:80",
			result: http.Header{"Forwarded": {
				`proto=https;for="212.53.1.6:54387";by="127.0.0.1:443", proto=http;for="127.0.0.1:56789";by="127.0.0.1:80"`,
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.result.Get("Forwarded"), func(t *testing.T) {
			addForwarded(test.header, test.proto, test.forAddr, test.byAddr)
			if !reflect.DeepEqual(test.header, test.result) {
				t.Error(test.header)
			}
		})
	}
}

func TestMakeVia(t *testing.T) {
	tests := []struct {
		version string
		host    string
		via     string
	}{
		{
			version: "1.1",
			host:    "localhost",
			via:     "1.1 localhost",
		},
	}

	for _, test := range tests {
		t.Run(test.via, func(t *testing.T) {
			if via := makeVia(test.version, test.host); via != test.via {
				t.Error(via)
			}
		})
	}
}

func TestAddVia(t *testing.T) {
	tests := []struct {
		version string
		host    string
		in      http.Header
		out     http.Header
	}{
		{
			version: "1.1",
			host:    "localhost",
			in:      http.Header{},
			out: http.Header{
				"Via": {"1.1 localhost"},
			},
		},
		{
			version: "1.1",
			host:    "localhost",
			in: http.Header{
				"Via": {"1.1 laptop"},
			},
			out: http.Header{
				"Via": {"1.1 laptop, 1.1 localhost"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.out.Get("Via"), func(t *testing.T) {
			addVia(test.in, test.version, test.host)

			if !reflect.DeepEqual(test.in, test.out) {
				t.Error(test.in)
			}
		})
	}
}

func TestMaxForwards(t *testing.T) {
	tests := []struct {
		in  http.Header
		max int
	}{
		{
			in:  http.Header{},
			max: -1,
		},
		{
			in:  http.Header{"Max-Forwards": {"42"}},
			max: 42,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			max, err := maxForwards(test.in)

			if err != nil {
				t.Error(err)
			}

			if max != test.max {
				t.Error("max =", max)
			}
		})
	}
}

func TestConnectionUpgrade(t *testing.T) {
	tests := []struct {
		in  http.Header
		out string
	}{
		{
			in:  http.Header{},
			out: "",
		},
		{
			in: http.Header{ // missing "Connection"
				"Upgrade": {"websocket"},
			},
			out: "",
		},
		{
			in: http.Header{ // missing "Upgrade"
				"Connection": {"Upgrade"},
			},
			out: "",
		},
		{
			in: http.Header{
				"Connection": {"Upgrade"},
				"Upgrade":    {"websocket"},
			},
			out: "websocket",
		},
	}

	for _, test := range tests {
		t.Run(test.out, func(t *testing.T) {
			if s := connectionUpgrade(test.in); s != test.out {
				t.Error(s)
			}
		})
	}
}

func TestHeaderValuesRemoveTokens(t *testing.T) {
	tests := []struct {
		values []string
		tokens []string
		result []string
	}{
		{ // empty values
			values: []string{},
			tokens: []string{},
			result: []string{},
		},
		{ // remove nothing
			values: []string{"A", "B", "C"},
			tokens: []string{"other"},
			result: []string{"A", "B", "C"},
		},
		{ // remove first
			values: []string{"A", "B", "C"},
			tokens: []string{"A"},
			result: []string{"B", "C"},
		},
		{ // remove middle
			values: []string{"A", "B", "C"},
			tokens: []string{"B"},
			result: []string{"A", "C"},
		},
		{ // remove last
			values: []string{"A", "B", "C"},
			tokens: []string{"C"},
			result: []string{"A", "B"},
		},
		{ // remove all
			values: []string{"A", "B", "C"},
			tokens: []string{"A", "B", "C"},
			result: []string{},
		},
		{ // remove inner (single)
			values: []string{"A, B", "C"},
			tokens: []string{"A"},
			result: []string{"B", "C"},
		},
		{ // remove inner (multi)
			values: []string{"A, B, C"},
			tokens: []string{"A", "B"},
			result: []string{"C"},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			result := headerValuesRemoveTokens(test.values, test.tokens...)

			if !reflect.DeepEqual(result, test.result) {
				t.Error(result)
			}
		})
	}
}

func TestIsIdempotent(t *testing.T) {
	tests := []struct {
		method string
		is     bool
	}{
		{"HEAD", true},
		{"GET", true},
		{"PUT", true},
		{"OPTIONS", true},
		{"DELETE", true},
		{"POST", false},
		{"TRACE", false},
		{"PATCH", false},
	}

	for _, test := range tests {
		t.Run(test.method, func(t *testing.T) {
			if is := isIdempotent(test.method); is != test.is {
				t.Error(is)
			}
		})
	}
}

func TestIsRetriable(t *testing.T) {
	tests := []struct {
		status int
		retry  bool
	}{
		{http.StatusOK, false},
		{http.StatusInternalServerError, true},
		{http.StatusNotImplemented, false},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
		{http.StatusHTTPVersionNotSupported, false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprint(test.status), func(t *testing.T) {
			if retry := isRetriable(test.status); retry != test.retry {
				t.Error(retry)
			}
		})
	}
}
