package publisher

import (
	"relay.mleku.dev/event"
)

type Typer interface {
	// Type returns a type identifier string to allow multiple self-registering publisher.I to
	// be used with an abstraction to allow multiple APIs to publish.
	Type() string
}

type I interface {
	Typer
	// Deliver the event, accounting for whether auth is required and if the subscriber is
	// authed for protected privacy of privileged messages. if publicReadable, then auth is
	// required if set for writing.
	Deliver(authRequired, publicReadable bool, ev *event.T)
	// Receive accepts a new subscription request, using the Typer to match it to the
	// publisher.I that handles it.
	Receive(msg Typer)
}

type Publishers []I
