package relay

import (
	"bytes"
	"fmt"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/context"
	"relay.mleku.dev/ec/schnorr"
	"relay.mleku.dev/event"
	"relay.mleku.dev/hex"
	"relay.mleku.dev/kind"
	"relay.mleku.dev/log"
	"relay.mleku.dev/tag"
	"relay.mleku.dev/tag/atag"
)

func (s *Server) acceptEvent(c context.T, evt *event.T, remote string,
	authedPubkey []byte) (accept bool, notice string, afterSave func()) {
	// if the authenticator is enabled we require auth to accept events
	if !s.AuthRequired() && len(s.owners) < 1 {
		return true, "", nil
	}
	if len(authedPubkey) != 32 && !s.PublicReadable() {
		return false, fmt.Sprintf("client not authed with auth required %s", remote), nil
	}
	if len(s.owners) > 0 {
		s.Lock()
		defer s.Unlock()
		if evt.Kind.Equal(kind.FollowList) || evt.Kind.Equal(kind.MuteList) {
			// if owner or any of their follows lists are updated we need to regenerate the
			// list this ensures that immediately a follow changes their list that newly
			// followed can access the relay and upload DM events and such for owner
			// followed users.
			for o := range s.ownersFollowed {
				if bytes.Equal([]byte(o), evt.Pubkey) {
					return true, "", func() {
						s.ZeroLists()
						s.CheckOwnerLists(context.Bg())
					}
				}
			}
		}
		for _, o := range s.owners {
			log.T.F("%0x,%0x", o, evt.Pubkey)
			if bytes.Equal(o, evt.Pubkey) {
				// prevent owners from deleting their own mute/follow lists in case of bad
				// client implementation
				if evt.Kind.Equal(kind.Deletion) {
					// check all a tags present are not follow/mute lists of the owners
					aTags := evt.Tags.GetAll(tag.New("a"))
					for _, at := range aTags.ToSliceOfTags() {
						a := &atag.T{}
						var rem []byte
						var err error
						if rem, err = a.Unmarshal(at.Value()); chk.E(err) {
							continue
						}
						if len(rem) > 0 {
							log.I.S("remainder", evt, rem)
						}
						if a.Kind.Equal(kind.Deletion) {
							// we don't delete delete events, period
							return false, "delete event kind may not be deleted", nil
						}
						// if the kind is not parameterised replaceable, the tag is invalid and the
						// delete event will not be saved.
						if !a.Kind.IsParameterizedReplaceable() {
							return false, "delete tags with a tags containing " +
								"non-parameterized-replaceable events cannot be processed", nil
						}
						for _, own := range s.owners {
							// don't allow owners to delete their mute or follow lists because
							// they should not want to, can simply replace it, and malicious
							// clients may do this specifically to attack the owner's relay (s)
							if bytes.Equal(own, a.PubKey) ||
								a.Kind.Equal(kind.MuteList) ||
								a.Kind.Equal(kind.FollowList) {
								return false, "owners may not delete their own " +
									"mute or follow lists, they can be replaced", nil
							}
						}
					}
					log.W.Ln("event is from owner")
					return true, "", nil
				}
			}
			// check the mute list, and reject events authored by muted pubkeys, even if
			// they come from a pubkey that is on the follow list.
			for pk := range s.Muted {
				if bytes.Equal(evt.Pubkey, []byte(pk)) {
					return false, "rejecting event with pubkey " + hex.Enc(evt.Pubkey) +
						" because on owner mute list", nil
				}
			}
			// for all else, check the authed pubkey is in the follow list
			for pk := range s.Followed {
				// allow all events from follows of owners
				if bytes.Equal(authedPubkey, []byte(pk)) {
					log.I.F("accepting event %0x because %0x on owner follow list",
						evt.Id, []byte(pk))
					return true, "", nil
				}
			}
		}
	}
	// if auth is enabled and there is no moderators we just check that the pubkey
	// has been loaded via the auth function.
	if len(authedPubkey) == schnorr.PubKeyBytesLen && s.AuthRequired() {
		notice = "auth required but user not authed"
	}
	return
}
