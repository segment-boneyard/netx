package httpx

import (
	"net/http"
	"strings"
	"sync"
)

// UpgradeMux maps protocol names to HTTP handlers that should be used to
// perform upgrades from HTTP to a different protocol.
//
// It is expected that the upgrade handler flushes and hijacks the connection
// after sending the response, and doesn't return from its ServeHTTP method
// until it's done serving the connection (or it would be closed prematuraly).
//
// UpgradeMux exposes the exact same API than http.ServeMux, therefore is safe
// to use by multiple concurrent goroutines.
type UpgradeMux struct {
	mutex    sync.RWMutex
	handlers map[string]http.Handler
}

// NewUpgradeMux allocates and returns a new UpgradeMux.
func NewUpgradeMux() *UpgradeMux {
	return &UpgradeMux{
		handlers: make(map[string]http.Handler),
	}
}

// Handle registers a handler for the given protocol name. If a handler already
// exists for name, Handle panics.
//
// A special-case is made for the name "*" which indicates that the handler is
// set as a the fallback upgrade handler to handler unrecognized protocols.
func (mux *UpgradeMux) Handle(name string, handler http.Handler) {
	var key string

	if name != "*" {
		key = strings.ToLower(name)
	}

	defer mux.mutex.Unlock()
	mux.mutex.Lock()

	if mux.handlers[key] != nil {
		panic("an upgrade handler already exists for " + name)
	}

	if mux.handlers == nil {
		mux.handlers = make(map[string]http.Handler)
	}

	mux.handlers[key] = handler
}

// HandleFunc registers a handler function for the given protocol name. If a
// handler already exists for name, HandleFunc panics.
func (mux *UpgradeMux) HandleFunc(name string, handler func(http.ResponseWriter, *http.Request)) {
	mux.Handle(name, http.HandlerFunc(handler))
}

// Handler retuns the appropriate http.Handler for serving req.
func (mux *UpgradeMux) Handler(req *http.Request) http.Handler {
	key := strings.ToLower(req.Header.Get("Upgrade"))

	if len(key) == 0 {
		return nil
	}

	mux.mutex.RLock()
	h := mux.handlers[key]

	if h == nil {
		h = mux.handlers[""]
	}

	mux.mutex.RUnlock()
	return h
}

// ServeHTTP satisfies the http.Handler interface os UpgradeMux can be used as
// handler on an HTTP server.
func (mux *UpgradeMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h := mux.Handler(req)

	if h == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	h.ServeHTTP(w, req)
}
