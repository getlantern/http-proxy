#!/bin/sh
set -e

echo "[+] Setting up LANTERN_SERVERMASQ iptables chain..."

if [ -z "$PROXY_ADDR" ] || [ -z "$PROXY_PORT" ] || [ -z "$MASQ_ADDR" ]; then
  echo "[~] Required environment variables not set, skipping iptables setup"
  exec "$@"
fi

# The iptables rules can be expalined as follows:
# 1. Create a new chain called LANTERN_SERVERMASQ.
# 2. Add a rule to the LANTERN_SERVERMASQ chain that matches packets destined for the proxy address
#     (PROXY_ADDR) that are not destined for the proxy port (PROXY_PORT), and redirects them to the masqAddr.
#  3. Add a rule to the PREROUTING chain that matches packets destined for the proxy address (PROXY_ADDR)
#     and redirects them to the LANTERN_SERVERMASQ chain.

iptables -t nat -N LANTERN_SERVERMASQ 2>/dev/null || true
iptables -t nat -F LANTERN_SERVERMASQ 2>/dev/null || true

iptables -t nat -A LANTERN_SERVERMASQ -d "$PROXY_ADDR" -p tcp ! --dport "$PROXY_PORT" -j DNAT --to-destination "$MASQ_ADDR"
iptables -t nat -A PREROUTING -d "$PROXY_ADDR" -j LANTERN_SERVERMASQ

echo "[+] LANTERN_SERVERMASQ setup complete: $@"
exec su-exec lantern "$@"