#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "$(readlink "$0")")" && pwd)"

echo "==> Starting dev"
echo "    --> Scratch dir"
echo "        Creating"

SCRATCH="$DIR/tmp"
mkdir -p "$SCRATCH/plugins"

echo "    --> Vault server"
echo "        Writing config"

tee "$SCRATCH/vault.hcl" > /dev/null <<EOF
plugin_directory = "$SCRATCH/plugins"
EOF

echo "    --> Configuring Shell Environment"
export VAULT_DEV_ROOT_TOKEN_ID="root"
export VAULT_ADDR="http://127.0.0.1:8200"

echo "    --> Starting Vault"
vault server \
  -dev \
  -log-level="debug" \
  -config="$SCRATCH/vault.hcl" \
  > "$SCRATCH/vault.log" 2>&1 &
sleep 3
VAULT_PID=$!

echo "    --> Starting PostgreSQL container"
docker run --rm \
           --publish 5432:5432 \
           --name vault-test-pg-cluster \
           --detach \
           -e POSTGRES_PASSWORD=secret  \
           -e POSTGRES_USER=super_admin \
           -e POSTGRES_DB=postgres \
           postgres:9.6.11 > /dev/null

function cleanup {
  echo ""
  echo "  ==> Cleaning up"
  kill -INT "$VAULT_PID"
  rm -rf "$SCRATCH"
  docker kill vault-test-pg-cluster > /dev/null
}
trap cleanup EXIT

echo "    --> Authenticating with vault"
vault login root &>/dev/null

echo "    --> Building plugin"
go build -o "$SCRATCH/plugins/vault-secrets-postgres-cluster"
SHASUM=$(shasum -a 256 "$SCRATCH/plugins/vault-secrets-postgres-cluster" | cut -d " " -f1)

echo "    --> Registering plugin"
vault write sys/plugins/catalog/secret/pg-cluster \
  sha_256="$SHASUM" \
  command="vault-secrets-postgres-cluster" | awk '{print "        " $0}'

echo "    --> Mounting plugin"
vault secrets enable -path=pg-cluster pg-cluster | awk '{print "        " $0}'

echo "    --> Reading out"
vault read pg-cluster/info | awk '{print "        " $0}'

echo "    --> Postgres is available:"
echo "        Port: 5432"
echo "        User: super_admin"
echo "        Password: secret"
echo "        Database: postgres"
echo ""
echo "    --> Vault is available:"
awk '/Unseal Key:|Root Token:/ { print "        " $0 }' "$SCRATCH/vault.log"
echo ""
echo "    --> See vault logs in $SCRATCH/vault.log"
echo "    --> See postgres logs with 'docker logs -f vault-test-pg-cluster'"
echo "    ==> Ready!"

# Only hold control if not being sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    wait $!
fi