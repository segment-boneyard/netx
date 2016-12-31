package httpx

import (
	"bytes"
	"fmt"
	"io"
)

// quoted is a type alias to string that implements the fmt.Stringer and
// fmt.Formatter interfaces to efficiently output HTTP tokens or quoted
// strings.
type quoted string

// parseQuoted parses s, potentially removing the quotes if it is a quoted
// string.
func parseQuoted(s string) (q quoted, err error) {
	if isToken(s) {
		q = quoted(s)
		return
	}

	n := len(s)

	if n < 2 || s[0] != '"' || s[n-1] != '"' {
		err = errorInvalidQuotedString(s)
		return
	}

	e := false
	b := bytes.Buffer{}
	b.Grow(n)

	for _, c := range s[1 : n-1] {
		if e {
			e = false
			switch c {
			case 'n':
				c = '\n'
			case 'r':
				c = '\r'
			case 't':
				c = '\t'
			case 'v':
				c = '\v'
			case 'f':
				c = '\f'
			case 'b':
				c = '\b'
			case '0':
				c = '\x00'
			}
		} else if c == '\\' {
			e = true
			continue
		} else if c == '"' {
			err = errorInvalidQuotedString(s)
			return
		}
		b.WriteRune(c)
	}

	if e {
		err = errorInvalidQuotedString(s)
		return
	}

	q = quoted(b.String())
	return
}

// String satisfies the fmt.Stringer interface.
func (q quoted) String() string {
	return fmt.Sprint(q)
}

// Format satisfies the fmt.Formatter interface.
func (q quoted) Format(w fmt.State, _ rune) {
	if s := string(q); isToken(s) {
		fmt.Fprint(w, s)
	} else {
		writeQuoted(w, s)
	}
}

// quote returns a quoted representation of s, not checking whether s is a valid
// HTTP token.
func quote(s string) string {
	b := &bytes.Buffer{}
	b.Grow(len(s) + 10)
	writeQuoted(b, s)
	return b.String()
}

// writeQuoted writes the quoted representation of s to w.
func writeQuoted(w io.Writer, s string) {
	fmt.Fprint(w, `"`)
	i := 0
	j := 0
	n := len(s)

	for j < n {
		c := s[j]
		j++

		switch c {
		case '\\', '"':
		case '\n':
			c = 'n'
		case '\r':
			c = 'r'
		case '\t':
			c = 't'
		case '\f':
			c = 'f'
		case '\v':
			c = 'v'
		case '\b':
			c = 'b'
		case '\x00':
			c = '0'
		default:
			continue
		}

		fmt.Fprintf(w, `%s\%c`, s[i:j-1], c)
		i = j
	}

	fmt.Fprintf(w, `%s"`, s[i:])
}

// split s at b, properly handling inner quoted strings in s, which means b
// cannot be a double-quote.
func split(s string, b byte) (head string, tail string) {
	e := false
	q := false

	for i := range s {
		if q {
			if e {
				e = false
			} else if s[i] == '\\' {
				e = true
			} else if s[i] == '"' {
				q = false
			}
			continue
		}

		if s[i] == '"' {
			q = true
			continue
		}

		if !q && s[i] == b {
			head = s[:i]
			tail = s[i+1:]
			return
		}
	}

	head = s
	return
}

// split s at b, properly handling inner quoted strings in s, and trimming the
// results for leading and trailing white spaces.
func splitTrimOWS(s string, b byte) (head string, tail string) {
	head, tail = split(s, b)
	head = trimOWS(head)
	tail = trimOWS(tail)
	return
}

func errorInvalidQuotedString(s string) error {
	return fmt.Errorf("invalid quoted string: %#v", s)
}
