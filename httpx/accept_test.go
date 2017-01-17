package httpx

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseAcceptItemSuccess(t *testing.T) {
	tests := []struct {
		s string
		a AcceptItem
	}{
		{
			s: `text/html`,
			a: AcceptItem{
				typ: "text",
				sub: "html",
				q:   1.0,
			},
		},
		{
			s: `text/*;q=0`,
			a: AcceptItem{
				typ: "text",
				sub: "*",
			},
		},
		{
			s: `text/html; param="Hello World!"; q=1.0; ext=value`,
			a: AcceptItem{
				typ:    "text",
				sub:    "html",
				q:      1.0,
				params: []MediaParam{{"param", "Hello World!"}},
				extens: []MediaParam{{"ext", "value"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.a.String(), func(t *testing.T) {
			a, err := ParseAcceptItem(test.s)

			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(a, test.a) {
				t.Error(a)
			}
		})
	}

}

func TestParseAcceptItemFailure(t *testing.T) {
	tests := []struct {
		s string
	}{
		{``},        // empty string
		{`garbage`}, // garbage
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if a, err := ParseAcceptItem(test.s); err == nil {
				t.Error(a)
			}
		})
	}
}

func TestParseAcceptSuccess(t *testing.T) {
	tests := []struct {
		s string
		a Accept
	}{
		{
			s: `text/html`,
			a: Accept{
				{
					typ: "text",
					sub: "html",
					q:   1.0,
				},
			},
		},
		{
			s: `text/*; q=0, text/html; param="Hello World!"; q=1.0; ext=value`,
			a: Accept{
				{
					typ:    "text",
					sub:    "html",
					q:      1.0,
					params: []MediaParam{{"param", "Hello World!"}},
					extens: []MediaParam{{"ext", "value"}},
				},
				{
					typ: "text",
					sub: "*",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.a.String(), func(t *testing.T) {
			a, err := ParseAccept(test.s)

			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(a, test.a) {
				t.Error(a)
			}
		})
	}
}

func TestParseAcceptFailure(t *testing.T) {
	tests := []struct {
		s string
	}{
		{`garbage`}, // garbage
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if a, err := ParseAccept(test.s); err == nil {
				t.Error(a)
			}
		})
	}
}

func TestAcceptSort(t *testing.T) {
	a, err := ParseAccept(`text/*, image/*;q=0.5, text/plain;q=1.0, text/html, text/html;level=1, */*`)

	if err != nil {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(a, Accept{
		{typ: "text", sub: "html", q: 1, params: []MediaParam{{"level", "1"}}},
		{typ: "text", sub: "html", q: 1},
		{typ: "text", sub: "plain", q: 1},
		{typ: "text", sub: "*", q: 1},
		{typ: "*", sub: "*", q: 1},
		{typ: "image", sub: "*", q: 0.5},
	}) {
		t.Error(a)
	}
}

func TestAcceptNegotiate(t *testing.T) {
	tests := []struct {
		t []string
		s string
	}{
		{
			t: nil,
			s: "",
		},
		{
			t: []string{"text/html"},
			s: "text/html",
		},
		{
			t: []string{"application/json"},
			s: "application/json",
		},
		{
			t: []string{"application/msgpack"},
			s: "application/msgpack",
		},
		{
			t: []string{"application/json", "application/msgpack"},
			s: "application/msgpack",
		},
		{
			t: []string{"application/msgpack", "application/json"},
			s: "application/msgpack",
		},
		{
			t: []string{"msgpack", "application/json"}, // first type is bad
			s: "application/json",
		},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if s := Negotiate("application/msgpack;q=1.0, application/json;q=0.5", test.t...); s != test.s {
				t.Error(s)
			}
		})
	}
}

func TestAcceptNegotiateEncoding(t *testing.T) {
	tests := []struct {
		c []string
		s string
	}{
		{
			c: nil,
			s: "",
		},
		{
			c: []string{"gzip"},
			s: "gzip",
		},
		{
			c: []string{"deflate"},
			s: "deflate",
		},
		{
			c: []string{"deflate", "gzip"},
			s: "gzip",
		},
	}

	for _, test := range tests {
		t.Run(strings.Join(test.c, ","), func(t *testing.T) {
			if s := NegotiateEncoding("gzip;q=1.0, deflate;q=0.5", test.c...); s != test.s {
				t.Error(s)
			}
		})
	}
}

func TestParseAcceptEncodingItemSuccess(t *testing.T) {
	tests := []struct {
		s string
		a AcceptEncodingItem
	}{
		{
			s: `gzip`,
			a: AcceptEncodingItem{
				coding: "gzip",
				q:      1.0,
			},
		},
		{
			s: `gzip;q=0`,
			a: AcceptEncodingItem{
				coding: "gzip",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.a.String(), func(t *testing.T) {
			a, err := ParseAcceptEncodingItem(test.s)

			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(a, test.a) {
				t.Error(a)
			}
		})
	}

}

func TestParseAcceptEncodingItemFailure(t *testing.T) {
	tests := []struct {
		s string
	}{
		{``},               // empty string
		{`q=`},             // missing value
		{`gzip;key=value`}, // not q=X
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if a, err := ParseAcceptEncodingItem(test.s); err == nil {
				t.Error(a)
			}
		})
	}
}

func TestParseAcceptEncodingSuccess(t *testing.T) {
	tests := []struct {
		s string
		a AcceptEncoding
	}{
		{
			s: `gzip;q=1.0, *;q=0, identity; q=0.5`,
			a: AcceptEncoding{
				{
					coding: "gzip",
					q:      1.0,
				},
				{
					coding: "identity",
					q:      0.5,
				},
				{
					coding: "*",
					q:      0.0,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.a.String(), func(t *testing.T) {
			a, err := ParseAcceptEncoding(test.s)

			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(a, test.a) {
				t.Error(a)
			}
		})
	}
}

func TestParseAcceptEncodingFailure(t *testing.T) {
	tests := []struct {
		s string
	}{
		{`gzip;`},          // missing q=X
		{`gzip;key=value`}, // not q=X
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if a, err := ParseAcceptEncoding(test.s); err == nil {
				t.Error(a)
			}
		})
	}
}
