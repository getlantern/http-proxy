package instrument

import (
	"time"

	"github.com/lucas-clemente/quic-go/quictrace"
)

const reportPeriod = time.Minute

// QuicTracer is a quictrace.Tracker implementation which counts the sent and
// lost packets and exports the data to Prometheus.
type QuicTracer struct {
	connStats map[string]*connStat
	inst      Instrument
	ch        chan idAndEvent
}

type connStat struct {
	sentPackets int
	lostPackets int
	lastActive  time.Time
}

type idAndEvent struct {
	connID string
	event  *quictrace.Event
}

func NewQuicTracer(inst Instrument) *QuicTracer {
	tracer := &QuicTracer{
		connStats: make(map[string]*connStat),
		ch:        make(chan idAndEvent, 100),
	}
	go tracer.run()
	return tracer
}

func (t *QuicTracer) Trace(connID quictrace.ConnectionID, event quictrace.Event) {
	t.ch <- idAndEvent{string(connID), &event}
}

func (t *QuicTracer) run() {
	tk := time.NewTicker(reportPeriod)
	for {
		select {
		case idAndEvent := <-t.ch:
			stats, exists := t.connStats[idAndEvent.connID]
			if !exists {
				stats = &connStat{}
			}
			stats.lastActive = time.Now()
			switch idAndEvent.event.EventType {
			case quictrace.PacketSent:
				stats.sentPackets++
			case quictrace.PacketLost:
				stats.lostPackets++
			}
		case now := <-tk.C:
			cutoff := now.Add(-reportPeriod)
			for connID, stats := range t.connStats {
				if stats.lastActive.Before(cutoff) {
					t.inst.quicPackets(stats.sentPackets, stats.lostPackets)
					delete(t.connStats, connID)
				}
			}
		}
	}
}

func (t *QuicTracer) GetAllTraces() map[string][]byte {
	panic("not implemented")
}
