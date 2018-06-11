#!/usr/bin/env bash

function die() {
  echo "$@"
  exit 1
}

ip=$1

echo "Disabling auto-update on $ip"
ssh lantern@$ip -t "sudo crontab -l | perl -p -e 's/^(.*update_proxy.bash.*)/#\1/g' | sudo crontab -" || die "Could not disable auto-updates"

echo "Uploading http-proxy-lantern"
scp dist/http-proxy lantern@$ip:http-proxy || die "Could not copy binary"

echo "Restarting http-proxy-lantern"
ssh lantern@$ip -t "sudo service http-proxy restart" || die "Could not restart"
