// Package badgerbadger is a test of the layer 2 that uses two instances of the ratel event
// store, meant for testing the layer 2 protocol with two tiers of the database a size limited
// cache and a large non-purging store.
package badgerbadger

import (
	"sync"

	"relay.mleku.dev/context"
	"relay.mleku.dev/event"
	"relay.mleku.dev/eventid"
	"relay.mleku.dev/filter"
	"relay.mleku.dev/layer2"
	"relay.mleku.dev/ratel"
	"relay.mleku.dev/store"
)

// Backend is a hybrid badger/badger eventstore where L1 will have GC enabled and L2 will not.
// This is mainly for testing, as both are local.
type Backend struct {
	*layer2.Backend
}

var _ store.I = (*Backend)(nil)

// GetBackend returns a l2.Backend that combines two differently configured backends... the
// settings need to be configured in the ratel.T data structure before calling this.
func GetBackend(c context.T, wg *sync.WaitGroup, L1, L2 *ratel.T) (es store.I) {
	// log.I.S(L1, L2)
	es = &layer2.Backend{Ctx: c, WG: wg, L1: L1, L2: L2}
	return
}

// Init sets up the badger event store and connects to the configured IC canister.
//
// required params are address, canister Id and the badger event store size limit (which can be
// 0)
func (b *Backend) Init(path string) (err error) { return b.Backend.Init(path) }

// Close the connection to the database. IC is a request/response API authing at each request.
func (b *Backend) Close() (err error) { return b.Backend.Close() }

// DeleteEvent removes an event from the event store.
func (b *Backend) DeleteEvent(c context.T, eid *eventid.T, noTombstone ...bool) (err error) {
	return b.Backend.DeleteEvent(c, eid, noTombstone...)
}

// QueryEvents searches for events that match a filter and returns them asynchronously over a
// provided channel.
func (b *Backend) QueryEvents(c context.T, f *filter.T) (ch event.Ts, err error) {
	return b.Backend.QueryEvents(c, f)
}

// SaveEvent writes an event to the event store.
func (b *Backend) SaveEvent(c context.T, ev *event.T) (err error) {
	return b.Backend.SaveEvent(c, ev)
}
