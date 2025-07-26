#!/bin/sh
set -e

echo "[+] Setting up LANTERN_SERVERMASQ iptables chain..."

PROXY_ADDR=$(hostname -i | awk '{print $1}')

if [ -z "$PROXY_ADDR" ] || [ -z "$PROXY_PORT" ] || [ -z "$MASQ_ADDR" ]; then
  echo "[~] Required environment variables not set, skipping iptables setup"
  exec "$@"
fi

# The iptables rules can be expalined as follows:
# 1. Create a new chain called LANTERN_SERVERMASQ.
# 2. Add a rule to the LANTERN_SERVERMASQ chain that matches packets destined for the proxy address
#     (PROXY_ADDR) that are not destined for the proxy port (PROXY_PORT), and redirects them to the MASQ_ADDR.
#      It is important to understand the context in which this docker container is running and this determines the
#      value of PROXY_ADDR. PROXY_ADDR is the container's internal IP address. See flow of traffic below:

# [ External Client (public internet) ]
#                  |
#                  v
#        [ Public IP of cloud provider ]
#                  |
#          (NAT to private IP)
#                  |
#                  v
#      [ VM Private IP (e.g., 10.52.x.x) ]
#                  |
#         (Host port → Container port binding)
#                  |
#          (Docker NAT to container IP)
#                  |
#                  v
#   [ Docker Container IP (e.g., 172.17.x.x) ]

#  3. Add a rule to the PREROUTING chain that matches packets destined for the proxy address (PROXY_ADDR)
#     and redirects them to the LANTERN_SERVERMASQ chain.
#  4. Add a rule to the POSTROUTING chain that matches packets destined for the MASQ_ADDR and masquerades them in order for responses to be sent back correctly to container.

iptables -t nat -N LANTERN_SERVERMASQ 2>/dev/null || true
iptables -t nat -F LANTERN_SERVERMASQ 2>/dev/null || true

iptables -t nat -A LANTERN_SERVERMASQ -d "$PROXY_ADDR" -p tcp ! --dport "$PROXY_PORT" -j DNAT --to-destination "$MASQ_ADDR"
iptables -t nat -A PREROUTING -d "$PROXY_ADDR" -j LANTERN_SERVERMASQ
iptables -t nat -A POSTROUTING -d "$MASQ_ADDR" -j MASQUERADE

echo "[+] LANTERN_SERVERMASQ setup complete: $@"
exec su-exec lantern "$@"