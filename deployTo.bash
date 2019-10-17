#!/usr/bin/env bash

function die() {
  echo "$@"
  exit 1
}

"${VERSION:?VERSION required}"

ip=$1

rm dist/*

echo "Building http-proxy-lantern for $ip"
make docker-distnochange || die "Could not make dist for http proxy"

./onlyDeployTo.bash $ip
