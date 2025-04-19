package relay

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"relay.mleku.dev/context"
	"relay.mleku.dev/event"
	"relay.mleku.dev/normalize"
	"relay.mleku.dev/publish"
	"relay.mleku.dev/store"

	"relay.mleku.dev/log"
)

var (
	NIP20prefixmatcher = regexp.MustCompile(`^\w+: `)
)

func (s *Server) addEvent(c context.T, ev *event.T,
	authedPubkey []byte, remote string) (accepted bool, message []byte) {

	if ev == nil {
		return false, normalize.Invalid.F("empty event")
	}
	// don't allow storing event with protected marker as per nip-70 with auth enabled.
	if (s.AuthRequired() || !s.PublicReadable()) && ev.Tags.ContainsProtectedMarker() {
		if len(authedPubkey) == 0 || !bytes.Equal(ev.Pubkey, authedPubkey) {
			return false,
				[]byte(fmt.Sprintf("event with relay marker tag '-' (nip-70 protected event) "+
					"may only be published by matching npub: %0x is not %0x",
					authedPubkey, ev.Pubkey))
		}
	}
	if ev.Kind.IsEphemeral() {
	} else {
		if saveErr := s.Publish(c, ev); saveErr != nil {
			if errors.Is(saveErr, store.ErrDupEvent) {
				return false, normalize.Error.F(saveErr.Error())
			}
			errmsg := saveErr.Error()
			if NIP20prefixmatcher.MatchString(errmsg) {
				if strings.Contains(errmsg, "tombstone") {
					return false, normalize.Blocked.F("event was deleted, not storing it again")
				}
				if strings.HasPrefix(errmsg, string(normalize.Blocked)) {
					return false, []byte(errmsg)
				}
				return false, normalize.Error.F(errmsg)
			} else {
				return false, normalize.Error.F("failed to save (%s)", errmsg)
			}
		}
	}
	var authRequired bool
	authRequired = s.AuthRequired()
	// notify subscribers
	publish.P.Deliver(authRequired, s.PublicReadable(), ev)
	accepted = true
	log.T.F("event id %0x stored", ev.Id)
	return
}
