package shadowsocks

import (
	"net"
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	onet "github.com/Jigsaw-Code/outline-ss-server/net"
	"github.com/Jigsaw-Code/outline-ss-server/service"
)

// TCPService is a Shadowsocks TCP service that can be started and stopped.
type TCPService interface {

	// SetTargetIPValidator sets the function to be used to validate the target IP addresses.
	SetTargetIPValidator(targetIPValidator onet.TargetIPValidator)
	// Serve adopts the listener, which will be closed before Serve returns.  Serve returns an error unless Stop() was called.
	Serve(listener net.TCPListener) error
	// Stop closes the listener but does not interfere with existing connections.
	Stop() error
	// GracefulStop calls Stop(), and then blocks until all resources have been cleaned up.
	GracefulStop() error
}

type tcpService struct {
	mu          sync.RWMutex // Protects .listeners and .stopped
	listener    *net.TCPListener
	stopped     bool
	ciphers     service.CipherList
	m           ShadowsocksMetrics
	running     sync.WaitGroup
	readTimeout time.Duration
	// `replayCache` is a pointer to SSServer.replayCache, to share the cache among all ports.
	replayCache       *service.ReplayCache
	targetIPValidator onet.TargetIPValidator
}

type llistener struct {
	wrapped      net.Listener
	connections  chan net.Conn
	closedSignal chan struct{}
	closeOnce    sync.Once
	closeError   error
	TCPHandler   service.TCPHandler
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
	ShouldHandleLocally   HandleLocalPredicate   // determines whether an upstream should be handled by the listener locally or dial upstream
	TargetIPValidator     onet.TargetIPValidator // determines validity of non-local upstream dials
	MaxPendingConnections int                    // defaults to 1000
	ShadowsocksMetrics    SSMetrics
	Accept                func(conn transport.StreamConn) error
}

type TCPServiceOptions struct {
	TargetIPValidator onet.TargetIPValidator
}
