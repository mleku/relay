package interfaces

import (
	"net/http"
	"time"

	"relay.mleku.dev/context"
	"relay.mleku.dev/event"
	"relay.mleku.dev/filters"
	"relay.mleku.dev/store"
)

type Server interface {
	AcceptReq(c context.T, hr *http.Request, id []byte, ff *filters.T,
		authedPubkey []byte) (allowed *filters.T,
		ok bool, modified bool)
	AcceptEvent(
		c context.T, ev *event.T, hr *http.Request, origin string,
		authedPubkey []byte) (accept bool, notice string, afterSave func())
	AddEvent(
		c context.T, ev *event.T, hr *http.Request,
		origin string, authedPubkey []byte) (accepted bool,
		message []byte)
	AdminAuth(r *http.Request,
		tolerance ...time.Duration) (authed bool, pubkey []byte)
	AuthRequired() bool
	Configuration() store.Configuration
	Context() context.T
	Owners() [][]byte
	PublicReadable() bool
	SetConfiguration(*store.Configuration)
	Shutdown()
	Storage() store.I
	Lock()
	Unlock()
	ZeroLists()
	CheckOwnerLists(c context.T)
	OwnersFollowed(pubkey string) (ok bool)
}
