package api

import (
	flag "github.com/spf13/pflag"
)

type APIConfig struct {
	Enable   bool     `koanf:"enable"`
	Endpoint string   `koanf:"endpoint"`
	API      []string `koanf:"allowed-apis"`
}

var APIConfigDefault = APIConfig{
	Enable:   false,
	Endpoint: "",
	API:      []string{"init", "update", "sign"},
}

func APIConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", APIConfigDefault.Enable, "Enable API server")
	f.String(prefix+".endpoint", APIConfigDefault.Endpoint, "API address")
	f.StringSlice(prefix+".allowed-apis", APIConfigDefault.API, "Allowed origins")
}
