#!/usr/bin/env bash

function die() {
  echo "$@"
  exit 1
}

"${VERSION:?VERSION required}"

ip=$1

echo "Building http-proxy-lantern"
make dist || "Could not make dist for http proxy"

./onlyDeployTo.bash $ip
