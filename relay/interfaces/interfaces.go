package interfaces

import (
	"net/http"
	"time"

	"relay.mleku.dev/context"
	"relay.mleku.dev/event"
	"relay.mleku.dev/filters"
	"relay.mleku.dev/relay/config"
	"relay.mleku.dev/store"
)

type Server interface {
	AcceptEvent(c context.T, ev *event.T, hr *http.Request,
		authedPubkey []byte, remote string) (accept bool, notice string, afterSave func())
	AcceptReq(c context.T, hr *http.Request, id []byte, ff *filters.T,
		authedPubkey []byte, remote string) (allowed *filters.T, ok bool, modified bool)
	AddEvent(c context.T, ev *event.T, hr *http.Request,
		authedPubkey []byte, remote string) (accepted bool, message []byte)
	AdminAuth(r *http.Request, remote string, tolerance ...time.Duration) (authed bool,
		pubkey []byte)
	AuthRequired() bool
	CheckOwnerLists(c context.T)
	Configuration() config.C
	Configured() bool
	Context() context.T
	HandleRelayInfo(w http.ResponseWriter, r *http.Request)
	Lock()
	Owners() [][]byte
	OwnersFollowed(pubkey string) (ok bool)
	PublicReadable() bool
	ServiceURL(req *http.Request) (s string)
	SetConfiguration(*config.C)
	UpdateConfiguration() (err error)
	Shutdown()
	Storage() store.I
	Unlock()
	ZeroLists()
}
