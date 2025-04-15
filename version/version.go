package version

import (
	_ "embed"
)

//go:embed version
var V string

var Description = "a simple, fast nostr relay"

var URL = "https://relay.mleku.dev"
