package main

import (
	"github.com/Shuttl-Tech/vault-plugin-postgres/backend"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/plugin"
	"log"
	"os"
)

func main() {
	apiClientMeta := &api.PluginAPIClientMeta{}
	_ = apiClientMeta.FlagSet().Parse(os.Args[1:])

	tlsConfig := apiClientMeta.GetTLSConfig()
	tlsProviderFunc := api.VaultPluginTLSProvider(tlsConfig)

	opts := &plugin.ServeOpts{
		BackendFactoryFunc: backend.Factory,
		TLSProviderFunc:    tlsProviderFunc,
	}

	err := plugin.Serve(opts)
	if err != nil {
		log.Fatalf("plugin exited with error: %s", err)
	}
}
