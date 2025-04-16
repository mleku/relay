// Package tombstone is a 16 byte truncated event Id for keys.Element used to mark an event as
// being deleted so it isn't saved again.
package tombstone

import (
	"io"

	"relay.mleku.dev/eventid"
	"relay.mleku.dev/ratel/keys"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/log"
)

const Len = 16

type T struct {
	val []byte
}

var _ keys.Element = &T{}

func Make(eid *eventid.T) (v []byte) {
	v = make([]byte, Len)
	copy(v, eid.Bytes())
	return
}

func New() (t *T) { return new(T) }

func NewWith(eid *eventid.T) (t *T) {
	t = &T{val: Make(eid)}
	return
}

func (t *T) Write(buf io.Writer) {
	buf.Write(t.val)
}

func (t *T) Read(buf io.Reader) (el keys.Element) {
	b := make([]byte, Len)
	if n, err := buf.Read(b); chk.E(err) || n < Len {
		log.I.S(n, err)
		return nil
	}
	t.val = b
	return &T{val: b}
}

func (t *T) Len() int { return Len }
