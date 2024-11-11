package v2ray

import (
	"context"
	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/infra/conf/serial"
	"io"
	"log"
	"net"
	"strings"

	// The following are necessary as they register handlers in their init functions.

	// Mandatory features. Can't remove unless there are replacements.
	_ "github.com/v2fly/v2ray-core/v5/app/dispatcher"
	_ "github.com/v2fly/v2ray-core/v5/app/proxyman/inbound"
	_ "github.com/v2fly/v2ray-core/v5/app/proxyman/outbound"

	// Other optional features.
	_ "github.com/v2fly/v2ray-core/v5/app/dns"
	_ "github.com/v2fly/v2ray-core/v5/app/dns/fakedns"
	_ "github.com/v2fly/v2ray-core/v5/app/log"
	_ "github.com/v2fly/v2ray-core/v5/app/policy"
	_ "github.com/v2fly/v2ray-core/v5/app/reverse"
	_ "github.com/v2fly/v2ray-core/v5/app/router"
	_ "github.com/v2fly/v2ray-core/v5/app/stats"

	// Fix dependency cycle caused by core import in internet package
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/tagged/taggedimpl"

	// Developer preview features
	_ "github.com/v2fly/v2ray-core/v5/app/observatory"

	// Inbound and outbound proxies.
	_ "github.com/v2fly/v2ray-core/v5/proxy/blackhole"
	_ "github.com/v2fly/v2ray-core/v5/proxy/dns"
	_ "github.com/v2fly/v2ray-core/v5/proxy/dokodemo"
	_ "github.com/v2fly/v2ray-core/v5/proxy/freedom"
	_ "github.com/v2fly/v2ray-core/v5/proxy/http"
	_ "github.com/v2fly/v2ray-core/v5/proxy/shadowsocks"
	_ "github.com/v2fly/v2ray-core/v5/proxy/socks"
	_ "github.com/v2fly/v2ray-core/v5/proxy/trojan"
	_ "github.com/v2fly/v2ray-core/v5/proxy/vless/inbound"
	_ "github.com/v2fly/v2ray-core/v5/proxy/vless/outbound"
	_ "github.com/v2fly/v2ray-core/v5/proxy/vmess/inbound"
	_ "github.com/v2fly/v2ray-core/v5/proxy/vmess/outbound"

	// Developer preview proxies
	_ "github.com/v2fly/v2ray-core/v5/proxy/vlite/inbound"
	_ "github.com/v2fly/v2ray-core/v5/proxy/vlite/outbound"

	_ "github.com/v2fly/v2ray-core/v5/proxy/hysteria2"
	_ "github.com/v2fly/v2ray-core/v5/proxy/shadowsocks2022"

	// Transports
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/domainsocket"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/grpc"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/http"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/kcp"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/quic"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/tcp"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/tls"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/tls/utls"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/udp"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/websocket"

	// Developer preview transports
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/request/assembly"

	_ "github.com/v2fly/v2ray-core/v5/transport/internet/request/assembler/simple"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/request/roundtripper/httprt"

	_ "github.com/v2fly/v2ray-core/v5/transport/internet/request/assembler/packetconn"

	_ "github.com/v2fly/v2ray-core/v5/transport/internet/request/stereotype/meek"

	_ "github.com/v2fly/v2ray-core/v5/transport/internet/dtls"

	_ "github.com/v2fly/v2ray-core/v5/transport/internet/httpupgrade"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/hysteria2"

	// Transport headers
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/http"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/noop"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/srtp"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/tls"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/utp"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/wechat"
	_ "github.com/v2fly/v2ray-core/v5/transport/internet/headers/wireguard"

	// Geo loaders
	_ "github.com/v2fly/v2ray-core/v5/infra/conf/geodata/memconservative"
	_ "github.com/v2fly/v2ray-core/v5/infra/conf/geodata/standard"

	// JSON, TOML, YAML config support. (jsonv4) This disable selective compile
	_ "github.com/v2fly/v2ray-core/v5/main/formats"
)

type TunnelListener struct {
	listener   net.Listener
	targetAddr string
}

func NewTunnelListener(listenAddr, targetAddr string) (*TunnelListener, error) {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}
	return &TunnelListener{listener: listener, targetAddr: targetAddr}, nil
}

func (t *TunnelListener) Accept() (net.Conn, error) {
	conn, err := t.listener.Accept()
	if err != nil {
		return nil, err
	}

	go t.handleConnection(conn)
	return conn, nil
}

func (t *TunnelListener) handleConnection(srcConn net.Conn) {
	defer srcConn.Close()

	// Connect to the target application
	dstConn, err := net.Dial("tcp", t.targetAddr)
	if err != nil {
		log.Println("Failed to connect to target:", err)
		return
	}
	defer dstConn.Close()

	// Start forwarding data between the source and destination connections
	go io.Copy(dstConn, srcConn)
	io.Copy(srcConn, dstConn)
}

func (t *TunnelListener) Close() error {
	return t.listener.Close()
}

func (t *TunnelListener) Addr() net.Addr {
	return t.listener.Addr()
}

const tmp = `
{
  "log": {
    "loglevel": "debug"
  },
  "outbounds": [
    {
      "protocol": "freedom",
      "settings": {}
    }
  ],

  "inbounds": [
    {
      "listen": "127.0.0.1",
      "port": 11082,
      "protocol": "socks",
      "settings": {
        "followRedirect": true,
        "network": "tcp,udp"
      },
      "sniffing": {
        "destOverride": ["http", "tls"],
        "enabled": true
      }
    }
  ]
}

`

// NewV2RayListener creates a V2Ray listener
func NewV2RayListener(ctx context.Context, address, cfgJSON string) (net.Listener, error) {
	config, err := serial.LoadJSONConfig(strings.NewReader(tmp))
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
	return NewTunnelListener(address, "0.0.0.0:11082")
}
