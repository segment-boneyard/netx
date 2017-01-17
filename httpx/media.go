package httpx

import (
	"errors"
	"fmt"
	"strings"
)

// MediaRange is a representation of a HTTP media range as described in
// https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
type MediaRange struct {
	typ    string
	sub    string
	params []MediaParam
}

// Param return the value of the parameter with name, which will be an empty
// string if none was found.
func (r MediaRange) Param(name string) string {
	for _, p := range r.params {
		if tokenEqual(p.name, name) {
			return p.value
		}
	}
	return ""
}

// String satisfies the fmt.Stringer interface.
func (r MediaRange) String() string {
	return fmt.Sprint(r)
}

// Format satisfies the fmt.Formatter interface.
func (r MediaRange) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%s/%s", r.typ, r.sub)

	for _, p := range r.params {
		fmt.Fprintf(w, ";%v", p)
	}
}

// ParseMediaRange parses a string representation of a HTTP media range from s.
func ParseMediaRange(s string) (r MediaRange, err error) {
	var s1 string
	var s2 string
	var s3 string
	var i int
	var j int
	var mp []MediaParam

	if i = strings.IndexByte(s, '/'); i < 0 {
		goto error
	}

	s1 = s[:i]

	if j = strings.IndexByte(s[i+1:], ';'); j < 0 {
		s2 = s[i+1:]
	} else {
		s2 = s[i+1 : i+1+j]
		s3 = s[i+j+2:]
	}

	if !isToken(s1) {
		goto error
	}
	if !isToken(s2) {
		goto error
	}

	for len(s3) != 0 {
		var p MediaParam

		if i = strings.IndexByte(s3, ';'); i < 0 {
			i = len(s3)
		}

		if p, err = ParseMediaParam(trimOWS(s3[:i])); err != nil {
			goto error
		}

		mp = append(mp, p)
		s3 = s3[i:]

		if len(s3) != 0 {
			s3 = s3[1:]
		}
	}

	r = MediaRange{
		typ:    s1,
		sub:    s2,
		params: mp,
	}
	return
error:
	err = errorInvalidMediaRange(s)
	return
}

func errorInvalidMediaRange(s string) error {
	return errors.New("invalid media range: " + s)
}

// MediaParam is a representation of a HTTP media parameter as described in
// https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
type MediaParam struct {
	name  string
	value string
}

// String satisfies the fmt.Stringer interface.
func (p MediaParam) String() string {
	return fmt.Sprint(p)
}

// Format satisfies the fmt.Formatter interface.
func (p MediaParam) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%v=%v", p.name, quoted(p.value))
}

// ParseMediaParam parses a string representation of a HTTP media parameter
// from s.
func ParseMediaParam(s string) (p MediaParam, err error) {
	var s1 string
	var s2 string
	var q quoted
	var i = strings.IndexByte(s, '=')

	if i < 0 {
		goto error
	}

	s1 = s[:i]
	s2 = s[i+1:]

	if !isToken(s1) {
		goto error
	}
	if q, err = parseQuoted(s2); err != nil {
		goto error
	}

	p = MediaParam{
		name:  s1,
		value: string(q),
	}
	return
error:
	err = errorInvalidMediaParam(s)
	return
}

// MediaType is a representation of a HTTP media type which is usually expressed
// in the form of a main and sub type as in "main/sub", where both may be the
// special wildcard token "*".
type MediaType struct {
	typ string
	sub string
}

// Contains returns true if t is a superset or is equal to t2.
func (t MediaType) Contains(t2 MediaType) bool {
	return t.typ == "*" || (t.typ == t2.typ && (t.sub == "*" || t.sub == t2.sub))
}

// String satisfies the fmt.Stringer interface.
func (t MediaType) String() string {
	return fmt.Sprint(t)
}

// Format satisfies the fmt.Formatter interface.
func (t MediaType) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%s/%s", t.typ, t.sub)
}

// ParseMediaType parses the media type in s.
func ParseMediaType(s string) (t MediaType, err error) {
	var s1 string
	var s2 string
	var i = strings.IndexByte(s, '/')

	if i < 0 {
		goto error
	}

	s1 = s[:i]
	s2 = s[i+1:]

	if !isToken(s1) || !isToken(s2) {
		goto error
	}

	t = MediaType{
		typ: s1,
		sub: s2,
	}
	return
error:
	err = errorInvalidMediaType(s)
	return
}

func mediaTypeLess(t1 string, t2 string) bool {
	if t1 == t2 {
		return false
	}
	if t1 == "*" {
		return false
	}
	if t2 == "*" {
		return true
	}
	return t1 < t2
}

func errorInvalidMediaParam(s string) error {
	return errors.New("invalid media parameter: " + s)
}

func errorInvalidMediaType(s string) error {
	return errors.New("invalid media type: " + s)
}
