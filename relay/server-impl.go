package relay

import (
	"net/http"
	"time"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/context"
	"relay.mleku.dev/event"
	"relay.mleku.dev/relay/interfaces"
	"relay.mleku.dev/store"
)

func (s *Server) AdminAuth(r *http.Request,
	tolerance ...time.Duration) (authed bool, pubkey []byte) {

	return s.adminAuth(r, tolerance...)
}

func (s *Server) Storage() store.I { return s.Store }

func (s *Server) Configuration() store.Configuration {
	s.configurationMx.Lock()
	defer s.configurationMx.Unlock()
	if s.configuration == nil {
		return store.Configuration{}
	}
	return *s.configuration
}

func (s *Server) SetConfiguration(cfg *store.Configuration) {
	s.configurationMx.Lock()
	s.configuration = cfg
	chk.E(s.UpdateConfiguration())
	s.configurationMx.Unlock()
}

func (s *Server) AddEvent(
	c context.T, ev *event.T, hr *http.Request, origin string,
	authedPubkey []byte) (accepted bool, message []byte) {

	return s.addEvent(c, ev, authedPubkey)
}

func (s *Server) AcceptEvent(
	c context.T, ev *event.T, hr *http.Request, origin string,
	authedPubkey []byte) (accept bool, notice string, afterSave func()) {

	return s.acceptEvent(c, ev, hr, origin, authedPubkey)
}

func (s *Server) PublicReadable() bool {
	s.configurationMx.Lock()
	defer s.configurationMx.Unlock()
	return s.configuration.PublicReadable
}

func (s *Server) Context() context.T { return s.Ctx }

func (s *Server) Owners() [][]byte { return s.owners }

func (s *Server) AuthRequired() bool {
	s.configurationMx.Lock()
	defer s.configurationMx.Unlock()
	return s.configuration.AuthRequired
}

func (s *Server) OwnersFollowed(pubkey string) (ok bool) {
	s.Lock()
	defer s.Unlock()
	_, ok = s.ownersFollowed[pubkey]
	return
}

var _ interfaces.Server = &Server{}
