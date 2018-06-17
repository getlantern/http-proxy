#!/usr/bin/env bash

ip=$1

function cleanup() {
  echo "Starting http-proxy-lantern"
  ssh lantern@$ip -t "sudo service http-proxy start" || die "Could not start http-proxy"
}

function die() {
  echo "$@"
  cleanup
  exit 1
}

echo "Disabling auto-update on $ip"
ssh lantern@$ip -t "sudo crontab -l | perl -p -e 's/^(.*update_proxy.bash.*)/#\1/g' | sudo crontab -" || die "Could not disable auto-updates"

echo "Stopping http-proxy-lantern to allow upload"
ssh lantern@$ip -t "sudo service http-proxy stop"

echo "Uploading http-proxy-lantern"
scp dist/http-proxy lantern@$ip:http-proxy || die "Could not copy binary"

# This is necessary for http-proxy to run on restricted ports.
echo "Calling setcap on http-proxy"
ssh lantern@$ip -t "sudo setcap 'cap_net_bind_service=+ep' /home/lantern/http-proxy" || die "Error calling setcap on http-proxy"

cleanup
