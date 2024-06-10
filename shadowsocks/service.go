package shadowsocks

import (
	"net"
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	onet "github.com/Jigsaw-Code/outline-ss-server/net"
	"github.com/Jigsaw-Code/outline-ss-server/service"
)

type llistener struct {
	wrapped      net.Listener
	connections  chan net.Conn
	closedSignal chan struct{}
	closeOnce    sync.Once
	closeError   error
}

type SSMetrics interface {
	service.TCPMetrics
	service.ShadowsocksTCPMetrics
}

type TCPConn interface {
	transport.StreamConn
	CloseRead() error
	CloseWrite() error
	SetKeepAlive(keepAlive bool) error
}

type TCPListener interface {
	net.Listener
	AcceptTCP() (TCPConn, error)
}

type ListenerOptions struct {
	Listener              TCPListener
	Ciphers               service.CipherList
	ReplayCache           *service.ReplayCache
	Timeout               time.Duration
	TargetIPValidator     onet.TargetIPValidator // determines validity of non-local upstream dials
	MaxPendingConnections int                    // defaults to 1000
	ShadowsocksMetrics    SSMetrics
}
