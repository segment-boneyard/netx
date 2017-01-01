package httpx

import (
	"reflect"
	"testing"
)

func TestParseAcceptItemSuccess(t *testing.T) {
	tests := []struct {
		s string
		a acceptItem
	}{
		{
			s: `text/html`,
			a: acceptItem{
				typ: "text",
				sub: "html",
				q:   1.0,
			},
		},
		{
			s: `text/*;q=0`,
			a: acceptItem{
				typ: "text",
				sub: "*",
			},
		},
		{
			s: `text/html; param="Hello World!"; q=1.0; ext=value`,
			a: acceptItem{
				typ:        "text",
				sub:        "html",
				q:          1.0,
				params:     []mediaParam{{"param", "Hello World!"}},
				extensions: []mediaParam{{"ext", "value"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.a.String(), func(t *testing.T) {
			a, err := parseAcceptItem(test.s)

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
			if a, err := parseAcceptItem(test.s); err == nil {
				t.Error(a)
			}
		})
	}
}

func TestParseAcceptSuccess(t *testing.T) {
	tests := []struct {
		s string
		a accept
	}{
		{
			s: `text/html`,
			a: accept{
				{
					typ: "text",
					sub: "html",
					q:   1.0,
				},
			},
		},
		{
			s: `text/*; q=0, text/html; param="Hello World!"; q=1.0; ext=value`,
			a: accept{
				{
					typ:        "text",
					sub:        "html",
					q:          1.0,
					params:     []mediaParam{{"param", "Hello World!"}},
					extensions: []mediaParam{{"ext", "value"}},
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
			a, err := parseAccept(test.s)

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
			if a, err := parseAccept(test.s); err == nil {
				t.Error(a)
			}
		})
	}
}

func TestAcceptSort(t *testing.T) {
	a, err := parseAccept(`text/*, image/*;q=0.5, text/plain;q=1.0, text/html, text/html;level=1, */*`)

	if err != nil {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(a, accept{
		{typ: "text", sub: "html", q: 1, params: []mediaParam{{"level", "1"}}},
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

	a := accept{
		{
			typ: "application",
			sub: "msgpack",
			q:   1.0,
		},
		{
			typ: "application",
			sub: "json",
			q:   0.5,
		},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if s := a.Negotiate(test.t...); s != test.s {
				t.Error(s)
			}
		})
	}
}

func TestParseAcceptEncodingItemSuccess(t *testing.T) {
	tests := []struct {
		s string
		a acceptEncodingItem
	}{
		{
			s: `gzip`,
			a: acceptEncodingItem{
				coding: "gzip",
				q:      1.0,
			},
		},
		{
			s: `gzip;q=0`,
			a: acceptEncodingItem{
				coding: "gzip",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.a.String(), func(t *testing.T) {
			a, err := parseAcceptEncodingItem(test.s)

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
			if a, err := parseAcceptEncodingItem(test.s); err == nil {
				t.Error(a)
			}
		})
	}
}

func TestParseAcceptEncodingSuccess(t *testing.T) {
	tests := []struct {
		s string
		a acceptEncoding
	}{
		{
			s: `gzip;q=1.0, *;q=0, identity; q=0.5`,
			a: acceptEncoding{
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
			a, err := parseAcceptEncoding(test.s)

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
			if a, err := parseAcceptEncoding(test.s); err == nil {
				t.Error(a)
			}
		})
	}
}
