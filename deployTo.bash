#!/usr/bin/env bash

function die() {
  echo "$@"
  exit 1
}

"${VERSION:?VERSION required}"

ip=$1

echo "Building http-proxy-lantern for $ip"
make dist || die "Could not make dist for http proxy"

./onlyDeployTo.bash $ip
