# PostgreSQL credentials plugin for vault

This repository contains the source code for vault plugin used to manage
fleets of Postgres clusters.  
The default Postgres plugin shipped with vault falls short when it comes
to provisioning credentials
for hundreds of different databases across tens of different clusters.
This plugin was written to address
following particular shortcomings of the default Postgres secret engine:

 1. Database is the only entity available. To work with several databases
    in same cluster you'll need to configure identical connections.
 1. Automation requires the administrative credentials to be accessible by 
    automation script
 1. Some bootstrapping work on database server is required before the
    connection can be configured on vault, this calls for a human or script
    with access to administrative credentials.
    
## Setup

The setup guide assumes some familiarity with Vault and Vault's plugin
ecosystem. You must have a Vault server already running, unsealed, and
authenticated.

1. Download and decompress the latest plugin binary from the Releases tab on
GitHub. Alternatively you can compile the plugin from source.

1. Move the compiled plugin into Vault's configured `plugin_directory`:

  ```sh
  $ mv vault-secret-plugin-postgres /etc/vault/plugins/vault-secret-plugin-postgres
  ```

1. Calculate the SHA256 of the plugin and register it in Vault's plugin catalog.
If you are downloading the pre-compiled binary, it is highly recommended that
you use the published checksums to verify integrity.

  ```sh
  $ export SHA256=$(shasum -a 256 "/etc/vault/plugins/vault-secret-plugin-postgres" | cut -d' ' -f1)

  $ vault write sys/plugins/catalog/secret-postgres \
      sha_256="${SHA256}" \
      command="vault-secret-plugin-postgres"
  ```

1. Mount the auth method:

  ```sh
  $ vault secrets enable -path=database secret-postgres
  ```
  
## API

For details on API endpoints and their usage see [docs](./docs).

## Contributing

All reasonable pull requests are welcome. Before you start working on things it is
a good idea to first search through existing pull requests and open issue to
make sure your work don't clash with another contributor. If you are unsure of
anything please feel free to open an issue and start the discussion.

## License

This code is licensed under the MPLv2 license.
