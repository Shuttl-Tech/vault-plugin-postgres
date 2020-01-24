#!/usr/bin/env bash
set -e

. ./dev.sh

mkdir -p docs/

function fmt_header {
  sed -e 's/^\(Request\)/    &/' -e 's/^\(Matching Route\)/    &/'
}

vault path-help pg-cluster/                     | fmt_header > docs/index.md
vault path-help pg-cluster/info                 | fmt_header > docs/info.md
vault path-help pg-cluster/clone/name           | fmt_header > docs/clone.md
vault path-help pg-cluster/cluster              | fmt_header > docs/cluster.md
echo -e "\n---\n"                                            >> docs/cluster.md
vault path-help pg-cluster/cluster/name         | fmt_header >> docs/cluster.md
vault path-help pg-cluster/cluster/c/database   | fmt_header > docs/database.md
vault path-help pg-cluster/roles/name           | fmt_header > docs/roles.md
echo -e "\n---\n"                                            >> docs/roles.md
vault path-help pg-cluster/roles                | fmt_header >> docs/roles.md
vault path-help pg-cluster/creds/c/d/r          | fmt_header > docs/creds.md
vault path-help pg-cluster/metadata             | fmt_header > docs/metadata.md
vault path-help pg-cluster/gc/clusters          | fmt_header > docs/gc.md
echo -e "\n---\n"                                            >> docs/gc.md
vault path-help pg-cluster/gc/cluster/c         | fmt_header >> docs/gc.md
echo -e "\n---\n"                                            >> docs/gc.md
vault path-help pg-cluster/gc/cluster/c/d       | fmt_header >> docs/gc.md

declare -a toc

for docname in docs/*.md; do
  titlef=$(basename "${docname}")
  toc+=(" - [${titlef%.md}](./${titlef})")
done

for docf in docs/*.md; do
  echo -e "\n\n### TOC\n" >> "${docf}"
  printf "%s\n" "${toc[@]}" >> "${docf}"
done