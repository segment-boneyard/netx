package httpx

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// AcceptItem is the representation of an item in an Accept header.
type AcceptItem struct {
	Type       string
	SubType    string
	Q          float32
	Params     []MediaParam
	Extensions []MediaParam
}

// String satisfies the fmt.Stringer interface.
func (item AcceptItem) String() string {
	return fmt.Sprint(item)
}

// String satisfies the fmt.Stringer interface.
func (item AcceptItem) Format(w fmt.State, _ rune) {
	fmt.Fprintf(w, "%s/%s", item.Type, item.SubType)

	for _, p := range item.Params {
		fmt.Fprintf(w, ";%v", p)
	}

	fmt.Fprintf(w, ";q=%.1f", item.Q)

	for _, e := range item.Extensions {
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
		Type:    r.Type,
		SubType: r.SubType,
		Q:       1.0,
		Params:  r.Params,
	}

	for i, p := range r.Params {
		if p.Name == "q" {
			item.Q = parseQ(p.Value)
			if item.Params = r.Params[:i]; len(item.Params) == 0 {
				item.Params = nil
			}
			if item.Extensions = r.Params[i+1:]; len(item.Extensions) == 0 {
				item.Extensions = nil
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
				Type:    acc.Type,
				SubType: acc.SubType,
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

	if ai.Q > aj.Q {
		return true
	}

	if ai.Q < aj.Q {
		return false
	}

	if ai.Type == aj.Type && ai.SubType == aj.SubType {
		n1 := len(ai.Params) + len(ai.Extensions)
		n2 := len(aj.Params) + len(aj.Extensions)
		return n1 > n2
	}

	if ai.Type != aj.Type {
		return mediaTypeLess(ai.Type, aj.Type)
	}

	return mediaTypeLess(ai.SubType, aj.SubType)
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
	Coding string
	Q      float32
}

// String satisfies the fmt.Stringer interface.
func (item AcceptEncodingItem) String() string {
	return fmt.Sprint(item)
}

// String satisfies the fmt.Stringer interface.
func (item AcceptEncodingItem) Format(w fmt.State, _ rune) {
	fmt.Fprint(w, "%s;q=%.1f", item.Coding, item.Q)
}

// ParseAcceptEncodingItem parses a single item in an Accept-Encoding header.
func ParseAcceptEncodingItem(s string) (item AcceptEncodingItem, err error) {
	if i := strings.IndexByte(s, ';'); i < 0 {
		item.Coding = s
	} else {
		var p MediaParam

		if p, err = ParseMediaParam(trimOWS(s[i+1:])); err != nil {
			err = errorInvalidAcceptEncoding(s)
			return
		}

		if p.Name != "q" {
			err = errorInvalidAcceptEncoding(s)
			return
		}

		item.Coding = s[:i]
		item.Q = parseQ(p.Value)
	}
	if !isToken(item.Coding) {
		err = errorInvalidAcceptEncoding(s)
		return
	}
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

// Less satisfies sort.Interface.
func (accept AcceptEncoding) Less(i int, j int) bool {
	ai, aj := &accept[i], &accept[j]
	return ai.Q < aj.Q || (ai.Q == aj.Q && ai.Coding < ai.Coding)
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

func parseQ(s string) float32 {
	q, _ := strconv.ParseFloat(s, 32)
	return float32(q)
}
