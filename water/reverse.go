package water

import (
	"context"
	"net"
	"sync"

	"github.com/getlantern/golog"
	"github.com/refraction-networking/water"
	_ "github.com/refraction-networking/water/transport/v0"
)

var log = golog.LoggerFor("water")

type llistener struct {
	net.Listener

	connections  chan net.Conn
	closedSignal chan struct{}
	closeOnce    sync.Once
	closeError   error
}

func NewReverseListener(ctx context.Context, address string, wasm []byte) (*llistener, error) {
	cfg := &water.Config{
		TransportModuleBin: wasm,
	}

	waterListener, err := cfg.ListenContext(ctx, "tcp", address)
	if err != nil {
		log.Errorf("error creating water listener: %v", err)
		return nil, err
	}

	l := &innerListener{Listener: waterListener}
	ll := &llistener{
		Listener:     l,
		connections:  make(chan net.Conn),
		closedSignal: make(chan struct{}),
	}

	go func() {
		for {
			conn, err := l.Listener.Accept()
			if err != nil {
				log.Errorf("failed accepting connection: %v", err)
				return
			}
			select {
			case ll.connections <- conn:
			case <-ll.closedSignal:
				ll.Close()
			}
		}
	}()
	return ll, nil
}

func (l *llistener) Accept() (net.Conn, error) {
	select {
	case c := <-l.connections:
		return c, nil
	case <-l.closedSignal:
		return nil, l.closeError
	}
}

func (l *llistener) Close() error {
	l.closeOnce.Do(func() {
		close(l.closedSignal)
		l.closeError = l.Close()
	})
	return l.closeError
}

type innerListener struct {
	net.Listener
}

func (il *innerListener) Accept() (net.Conn, error) {
	c, err := il.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return c, nil
}
