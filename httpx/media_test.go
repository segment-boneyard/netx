package httpx

import (
	"reflect"
	"testing"
)

func TestParseMediaTypeSuccess(t *testing.T) {
	tests := []struct {
		s string
		m mediaType
	}{
		{
			s: `*/*`,
			m: mediaType{typ: "*", sub: "*"},
		},
		{
			s: `text/*`,
			m: mediaType{typ: "text", sub: "*"},
		},
		{
			s: `text/plain`,
			m: mediaType{typ: "text", sub: "plain"},
		},
	}

	for _, test := range tests {
		t.Run(test.m.String(), func(t *testing.T) {
			m, err := parseMediaType(test.s)

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
			if m, err := parseMediaType(test.s); err == nil {
				t.Error(m)
			}
		})
	}
}

func TestMediaTypeContainsTrue(t *testing.T) {
	tests := []struct {
		t1 mediaType
		t2 mediaType
	}{
		{
			t1: mediaType{typ: "*", sub: "*"},
			t2: mediaType{typ: "text", sub: "plain"},
		},
		{
			t1: mediaType{typ: "text", sub: "*"},
			t2: mediaType{typ: "text", sub: "plain"},
		},
		{
			t1: mediaType{typ: "text", sub: "plain"},
			t2: mediaType{typ: "text", sub: "plain"},
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
		t1 mediaType
		t2 mediaType
	}{
		{
			t1: mediaType{typ: "text", sub: "*"},
			t2: mediaType{typ: "image", sub: "png"},
		},
		{
			t1: mediaType{typ: "text", sub: "plain"},
			t2: mediaType{typ: "text", sub: "html"},
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
		p mediaParam
	}{
		{
			s: `key=value`,
			p: mediaParam{name: "key", value: "value"},
		},
		{
			s: `key="你好"`,
			p: mediaParam{name: "key", value: "你好"},
		},
	}

	for _, test := range tests {
		t.Run(test.p.String(), func(t *testing.T) {
			p, err := parseMediaParam(test.s)

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
			if p, err := parseMediaParam(test.s); err == nil {
				t.Error(p)
			}
		})
	}
}

func TestParseMediaRangeSuccess(t *testing.T) {
	tests := []struct {
		s string
		r mediaRange
	}{
		{
			s: `image/*`,
			r: mediaRange{
				typ: "image",
				sub: "*",
			},
		},
		{
			s: `image/*;`, // trailing ';'
			r: mediaRange{
				typ: "image",
				sub: "*",
			},
		},
		{
			s: `text/html;key1=hello;key2="你好"`,
			r: mediaRange{
				typ:    "text",
				sub:    "html",
				params: []mediaParam{{"key1", "hello"}, {"key2", "你好"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.r.String(), func(t *testing.T) {
			r, err := parseMediaRange(test.s)

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
			if m, err := parseMediaRange(test.s); err == nil {
				t.Error(m)
			}
		})
	}
}

func TestMediaRangeParam(t *testing.T) {
	r := mediaRange{
		typ:    "image",
		sub:    "*",
		params: []mediaParam{{"answer", "42"}},
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

func TestMediaTypeLess(t *testing.T) {
	tests := []struct {
		t1   string
		t2   string
		less bool
	}{
		{
			t1:   "",
			t2:   "",
			less: false,
		},
		{
			t1:   "*",
			t2:   "*",
			less: false,
		},
		{
			t1:   "*",
			t2:   "text",
			less: false,
		},
		{
			t1:   "text",
			t2:   "*",
			less: true,
		},
		{
			t1:   "text",
			t2:   "text",
			less: false,
		},
		{
			t1:   "plain",
			t2:   "html",
			less: false,
		},
		{
			t1:   "html",
			t2:   "plain",
			less: true,
		},
	}

	for _, test := range tests {
		t.Run(test.t1+"<"+test.t2, func(t *testing.T) {
			if less := mediaTypeLess(test.t1, test.t2); less != test.less {
				t.Error(less)
			}
		})
	}
}
