package relay

import (
	"bytes"
	"net/http"

	"relay.mleku.dev/context"
	"relay.mleku.dev/ec/schnorr"
	"relay.mleku.dev/filters"
	"relay.mleku.dev/log"
)

func (s *Server) AcceptReq(c context.T, hr *http.Request, id []byte,
	ff *filters.T, authedPubkey []byte, remote string) (allowed *filters.T, ok bool,
	modified bool) {

	log.T.F("%s AcceptReq pubkey %0x", remote, authedPubkey)
	s.Lock()
	defer s.Unlock()
	if s.PublicReadable() && len(s.Owners()) == 0 && !s.AuthRequired() {
		log.T.F("%s accept because public readable and auth not required", remote)
		allowed = ff
		ok = true
		return
	}
	allowed = ff
	// client is permitted, pass through the filter so request/count processing does
	// not need logic and can just use the returned filter.
	// check that the client is authed to a pubkey in the owner follow list
	if len(s.Owners()) > 0 {
		for pk := range s.Followed {
			if bytes.Equal(authedPubkey, []byte(pk)) {
				ok = true
				return
			}
		}
		// if the authed pubkey was not found, reject the request.
		return
	}
	// if auth is enabled and there is no moderators we just check that the pubkey
	// has been loaded via the auth function.
	ok = len(authedPubkey) == schnorr.PubKeyBytesLen
	return
}
