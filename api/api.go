// Package api is a HTTP API router that includes optional middlewares, switches on paths and
// then further can switch on a header key/value pair, and if all else fails, returns a 404.
package api

import (
	"net/http"
	"strings"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/log"
	"relay.mleku.dev/servemux"
)

type Header struct{ Key, Value string }

type Headers []Header

// Handler is an HTTP handler with a prescribed root path and net.Listener for handling HTTP
// requests.
type Handler struct {
	*servemux.S
	// Path is the root path for the Handler, the Router selects which handler to pass the request
	// to from this.
	Path string
	// Header is a header key/value pair that must match for the handler to be called.
	Headers
}

type Handlers []*Handler

type Middleware func(w http.ResponseWriter, r *http.Request) (err error)

type Middlewares []Middleware

type A struct {
	Middlewares
	Handlers
}

var Handle = &A{}

func RegisterHandler(h *Handler) { Handle.Handlers = append(Handle.Handlers, h) }

func RegisterMiddleware(m Middleware) { Handle.Middlewares = append(Handle.Middlewares, m) }

// Router processes a request according to the registered Handlers.
func (a *A) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, m := range Handle.Middlewares {
		if err := m(w, r); chk.E(err) {
			return
		}
	}
	for _, h := range Handle.Handlers {
		if strings.HasPrefix(r.URL.Path, h.Path) {
			var count int
			for _, hdr := range h.Headers {
				if r.Header.Get(hdr.Key) == hdr.Value {
					count++
				}
			}
			if count == len(h.Headers) {
				h.ServeMux.ServeHTTP(w, r)
				return
			}
		}
	}
	log.D.F("handler for path %s not found", r.URL.Path)
	w.WriteHeader(http.StatusNotFound)
}
