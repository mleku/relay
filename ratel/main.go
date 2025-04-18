// Package ratel is a badger DB based event store.
package ratel

import (
	"encoding/binary"
	"sync"

	"github.com/dgraph-io/badger/v4"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/context"
	"relay.mleku.dev/ratel/keys/serial"
	"relay.mleku.dev/ratel/prefixes"
	"relay.mleku.dev/store"
)

// DefaultMaxLimit is set to a size that means the usual biggest batch of events sent to a
// client usually is at most about 256kb or so.
const DefaultMaxLimit = 2048

// T is a badger event store database with layer2 and garbage collection.
type T struct {
	Ctx            context.T
	WG             *sync.WaitGroup
	dataDir        string
	HasL2          bool
	BlockCacheSize int
	InitLogLevel   int32
	Logger         *logger
	// DB is the badger db
	*badger.DB
	// seq is the monotonic collision free index for raw event storage.
	seq *badger.Sequence
	// Threads is how many CPU threads we dedicate to concurrent actions, flatten and GC mark
	Threads int
	// MaxLimit is a default limit that applies to a query without a limit, to avoid sending out
	// too many events to a client from a malformed or excessively broad filter.
	MaxLimit int
	// Flatten should be set to true to trigger a flatten at close... this is mainly triggered
	// by running an import
	Flatten bool
	// UseCompact uses a compact encoding based on the canonical format (generate hash of it to
	// get Id field with the signature in raw binary after.
	UseCompact bool
	// Compression sets the compression to use, none/snappy/zstd. If zstd compression is enabled
	// there is less benefit to UseCompact, and instead of having to re-marshal the event it can
	// be directly delivered from the form returned from the database.
	Compression string
}

var _ store.I = (*T)(nil)

// BackendParams is the configurations used in creating a new ratel.T.
type BackendParams struct {
	Ctx                                context.T
	WG                                 *sync.WaitGroup
	HasL2, UseCompact                  bool
	BlockCacheSize, LogLevel, MaxLimit int
	Compression                        string // none,snappy,zstd
	Extra                              []int
}

// New configures a a new ratel.T event store.
func New(p BackendParams) *T {
	return GetBackend(p.Ctx, p.WG, p.HasL2, p.UseCompact, p.BlockCacheSize, p.LogLevel,
		p.MaxLimit, p.Compression)
}

// GetBackend returns a reasonably configured badger.Backend.
//
// The variadic params correspond to DBSizeLimit, DBLowWater, DBHighWater and GCFrequency as an
// integer multiplier of number of seconds.
//
// Note that the cancel function for the context needs to be managed by the caller.
//
// Deprecated: use New instead.
func GetBackend(Ctx context.T, WG *sync.WaitGroup, hasL2, useCompact bool,
	blockCacheSize, logLevel, maxLimit int, compression string) (b *T) {
	// if unset, assume a safe maximum limit for unlimited filters.
	if maxLimit == 0 {
		maxLimit = DefaultMaxLimit
	}
	b = &T{
		Ctx:            Ctx,
		WG:             WG,
		HasL2:          hasL2,
		BlockCacheSize: blockCacheSize,
		InitLogLevel:   int32(logLevel),
		MaxLimit:       maxLimit,
		UseCompact:     useCompact,
		Compression:    compression,
	}
	return
}

// Path returns the path where the database files are stored.
func (r *T) Path() string { return r.dataDir }

// SerialKey returns a key used for storing events, and the raw serial counter bytes to copy
// into index keys.
func (r *T) SerialKey() (idx []byte, ser *serial.T) {
	var err error
	var s []byte
	if s, err = r.SerialBytes(); chk.E(err) {
		panic(err)
	}
	ser = serial.New(s)
	return prefixes.Event.Key(ser), ser
}

// Serial returns the next monotonic conflict free unique serial on the database.
func (r *T) Serial() (ser uint64, err error) {
	if ser, err = r.seq.Next(); chk.E(err) {
	}
	// log.T.ToSliceOfBytes("serial %x", ser)
	return
}

// SerialBytes returns a new serial value, used to store an event record with a conflict-free
// unique code (it is a monotonic, atomic, ascending counter).
func (r *T) SerialBytes() (ser []byte, err error) {
	var serU64 uint64
	if serU64, err = r.Serial(); chk.E(err) {
		panic(err)
	}
	ser = make([]byte, serial.Len)
	binary.BigEndian.PutUint64(ser, serU64)
	return
}
