package v2ray

import (
	"context"
	"fmt"
	"github.com/v2fly/v2ray-core/v5/features/inbound"
	"net"
	"strings"

	core "github.com/v2fly/v2ray-core/v5"
	_ "github.com/v2fly/v2ray-core/v5/app/dispatcher"
	_ "github.com/v2fly/v2ray-core/v5/app/dns"
	_ "github.com/v2fly/v2ray-core/v5/app/dns/fakedns"
	_ "github.com/v2fly/v2ray-core/v5/app/observatory"
	_ "github.com/v2fly/v2ray-core/v5/app/policy"
	_ "github.com/v2fly/v2ray-core/v5/app/proxyman/inbound"
	_ "github.com/v2fly/v2ray-core/v5/app/proxyman/outbound"
	_ "github.com/v2fly/v2ray-core/v5/app/reverse"
	_ "github.com/v2fly/v2ray-core/v5/app/router"
	_ "github.com/v2fly/v2ray-core/v5/app/stats"
	_ "github.com/v2fly/v2ray-core/v5/infra/conf/geodata/memconservative"
	_ "github.com/v2fly/v2ray-core/v5/infra/conf/geodata/standard"
	"github.com/v2fly/v2ray-core/v5/infra/conf/serial"
	_ "github.com/v2fly/v2ray-core/v5/main/formats"
	_ "github.com/v2fly/v2ray-core/v5/proxy/blackhole"
	_ "github.com/v2fly/v2ray-core/v5/proxy/dns"
	_ "github.com/v2fly/v2ray-core/v5/proxy/dokodemo"
	_ "github.com/v2fly/v2ray-core/v5/proxy/freedom"
	_ "github.com/v2fly/v2ray-core/v5/proxy/http"
	_ "github.com/v2fly/v2ray-core/v5/proxy/hysteria2"
	_ "github.com/v2fly/v2ray-core/v5/proxy/shadowsocks"
	_ "github.com/v2fly/v2ray-core/v5/proxy/shadowsocks2022"
	_ "github.com/v2fly/v2ray-core/v5/proxy/socks"
	_ "github.com/v2fly/v2ray-core/v5/proxy/trojan"
	_ "github.com/v2fly/v2ray-core/v5/proxy/vless/inbound"
	_ "github.com/v2fly/v2ray-core/v5/proxy/vless/outbound"
	_ "github.com/v2fly/v2ray-core/v5/proxy/vlite/inbound"
	_ "github.com/v2fly/v2ray-core/v5/proxy/vlite/outbound"
	_ "github.com/v2fly/v2ray-core/v5/proxy/vmess/inbound"
	_ "github.com/v2fly/v2ray-core/v5/proxy/vmess/outbound"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/domainsocket"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/dtls"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/grpc"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/http"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/noop"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/srtp"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/tls"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/utp"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/wechat"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/wireguard"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/http"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/httpupgrade"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/hysteria2"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/kcp"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/quic"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/request/assembler/packetconn"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/request/assembler/simple"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/request/assembly"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/request/roundtripper/httprt"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/request/stereotype/meek"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/tagged/taggedimpl"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/tcp"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/tls"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/tls/utls"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/udp"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/websocket"
)

// NewV2RayListener creates a V2Ray listener
// the cfgJSON is a JSON string that contains the configuration of the listener in v2ray format
// it should have one inbound and one outbound protocol
// the inbound protocol should have a tag "ingress"
// the listening address and port should be "LISTEN_ADDRESS" and "LISTEN_PORT" respectively and will be replaced
// by the actual address and port
func NewV2RayListener(ctx context.Context, address, cfgJSON string) (net.Listener, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	configStr := strings.Replace(strings.Replace(cfgJSON, "LISTEN_ADDRESS", host, 1), "LISTEN_PORT", port, 1)

	config, err := serial.LoadJSONConfig(strings.NewReader(configStr))
	if err != nil {
		return nil, err
	}

	server, err := core.New(config)
	if err != nil {
		return nil, err
	}
	if err = server.Start(); err != nil {
		return nil, err
	}

	inboundManager := server.GetFeature(inbound.ManagerType()).(inbound.Manager)
	h, err := inboundManager.GetHandler(ctx, "ingress")
	if err != nil {
		return nil, fmt.Errorf("failed to get handler with tag ingress: %w", err)
	}
	if len(h.GetWorkers()) == 0 {
		return nil, fmt.Errorf("no worker found in handler with tag ingress")
	}

	return h.GetWorkers()[0].Hub().GetNetListener(), nil
}
