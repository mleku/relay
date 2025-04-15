package main

import (
	"relay.mleku.dev/config"
	"relay.mleku.dev/log"
	"relay.mleku.dev/version"
)

func main() {
	cfg := config.New()
	log.I.F("starting %s %s", cfg.AppName, version.V)

}
