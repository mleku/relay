package event

import (
	"bytes"

	"relay.mleku.dev/errorf"
	"relay.mleku.dev/log"
	"relay.mleku.dev/p256k"
	"relay.mleku.dev/signer"

	"relay.mleku.dev/chk"
)

// Sign the event using the signer.I. Uses github.com/bitcoin-core/secp256k1 if available for
// much faster signatures.
//
// Note that this only populates the Pubkey, Id and Sig. The caller must set the CreatedAt
// timestamp as intended.
func (ev *T) Sign(keys signer.I) (err error) {
	ev.Pubkey = keys.Pub()
	ev.Id = ev.GetIDBytes()
	if ev.Sig, err = keys.Sign(ev.Id); chk.E(err) {
		return
	}
	return
}

// Verify an event is signed by the pubkey it contains. Uses github.com/bitcoin-core/secp256k1
// if available for faster verification.
func (ev *T) Verify() (valid bool, err error) {
	keys := p256k.Signer{}
	if err = keys.InitPub(ev.Pubkey); chk.E(err) {
		return
	}
	if valid, err = keys.Verify(ev.Id, ev.Sig); chk.T(err) {
		// check that this isn't because of a bogus Id
		id := ev.GetIDBytes()
		if !bytes.Equal(id, ev.Id) {
			log.E.Ln("event Id incorrect")
			ev.Id = id
			err = nil
			if valid, err = keys.Verify(ev.Id, ev.Sig); chk.E(err) {
				return
			}
			err = errorf.W("event Id incorrect but signature is valid on correct Id")
		}
		return
	}
	return
}
