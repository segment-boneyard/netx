package httpx

import (
	"errors"
	"fmt"
	"strings"
)

// MediaRange is a representation of a HTTP media range as described in
// https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
type MediaRange struct {
	Type    string
	SubType string
	Params  []MediaParam
}

// Param return the value of the parameter with name, which will be an empty
// string if none was found.
func (r MediaRange) Param(name string) string {
	for _, p := range r.Params {
		if tokenEqual(p.Name, name) {
			return p.Value
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
	fmt.Fprintf(w, "%s/%s", r.Type, r.SubType)

	for _, p := range r.Params {
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
		err = errorInvalidMediaRange(s)
		return
	}

	s1 = s[:i]

	if j = strings.IndexByte(s[i+1:], ';'); j < 0 {
		s2 = s[i+1:]
	} else {
		s2 = s[i+1 : i+1+j]
		s3 = s[i+j+2:]
	}

	if !isToken(s1) {
		err = errorInvalidMediaRange(s)
		return
	}

	if !isToken(s2) {
		err = errorInvalidMediaRange(s)
		return
	}

	for len(s3) != 0 {
		var p MediaParam

		if i = strings.IndexByte(s3, ';'); i < 0 {
			i = len(s3)
		}

		if p, err = ParseMediaParam(trimOWS(s3[:i])); err != nil {
			err = errorInvalidMediaParam(s)
			return
		}

		mp = append(mp, p)
		s3 = s3[i:]

		if len(s3) != 0 {
			s3 = s3[1:]
		}
	}

	r = MediaRange{
		Type:    s1,
		SubType: s2,
		Params:  mp,
	}
	return
}

func errorInvalidMediaRange(s string) error {
	return errors.New("invalid media range: " + s)
}

// MediaParam is a representation of a HTTP media parameter as described in
// https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
type MediaParam struct {
	Name  string
	Value string
}

// String satisfies the fmt.Stringer interface.
func (p MediaParam) String() string {
	return fmt.Sprint(p)
}

// Format satisfies the fmt.Formatter interface.
func (p MediaParam) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%v=%v", p.Name, quoted(p.Value))
}

// ParseMediaParam parses a string representation of a HTTP media parameter
// from s.
func ParseMediaParam(s string) (p MediaParam, err error) {
	i := strings.IndexByte(s, '=')

	if i < 0 {
		err = errorInvalidMediaParam(s)
		return
	}

	var s1 = s[:i]
	var s2 = s[i+1:]
	var q quoted

	if !isToken(s1) {
		err = errorInvalidMediaParam(s)
		return
	}

	if q, err = parseQuoted(s2); err != nil {
		err = errorInvalidMediaParam(s)
		return
	}

	p = MediaParam{
		Name:  s1,
		Value: string(q),
	}
	return
}

// MediaType is a representation of a HTTP media type which is usually expressed
// in the form of a main and sub type as in "main/sub", where both may be the
// special wildcard token "*".
type MediaType struct {
	Type    string
	SubType string
}

// Contains returns true if t is a superset or is equal to t2.
func (t MediaType) Contains(t2 MediaType) bool {
	return t.Type == "*" || (t.Type == t2.Type && (t.SubType == "*" || t.SubType == t2.SubType))
}

// String satisfies the fmt.Stringer interface.
func (t MediaType) String() string {
	return fmt.Sprint(t)
}

// Format satisfies the fmt.Formatter interface.
func (t MediaType) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%s/%s", t.Type, t.SubType)
}

// ParseMediaType parses the media type in s.
func ParseMediaType(s string) (t MediaType, err error) {
	i := strings.IndexByte(s, '/')

	if i < 0 {
		err = errorInvalidMediaType(s)
		return
	}

	s1 := s[:i]
	s2 := s[i+1:]

	if !isToken(s1) || !isToken(s2) {
		err = errorInvalidMediaType(s)
		return
	}

	t = MediaType{
		Type:    s1,
		SubType: s2,
	}
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
