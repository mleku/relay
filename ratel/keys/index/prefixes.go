package index

import (
	"relay.mleku.dev/ratel/keys"
)

type P byte

// Key writes a key with the P prefix byte and an arbitrary list of keys.Element.
func (p P) Key(element ...keys.Element) (b []byte) {
	b = keys.Write(
		append([]keys.Element{New(byte(p))}, element...)...)
	return
}

// B returns the index.P as a byte.
func (p P) B() byte { return byte(p) }

// I returns the index.P as an int (for use with the KeySizes.
func (p P) I() int { return int(p) }
