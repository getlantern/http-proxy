#!/usr/bin/env bash

ip=$1

function die() {
  echo "$@"
  exit 1
}

echo "Enabling auto-update on $ip"
ssh lantern@$ip -t "sudo crontab -l | perl -p -e 's/^#(.*update_proxy.bash.*)/\1/g' | sudo crontab -" || die "Could not reenable auto-updates"
