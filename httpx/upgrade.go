package httpx

import (
	"net/http"
	"sync"
)

// UpgradeMap maps protocol names to HTTP handlers that should be used to
// perform upgrades from HTTP to a different protocol.
//
// It is expected that the upgrade handler flushes and hijacks the connection
// after sending the response, and doesn't return from its ServeHTTP method
// until it's done serving the connection (or it would be closed prematuraly).
//
// A special-case is made for the name "*" which indicates that the handler is
// set as a the fallback upgrade handler to handle unrecognized protocols.
//
// Keys in the UpgradeMap map should be formatted with http.CanonicalHeaderKey.
type UpgradeMap map[string]http.Handler

// ServeHTTP satisfies the http.Handler interface so an UpgradeMap can be used as
// handler on an HTTP server.
func (u UpgradeMap) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h := u[http.CanonicalHeaderKey(req.Header.Get("Upgrade"))]

	if h == nil {
		h = u["*"]
	}

	if h == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	h.ServeHTTP(w, req)
}

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
	upgrader UpgradeMap
}

// NewUpgradeMux allocates and returns a new UpgradeMux.
func NewUpgradeMux() *UpgradeMux {
	return &UpgradeMux{}
}

// Handle registers a handler for the given protocol name. If a handler already
// exists for name, Handle panics.
//
// A special-case is made for the name "*" which indicates that the handler is
// set as a the fallback upgrade handler to handle unrecognized protocols.
func (mux *UpgradeMux) Handle(name string, handler http.Handler) {
	var key string

	if name != "*" {
		key = http.CanonicalHeaderKey(name)
	}

	defer mux.mutex.Unlock()
	mux.mutex.Lock()

	if mux.upgrader[key] != nil {
		panic("an upgrade handler already exists for " + name)
	}

	if mux.upgrader == nil {
		mux.upgrader = make(UpgradeMap)
	}

	mux.upgrader[key] = handler
}

// HandleFunc registers a handler function for the given protocol name. If a
// handler already exists for name, HandleFunc panics.
func (mux *UpgradeMux) HandleFunc(name string, handler func(http.ResponseWriter, *http.Request)) {
	mux.Handle(name, http.HandlerFunc(handler))
}

// Handler returns the appropriate http.Handler for serving req.
func (mux *UpgradeMux) Handler(req *http.Request) http.Handler {
	key := http.CanonicalHeaderKey(req.Header.Get("Upgrade"))

	if len(key) == 0 {
		return nil
	}

	mux.mutex.RLock()
	h := mux.upgrader[key]
	mux.mutex.RUnlock()
	return h
}

// ServeHTTP satisfies the http.Handler interface so UpgradeMux can be used as
// handler on an HTTP server.
func (mux *UpgradeMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h := mux.Handler(req)

	if h == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	h.ServeHTTP(w, req)
}
