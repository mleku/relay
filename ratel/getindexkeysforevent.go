package ratel

import (
	"bytes"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/event"
	"relay.mleku.dev/eventid"
	"relay.mleku.dev/log"
	"relay.mleku.dev/ratel/keys"
	"relay.mleku.dev/ratel/keys/createdat"
	"relay.mleku.dev/ratel/keys/fullid"
	"relay.mleku.dev/ratel/keys/fullpubkey"
	"relay.mleku.dev/ratel/keys/id"
	"relay.mleku.dev/ratel/keys/index"
	"relay.mleku.dev/ratel/keys/kinder"
	"relay.mleku.dev/ratel/keys/pubkey"
	"relay.mleku.dev/ratel/keys/serial"
	"relay.mleku.dev/ratel/prefixes"
	"relay.mleku.dev/tag"
)

// GetIndexKeysForEvent generates all the index keys required to filter for events. evtSerial
// should be the output of Serial() which gets a unique, monotonic counter value for each new
// event.
func GetIndexKeysForEvent(ev *event.T, ser *serial.T) (keyz [][]byte) {

	var err error
	keyz = make([][]byte, 0, 18)
	ID := id.New(eventid.NewWith(ev.Id))
	CA := createdat.New(ev.CreatedAt)
	K := kinder.New(ev.Kind.ToU16())
	PK, _ := pubkey.New(ev.Pubkey)
	FID := fullid.New(eventid.NewWith(ev.Id))
	FPK := fullpubkey.New(ev.Pubkey)
	// indexes
	{ // ~ by id
		k := prefixes.Id.Key(ID, ser)
		keyz = append(keyz, k)
	}
	{ // ~ by pubkey+date
		k := prefixes.Pubkey.Key(PK, CA, ser)
		keyz = append(keyz, k)
	}
	{ // ~ by kind+date
		k := prefixes.Kind.Key(K, CA, ser)
		keyz = append(keyz, k)
	}
	{ // ~ by pubkey+kind+date
		k := prefixes.PubkeyKind.Key(PK, K, CA, ser)
		keyz = append(keyz, k)
	}
	// ~ by tag value + date
	for i, t := range ev.Tags.ToSliceOfTags() {
		// there is no value field
		if t.Len() < 2 ||
			// the tag is not a-zA-Z probably (this would permit arbitrary other single byte
			// chars)
			len(t.ToSliceOfBytes()[0]) != 1 ||
			// the second field is zero length
			len(t.ToSliceOfBytes()[1]) == 0 ||
			// the second field is more than 100 characters long
			len(t.ToSliceOfBytes()[1]) > 100 {
			// any of the above is true then the tag is not indexable
			continue
		}
		var firstIndex int
		var tt *tag.T
		for firstIndex, tt = range ev.Tags.ToSliceOfTags() {
			if tt.Len() >= 2 && bytes.Equal(tt.B(1), t.B(1)) {
				break
			}
		}
		if firstIndex != i {
			// duplicate
			continue
		}
		// get key prefix (with full length) and offset where to write the last parts
		prf, elems := index.P(0), []keys.Element(nil)
		if prf, elems, err = Create_a_Tag(string(t.ToSliceOfBytes()[0]),
			string(t.ToSliceOfBytes()[1]), CA,
			ser); chk.E(err) {
			log.I.F("%v", t.ToStringSlice())
			return
		}
		k := prf.Key(elems...)
		keyz = append(keyz, k)
	}
	{ // ~ by date only
		k := prefixes.CreatedAt.Key(CA, ser)
		keyz = append(keyz, k)
	}
	{ // Counter index - for storing last access time of events.
		k := GetCounterKey(ser)
		keyz = append(keyz, k)
	}
	{ // - full Id index - enabling retrieving the event Id without unmarshalling the data
		k := prefixes.FullIndex.Key(ser, FID, FPK, CA)
		keyz = append(keyz, k)
	}
	return
}
