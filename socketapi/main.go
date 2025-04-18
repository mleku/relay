package socketapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/fasthttp/websocket"

	"relay.mleku.dev/chk"
	"relay.mleku.dev/context"
	"relay.mleku.dev/envelopes/authenvelope"
	"relay.mleku.dev/log"
	"relay.mleku.dev/publish"
	"relay.mleku.dev/relay/helpers"
	"relay.mleku.dev/relay/interfaces"
	"relay.mleku.dev/servemux"
	"relay.mleku.dev/units"
	"relay.mleku.dev/ws"
)

const (
	DefaultWriteWait      = 10 * time.Second
	DefaultPongWait       = 60 * time.Second
	DefaultPingWait       = DefaultPongWait / 2
	DefaultMaxMessageSize = 1 * units.Mb
)

type A struct {
	Ctx      context.T
	Listener *ws.Listener
	interfaces.Server
}

func New(s interfaces.Server, path string, sm *servemux.S) {
	a := &A{Server: s}
	sm.Handle(path, a)
	return
}

func (a *A) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remote := helpers.GetRemoteFromReq(r)
	if !a.Server.Configured() {
		log.T.F("not configured %s", remote)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable),
			http.StatusServiceUnavailable)
		return
	}
	if r.Header.Get("Upgrade") != "websocket" && r.Header.Get("Accept") == "application/nostr+json" {
		log.T.F("serving relay info %s", remote)
		a.Server.HandleRelayInfo(w, r)
		return
	}
	if r.Header.Get("Upgrade") != "websocket" {
		// todo: we can put a website here
		http.Error(w, http.StatusText(http.StatusUpgradeRequired), http.StatusUpgradeRequired)
		return
	}
	var err error
	ticker := time.NewTicker(DefaultPingWait)
	var cancel context.F
	a.Ctx, cancel = context.Cancel(a.Server.Context())
	var conn *websocket.Conn
	if conn, err = Upgrader.Upgrade(w, r, nil); err != nil {
		log.E.F("%s failed to upgrade websocket: %v", remote, err)
		return
	}
	log.T.F("upgraded to websocket %s", remote)
	a.Listener = GetListener(conn, r)

	defer func() {
		log.D.F("%s closing connection", remote)
		cancel()
		ticker.Stop()
		publish.P.Receive(&W{
			Cancel:   true,
			Listener: a.Listener,
		})
		chk.E(a.Listener.Conn.Close())
	}()
	conn.SetReadLimit(DefaultMaxMessageSize)
	chk.E(conn.SetReadDeadline(time.Now().Add(DefaultPongWait)))
	conn.SetPongHandler(func(string) error {
		chk.E(conn.SetReadDeadline(time.Now().Add(DefaultPongWait)))
		return nil
	})
	if a.Server.AuthRequired() {
		log.T.F("requesting auth from %s", remote)
		a.Listener.RequestAuth()
	}
	if a.Server.AuthRequired() && a.Listener.AuthRequested() && len(a.Listener.Authed()) == 0 {
		log.T.F("requesting auth from client again from %s", a.Listener.RealRemote())
		if err = authenvelope.NewChallengeWith(a.Listener.Challenge()).Write(a.Listener); chk.E(err) {
			return
		}
		// return
	}
	go a.Pinger(a.Ctx, ticker, cancel, remote)
	var message []byte
	var typ int
	for {
		select {
		case <-a.Ctx.Done():
			log.I.F("%s closing connection", remote)
			a.Listener.Close()
			return
		default:
		}
		typ, message, err = conn.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
				websocket.CloseAbnormalClosure,
			) {
				log.W.F("unexpected close error from %s: %v",
					a.Listener.Request.Header.Get("X-Forwarded-For"), err)
			}
			return
		}
		if typ == websocket.PingMessage {
			log.T.F("pinging %s", remote)
			if err = a.Listener.WriteMessage(websocket.PongMessage, nil); chk.E(err) {
			}
			continue
		}
		go a.HandleMessage(message, remote)
	}
}
