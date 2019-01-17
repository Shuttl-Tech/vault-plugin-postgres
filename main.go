package main

import (
	"github.com/Shuttl-Tech/vault-plugin-postgres-cluster/backend"
	"github.com/hashicorp/vault/helper/pluginutil"
	"github.com/hashicorp/vault/logical/plugin"
	"log"
	"os"
)

func main() {
	apiClientMeta := &pluginutil.APIClientMeta{}
	_ = apiClientMeta.FlagSet().Parse(os.Args[1:])

	tlsConfig := apiClientMeta.GetTLSConfig()
	tlsProviderFunc := pluginutil.VaultPluginTLSProvider(tlsConfig)

	opts := &plugin.ServeOpts{
		BackendFactoryFunc: backend.Factory,
		TLSProviderFunc:    tlsProviderFunc,
	}

	err := plugin.Serve(opts)
	if err != nil {
		log.Fatal(err)
	}
}
