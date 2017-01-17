package httpx

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Negotiate performs an Accept header negotiation where the server can expose
// the content in the given list of types.
//
// If none types match the method returns the first element in the list of
// types.
//
// Here's an example of a typical use of this function:
//
//	accept := Negotiate(req.Header.Get("Accept"), "image/png", "image/jpg")
//
func Negotiate(accept string, types ...string) string {
	a, _ := ParseAccept(accept)
	return a.Negotiate(types...)
}

// NegotiateEncoding performs an Accept-Encoding header negotiation where the
// server can expose the content in the given list of codings.
//
// If none types match the method returns an empty string to indicate that the
// server should not apply any encoding to its response.
//
// Here's an exmaple of a typical use of this function:
//
//	encoding := NegotiateEncoding(req.Get("Accept-Encoding"), "gzip", "deflate")
//
func NegotiateEncoding(accept string, codings ...string) string {
	a, _ := ParseAcceptEncoding(accept)
	return a.Negotiate(codings...)
}

// AcceptItem is the representation of an item in an Accept header.
type AcceptItem struct {
	typ    string
	sub    string
	q      float32
	params []MediaParam
	extens []MediaParam
}

// String satisfies the fmt.Stringer interface.
func (item AcceptItem) String() string {
	return fmt.Sprint(item)
}

// Format satisfies the fmt.Formatter interface.
func (item AcceptItem) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%s/%s", item.typ, item.sub)

	for _, p := range item.params {
		fmt.Fprintf(w, ";%v", p)
	}

	fmt.Fprintf(w, ";q=%.1f", item.q)

	for _, e := range item.extens {
		fmt.Fprintf(w, ";%v", e)
	}
}

// ParseAcceptItem parses a single item in an Accept header.
func ParseAcceptItem(s string) (item AcceptItem, err error) {
	var r MediaRange

	if r, err = ParseMediaRange(s); err != nil {
		err = errorInvalidAccept(s)
		return
	}

	item = AcceptItem{
		typ:    r.typ,
		sub:    r.sub,
		q:      1.0,
		params: r.params,
	}

	for i, p := range r.params {
		if p.name == "q" {
			item.q = q(p.value)
			if item.params = r.params[:i]; len(item.params) == 0 {
				item.params = nil
			}
			if item.extens = r.params[i+1:]; len(item.extens) == 0 {
				item.extens = nil
			}
			break
		}
	}

	return
}

// Accept is the representation of an Accept header.
type Accept []AcceptItem

// String satisfies the fmt.Stringer interface.
func (accept Accept) String() string {
	return fmt.Sprint(accept)
}

// Format satisfies the fmt.Formatter interface.
func (accept Accept) Format(w fmt.State, r rune) {
	for i, item := range accept {
		if i != 0 {
			fmt.Fprint(w, ", ")
		}
		item.Format(w, r)
	}
}

// Negotiate performs an Accept header negotiation where the server can expose
// the content in the given list of types.
//
// If none types match the method returns the first element in the list of
// types.
func (accept Accept) Negotiate(types ...string) string {
	if len(types) == 0 {
		return ""
	}
	for _, acc := range accept {
		for _, typ := range types {
			t2, err := ParseMediaType(typ)
			if err != nil {
				continue
			}
			t1 := MediaType{
				typ: acc.typ,
				sub: acc.sub,
			}
			if t1.Contains(t2) {
				return typ
			}
		}
	}
	return types[0]
}

// Less satisfies sort.Interface.
func (accept Accept) Less(i int, j int) bool {
	ai, aj := &accept[i], &accept[j]

	if ai.q > aj.q {
		return true
	}

	if ai.q < aj.q {
		return false
	}

	if ai.typ == aj.typ && ai.sub == aj.sub {
		n1 := len(ai.params) + len(ai.extens)
		n2 := len(aj.params) + len(aj.extens)
		return n1 > n2
	}

	if ai.typ != aj.typ {
		return mediaTypeLess(ai.typ, aj.typ)
	}

	return mediaTypeLess(ai.sub, aj.sub)
}

