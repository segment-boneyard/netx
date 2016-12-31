package httpx

import "testing"

var quotedTests = []struct {
	s string
	q quoted
}{
	{`""`, ``},
	{`hello`, `hello`},
	{`"你好"`, `你好`},
	{`"hello\\world"`, "hello\\world"},
	{`"hello\"world"`, "hello\"world"},
	{`"hello\nworld"`, "hello\nworld"},
	{`"hello\rworld"`, "hello\rworld"},
	{`"hello\tworld"`, "hello\tworld"},
	{`"hello\vworld"`, "hello\vworld"},
	{`"hello\fworld"`, "hello\fworld"},
	{`"hello\bworld"`, "hello\bworld"},
	{`"hello\0world"`, "hello\x00world"},
}

func TestParseQuotedFailure(t *testing.T) {
	tests := []struct {
		s string
	}{
		{`,`},             // non-token and non-quoted-string (single byte)
		{`,,,`},           // non-token and non-quoted-string (multi byte)
		{`"hello"world"`}, // unexpected double-quote
		{`"hello\"`},      // terminated by an escaped quote
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if q, err := parseQuoted(test.s); err == nil {
				t.Error(q)
			}
		})
	}
}

func TestParseQuotedSuccess(t *testing.T) {
	for _, test := range quotedTests {
		t.Run(test.s, func(t *testing.T) {
			q, err := parseQuoted(test.s)

			if err != nil {
				t.Error(err)
			}

			if q != test.q {
				t.Error(q)
			}
		})
	}
}

func TestQuotedString(t *testing.T) {
	for _, test := range quotedTests {
		t.Run(string(test.q), func(t *testing.T) {
			if s := test.q.String(); s != test.s {
				t.Error(s)
			}
		})
	}
}

func TestSplitTrimOWS(t *testing.T) {
	tests := []struct {
		s    string
		b    byte
		head string
		tail string
	}{
		{
			s:    ``,
			b:    ',',
			head: "",
			tail: "",
		},
		{
			s:    `hello, world`,
			b:    ',',
			head: "hello",
			tail: "world",
		},
		{
			s:    `key1="hello, world", key2=`,
			b:    ',',
			head: `key1="hello, world"`,
			tail: `key2=`,
		},
		{
			s:    `key1="message: \"hello, world\"", key2=`,
			b:    ',',
			head: `key1="message: \"hello, world\""`,
			tail: `key2=`,
		},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			head, tail := splitTrimOWS(test.s, test.b)

			if head != test.head {
				t.Errorf("bad head: %#v", head)
			}

			if tail != test.tail {
				t.Errorf("bad tail: %#v", tail)
			}
		})
	}
}
