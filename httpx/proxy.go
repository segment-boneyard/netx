package httpx

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"time"

	"github.com/segmentio/netx"
)

// ReverseProxy is a HTTP handler which implements the logic of a reverse HTTP
// proxy, forwarding incoming requests to backend servers.
//
// The implementation is similar to httputil.ReverseProxy but the implementation
// has some differences. Instead of using a Director function to rewrite the
// request to its destination the proxy expects the request it receives to be
// already well constructed to be forwarded to a backend server. Any conforming
// HTTP client aware of being behing a proxy would have included the full URL in
// the request line which the proxy will use to extract the backend address.
//
// The proxy also converts the X-Forwarded headers to Forwarded as defined by
// RFC 7239 (see https://tools.ietf.org/html/rfc7239).
//
// HTTP upgrades are also supported by this reverse HTTP proxy implementation,
// the proxy forwards the HTTP handshake requesting an upgrade to the backend
// server, then if it gets a successful 101 Switching Protocol response it will
// start acting as a simple TCP tunnel between the client and backend server.
//
// Finally, the proxy also properly handles the Max-Forward header for TRACE and
// OPTIONS methods, decrementing the value or directly responding to the client
// if it reaches zero.
type ReverseProxy struct {
	// Transport is used to forward HTTP requests to backend servers. If nil,
	// http.DefaultTransport is used instead.
	Transport http.RoundTripper
}

// ServeHTTP satisfies the http.Handler interface.
func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	remoteAddr := req.RemoteAddr
	localAddr := requestLocalAddr(req)

	// Forwarded requests always use the HTTP/1.1 protocol when talking to the
	// backend server.
	outurl := *req.URL
	outreq := *req
	outreq.URL = &outurl
	outreq.Header = make(http.Header, len(req.Header))
	outreq.Proto = "HTTP/1.1"
	outreq.ProtoMajor = 1
	outreq.ProtoMinor = 1
	outreq.Close = false

	// No target host was set on the request URL, assuming the client intended
	// to read req.Host then.
	if len(outreq.URL.Host) == 0 {
		outreq.URL.Host = req.Host
	}

	// No target protocol was set, attempting to guess it from the port that the
	// client is trying to connect to (fail later otherwise).
	scheme := outreq.URL.Scheme
	if len(scheme) == 0 {
		outreq.URL.Scheme = guessScheme(localAddr, req.URL.Host)
	}

	// Remove hop-by-hop headers from the request so they aren't forwarded, save
	// a potential upgrade header because we'd need to put it back later.
	copyHeader(outreq.Header, req.Header)
	deleteHopFields(outreq.Header)

	// Add proxy headers, Forwarded, Via, and convert X-Forwarded-For.
	if _, hasFwd := outreq.Header["Forwarded"]; !hasFwd {
		translateXForwarded(outreq.Header)
	}
	addForwarded(outreq.Header, scheme, remoteAddr, localAddr)
	addVia(outreq.Header, protoVersion(req), localAddr)

	// Decrement the Max-Forward header for TRACE and OPTIONS requests.
	if method := req.Method; method == "TRACE" || method == "OPTIONS" {
		max, err := maxForward(outreq.Header)
		if max--; max == 0 || err != nil {
			if method == "TRACE" {
				p.serveTRACE(w, &outreq)
			} else {
				p.serveOPTIONS(w, &outreq)
			}
			return
		}
		outreq.Header.Set("Max-Forward", strconv.Itoa(max))
	}

	// The proxy has to forward a protocol upgrade, we open a new connection to
	// the target host that we can make exclusive use of, then the handshake is
	// performed and the proxy starts passing bytes back and forth.
	if upgrade := connectionUpgrade(req.Header); len(upgrade) != 0 {
		outreq.Header.Set("Connection", "Upgrade")
		outreq.Header.Set("Upgrade", upgrade)
		p.serveUpgrade(w, &outreq)
	} else {
		p.serveHTTP(w, &outreq)
	}
}

func (p *ReverseProxy) serveHTTP(w http.ResponseWriter, req *http.Request) {
	transport := p.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	res, err := transport.RoundTrip(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	deleteHopFields(res.Header)
	copyHeader(w.Header(), res.Header)

	w.WriteHeader(res.StatusCode)
	netx.Copy(w, res.Body)
	res.Body.Close()

	deleteHopFields(res.Trailer)
	copyHeader(w.Header(), res.Trailer)
}

func (p *ReverseProxy) serveOPTIONS(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (p *ReverseProxy) serveTRACE(w http.ResponseWriter, req *http.Request) {
	content, err := httputil.DumpRequest(req, true)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "message/http")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func (p *ReverseProxy) serveUpgrade(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	backend, err := p.dial(ctx, "tcp", req.URL.Host)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer backend.Close()

	res, err := (&ConnTransport{
		Conn: backend,
		ResponseHeaderTimeout: 10 * time.Second,
	}).RoundTrip(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	// Forward the response to the protocol upgrade request, removing the
	// hop-by-hop headers, except the Upgrade header which is used by some
	// protocol upgrades.
	upgrade := res.Header["Upgrade"]
	deleteHopFields(res.Header)
	if len(upgrade) != 0 {
		res.Header["Upgrade"] = upgrade
		res.Header["Connection"] = []string{"Upgrade"}
	}
	copyHeader(w.Header(), res.Header)
	w.WriteHeader(res.StatusCode)
	netx.Copy(w, res.Body)
	res.Body.Close()

	// Switching to a different protocol failed apparently, stopping here and
	// the server will wait for the next request on that connection.
	if res.StatusCode != http.StatusSwitchingProtocols {
		return
	}

	frontend, rw, err := w.(http.Hijacker).Hijack()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer frontend.Close()

	if err := rw.Writer.Flush(); err != nil {
		return // the client is gone
	}

	done := make(chan struct{}, 2)
	go forward(rw.Writer, backend, done)
	go forward(backend, rw.Reader, done)

	// Wait for either the connections to be closed or the context to be
	// canceled.
	select {
	case <-done:
	case <-ctx.Done():
	}
}

func (p *ReverseProxy) dial(ctx context.Context, network string, address string) (conn net.Conn, err error) {
	if conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", address); err != nil {
		return
	}
	switch network {
	case "https", "wss", "tls":
		conn = tls.Client(conn, &tls.Config{})
	}
	return
}

// guessScheme attempts to guess the protocol that should be used for a proxied
// request (either http or https).
func guessScheme(localAddr string, remoteAddr string) string {
	switch _, port, _ := net.SplitHostPort(remoteAddr); port {
	case "", "80":
		return "http"
	case "443":
		return "https"
	}
	if scheme, _ := splitProtoAddr(localAddr); scheme == "tls" {
		return "https"
	}
	return "http"
}

// forward copies bytes from r to w, sending a signal on the done channel when
// the copy completes.
func forward(w io.Writer, r io.Reader, done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	netx.Copy(w, r)
}

// requestLocalAddr looks for the request's local address in its context and
// returns the string representation.
func requestLocalAddr(req *http.Request) string {
	addr := contextLocalAddr(req.Context())
	if addr == nil {
		return ""
	}
	return addr.String()
}

// contextLocalAddr looks for the request's local address in ctx and returns it.
func contextLocalAddr(ctx context.Context) net.Addr {
	val := ctx.Value(http.LocalAddrContextKey)
	if val == nil {
		return nil
	}
	addr, _ := val.(net.Addr)
	return addr
}
