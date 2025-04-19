package openapi

import (
	"github.com/danielgtaylor/huma/v2"

	"relay.mleku.dev/relay/interfaces"
	"relay.mleku.dev/router"
	"relay.mleku.dev/servemux"
)

type Operations struct {
	interfaces.Server
	path string
	*servemux.S
}

// New creates a new openapi.Operations and registers its methods.
func New(s interfaces.Server, name, version, description string, path string,
	sm *servemux.S) (handler *router.Handler) {

	handler = &router.Handler{Path: path, S: sm}
	a := NewHuma(sm, name, version, description)
	huma.AutoRegister(a, &Operations{Server: s, path: path})
	return
}
