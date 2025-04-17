package main

import (
	"errors"
	"net"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/adrg/xdg"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/config"
	"relay.mleku.dev/context"
	"relay.mleku.dev/interrupt"
	"relay.mleku.dev/log"
	"relay.mleku.dev/lol"
	"relay.mleku.dev/ratel"
	"relay.mleku.dev/relay"
	"relay.mleku.dev/store"
	"relay.mleku.dev/units"
	"relay.mleku.dev/version"
)

func main() {
	cfg := config.New()
	log.I.F("starting %s %s", cfg.AppName, version.V)
	wg := &sync.WaitGroup{}
	c, cancel := context.Cancel(context.Bg())
	interrupt.AddHandler(func() { cancel() })
	storage := ratel.New(
		ratel.BackendParams{
			Ctx:            c,
			WG:             wg,
			BlockCacheSize: 250 * units.Mb,
			LogLevel:       lol.Info,
			MaxLimit:       ratel.DefaultMaxLimit,
			UseCompact:     false,
			Compression:    "zstd",
		},
	)
	var err error
	if err = storage.Init(filepath.Join(xdg.DataHome, cfg.AppName)); chk.E(err) {
		os.Exit(1)
	}
	s := relay.Server{
		Ctx:             c,
		Cancel:          cancel,
		WG:              wg,
		Address:         net.JoinHostPort(cfg.Listen, strconv.Itoa(cfg.Port)),
		ConfigurationMx: &sync.Mutex{},
		Configuration:   &store.Configuration{},
		Store:           storage,
	}
	interrupt.AddHandler(func() { s.Shutdown() })
	if err = s.Start(); err != nil {
		if errors.Is(err, httputil.ErrClosed) {
			os.Exit(0)
		}
		os.Exit(1)
	}
}
