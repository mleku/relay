package socketapi

import (
	"time"

	"github.com/fasthttp/websocket"

	"relay.mleku.dev/context"
	"relay.mleku.dev/log"
)

func (a *A) Pinger(ctx context.T, ticker *time.Ticker, cancel context.F) {
	defer func() {
		cancel()
		ticker.Stop()
		_ = a.Listener.Conn.Close()
	}()
	var err error
	for {
		select {
		case <-ticker.C:
			err = a.Listener.Conn.WriteControl(websocket.PingMessage, nil,
				time.Now().Add(DefaultPingWait))
			if err != nil {
				log.E.F("error writing ping: %v; closing websocket", err)
				return
			}
			a.Listener.RealRemote()
		case <-ctx.Done():
			return
		}
	}
}