// Swap satisfies sort.Interface.
func (accept Accept) Swap(i int, j int) {
	accept[i], accept[j] = accept[j], accept[i]
}

// Len satisfies sort.Interface.
func (accept Accept) Len() int {
	return len(accept)
}

// ParseAccept parses the value of an Accept header from s.
func ParseAccept(s string) (accept Accept, err error) {
	var head string
	var tail = s

	for len(tail) != 0 {
		var item AcceptItem
		head, tail = splitTrimOWS(tail, ',')

		if item, err = ParseAcceptItem(head); err != nil {
			return
		}

		accept = append(accept, item)
	}

	sort.Sort(accept)
	return
}

// AcceptEncodingItem represents a single item in an Accept-Encoding header.
type AcceptEncodingItem struct {
	coding string
	q      float32
}

// String satisfies the fmt.Stringer interface.
func (item AcceptEncodingItem) String() string {
	return fmt.Sprint(item)
}

// Format satisfies the fmt.Formatter interface.
func (item AcceptEncodingItem) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%s;q=%.1f", item.coding, item.q)
}

// ParseAcceptEncodingItem parses a single item in an Accept-Encoding header.
func ParseAcceptEncodingItem(s string) (item AcceptEncodingItem, err error) {
	if i := strings.IndexByte(s, ';'); i < 0 {
		item.coding = s
		item.q = 1.0
	} else {
		var p MediaParam

		if p, err = ParseMediaParam(trimOWS(s[i+1:])); err != nil {
			goto error
		}
		if p.name != "q" {
			goto error
		}

		item.coding = s[:i]
		item.q = q(p.value)
	}
	if !isToken(item.coding) {
		goto error
	}
	return
error:
	err = errorInvalidAcceptEncoding(s)
	return
}

// AcceptEncoding respresents an Accept-Encoding header.
type AcceptEncoding []AcceptEncodingItem

// String satisfies the fmt.Stringer interface.
func (accept AcceptEncoding) String() string {
	return fmt.Sprint(accept)
}

// Format satisfies the fmt.Formatter interface.
func (accept AcceptEncoding) Format(w fmt.State, r rune) {
	for i, item := range accept {
		if i != 0 {
			fmt.Fprint(w, ", ")
		}
		item.Format(w, r)
	}
}

// Negotiate performs an Accept-Encoding header negotiation where the server can
// expose the content in the given list of codings.
//
// If none types match the method returns an empty string to indicate that the
// server should not apply any encoding to its response.
func (accept AcceptEncoding) Negotiate(codings ...string) string {
	for _, acc := range accept {
		for _, coding := range codings {
			if coding == acc.coding {
				return coding
			}
		}
	}
	return ""
}

// Less satisfies sort.Interface.
func (accept AcceptEncoding) Less(i int, j int) bool {
	ai, aj := &accept[i], &accept[j]
	return ai.q > aj.q || (ai.q == aj.q && mediaTypeLess(ai.coding, aj.coding))
}

// Swap satisfies sort.Interface.
func (accept AcceptEncoding) Swap(i int, j int) {
	accept[i], accept[j] = accept[j], accept[i]
}

// Len satisfies sort.Interface.
func (accept AcceptEncoding) Len() int {
	return len(accept)
}

// ParseAcceptEncoding parses an Accept-Encoding header value from s.
func ParseAcceptEncoding(s string) (accept AcceptEncoding, err error) {
	var head string
	var tail = s

	for len(tail) != 0 {
		var item AcceptEncodingItem
		head, tail = splitTrimOWS(tail, ',')

		if item, err = ParseAcceptEncodingItem(head); err != nil {
			return
		}

		accept = append(accept, item)
	}

	sort.Sort(accept)
	return
}

func errorInvalidAccept(s string) error {
	return errors.New("invalid Accept header value: " + s)
}

func errorInvalidAcceptEncoding(s string) error {
	return errors.New("invalid Accept-Encoding header value: " + s)
}

func q(s string) float32 {
	q, _ := strconv.ParseFloat(s, 32)
	return float32(q)
}
