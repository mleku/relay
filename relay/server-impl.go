package relay

import (
	"net/http"
	"time"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/context"
	"relay.mleku.dev/event"
	"relay.mleku.dev/log"
	"relay.mleku.dev/relay/config"
	"relay.mleku.dev/relay/interfaces"
	"relay.mleku.dev/store"
)

func (s *Server) AdminAuth(r *http.Request, remote string,
	tolerance ...time.Duration) (authed bool, pubkey []byte) {

	return s.adminAuth(r, remote, tolerance...)
}

func (s *Server) Storage() store.I { return s.Store }

func (s *Server) Configuration() config.C {
	s.configurationMx.Lock()
	defer s.configurationMx.Unlock()
	if s.configuration == nil {
		s.configured = false
		return config.C{}
	}
	return *s.configuration
}

func (s *Server) SetConfiguration(cfg *config.C) {
	s.configurationMx.Lock()
	s.configuration = cfg
	s.configured = true
	chk.E(s.UpdateConfiguration())
	s.configurationMx.Unlock()
}

func (s *Server) AddEvent(
	c context.T, ev *event.T, hr *http.Request, authedPubkey []byte,
	remote string) (accepted bool, message []byte) {

	return s.addEvent(c, ev, authedPubkey, remote)
}

func (s *Server) AcceptEvent(
	c context.T, ev *event.T, hr *http.Request, authedPubkey []byte,
	remote string) (accept bool, notice string, afterSave func()) {

	return s.acceptEvent(c, ev, remote, authedPubkey)
}

func (s *Server) PublicReadable() bool {
	s.configurationMx.Lock()
	defer s.configurationMx.Unlock()
	pr := s.configuration.PublicReadable
	log.T.F("public readable %v", pr)
	return pr
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
