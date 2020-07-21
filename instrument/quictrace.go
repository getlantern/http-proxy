package instrument

import (
	"github.com/lucas-clemente/quic-go/quictrace"
)

// QuicTracer is a quictrace.Tracker implementation which counts the sent and
// lost packets and exports the data to Prometheus.
type QuicTracer struct {
	inst Instrument
}

func NewQuicTracer(inst Instrument) *QuicTracer {
	tracer := &QuicTracer{inst: inst}
	return tracer
}

func (t *QuicTracer) Trace(connID quictrace.ConnectionID, event quictrace.Event) {
	// Trace is called for each packet handled but some bug causes connID to be
	// always empty, so we can not track per connection packet loss rate.
	switch event.EventType {
	case quictrace.PacketSent:
		t.inst.quicSentPacket()
	case quictrace.PacketLost:
		t.inst.quicLostPacket()
	}
}

func (t *QuicTracer) GetAllTraces() map[string][]byte {
	panic("not implemented")
}
