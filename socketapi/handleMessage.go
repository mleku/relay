package socketapi

import (
	"fmt"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/envelopes"
	"relay.mleku.dev/envelopes/authenvelope"
	"relay.mleku.dev/envelopes/closeenvelope"
	"relay.mleku.dev/envelopes/eventenvelope"
	"relay.mleku.dev/envelopes/noticeenvelope"
	"relay.mleku.dev/envelopes/reqenvelope"
	"relay.mleku.dev/log"
)

func (a *A) HandleMessage(msg []byte) {
	var notice []byte
	var err error
	var t string
	var rem []byte
	if t, rem, err = envelopes.Identify(msg); chk.E(err) {
		notice = []byte(err.Error())
	}
	switch t {
	case eventenvelope.L:
		notice = a.HandleEvent(a.Ctx, rem, a.Server)
	case reqenvelope.L:
		notice = a.HandleReq(a.Ctx, rem, a.Server)
	case closeenvelope.L:
		notice = a.HandleClose(rem, a.Server)
	case authenvelope.L:
		notice = a.HandleAuth(rem, a.Server)
	default:
		notice = []byte(fmt.Sprintf("unknown envelope type %s\n%s", t, rem))
	}
	if len(notice) > 0 {
		log.D.F("notice->%s %s", a.Listener.RealRemote(), notice)
		if err = noticeenvelope.NewFrom(notice).Write(a.Listener); err != nil {
			return
		}
	}

}
