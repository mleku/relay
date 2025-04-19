package socketapi

import (
	"relay.mleku.dev/auth"
	"relay.mleku.dev/chk"
	"relay.mleku.dev/envelopes/authenvelope"
	"relay.mleku.dev/envelopes/okenvelope"
	"relay.mleku.dev/log"
	"relay.mleku.dev/normalize"
	"relay.mleku.dev/relay/interfaces"
)

func (a *A) HandleAuth(req []byte, srv interfaces.Server, remote string) (msg []byte) {
	log.T.F("%s handling auth %s", remote, req)
	svcUrl := srv.ServiceURL(a.Listener.Req())
	if svcUrl == "" {
		return
	}
	log.T.F("received auth response,%s", req)
	var err error
	var rem []byte
	env := authenvelope.NewResponse()
	if rem, err = env.Unmarshal(req); chk.E(err) {
		return
	}
	if len(rem) > 0 {
		log.I.F("extra '%s'", rem)
	}
	var valid bool
	if valid, err = auth.Validate(env.Event, []byte(a.Listener.Challenge()),
		svcUrl); chk.E(err) {
		e := err.Error()
		if err = okenvelope.NewFrom(env.Event.Id, false,
			normalize.Error.F(err.Error())).Write(a.Listener); chk.E(err) {
			return []byte(err.Error())
		}
		return normalize.Error.F(e)
	} else if !valid {
		if err = okenvelope.NewFrom(env.Event.Id, false,
			normalize.Error.F("failed to authenticate")).Write(a.Listener); chk.E(err) {
			return []byte(err.Error())
		}
		return normalize.Restricted.F("auth response does not validate")
	} else {
		if err = okenvelope.NewFrom(env.Event.Id, true,
			[]byte{}).Write(a.Listener); chk.E(err) {
			return
		}
		log.D.F("%s authed to pubkey,%0x", a.Listener.RealRemote(), env.Event.Pubkey)
		a.Listener.SetAuthed(string(env.Event.Pubkey))
	}
	return
}
