// Package createdat implements a badger key index keys.Element for timestamps.
package createdat

import (
	"encoding/binary"
	"io"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/errorf"
	"relay.mleku.dev/ratel/keys"
	serial2 "relay.mleku.dev/ratel/keys/serial"
	"relay.mleku.dev/timestamp"
)

const Len = 8

type T struct {
	Val *timestamp.T
}

var _ keys.Element = &T{}

func New(c *timestamp.T) (p *T) { return &T{Val: c} }

func (c *T) Write(buf io.Writer) { buf.Write(c.Val.Bytes()) }

func (c *T) Read(buf io.Reader) (el keys.Element) {
	b := make([]byte, Len)
	if n, err := buf.Read(b); chk.E(err) || n != Len {
		return nil
	}
	c.Val = timestamp.FromUnix(int64(binary.BigEndian.Uint64(b)))
	return c
}

func (c *T) Len() int { return Len }

// FromKey expects to find a datestamp in the 8 bytes before a serial in a key.
func FromKey(k []byte) (p *T) {
	if len(k) < Len+serial2.Len {
		err := errorf.F("cannot get a serial without at least %d bytes", Len+serial2.Len)
		panic(err)
	}
	key := make([]byte, 0, Len)
	key = append(key, k[len(k)-Len-serial2.Len:len(k)-serial2.Len]...)
	return &T{Val: timestamp.FromBytes(key)}
}
