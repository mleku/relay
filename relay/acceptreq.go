package relay

import (
	"bytes"
	"net/http"

	"relay.mleku.dev/context"
	"relay.mleku.dev/ec/schnorr"
	"relay.mleku.dev/filters"
	"relay.mleku.dev/kinds"
)

func (s *Server) AcceptReq(c context.T, hr *http.Request, id []byte,
	ff *filters.T, authedPubkey []byte) (allowed *filters.T, ok bool, modified bool) {

	if s.PublicReadable() {
		allowed = ff
		ok = true
		return
	}
	// if client isn't authed but there are kinds in the filters that are
	// kind.Directory type then trim the filter down and only respond to the queries
	// that blanket should deliver events in order to facilitate non-authorized users
	// to interact with users, even just such as to see their profile metadata or
	// learn about deleted events.
	if len(authedPubkey) == 0 {
		for _, f := range ff.F {
			fk := f.Kinds.K
			allowedKinds := kinds.New()
			for _, fkk := range fk {
				if fkk.IsDirectoryEvent() || (!fkk.IsPrivileged() && s.PublicReadable()) {
					allowedKinds.K = append(allowedKinds.K, fkk)
				}
			}
			// if none of the kinds in the req are permitted, continue to the next filter.
			if len(allowedKinds.K) == 0 {
				continue
			}
			// if no filters have yet been added, initialize one
			if allowed == nil {
				allowed = &filters.T{}
			}
			// overwrite the kinds that have been permitted
			if len(f.Kinds.K) != len(allowedKinds.K) {
				modified = true
			}
			f.Kinds.K = allowedKinds.K
			allowed.F = append(allowed.F, f)
		}
		if allowed != nil {
			// request has been filtered and can be processed. note that the caller should
			// still send out an auth request after the filter has been processed.
			ok = true
			return
		}
	}
	// if the client hasn't authed, reject
	if len(authedPubkey) == 0 {
		return
	}
	allowed = ff
	// client is permitted, pass through the filter so request/count processing does
	// not need logic and can just use the returned filter.
	// check that the client is authed to a pubkey in the owner follow list
	s.Lock()
	defer s.Unlock()
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
