package ratel

import (
	"github.com/dgraph-io/badger/v4"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/context"
	"relay.mleku.dev/errorf"
	"relay.mleku.dev/event"
	"relay.mleku.dev/eventid"
	"relay.mleku.dev/ratel/keys"
	"relay.mleku.dev/ratel/keys/createdat"
	"relay.mleku.dev/ratel/keys/id"
	"relay.mleku.dev/ratel/keys/index"
	"relay.mleku.dev/ratel/keys/serial"
	"relay.mleku.dev/ratel/prefixes"
	"relay.mleku.dev/sha256"
	eventstore "relay.mleku.dev/store"
	"relay.mleku.dev/timestamp"
)

func (r *T) SaveEvent(c context.T, ev *event.T) (err error) {
	if ev.Kind.IsEphemeral() {
		return
	}
	// make sure Close waits for this to complete
	r.WG.Add(1)
	defer r.WG.Done()
	// first, search to see if the event Id already exists.
	var foundSerial []byte
	var deleted bool
	seri := serial.New(nil)
	var ts []byte
	err = r.View(func(txn *badger.Txn) (err error) {
		// query event by id to ensure we don't try to save duplicates
		prf := prefixes.Id.Key(id.New(eventid.NewWith(ev.Id)))
		it := txn.NewIterator(badger.IteratorOptions{})
		defer it.Close()
		it.Seek(prf)
		if it.ValidForPrefix(prf) {
			var k []byte
			// get the serial
			k = it.Item().Key()
			// copy serial out
			keys.Read(k, index.Empty(), id.New(&eventid.T{}), seri)
			// save into foundSerial
			foundSerial = seri.Val
		}
		// if the event was deleted we don't want to save it again
		ts = prefixes.Tombstone.Key(id.New(eventid.NewWith(ev.Id)))
		it.Seek(ts)
		if it.ValidForPrefix(ts) {
			deleted = true
		}
		return
	})
	if chk.E(err) {
		return
	}
	if deleted {
		return errorf.W("tombstone found %0x, event will not be saved", ts)
	}
	if foundSerial != nil {
		err = r.Update(func(txn *badger.Txn) (err error) {
			// retrieve the event record
			evKey := keys.Write(index.New(prefixes.Event), seri)
			it := txn.NewIterator(badger.IteratorOptions{})
			defer it.Close()
			it.Seek(evKey)
			if it.ValidForPrefix(evKey) {
				if it.Item().ValueSize() != sha256.Size {
					// not a stub, we already have it
					return eventstore.ErrDupEvent
				}
				// we only need to restore the event binary and write the access counter key
				// encode to binary
				var bin []byte
				bin = r.Marshal(ev, bin)
				if err = txn.Set(it.Item().Key(), bin); chk.E(err) {
					return
				}
				// bump counter key
				counterKey := GetCounterKey(seri)
				val := keys.Write(createdat.New(timestamp.Now()))
				if err = txn.Set(counterKey, val); chk.E(err) {
					return
				}
				return
			}
			return
		})
		// if it was a dupe, we are done.
		if err != nil {
			return
		}
		return
	}
	var bin []byte
	bin = r.Marshal(ev, bin)
	// otherwise, save new event record.
	if err = r.Update(func(txn *badger.Txn) (err error) {
		var idx []byte
		var ser *serial.T
		idx, ser = r.SerialKey()
		if err = txn.Set(idx, bin); chk.E(err) {
			return
		}
		// 	add the indexes
		var indexKeys [][]byte
		indexKeys = GetIndexKeysForEvent(ev, ser)
		for _, k := range indexKeys {
			var val []byte
			if k[0] == prefixes.Counter.B() {
				val = keys.Write(createdat.New(timestamp.Now()))
			}
			if err = txn.Set(k, val); chk.E(err) {
				return
			}
		}
		return
	}); chk.E(err) {
		return
	}
	return
}

func (r *T) Sync() (err error) { return r.DB.Sync() }
