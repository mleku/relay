package relay

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rs/cors"

	"relay.mleku.dev/api"
	"relay.mleku.dev/chk"
	"relay.mleku.dev/context"
	"relay.mleku.dev/log"
	"relay.mleku.dev/store"
)

type Server struct {
	Ctx             context.T
	Cancel          context.F
	WG              *sync.WaitGroup
	Address         string
	ConfigurationMx *sync.Mutex
	Configuration   *store.Configuration
	HTTPServer      *http.Server
	Store           store.I
}

func (s *Server) Start() (err error) {
	var listener net.Listener
	if listener, err = net.Listen("tcp", s.Address); chk.E(err) {
		return
	}
	s.HTTPServer = &http.Server{
		Handler:           cors.Default().Handler(api.Handle),
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
