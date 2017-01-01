package httpx

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// acceptItem is the representation of an item in an Accept header.
type acceptItem struct {
	typ    string
	sub    string
	q      float32
	params []mediaParam
	extens []mediaParam
}

// String satisfies the fmt.Stringer interface.
func (item acceptItem) String() string {
	return fmt.Sprint(item)
}

// Format satisfies the fmt.Formatter interface.
func (item acceptItem) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%s/%s", item.typ, item.sub)

	for _, p := range item.params {
		fmt.Fprintf(w, ";%v", p)
	}

	fmt.Fprintf(w, ";q=%.1f", item.q)

	for _, e := range item.extens {
		fmt.Fprintf(w, ";%v", e)
	}
}

// parseAcceptItem parses a single item in an Accept header.
func parseAcceptItem(s string) (item acceptItem, err error) {
	var r mediaRange

	if r, err = parseMediaRange(s); err != nil {
		err = errorInvalidAccept(s)
		return
	}

	item = acceptItem{
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

// accept is the representation of an Accept header.
type accept []acceptItem

// String satisfies the fmt.Stringer interface.
func (accept accept) String() string {
	return fmt.Sprint(accept)
}

// Format satisfies the fmt.Formatter interface.
func (accept accept) Format(w fmt.State, r rune) {
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
func (accept accept) Negotiate(types ...string) string {
	if len(types) == 0 {
		return ""
	}
	for _, acc := range accept {
		for _, typ := range types {
			t2, err := parseMediaType(typ)
			if err != nil {
				continue
			}
			t1 := mediaType{
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
func (accept accept) Less(i int, j int) bool {
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
func (accept accept) Swap(i int, j int) {
	accept[i], accept[j] = accept[j], accept[i]
}

// Len satisfies sort.Interface.
func (accept accept) Len() int {
	return len(accept)
}

// parseAccept parses the value of an Accept header from s.
func parseAccept(s string) (accept accept, err error) {
	var head string
	var tail = s

	for len(tail) != 0 {
		var item acceptItem
		head, tail = splitTrimOWS(tail, ',')

		if item, err = parseAcceptItem(head); err != nil {
			return
		}

		accept = append(accept, item)
	}

	sort.Sort(accept)
	return
}

// acceptEncodingItem represents a single item in an Accept-Encoding header.
type acceptEncodingItem struct {
	coding string
	q      float32
}

// String satisfies the fmt.Stringer interface.
func (item acceptEncodingItem) String() string {
	return fmt.Sprint(item)
}

// Format satisfies the fmt.Formatter interface.
func (item acceptEncodingItem) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%s;q=%.1f", item.coding, item.q)
}

// parseAcceptEncodingItem parses a single item in an Accept-Encoding header.
func parseAcceptEncodingItem(s string) (item acceptEncodingItem, err error) {
	if i := strings.IndexByte(s, ';'); i < 0 {
		item.coding = s
		item.q = 1.0
	} else {
		var p mediaParam

		if p, err = parseMediaParam(trimOWS(s[i+1:])); err != nil {
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

// acceptEncoding respresents an Accept-Encoding header.
type acceptEncoding []acceptEncodingItem

// String satisfies the fmt.Stringer interface.
func (accept acceptEncoding) String() string {
	return fmt.Sprint(accept)
}

// Format satisfies the fmt.Formatter interface.
func (accept acceptEncoding) Format(w fmt.State, r rune) {
	for i, item := range accept {
		if i != 0 {
			fmt.Fprint(w, ", ")
		}
		item.Format(w, r)
	}
}

// Less satisfies sort.Interface.
func (accept acceptEncoding) Less(i int, j int) bool {
	ai, aj := &accept[i], &accept[j]
	return ai.q > aj.q || (ai.q == aj.q && mediaTypeLess(ai.coding, aj.coding))
}

// Swap satisfies sort.Interface.
func (accept acceptEncoding) Swap(i int, j int) {
	accept[i], accept[j] = accept[j], accept[i]
}

// Len satisfies sort.Interface.
func (accept acceptEncoding) Len() int {
	return len(accept)
}

// parseAcceptEncoding parses an Accept-Encoding header value from s.
func parseAcceptEncoding(s string) (accept acceptEncoding, err error) {
	var head string
	var tail = s

	for len(tail) != 0 {
		var item acceptEncodingItem
		head, tail = splitTrimOWS(tail, ',')

		if item, err = parseAcceptEncodingItem(head); err != nil {
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
