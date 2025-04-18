package socketapi

import (
	"bytes"
	"errors"

	"github.com/dgraph-io/badger/v4"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/context"
	"relay.mleku.dev/envelopes/authenvelope"
	"relay.mleku.dev/envelopes/closedenvelope"
	"relay.mleku.dev/envelopes/eoseenvelope"
	"relay.mleku.dev/envelopes/eventenvelope"
	"relay.mleku.dev/envelopes/reqenvelope"
	"relay.mleku.dev/event"
	"relay.mleku.dev/filter"
	"relay.mleku.dev/hex"
	"relay.mleku.dev/kind"
	"relay.mleku.dev/kinds"
	"relay.mleku.dev/log"
	"relay.mleku.dev/normalize"
	"relay.mleku.dev/pointers"
	"relay.mleku.dev/publish"
	"relay.mleku.dev/relay/interfaces"
	"relay.mleku.dev/tag"
)

func (a *A) HandleReq(c context.T, req []byte, srv interfaces.Server,
	remote string) (r []byte) {

	log.T.F("%s handleReq %s", remote, req)
	sto := srv.Storage()
	var err error
	var rem []byte
	env := reqenvelope.New()
	if rem, err = env.Unmarshal(req); chk.E(err) {
		return normalize.Error.F(err.Error())
	}
	if len(rem) > 0 {
		log.I.F("%s extra '%s'", remote, rem)
	}
	allowed := env.Filters
	var accept, modified bool
	allowed, accept, modified = a.Server.AcceptReq(c, a.Listener.Req(),
		env.Subscription.T,
		env.Filters,
		[]byte(a.Listener.Authed()), remote)
	log.T.F("%s accept %v modified %v", remote, accept, modified)
	if !accept || allowed == nil || modified {
		if a.Server.AuthRequired() && !a.Listener.AuthRequested() {
			a.Listener.RequestAuth()
			if err = closedenvelope.NewFrom(env.Subscription,
				normalize.AuthRequired.F("auth required for request processing")).Write(a.Listener); chk.E(err) {
			}
			log.T.F("requesting auth from client from %s, challenge '%s'",
				a.Listener.RealRemote(), a.Listener.Challenge())
			if err = authenvelope.NewChallengeWith(a.Listener.Challenge()).Write(a.Listener); chk.E(err) {
				return
			}
			if !modified {
				return
			}
		}
	}
	if allowed != env.Filters {
		defer func() {
			if a.AuthRequired() &&
				!a.Listener.AuthRequested() {
				a.Listener.RequestAuth()
				if err = closedenvelope.NewFrom(env.Subscription,
					normalize.AuthRequired.F("auth required for request processing")).Write(a.Listener); chk.E(err) {
				}
				log.T.F("requesting auth from client from %s, challenge '%s'",
					a.Listener.RealRemote(),
					a.Listener.Challenge())
				if err = authenvelope.NewChallengeWith(a.Listener.Challenge()).Write(a.Listener); chk.E(err) {
					return
				}
				return
			}
		}()
	}
	if allowed == nil {
		return
	}
	for _, f := range allowed.F {
		var i uint
		if pointers.Present(f.Limit) {
			if *f.Limit == 0 {
				continue
			}
			i = *f.Limit
		}
		if a.Server.AuthRequired() {
			if f.Kinds.IsPrivileged() {
				log.T.F("privileged request\n%s", f.Serialize())
				senders := f.Authors
				receivers := f.Tags.GetAll(tag.New("#p"))
				switch {
				case len(a.Listener.Authed()) == 0:
					// a.RequestAuth()
					if err = closedenvelope.NewFrom(env.Subscription,
						normalize.AuthRequired.F("auth required for processing request due to presence of privileged kinds (DMs, app specific data)")).Write(a.Listener); chk.E(err) {
					}
					log.I.F("requesting auth from client from %s", a.Listener.RealRemote())
					if err = authenvelope.NewChallengeWith(a.Listener.Challenge()).Write(a.Listener); chk.E(err) {
						return
					}
					notice := normalize.Restricted.F("this realy does not serve DMs or Application Specific Data " +
						"to unauthenticated users or to npubs not found in the event tags or author fields, does your " +
						"client implement NIP-42?")
					return notice
				case senders.Contains(a.Listener.AuthedBytes()) ||
					receivers.ContainsAny([]byte("#p"), tag.New(a.Listener.AuthedBytes())):
					log.T.F("user %0x from %s allowed to query for privileged event",
						a.Listener.AuthedBytes(), a.Listener.RealRemote())
				default:
					return normalize.Restricted.F("authenticated user %0x does not have authorization for "+
						"requested filters", a.Listener.AuthedBytes())
				}
			}
		}
		var events event.Ts
		log.D.F("query from %s %0x,%s", a.Listener.RealRemote(), a.Listener.AuthedBytes(),
			f.Serialize())
		if events, err = sto.QueryEvents(c, f); err != nil {
			log.E.F("eventstore: %v", err)
			if errors.Is(err, badger.ErrDBClosed) {
				return
			}
			continue
		}
		aut := a.Listener.AuthedBytes()
		// remove events from muted authors if we have the authed user's mute list.
		if a.Listener.IsAuthed() {
			var mutes event.Ts
			if mutes, err = sto.QueryEvents(c, &filter.T{Authors: tag.New(aut),
				Kinds: kinds.New(kind.MuteList)}); !chk.E(err) {
				var mutePubs [][]byte
				for _, ev := range mutes {
					for _, t := range ev.Tags.ToSliceOfTags() {
						if bytes.Equal(t.Key(), []byte("p")) {
							var p []byte
							if p, err = hex.Dec(string(t.Value())); chk.E(err) {
								continue
							}
							mutePubs = append(mutePubs, p)
						}
					}
				}
				var tmp event.Ts
				for _, ev := range events {
					for _, pk := range mutePubs {
						if bytes.Equal(ev.Pubkey, pk) {
							continue
						}
						tmp = append(tmp, ev)
					}
				}
				// remove privileged events
				events = tmp
			}
		}
		// remove privileged events as they come through in scrape queries
		var tmp event.Ts
		for _, ev := range events {
			receivers := f.Tags.GetAll(tag.New("#p"))
			// if auth is required, kind is privileged and there is no authed pubkey, skip
			if srv.AuthRequired() && ev.Kind.IsPrivileged() && len(aut) == 0 {
				// log.I.ToSliceOfBytes("skipping event because event kind is %d and no auth", ev.Kind.K)
				if err = closedenvelope.NewFrom(env.Subscription,
					normalize.AuthRequired.F("auth required for processing request due to presence of privileged kinds (DMs, app specific data)")).Write(a.Listener); chk.E(err) {
				}
				log.I.F("requesting auth from client from %s", a.Listener.RealRemote())
				if err = authenvelope.NewChallengeWith(a.Listener.Challenge()).Write(a.Listener); chk.E(err) {
					return
				}
				notice := normalize.Restricted.F("this realy does not serve DMs or Application Specific Data " +
					"to unauthenticated users or to npubs not found in the event tags or author fields, does your " +
					"client implement NIP-42?")
				return notice
				continue
			}
			// if the authed pubkey is not present in the pubkey or p tags, skip
			if ev.Kind.IsPrivileged() && (!bytes.Equal(ev.Pubkey, aut) ||
				!receivers.ContainsAny([]byte("#p"), tag.New(a.Listener.AuthedBytes()))) {
				// log.I.ToSliceOfBytes("skipping event %0x because authed key %0x is in neither pubkey or p tag",
				// 	ev.Id, aut)
				if err = closedenvelope.NewFrom(env.Subscription,
					normalize.AuthRequired.F("auth required for processing request due to presence of privileged kinds (DMs, app specific data)")).Write(a.Listener); chk.E(err) {
				}
				log.I.F("requesting auth from client from %s", a.Listener.RealRemote())
				if err = authenvelope.NewChallengeWith(a.Listener.Challenge()).Write(a.Listener); chk.E(err) {
					return
				}
				notice := normalize.Restricted.F("this realy does not serve DMs or Application Specific Data " +
					"to unauthenticated users or to npubs not found in the event tags or author fields, does your " +
					"client implement NIP-42?")
				return notice
			}
			tmp = append(tmp, ev)
		}
		events = tmp
		// write out the events to the socket
		for _, ev := range events {
			i--
			if i < 0 {
				break
			}
			var res *eventenvelope.Result
			if res, err = eventenvelope.NewResultWith(env.Subscription.T,
				ev); chk.E(err) {
				return
			}
			if err = res.Write(a.Listener); chk.E(err) {
				return
			}
		}
	}
	if err = eoseenvelope.NewFrom(env.Subscription).Write(a.Listener); chk.E(err) {
		return
	}
	if env.Filters != allowed {
		return
	}
	receiver := make(event.C, 32)
	publish.P.Receive(&W{
		Listener: a.Listener,
		Id:       env.Subscription.String(),
		Receiver: receiver,
		Filters:  env.Filters,
	})
	return
}
