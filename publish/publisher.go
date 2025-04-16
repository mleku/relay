// Package publisher is a top level router for publishing to registered publishers.
package publish

import (
	"relay.mleku.dev/event"
	"relay.mleku.dev/publish/publisher"
	"relay.mleku.dev/typer"
)

var registry publisher.Publishers

func Register(p publisher.I) {
	registry = append(registry, p)
}

// S is the control structure for the subscription management scheme.
type S struct{ publisher.Publishers }

var _ publisher.I = &S{}

// New creates a new publish.S using the registered publisher.Publishers that have added
// themselves.
func New() (s *S) { return &S{Publishers: registry} }

func (s *S) Type() string { return "publish" }

func (s *S) Deliver(authRequired, publicReadable bool, ev *event.T) {
	for _, p := range s.Publishers {
		p.Deliver(authRequired, publicReadable, ev)
		return
	}
}

func (s *S) Receive(msg typer.T) {
	t := msg.Type()
	for _, p := range s.Publishers {
		if p.Type() == t {
			p.Receive(msg)
			return
		}
	}
}
