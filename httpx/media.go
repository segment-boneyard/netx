package httpx

import (
	"errors"
	"fmt"
	"strings"
)

// mediaRange is a representation of a HTTP media range as described in
// https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
type mediaRange struct {
	typ    string
	sub    string
	params []mediaParam
}

// Param return the value of the parameter with name, which will be an empty
// string if none was found.
func (r mediaRange) Param(name string) string {
	for _, p := range r.params {
		if tokenEqual(p.name, name) {
			return p.value
		}
	}
	return ""
}

// String satisfies the fmt.Stringer interface.
func (r mediaRange) String() string {
	return fmt.Sprint(r)
}

// Format satisfies the fmt.Formatter interface.
func (r mediaRange) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%s/%s", r.typ, r.sub)

	for _, p := range r.params {
		fmt.Fprintf(w, ";%v", p)
	}
}

// parseMediaRange parses a string representation of a HTTP media range from s.
func parseMediaRange(s string) (r mediaRange, err error) {
	var s1 string
	var s2 string
	var s3 string
	var i int
	var j int
	var mp []mediaParam

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
		var p mediaParam

		if i = strings.IndexByte(s3, ';'); i < 0 {
			i = len(s3)
		}

		if p, err = parseMediaParam(trimOWS(s3[:i])); err != nil {
			goto error
		}

		mp = append(mp, p)
		s3 = s3[i:]

		if len(s3) != 0 {
			s3 = s3[1:]
		}
	}

	r = mediaRange{
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

// mediaParam is a representation of a HTTP media parameter as described in
// https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
type mediaParam struct {
	name  string
	value string
}

// String satisfies the fmt.Stringer interface.
func (p mediaParam) String() string {
	return fmt.Sprint(p)
}

// Format satisfies the fmt.Formatter interface.
func (p mediaParam) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%v=%v", p.name, quoted(p.value))
}

// parseMediaParam parses a string representation of a HTTP media parameter
// from s.
func parseMediaParam(s string) (p mediaParam, err error) {
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

	p = mediaParam{
		name:  s1,
		value: string(q),
	}
	return
error:
	err = errorInvalidMediaParam(s)
	return
}

// mediaType is a representation of a HTTP media type which is usually expressed
// in the form of a main and sub type as in "main/sub", where both may be the
// special wildcard token "*".
type mediaType struct {
	typ string
	sub string
}

// Contains returns true if t is a superset or is equal to t2.
func (t mediaType) Contains(t2 mediaType) bool {
	return t.typ == "*" || (t.typ == t2.typ && (t.sub == "*" || t.sub == t2.sub))
}

// String satisfies the fmt.Stringer interface.
func (t mediaType) String() string {
	return fmt.Sprint(t)
}

// Format satisfies the fmt.Formatter interface.
func (t mediaType) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%s/%s", t.typ, t.sub)
}

// parseMediaType parses the media type in s.
func parseMediaType(s string) (t mediaType, err error) {
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

	t = mediaType{
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
