package relay

import (
	"relay.mleku.dev/bech32encoding"
	"relay.mleku.dev/chk"
	"relay.mleku.dev/hex"
	"relay.mleku.dev/log"
	"relay.mleku.dev/p256k"
	"relay.mleku.dev/signer"
	"relay.mleku.dev/store"
)

func (s *Server) UpdateConfiguration() (err error) {
	if c, ok := s.Store.(store.Configurationer); ok {
		var cfg *store.Configuration
		if cfg, err = c.GetConfiguration(); chk.E(err) {
			return
		}
		// first update the admins
		var administrators []signer.I
		log.I.S(cfg.Admins)
		for _, src := range cfg.Admins {
			if len(src) < 1 {
				continue
			}
			dst := make([]byte, len(src)/2)
			if _, err = hex.DecBytes(dst, []byte(src)); chk.E(err) {
				if dst, err = bech32encoding.NpubToBytes([]byte(src)); chk.E(err) {
					continue
				}
			}
			log.T.S(dst)
			sign := &p256k.Signer{}
			if err = sign.InitPub(dst); chk.E(err) {
				return
			}
			administrators = append(administrators, sign)
			log.I.F("administrator pubkey: %0x", sign.Pub())
		}
		log.I.F("administrators: %v", administrators)
		s.admins = administrators
	}
	return
}
