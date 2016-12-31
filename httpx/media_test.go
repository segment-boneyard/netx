package httpx

import (
	"reflect"
	"testing"
)

func TestParseMediaTypeSuccess(t *testing.T) {
	tests := []struct {
		s string
		m MediaType
	}{
		{
			s: `*/*`,
			m: MediaType{Type: "*", SubType: "*"},
		},
		{
			s: `text/*`,
			m: MediaType{Type: "text", SubType: "*"},
		},
		{
			s: `text/plain`,
			m: MediaType{Type: "text", SubType: "plain"},
		},
	}

	for _, test := range tests {
		t.Run(test.m.String(), func(t *testing.T) {
			m, err := ParseMediaType(test.s)

			if err != nil {
				t.Error(err)
			}

			if m != test.m {
				t.Error(m)
			}
		})
	}
}

func TestParseMediaTypeFailure(t *testing.T) {
	tests := []struct {
		s string
	}{
		{``},        // empty string
		{`/`},       // missing type and subtype
		{`text`},    // missing separator
		{`,/plain`}, // bad type
		{`text/,`},  // bad subtype
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if m, err := ParseMediaType(test.s); err == nil {
				t.Error(m)
			}
		})
	}
}

func TestMediaTypeContainsTrue(t *testing.T) {
	tests := []struct {
		t1 MediaType
		t2 MediaType
	}{
		{
			t1: MediaType{Type: "*", SubType: "*"},
			t2: MediaType{Type: "text", SubType: "plain"},
		},
		{
			t1: MediaType{Type: "text", SubType: "*"},
			t2: MediaType{Type: "text", SubType: "plain"},
		},
		{
			t1: MediaType{Type: "text", SubType: "plain"},
			t2: MediaType{Type: "text", SubType: "plain"},
		},
	}

	for _, test := range tests {
		t.Run(test.t1.String()+":"+test.t2.String(), func(t *testing.T) {
			if !test.t1.Contains(test.t2) {
				t.Error("nope")
			}
		})
	}
}

func TestMediaTypeContainsFalse(t *testing.T) {
	tests := []struct {
		t1 MediaType
		t2 MediaType
	}{
		{
			t1: MediaType{Type: "text", SubType: "*"},
			t2: MediaType{Type: "image", SubType: "png"},
		},
		{
			t1: MediaType{Type: "text", SubType: "plain"},
			t2: MediaType{Type: "text", SubType: "html"},
		},
	}

	for _, test := range tests {
		t.Run(test.t1.String()+":"+test.t2.String(), func(t *testing.T) {
			if test.t1.Contains(test.t2) {
				t.Error("nope")
			}
		})
	}
}

func TestParseMediaParamSuccess(t *testing.T) {
	tests := []struct {
		s string
		p MediaParam
	}{
		{
			s: `key=value`,
			p: MediaParam{Name: "key", Value: "value"},
		},
		{
			s: `key="你好"`,
			p: MediaParam{Name: "key", Value: "你好"},
		},
	}

	for _, test := range tests {
		t.Run(test.p.String(), func(t *testing.T) {
			p, err := ParseMediaParam(test.s)

			if err != nil {
				t.Error(err)
			}

			if p != test.p {
				t.Error(p)
			}
		})
	}
}

func TestParseMediaParamFailure(t *testing.T) {
	tests := []struct {
		s string
	}{
		{``},       // empty string
		{`key`},    // missing =
		{`key=`},   // missing value
		{`=value`}, // missing key
		{`=`},      // missing key and value
		{`key=你好`}, // non-token and non-quoted value
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if p, err := ParseMediaParam(test.s); err == nil {
				t.Error(p)
			}
		})
	}
}

func TestParseMediaRangeSuccess(t *testing.T) {
	tests := []struct {
		s string
		r MediaRange
	}{
		{
			s: `image/*`,
			r: MediaRange{
				Type:    "image",
				SubType: "*",
			},
		},
		{
			s: `image/*;`, // trailing ';'
			r: MediaRange{
				Type:    "image",
				SubType: "*",
			},
		},
		{
			s: `text/html;key1=hello;key2="你好"`,
			r: MediaRange{
				Type:    "text",
				SubType: "html",
				Params:  []MediaParam{{"key1", "hello"}, {"key2", "你好"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.r.String(), func(t *testing.T) {
			r, err := ParseMediaRange(test.s)

			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(r, test.r) {
				t.Error(r)
			}
		})
	}
}

func TestParseMediaRangeFailure(t *testing.T) {
	tests := []struct {
		s string
	}{
		{``},            // empty string
		{`image`},       // no media type
		{`/`},           // bad media type
		{`image/,`},     // bad sub-type
		{`image/*;bad`}, // bad parameters
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if m, err := ParseMediaRange(test.s); err == nil {
				t.Error(m)
			}
		})
	}
}

func TestMediaRangeParam(t *testing.T) {
	r := MediaRange{
		Type:    "image",
		SubType: "*",
		Params:  []MediaParam{{"answer", "42"}},
	}

	p1 := r.Param("answer")
	p2 := r.Param("other")

	if p1 != "42" {
		t.Error("found bad media parameter:", p1)
	}

	if p2 != "" {
		t.Error("found non-existing media parameter:", p2)
	}
}
