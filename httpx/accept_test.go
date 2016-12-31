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
