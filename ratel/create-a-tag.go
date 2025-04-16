package ratel

import (
	"strings"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/ec/schnorr"
	"relay.mleku.dev/hex"
	"relay.mleku.dev/log"
	"relay.mleku.dev/ratel/keys"
	"relay.mleku.dev/ratel/keys/arb"
	"relay.mleku.dev/ratel/keys/createdat"
	"relay.mleku.dev/ratel/keys/index"
	"relay.mleku.dev/ratel/keys/kinder"
	"relay.mleku.dev/ratel/keys/pubkey"
	"relay.mleku.dev/ratel/keys/serial"
	"relay.mleku.dev/ratel/prefixes"
	"relay.mleku.dev/tag/atag"
)

// Create_a_Tag generates tag indexes from a tag key, tag value, created_at timestamp and the
// event serial.
func Create_a_Tag(tagKey, tagValue string, CA *createdat.T,
	ser *serial.T) (prf index.P, elems []keys.Element, err error) {

	var pkb []byte
	// first check if it might be a public key, fastest test
	if len(tagValue) == 2*schnorr.PubKeyBytesLen {
		// this could be a pubkey
		pkb, err = hex.Dec(tagValue)
		if err == nil {
			// it's a pubkey
			var pkk keys.Element
			if pkk, err = pubkey.NewFromBytes(pkb); chk.E(err) {
				return
			}
			prf, elems = prefixes.Tag32, keys.Make(pkk, ser)
			return
		} else {
			err = nil
		}
	}
	// check for `a` tag
	if tagKey == "a" && strings.Count(tagValue, ":") == 2 {
		a := &atag.T{}
		var rem []byte
		if rem, err = a.Unmarshal([]byte(tagValue)); chk.E(err) {
			return
		}
		if len(rem) > 0 {
			log.I.S("remainder", tagKey, tagValue, rem)
		}
		prf = prefixes.TagAddr
		var pk *pubkey.T
		if pk, err = pubkey.NewFromBytes(a.PubKey); chk.E(err) {
			return
		}
		elems = keys.Make(kinder.New(a.Kind.K), pk, arb.New(a.DTag), CA,
			ser)
		return
	}
	// store whatever as utf-8
	prf = prefixes.Tag
	elems = keys.Make(arb.New(tagValue), CA, ser)
	return
}
