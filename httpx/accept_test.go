package httpx

import (
	"reflect"
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
				Type:    "text",
				SubType: "html",
				Q:       1.0,
			},
		},
		{
			s: `text/*;q=0`,
			a: AcceptItem{
				Type:    "text",
				SubType: "*",
			},
		},
		{
			s: `text/html; param="Hello World!"; q=1.0; ext=value`,
			a: AcceptItem{
				Type:       "text",
				SubType:    "html",
				Q:          1.0,
				Params:     []MediaParam{{"param", "Hello World!"}},
				Extensions: []MediaParam{{"ext", "value"}},
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
					Type:    "text",
					SubType: "html",
					Q:       1.0,
				},
			},
		},
		{
			s: `text/*; q=0, text/html; param="Hello World!"; q=1.0; ext=value`,
			a: Accept{
				{
					Type:       "text",
					SubType:    "html",
					Q:          1.0,
					Params:     []MediaParam{{"param", "Hello World!"}},
					Extensions: []MediaParam{{"ext", "value"}},
				},
				{
					Type:    "text",
					SubType: "*",
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
		{Type: "text", SubType: "html", Q: 1, Params: []MediaParam{{"level", "1"}}},
		{Type: "text", SubType: "html", Q: 1},
		{Type: "text", SubType: "plain", Q: 1},
		{Type: "text", SubType: "*", Q: 1},
		{Type: "*", SubType: "*", Q: 1},
		{Type: "image", SubType: "*", Q: 0.5},
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

	a := Accept{
		{
			Type:    "application",
			SubType: "msgpack",
			Q:       1.0,
		},
		{
			Type:    "application",
			SubType: "json",
			Q:       0.5,
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
		a AcceptEncodingItem
	}{
		{
			s: `gzip`,
			a: AcceptEncodingItem{
				Coding: "gzip",
				Q:      1.0,
			},
		},
		{
			s: `gzip;q=0`,
			a: AcceptEncodingItem{
				Coding: "gzip",
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
					Coding: "gzip",
					Q:      1.0,
				},
				{
					Coding: "identity",
					Q:      0.5,
				},
				{
					Coding: "*",
					Q:      0.0,
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
