#!/bin/sh
set -e

echo "[+] Setting up LANTERN_SERVERMASQ iptables chain..."

if [ -z "$PROXY_ADDR" ] || [ -z "$PROXY_PORT" ] || [ -z "$MASQ_ADDR" ]; then
  echo "[~] Required environment variables not set, skipping iptables setup"
  exec "$@"
fi

iptables -t nat -N LANTERN_SERVERMASQ 2>/dev/null || true
iptables -t nat -F LANTERN_SERVERMASQ 2>/dev/null || true

iptables -t nat -A LANTERN_SERVERMASQ -d "$PROXY_ADDR" ! --dport "$PROXY_PORT" -j DNAT --to-destination "$MASQ_ADDR"
iptables -t nat -A PREROUTING -d "$PROXY_ADDR" -j LANTERN_SERVERMASQ

echo "[+] LANTERN_SERVERMASQ setup complete: $@"
exec "$@"