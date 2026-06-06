package options

import (
	"github.com/spf13/pflag"
)

type Option struct {
	ServerPort   uint32
	ProviderType string
	ConfigFile   string
	AuthKey      string
}

func NewOption() *Option {
	return &Option{}
}

func (o *Option) AddFlags(flags *pflag.FlagSet) {
	flags.Uint32Var(&o.ServerPort, "server-port", 3000, "The server port binds to.")
	flags.StringVar(&o.ProviderType, "provider-type", "", "The provider type to use. If not specified, all routes will be enabled.")
	flags.StringVar(&o.ConfigFile, "config", "", "Path to the YAML config file for enabling specific providers. Overrides --provider-type when set.")
	flags.StringVar(&o.AuthKey, "auth", "", "API key for request authentication. Requests must include 'Authorization: Bearer <key>' or 'x-api-key: <key>' header.")
}
