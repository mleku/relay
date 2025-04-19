package socketapi

import (
	"relay.mleku.dev/chk"
	"relay.mleku.dev/envelopes/closeenvelope"
	"relay.mleku.dev/log"
	"relay.mleku.dev/publish"
	"relay.mleku.dev/relay/interfaces"
)

func (a *A) HandleClose(req []byte,
	srv interfaces.Server, remote string) (note []byte) {
	var err error
	var rem []byte
	env := closeenvelope.New()
	if rem, err = env.Unmarshal(req); chk.E(err) {
		return []byte(err.Error())
	}
	if len(rem) > 0 {
		log.T.F("%s extra '%s'", remote, rem)
	}
	if env.ID.String() == "" {
		log.T.F("%s close has no <id>", remote)
		return []byte("CLOSE has no <id>")
	}
	log.T.F("%s cancelling subscription %s", env.ID.String())
	publish.P.Receive(&W{
		Cancel:   true,
		Listener: a.Listener,
		Id:       env.ID.String(),
	})
	return
}
