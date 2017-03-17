package bbr

import (
	"math"
	"sync"
	"time"

	"github.com/getlantern/ema"
	"github.com/gonum/stat"
)

const (
	limit             = 25
	minSamples        = 5
	abeThreshold      = 2500000
	minBytesThreshold = 2048
)

type stats struct {
	sent    []float64
	abe     []float64
	weights []float64
	times   []time.Time
	emaABE  *ema.EMA
	size    int
	idx     int
	mx      sync.RWMutex
}

func newStats() *stats {
	return &stats{
		sent:    make([]float64, limit),
		abe:     make([]float64, limit),
		weights: make([]float64, limit),
		times:   make([]time.Time, limit),
		emaABE:  ema.New(0, 0.1),
	}
}

func (s *stats) update(sent float64, abe float64) {
	if sent < minBytesThreshold {
		// Don't bother recording values with too little data on connection
		return
	}

	now := time.Now()
	logOfSent := math.Log1p(sent)
	logOfABE := math.Log1p(abe)
	s.mx.Lock()
	s.sent[s.idx] = logOfSent
	s.abe[s.idx] = logOfABE
	s.weights[s.idx] = sent // give more weight to measurements from larger bytes sent
	s.times[s.idx] = now
	s.idx++
	if s.idx == limit {
		// Wrap
		s.idx = 0
	}
	if s.size < limit {
		s.size++
	}
	hasDelta := false
	for i := 1; i < s.size; i++ {
		if s.sent[i] != s.sent[i-1] {
			hasDelta = true
			break
		}
	}
	if !hasDelta {
		// There's no way to apply a regression, just ignore
		s.mx.Unlock()
		return
	}

	weights := make([]float64, 0, s.size)
	for i := 0; i < s.size; i++ {
		// Give more weight to more recent values
		age := now.Sub(s.times[i]).Seconds() + 1
		weights = append(weights, s.weights[i]/age)
	}

	// Estimate by applying a linear regression
	alpha, beta := stat.LinearRegression(s.sent[:s.size], s.abe[:s.size], weights, false)
	s.mx.Unlock()
	newEstimate := math.Expm1(alpha + beta*math.Log1p(abeThreshold))
	if math.IsNaN(newEstimate) {
		// ignore
		return
	}
	if newEstimate < 0 {
		newEstimate = 0
	}
	updated := s.emaABE.Update(newEstimate)
	log.Debugf("%.0f at %.2f -> %.2f", sent, abe, updated)
	if updated <= 0 {
		log.Debugf("Calculated negative EBA of %.2f?!, setting to small value instead")
		// Set estimate to a small value to show that we have something
		s.emaABE.Set(0.01)
	}
}

func (s *stats) clear() {
	s.mx.Lock()
	s.emaABE.Clear()
	s.size = 0
	s.idx = 0
	s.mx.Unlock()
}

// estABE estimates the ABE at bytes_sent = 2.5 MB using a logarithmic
// regression on the most recent measurements
func (s *stats) estABE() float64 {
	s.mx.RLock()
	enoughData := s.size >= minSamples
	s.mx.RUnlock()
	if !enoughData {
		return 0
	}
	return s.emaABE.Get()
}
