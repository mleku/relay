package main

import (
	"sync"

	"relay.mleku.dev/config"
	"relay.mleku.dev/context"
	"relay.mleku.dev/interrupt"
	"relay.mleku.dev/log"
	"relay.mleku.dev/lol"
	"relay.mleku.dev/ratel"
	"relay.mleku.dev/units"
	"relay.mleku.dev/version"
)

func main() {
	cfg := config.New()
	log.I.F("starting %s %s", cfg.AppName, version.V)
	var wg sync.WaitGroup
	c, cancel := context.Cancel(context.Bg())
	interrupt.AddHandler(func() { cancel() })
	storage := ratel.New(
		ratel.BackendParams{
			Ctx:            c,
			WG:             &wg,
			BlockCacheSize: units.Gb,
			LogLevel:       lol.GetLogLevel(cfg.LogLevel),
			MaxLimit:       ratel.DefaultMaxLimit,
			UseCompact:     false,
			Compression:    "zstd",
		},
	)
	_ = storage
}
