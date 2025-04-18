package openapi

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"relay.mleku.dev/bech32encoding"
	"relay.mleku.dev/chk"
	"relay.mleku.dev/context"
	"relay.mleku.dev/hex"
	"relay.mleku.dev/log"
	"relay.mleku.dev/relay/config"
	"relay.mleku.dev/relay/helpers"
)

// ConfigurationSetInput is the parameters for HTTP API method to set Configuration.
type ConfigurationSetInput struct {
	Auth string    `header:"Authorization" doc:"nostr nip-98 (and expiring variant)" required:"true"`
	Body *config.C `doc:"the new configuration"`
}

// ConfigurationGetInput is the parameters for HTTP API method to get Configuration.
type ConfigurationGetInput struct {
	Auth   string `header:"Authorization" doc:"nostr nip-98 (and expiring variant)" required:"true"`
	Accept string `header:"Accept" default:"application/json" enum:"application/json" required:"true"`
}

// ConfigurationGetOutput is the result of getting Configuration.
type ConfigurationGetOutput struct {
	Body config.C `doc:"the current configuration"`
}

// RegisterConfigurationSet implements the HTTP API for setting Configuration.
func (x *Operations) RegisterConfigurationSet(api huma.API) {
	name := "ConfigurationSet"
	description := "Set the configuration"
	path := x.path + "/configuration/set"
	scopes := []string{"admin", "write"}
	method := http.MethodPost
	huma.Register(api, huma.Operation{
		OperationID: name,
		Summary:     name,
		Path:        path,
		Method:      method,
		Tags:        []string{"admin"},
		Description: helpers.GenerateDescription(description, scopes),
		Security:    []map[string][]string{{"auth": scopes}},
	}, func(ctx context.T, input *ConfigurationSetInput) (wgh *struct{}, err error) {
		r := ctx.Value("http-request").(*http.Request)
		remote := helpers.GetRemoteFromReq(r)
		authed, _ := x.AdminAuth(r, remote)
		if !authed {
			log.I.F("checking first time password %s %s %v",
				input.Auth, x.Configuration().FirstTime,
				input.Auth != x.Configuration().FirstTime)
			if input.Auth != x.Configuration().FirstTime {
				err = huma.Error401Unauthorized("authorization required")
				return
			} else {
				var found bool
				for _, a := range input.Body.Admins {
					if len(a) < 1 {
						continue
					}
					dst := make([]byte, len(a)/2)
					if _, err = hex.DecBytes(dst, []byte(a)); chk.E(err) {
						if dst, err = bech32encoding.NpubToBytes([]byte(a)); chk.E(err) {
							continue
						}
					}
					log.T.S(dst)
					found = true
				}
				if !found {
					err = huma.Error401Unauthorized("at least one valid admin pubkey must be set")
					return
				}
			}
		}
		x.SetConfiguration(input.Body)
		return
	})
}

// RegisterConfigurationGet implements the HTTP API for getting the Configuration.
func (x *Operations) RegisterConfigurationGet(api huma.API) {
	name := "ConfigurationGet"
	description := "Fetch the current configuration"
	path := x.path + "/configuration/get"
	scopes := []string{"admin", "read"}
	method := http.MethodGet
	huma.Register(api, huma.Operation{
		OperationID: name,
		Summary:     name,
		Path:        path,
		Method:      method,
		Tags:        []string{"admin"},
		Description: helpers.GenerateDescription(description, scopes),
		Security:    []map[string][]string{{"auth": scopes}},
	}, func(ctx context.T, input *ConfigurationGetInput) (output *ConfigurationGetOutput,
		err error) {
		if !x.Server.Configured() {
			err = huma.Error404NotFound("server is not configured")
			return
		}
		r := ctx.Value("http-request").(*http.Request)
		remote := helpers.GetRemoteFromReq(r)
		authed, _ := x.AdminAuth(r, remote)
		if !authed {
			err = huma.Error401Unauthorized("authorization required")
			return
		}
		output = &ConfigurationGetOutput{Body: x.Configuration()}
		// }
		return
	})
}
