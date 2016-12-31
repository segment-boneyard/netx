package httpx

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/segmentio/netx"
)

// protoEqual checks if the protocol version used for req is equal to
// HTTP/major.minor
func protoEqual(req *http.Request, major int, minor int) bool {
	return req.ProtoMajor == major && req.ProtoMinor == minor
}

// protoVersion returns the version part of the protocol identifier of req.
func protoVersion(req *http.Request) string {
	proto := req.Proto
	if strings.HasPrefix(proto, "HTTP/") {
		proto = proto[5:]
	}
	return proto
}

// splitProtoAddr splits uri into the leading protocol scheme and trailing
// address.
func splitProtoAddr(uri string) (proto string, addr string) {
	if off := strings.Index(uri, "://"); off >= 0 {
		proto, addr = uri[:off], uri[off+3:]
	} else {
		addr = uri
	}
	return
}

// copyHeader copies the HTTP header src into dst.
func copyHeader(dst http.Header, src http.Header) {
	for name, values := range src {
		dst[name] = append(make([]string, 0, len(values)), values...)
	}
}

// deleteHopFields deletes the hop-by-hop fields from header.
func deleteHopFields(h http.Header) {
	forEachHeaderValues(h["Connection"], func(v string) {
		if v != "close" {
			h.Del(v)
		}
	})
	h.Del("Connection")
	h.Del("Keep-Alive")
	h.Del("Proxy-Authenticate")
	h.Del("Proxy-Authorization")
	h.Del("Proxy-Connection")
	h.Del("Te")
	h.Del("Trailer")
	h.Del("Transfer-Encoding")
	h.Del("Upgrade")
}

// translateXForwardedFor converts the X-Forwarded-* headers in their equivalent
// Forwarded header representation.
func translateXForwarded(h http.Header) {
	xFor := h.Get("X-Forwarded-For")
	xBy := h.Get("X-Forwarded-By")
	xPort := h.Get("X-Forwarded-Port")
	xProto := h.Get("X-Forwarded-Proto")
	forwarded := ""

	// If there's more than one entry in the X-Forwarded-For header it gets way
	// too complex to report all the different combinations of X-Forwarded-*
	// headers, and there's no standard saying which ones should or shouldn't be
	// included so we just translate the X-Forwarded-For list and pass on the
	// other ones.
	if n := strings.Count(xFor, ","); n != 0 {
		s := make([]string, 0, n+1)
		forEachHeaderValues([]string{xFor}, func(v string) {
			s = append(s, "for="+quoteForwarded(v))
		})
		forwarded = strings.Join(s, ", ")
	} else {
		if len(xPort) != 0 {
			xFor = net.JoinHostPort(trimOWS(xFor), trimOWS(xPort))
		}
		forwarded = makeForwarded(trimOWS(xProto), trimOWS(xFor), trimOWS(xBy))
	}

	if len(forwarded) != 0 {
		h.Set("Forwarded", forwarded)
	}

	h.Del("X-Forwarded-For")
	h.Del("X-Forwarded-By")
	h.Del("X-Forwarded-Port")
	h.Del("X-Forwarded-Proto")
}

// quoteForwarded returns addr, quoted if necessary in order to be used in the
// Forwarded header.
func quoteForwarded(addr string) string {
	if netx.IsIPv4(addr) {
		return addr
	}
	if netx.IsIPv6(addr) {
		return quote("[" + addr + "]")
	}
	return quote(addr)
}

// mameForwarded builds a Forwarded header value from proto, forAddr, and byAddr.
func makeForwarded(proto string, forAddr string, byAddr string) string {
	s := make([]string, 0, 4)
	if len(proto) != 0 {
		s = append(s, "proto="+quoted(proto).String())
	}
	if len(forAddr) != 0 {
		s = append(s, "for="+quoteForwarded(forAddr))
	}
	if len(byAddr) != 0 {
		s = append(s, "by="+quoteForwarded(byAddr))
	}
	return strings.Join(s, ";")
}

// addForwarded adds proto, forAddr, and byAddr to the Forwarded header.
func addForwarded(header http.Header, proto string, forAddr string, byAddr string) {
	addHeaderValue(header, "Forwarded", makeForwarded(proto, forAddr, byAddr))
}

// makeVia creates a Via header value from version and host.
func makeVia(version string, host string) string {
	return version + " " + host
}

// addVia adds version and host to the Via header.
func addVia(header http.Header, version string, host string) {
	addHeaderValue(header, "Via", makeVia(version, host))
}

// addHeaderValue adds value to the name header.
func addHeaderValue(header http.Header, name string, value string) {
	if prev := header.Get(name); len(prev) != 0 {
		value = prev + ", " + value
	}
	header.Set(name, value)
}

// maxForwards returns the value of the Max-Forward header.
func maxForwards(header http.Header) (max int, err error) {
	if s := header.Get("Max-Forwards"); len(s) == 0 {
		max = -1
	} else {
		max, err = strconv.Atoi(s)
	}
	return
}

// connectionUpgrade returns the value of the Upgrade header if it is present in
// the Connection header.
func connectionUpgrade(header http.Header) string {
	if !headerValuesContainsToken(header["Connection"], "Upgrade") {
		return ""
	}
	return header.Get("Upgrade")
}

// headerValuesRemoveTokens removes tokens from values, returning a new list of values.
func headerValuesRemoveTokens(values []string, tokens ...string) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		var item []string
	forEachValue:
		for len(v) != 0 {
			var s string
			s, v = readHeaderValue(v)
			for _, t := range tokens {
				if tokenEqual(t, s) {
					continue forEachValue
				}
			}
			item = append(item, s)
		}
		if len(item) != 0 {
			result = append(result, strings.Join(item, ", "))
		}
	}
	return result
}

// forEachHeaderValues through each value of l, where each element of l is a
// comma-separated list of values, calling f on each element.
func forEachHeaderValues(l []string, f func(string)) {
	for _, a := range l {
		for len(a) != 0 {
			var s string
			s, a = readHeaderValue(a)
			f(s)
		}
	}
}

// readHeaderValue tries to read the next value in a comma-separated list.
func readHeaderValue(s string) (value string, tail string) {
	if off := strings.IndexByte(s, ','); off >= 0 {
		value, tail = s[:off], s[off+1:]
	} else {
		value = s
	}
	value = trimOWS(value)
	return
}

// isIdempotent returns true if method is idempotent.
func isIdempotent(method string) bool {
	switch method {
	case "HEAD", "GET", "PUT", "DELETE", "OPTIONS":
		return true
	}
	return false
}

// isRetriable returns true if the status is a retriable error.
func isRetriable(status int) bool {
	switch status {
	case http.StatusInternalServerError:
	case http.StatusBadGateway:
	case http.StatusServiceUnavailable:
	case http.StatusGatewayTimeout:
	default:
		return false
	}
	return true
}
