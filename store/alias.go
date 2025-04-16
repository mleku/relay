package store

import (
	"net/http"

	"relay.mleku.dev/envelopes/okenvelope"
	"relay.mleku.dev/subscription"
)

type SubID = subscription.Id
type Responder = http.ResponseWriter
type Req = *http.Request
type OK = okenvelope.T
