package starbridge

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	replicant "github.com/OperatorFoundation/Replicant-go/Replicant/v3"
	"github.com/OperatorFoundation/Replicant-go/Replicant/v3/polish"
	"github.com/OperatorFoundation/Replicant-go/Replicant/v3/toneburst"
	"github.com/OperatorFoundation/Starbridge-go/Starbridge/v3"
)

// The starbridge client and server authenticate (amongst other things) the server address during
// the handshake. The problem with this is that our proxies do not necessarily listen on their
// public IP address (particularly in the triangle routing case). To get around this, we use the
// same fake server address on both client and server. To be clear, nothing binds on or dials this
// address. It is used exclusively for coordinating client and server authentication.
//
// We are not concerned about degradation of security here. The keypair exchanged out-of-band should
// be sufficient to authenticate the proxy. The address authentication scheme provides no security
// anyway; a bad actor could set the authentication address to any value they want, just as we do
// here.
//
// As an entry point to this logic, see:
// https://github.com/OperatorFoundation/go-shadowsocks2/blob/v1.1.12/darkstar/client.go#L189
const fakeListenAddr = "1.2.3.4:5678"

func Wrap(l net.Listener, privateKey string) (net.Listener, error) {
	// Key checks taken from:
	// https://github.com/OperatorFoundation/Starbridge-go/blob/v3.0.12/Starbridge/v3/starbridge.go#L69-L77

	key, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("key decode error: %w", err)
	}

	if !Starbridge.CheckPrivateKey(key) {
		return nil, errors.New("bad key")
	}

	return newStarbridgeListener(l, key), nil
}

// Port of Starbridge.starbridgeTransportListener:
// https://github.com/OperatorFoundation/Starbridge-go/blob/v3.0.12/Starbridge/v3/starbridge.go#L51
type starbridgeListener struct {
	net.Listener
	cfg replicant.ServerConfig
}

func newStarbridgeListener(transport net.Listener, privateKey []byte) starbridgeListener {
	return starbridgeListener{
		Listener: transport,
		cfg:      getServerConfig(privateKey),
	}
}

func (l starbridgeListener) Accept() (net.Conn, error) {
	transportConn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return &delayedHandshakeConn{
		wrapped: transportConn,
		doHandshake: func(transport net.Conn) (net.Conn, error) {
			return Starbridge.NewServerConnection(l.cfg, transportConn)
		},
	}, nil
}

// Adapted from https://github.com/OperatorFoundation/Starbridge-go/blob/v3.0.12/Starbridge/v3/starbridge.go#L255-L271
func getServerConfig(serverPrivateKey []byte) replicant.ServerConfig {
	polishServerConfig := polish.DarkStarPolishServerConfig{
		ServerAddress:    fakeListenAddr,
		ServerPrivateKey: base64.StdEncoding.EncodeToString(serverPrivateKey),
	}

	toneburstServerConfig := toneburst.StarburstConfig{
		Mode: "SMTPServer",
	}

	serverConfig := replicant.ServerConfig{
		Toneburst: toneburstServerConfig,
		Polish:    polishServerConfig,
	}

	return serverConfig
}

// Starbridge.NewServerConnection conducts the handshake right away (as opposed to on the first call
// to Read or Write). This is a problem for us as it means handshake errors look like Accept errors.
// Accept errors are generally treated as significant and, in our case, lead to fatal shutdown of
// proxy (as an error is returned by ListenAndServe).
//
// delayedHandshakeConn alleviates this issue by moving the connection initialization (and thus the
// handshake) to the first Read or Write operation.
type delayedHandshakeConn struct {
	// Before the handshake, this is the underlying transport connection. After the handshake, this
	// is the starbridge connection.
	wrapped net.Conn

	doHandshake   func(transport net.Conn) (net.Conn, error)
	handshakeErr  error
	handshakeDone bool

	sync.Mutex
}

// Implements netx.WrappedConn.
func (conn *delayedHandshakeConn) Wrapped() net.Conn {
	conn.Lock()
	defer conn.Unlock()
	return conn.wrapped
}

// Safe to call concurrently.
func (conn *delayedHandshakeConn) handshake() error {
	conn.Lock()
	defer conn.Unlock()

	if conn.handshakeDone {
		return conn.handshakeErr
	}

	newConn, err := conn.doHandshake(conn.wrapped)
	if err == nil {
		conn.wrapped = newConn
	} else {
		conn.handshakeErr = err
	}
	conn.handshakeDone = true

	return conn.handshakeErr
}

func (conn *delayedHandshakeConn) Read(b []byte) (n int, err error) {
	if err := conn.handshake(); err != nil {
		return 0, fmt.Errorf("handshake error: %w", err)
	}
	return conn.wrapped.Read(b)
}

func (conn *delayedHandshakeConn) Write(b []byte) (n int, err error) {
	if err := conn.handshake(); err != nil {
		return 0, fmt.Errorf("handshake error: %w", err)
	}
	return conn.wrapped.Write(b)
}

func (conn *delayedHandshakeConn) Close() error {
	conn.Lock()
	defer conn.Unlock()

	return conn.wrapped.Close()
}

func (conn *delayedHandshakeConn) LocalAddr() net.Addr {
	conn.Lock()
	defer conn.Unlock()

	return conn.wrapped.LocalAddr()
}

func (conn *delayedHandshakeConn) RemoteAddr() net.Addr {
	conn.Lock()
	defer conn.Unlock()

	return conn.wrapped.RemoteAddr()
}

func (conn *delayedHandshakeConn) SetDeadline(t time.Time) error {
	conn.Lock()
	defer conn.Unlock()

	return conn.wrapped.SetDeadline(t)
}

func (conn *delayedHandshakeConn) SetReadDeadline(t time.Time) error {
	conn.Lock()
	defer conn.Unlock()

	return conn.wrapped.SetReadDeadline(t)
}

func (conn *delayedHandshakeConn) SetWriteDeadline(t time.Time) error {
	conn.Lock()
	defer conn.Unlock()

	return conn.wrapped.SetWriteDeadline(t)
}
