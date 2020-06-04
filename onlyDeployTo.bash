#!/usr/bin/env bash

ip=$1

function cleanup() {
  echo "$@"
  echo "Starting http-proxy-lantern"
  ssh lantern@$ip -t "sudo service http-proxy start" || die "Could not start http-proxy"
  exit 1
}

function die() {
  echo "$@"
  exit 1
}

if [ $# -ne "1" ]
then
    die "$0: Received $# args... IP required"
fi

echo "Disabling auto-update on $ip"
ssh lantern@$ip -t "sudo crontab -l | perl -p -e 's/^(.*update_proxy.bash.*)/#\1/g' | sudo crontab -" || die "Could not disable auto-updates"

echo "Uploading http-proxy-lantern"
scp dist/http-proxy lantern@$ip:http-proxy.tmp || die "Could not copy binary"

echo "Stopping http-proxy-lantern to allow replacing binary"
ssh lantern@$ip -t "sudo service http-proxy stop" 

echo "Replacing binary"
ssh lantern@$ip -t "sudo cp /home/lantern/http-proxy.tmp /home/lantern/http-proxy" || cleanup "Could not replace binary"

# This is necessary for http-proxy to run on restricted ports.
echo "Calling setcap on http-proxy"
ssh lantern@$ip -t "sudo setcap 'cap_net_raw+eip cap_net_admin+eip cap_net_bind_service+ep' /home/lantern/http-proxy" || cleanup "Error calling setcap on http-proxy"

cleanup
