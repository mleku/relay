package socketapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/fasthttp/websocket"

	"relay.mleku.dev/api"
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

func New(s interfaces.Server, path string, sm *servemux.S) (handler *api.Handler) {
	a := &A{Server: s}
	sm.HandleFunc(path, a.ServeHTTP)
	handler = &api.Handler{Path: path, S: sm,
		Headers: api.Headers{
			// api.Header{Key: "Upgrade", Value: "websocket"},
			// api.Header{Key: "Connection", Value: "upgrade"},
		},
	}
	return
}

func (a *A) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !a.Server.Configured() {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable),
			http.StatusServiceUnavailable)
		return
	}
	remote := helpers.GetRemoteFromReq(r)
	for _, a := range a.Server.Configuration().BlockList {
		if strings.HasPrefix(remote, a) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
	}
	if r.Header.Get("Upgrade") != "websocket" {
		a.Server.HandleRelayInfo(w, r)
		return
	}
	var err error
	ticker := time.NewTicker(DefaultPingWait)
	var cancel context.F
	a.Ctx, cancel = context.Cancel(a.Server.Context())
	var conn *websocket.Conn
	if conn, err = Upgrader.Upgrade(w, r, nil); err != nil {
		log.E.F("%s failed to upgrade websocket: %v", a.Listener.RealRemote(), err)
		return
	}
	a.Listener = GetListener(conn, r)

	defer func() {
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
		a.Listener.RequestAuth()
	}
	if a.Server.AuthRequired() && a.Listener.AuthRequested() && len(a.Listener.Authed()) == 0 {
		log.I.F("requesting auth from client from %s", a.Listener.RealRemote())
		if err = authenvelope.NewChallengeWith(a.Listener.Challenge()).Write(a.Listener); chk.E(err) {
			return
		}
		// return
	}
	go a.Pinger(a.Ctx, ticker, cancel)
	var message []byte
	var typ int
	for {
		select {
		case <-a.Ctx.Done():
			a.Listener.Close()
			return
		case <-a.Context().Done():
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
			if err = a.Listener.WriteMessage(websocket.PongMessage, nil); chk.E(err) {
			}
			continue
		}
		go a.HandleMessage(message)
	}
}
