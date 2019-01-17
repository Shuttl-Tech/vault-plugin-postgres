#!/usr/bin/env bash
set -e

. ./dev.sh

mkdir -p docs/

vault path-help pg-cluster/                     > docs/backend.md
vault path-help pg-cluster/info                 > docs/info.md
vault path-help pg-cluster/clone/name           > docs/clone.md
vault path-help pg-cluster/cluster/name         > docs/cluster.md
vault path-help pg-cluster/cluster/c/database   > docs/database.md
vault path-help pg-cluster/roles/name           > docs/roles.md      && echo -e "\n---\n" >> docs/roles.md
vault path-help pg-cluster/roles                >> docs/roles.md
vault path-help pg-cluster/creds/c/d/r          > docs/creds.md
