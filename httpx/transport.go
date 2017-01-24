package httpx

import "net/http"

// RoundTripperFunc makes it possible to use regular functions as transports for
// HTTP clients.
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls f.
func (f RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
