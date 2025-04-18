package socketapi

import (
	"bytes"
	"strings"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/context"
	"relay.mleku.dev/envelopes/authenvelope"
	"relay.mleku.dev/envelopes/eventenvelope"
	"relay.mleku.dev/envelopes/okenvelope"
	"relay.mleku.dev/event"
	"relay.mleku.dev/filter"
	"relay.mleku.dev/hex"
	"relay.mleku.dev/ints"
	"relay.mleku.dev/kind"
	"relay.mleku.dev/log"
	"relay.mleku.dev/normalize"
	"relay.mleku.dev/relay/interfaces"
	"relay.mleku.dev/sha256"
	"relay.mleku.dev/tag"
)

func (a *A) HandleEvent(c context.T, req []byte, srv interfaces.Server,
	remote string) (msg []byte) {

	log.T.F("%s handleEvent %s", remote, req)
	var err error
	var ok bool
	var rem []byte
	sto := srv.Storage()
	if sto == nil {
		panic("no event store has been set to store event")
	}
	env := eventenvelope.NewSubmission()
	if rem, err = env.Unmarshal(req); chk.E(err) {
		return
	}
	if len(rem) > 0 {
		log.T.F("%s extra '%s'", remote, rem)
	}
	accept, notice, after := a.Server.AcceptEvent(c, env.T, a.Listener.Req(),
		a.Listener.AuthedBytes(), remote)
	log.T.F("%s accepted %s %v", remote, accept)
	if !accept {
		if strings.Contains(notice, "mute") {
			if err = okenvelope.NewFrom(env.Id, false,
				normalize.Blocked.F(notice)).Write(a.Listener); chk.T(err) {
			}
		} else {
			if !a.Listener.AuthRequested() {
				a.Listener.RequestAuth()
				log.I.F("requesting auth from client %s", a.Listener.RealRemote())
				if err = authenvelope.NewChallengeWith(a.Listener.Challenge()).Write(a.Listener); chk.T(err) {
					return
				}
				if err = okenvelope.NewFrom(env.Id, false,
					normalize.AuthRequired.F("auth required for storing events")).Write(a.Listener); chk.T(err) {
				}
				return
			} else {
				log.I.F("requesting auth again from client %s", a.Listener.RealRemote())
				if err = authenvelope.NewChallengeWith(a.Listener.Challenge()).Write(a.Listener); chk.T(err) {
					return
				}
				if err = okenvelope.NewFrom(env.Id, false,
					normalize.AuthRequired.F("auth required for storing events")).Write(a.Listener); chk.T(err) {
				}
				return
			}
		}
		if err = okenvelope.NewFrom(env.Id, false,
			normalize.Invalid.F(notice)).Write(a.Listener); chk.T(err) {
		}
		return
	}
	if !bytes.Equal(env.GetIDBytes(), env.Id) {
		if err = okenvelope.NewFrom(env.Id, false,
			normalize.Invalid.F("event id is computed incorrectly")).Write(a.Listener); chk.E(err) {
			return
		}
		return
	}
	if ok, err = env.Verify(); chk.T(err) {
		if err = okenvelope.NewFrom(env.Id, false,
			normalize.Error.F("failed to verify signature")).Write(a.Listener); chk.E(err) {
			return
		}
	} else if !ok {
		if err = okenvelope.NewFrom(env.Id, false,
			normalize.Error.F("signature is invalid")).Write(a.Listener); chk.E(err) {
			return
		}
		return
	}
	if env.T.Kind.K == kind.Deletion.K {
		log.I.F("delete event\n%s", env.T.Serialize())
		for _, t := range env.Tags.ToSliceOfTags() {
			var res []*event.T
			if t.Len() >= 2 {
				switch {
				case bytes.Equal(t.Key(), []byte("e")):
					evId := make([]byte, sha256.Size)
					if _, err = hex.DecBytes(evId, t.Value()); chk.E(err) {
						continue
					}
					res, err = sto.QueryEvents(c, &filter.T{IDs: tag.New(evId)})
					if err != nil {
						if err = okenvelope.NewFrom(env.Id, false,
							normalize.Error.F("failed to query for target event")).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
					for i := range res {
						if res[i].Kind.Equal(kind.Deletion) {
							if err = okenvelope.NewFrom(env.Id, false,
								normalize.Blocked.F("not processing or storing delete event containing delete event references")).Write(a.Listener); chk.E(err) {
								return
							}
							return
						}
						if !bytes.Equal(res[i].Pubkey, env.T.Pubkey) {
							if err = okenvelope.NewFrom(env.Id, false,
								normalize.Blocked.F("cannot delete other users' events (delete by e tag)")).Write(a.Listener); chk.E(err) {
								return
							}
							return
						}
					}
				case bytes.Equal(t.Key(), []byte("a")):
					split := bytes.Split(t.Value(), []byte{':'})
					if len(split) != 3 {
						continue
					}
					var pk []byte
					if pk, err = hex.DecAppend(nil, split[1]); chk.E(err) {
						if err = okenvelope.NewFrom(env.Id, false,
							normalize.Invalid.F("delete event a tag pubkey value invalid: %s",
								t.Value())).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
					kin := ints.New(uint16(0))
					if _, err = kin.Unmarshal(split[0]); chk.E(err) {
						if err = okenvelope.NewFrom(env.Id, false,
							normalize.Invalid.F("delete event a tag kind value invalid: %s",
								t.Value())).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
					kk := kind.New(kin.Uint16())
					if kk.Equal(kind.Deletion) {
						if err = okenvelope.NewFrom(env.Id, false,
							normalize.Blocked.F("delete event kind may not be deleted")).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
					if !kk.IsParameterizedReplaceable() {
						if err = okenvelope.NewFrom(env.Id, false,
							normalize.Error.F("delete tags with a tags containing non-parameterized-replaceable events cannot be processed")).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
					if !bytes.Equal(pk, env.T.Pubkey) {
						log.I.S(pk, env.T.Pubkey, env.T)
						if err = okenvelope.NewFrom(env.Id, false,
							normalize.Blocked.F("cannot delete other users' events (delete by a tag)")).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
					f := filter.New()
					f.Kinds.K = []*kind.T{kk}
					// aut := make(by, 0, len(pk)/2)
					// if aut, err = hex.DecAppend(aut, pk); chk.E(err) {
					// 	return
					// }
					f.Authors.Append(pk)
					f.Tags.AppendTags(tag.New([]byte{'#', 'd'}, split[2]))
					res, err = sto.QueryEvents(c, f)
					if err != nil {
						if err = okenvelope.NewFrom(env.Id, false,
							normalize.Error.F("failed to query for target event")).Write(a.Listener); chk.E(err) {
							return
						}
						return
					}
				}
			}
			if len(res) < 1 {
				continue
			}
			var resTmp []*event.T
			for _, v := range res {
				if env.T.CreatedAt.U64() >= v.CreatedAt.U64() {
					resTmp = append(resTmp, v)
				}
			}
			res = resTmp
			for _, target := range res {
				if target.Kind.K == kind.Deletion.K {
					if err = okenvelope.NewFrom(env.Id, false,
						normalize.Error.F("cannot delete delete event %s",
							env.Id)).Write(a.Listener); chk.E(err) {
						return
					}
				}
				if target.CreatedAt.Int() > env.T.CreatedAt.Int() {
					log.I.F("not deleting\n%d%\nbecause delete event is older\n%d",
						target.CreatedAt.Int(), env.T.CreatedAt.Int())
					continue
				}
				if !bytes.Equal(target.Pubkey, env.Pubkey) {
					if err = okenvelope.NewFrom(env.Id, false,
						normalize.Error.F("only author can delete event")).Write(a.Listener); chk.E(err) {
						return
					}
					return
				}
				if err = sto.DeleteEvent(c, target.EventId()); chk.T(err) {
					if err = okenvelope.NewFrom(env.Id, false,
						normalize.Error.F(err.Error())).Write(a.Listener); chk.E(err) {
						return
					}
					return
				}
			}
			res = nil
		}
		if err = okenvelope.NewFrom(env.Id, true).Write(a.Listener); chk.E(err) {
			return
		}
	}
	var reason []byte
	ok, reason = srv.AddEvent(c, env.T, a.Listener.Req(), a.Listener.AuthedBytes(), remote)
	log.T.F("event added %v", ok)
	if err = okenvelope.NewFrom(env.Id, ok, reason).Write(a.Listener); chk.E(err) {
		return
	}
	if after != nil {
		after()
	}
	return
}
