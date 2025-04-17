package relay

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/cors"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/context"
	"relay.mleku.dev/log"
	"relay.mleku.dev/relay/helpers"
	"relay.mleku.dev/servemux"
	"relay.mleku.dev/signer"
	"relay.mleku.dev/store"
)

type List map[string]struct{}

type Server struct {
	Name       string
	Ctx        context.T
	Cancel     context.F
	WG         *sync.WaitGroup
	Address    string
	HTTPServer *http.Server
	Mux        *servemux.S
	huma.API
	Store    store.I
	MaxLimit int

	configurationMx sync.Mutex
	configuration   store.Configuration

	sync.Mutex
	admins []signer.I
	owners [][]byte
	// Followed are the pubkeys that are in the Owners' follow lists and have full
	// access permission.
	Followed List
	// OwnersFollowed are "guests" of the Followed and have full access but with
	// rate limiting enabled.
	ownersFollowed List
	// Muted are on Owners' mute lists and do not have write access to the relay,
	// even if they would be in the OwnersFollowed list, they can only read.
	Muted List
	// OwnersFollowLists are the event IDs of owners follow lists, which must not be
	// deleted, only replaced.
	OwnersFollowLists [][]byte
	// OwnersMuteLists are the event IDs of owners mute lists, which must not be
	// deleted, only replaced.
	OwnersMuteLists [][]byte
}

func (s *Server) Start() (err error) {
	s.Init()
	var listener net.Listener
	if listener, err = net.Listen("tcp", s.Address); chk.E(err) {
		return
	}
	s.HTTPServer = &http.Server{
		Handler:           cors.Default().Handler(s),
		Addr:              s.Address,
		ReadHeaderTimeout: 7 * time.Second,
		IdleTimeout:       28 * time.Second,
	}
	log.I.F("listening on %s", s.Address)
	if err = s.HTTPServer.Serve(listener); errors.Is(err, http.ErrServerClosed) {
		return
	} else if chk.E(err) {
		return
	}
	return
}

// ServeHTTP is the server http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remote := helpers.GetRemoteFromReq(r)
	for _, a := range s.Configuration().BlockList {
		if strings.HasPrefix(remote, a) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
	}
	// standard nostr protocol only governs the "root" path of the relay and websockets
	// if r.URL.Path == "/" && r.Header.Get("Accept") == "application/nostr+json" {
	// 	s.handleRelayInfo(w, r)
	// 	return
	// }
	// if r.URL.Path == "/" && r.Header.Get("Upgrade") == "websocket" {
	// 	(&socketapi.A{Server: s}).ServeHTTP(w, r)
	// 	return
	// }
	log.I.F("http request: %s from %s", r.URL.String(), helpers.GetRemoteFromReq(r))
	s.Mux.ServeHTTP(w, r)
}

func (s *Server) Shutdown() {
	log.W.Ln("shutting down relay")
	s.Cancel()
	log.W.Ln("waiting for current operations to stop")
	s.WG.Wait()
	log.W.Ln("closing event store")
	chk.E(s.Store.Close())
	log.W.Ln("shutting down relay listener")
	chk.E(s.HTTPServer.Shutdown(s.Ctx))
}
