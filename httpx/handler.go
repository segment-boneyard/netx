package httpx

import "net/http"

// StatusHandler returns a HTTP handler that always responds with status and an
// empty body.
func StatusHandler(status int) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(status)
	})
}
